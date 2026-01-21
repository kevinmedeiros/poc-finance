package services

import (
	"errors"
	"fmt"
	"time"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
)

var (
	ErrBudgetNotFound     = errors.New("orçamento não encontrado")
	ErrCategoryNotFound   = errors.New("categoria não encontrada")
	ErrInvalidBudgetMonth = errors.New("mês inválido (deve estar entre 1-12)")
	ErrInvalidBudgetYear  = errors.New("ano inválido")
)

type BudgetService struct {
	groupService        *GroupService
	notificationService *NotificationService
}

func NewBudgetService() *BudgetService {
	return &BudgetService{
		groupService:        NewGroupService(),
		notificationService: NewNotificationService(),
	}
}

// CreateBudget creates a new budget for a user or group
func (s *BudgetService) CreateBudget(userID uint, groupID *uint, year, month int, name string, categories []struct {
	Category string
	Limit    float64
}) (*models.Budget, error) {
	// Validate month and year
	if month < 1 || month > 12 {
		return nil, ErrInvalidBudgetMonth
	}
	if year < 1900 || year > 3000 {
		return nil, ErrInvalidBudgetYear
	}

	// If group budget, verify user is group member
	if groupID != nil {
		if !s.groupService.IsGroupMember(*groupID, userID) {
			return nil, ErrUnauthorized
		}
	}

	// Create budget
	budget := &models.Budget{
		UserID: userID,
		GroupID: groupID,
		Year:   year,
		Month:  month,
		Name:   name,
		Status: models.BudgetStatusActive,
	}

	if err := database.DB.Create(budget).Error; err != nil {
		return nil, err
	}

	// Create categories
	for _, cat := range categories {
		category := &models.BudgetCategory{
			BudgetID: budget.ID,
			Category: cat.Category,
			Limit:    cat.Limit,
			Spent:    0,
		}
		if err := database.DB.Create(category).Error; err != nil {
			return nil, err
		}
	}

	// Reload with associations
	database.DB.Preload("User").Preload("Group").Preload("Categories").First(budget, budget.ID)
	return budget, nil
}

// GetUserBudgets returns all budgets for a user (personal budgets only)
func (s *BudgetService) GetUserBudgets(userID uint) ([]models.Budget, error) {
	var budgets []models.Budget
	database.DB.Where("user_id = ? AND group_id IS NULL", userID).
		Preload("Categories").
		Order("year DESC, month DESC").
		Find(&budgets)

	return budgets, nil
}

// GetGroupBudgets returns all budgets for a group
func (s *BudgetService) GetGroupBudgets(groupID, userID uint) ([]models.Budget, error) {
	// Verify user is group member
	if !s.groupService.IsGroupMember(groupID, userID) {
		return nil, ErrUnauthorized
	}

	var budgets []models.Budget
	database.DB.Where("group_id = ?", groupID).
		Preload("User").
		Preload("Categories").
		Order("year DESC, month DESC").
		Find(&budgets)

	return budgets, nil
}

// GetBudgetByID retrieves a budget by ID with authorization check
func (s *BudgetService) GetBudgetByID(budgetID, userID uint) (*models.Budget, error) {
	var budget models.Budget
	if err := database.DB.Preload("User").Preload("Group").Preload("Categories").
		First(&budget, budgetID).Error; err != nil {
		return nil, ErrBudgetNotFound
	}

	// Check authorization
	if !s.canAccessBudget(&budget, userID) {
		return nil, ErrUnauthorized
	}

	return &budget, nil
}

// GetActiveBudget retrieves the active budget for a specific month and year
func (s *BudgetService) GetActiveBudget(userID uint, groupID *uint, year, month int) (*models.Budget, error) {
	var budget models.Budget
	query := database.DB.Where("year = ? AND month = ? AND status = ?", year, month, models.BudgetStatusActive)

	if groupID != nil {
		// Verify user is group member
		if !s.groupService.IsGroupMember(*groupID, userID) {
			return nil, ErrUnauthorized
		}
		query = query.Where("group_id = ?", *groupID)
	} else {
		query = query.Where("user_id = ? AND group_id IS NULL", userID)
	}

	if err := query.Preload("User").Preload("Group").Preload("Categories").First(&budget).Error; err != nil {
		return nil, ErrBudgetNotFound
	}

	return &budget, nil
}

// UpdateBudget updates budget details
func (s *BudgetService) UpdateBudget(budgetID, userID uint, name string, year, month int) error {
	budget, err := s.GetBudgetByID(budgetID, userID)
	if err != nil {
		return err
	}

	// Only budget creator or group admin can update
	if !s.canModifyBudget(budget, userID) {
		return ErrUnauthorized
	}

	// Validate month and year
	if month < 1 || month > 12 {
		return ErrInvalidBudgetMonth
	}
	if year < 1900 || year > 3000 {
		return ErrInvalidBudgetYear
	}

	return database.DB.Model(budget).Updates(map[string]interface{}{
		"name":  name,
		"year":  year,
		"month": month,
	}).Error
}

// DeleteBudget deletes a budget
func (s *BudgetService) DeleteBudget(budgetID, userID uint) error {
	budget, err := s.GetBudgetByID(budgetID, userID)
	if err != nil {
		return err
	}

	// Only budget creator or group admin can delete
	if !s.canModifyBudget(budget, userID) {
		return ErrUnauthorized
	}

	// Delete categories first
	database.DB.Where("budget_id = ?", budgetID).Delete(&models.BudgetCategory{})

	return database.DB.Delete(budget).Error
}

// ArchiveBudget archives a budget
func (s *BudgetService) ArchiveBudget(budgetID, userID uint) error {
	budget, err := s.GetBudgetByID(budgetID, userID)
	if err != nil {
		return err
	}

	// Only budget creator or group admin can archive
	if !s.canModifyBudget(budget, userID) {
		return ErrUnauthorized
	}

	return database.DB.Model(budget).Update("status", models.BudgetStatusArchived).Error
}

// AddCategory adds a new category to a budget
func (s *BudgetService) AddCategory(budgetID, userID uint, categoryName string, limit float64) (*models.BudgetCategory, error) {
	budget, err := s.GetBudgetByID(budgetID, userID)
	if err != nil {
		return nil, err
	}

	// Only budget creator or group admin can modify
	if !s.canModifyBudget(budget, userID) {
		return nil, ErrUnauthorized
	}

	category := &models.BudgetCategory{
		BudgetID: budgetID,
		Category: categoryName,
		Limit:    limit,
		Spent:    0,
	}

	if err := database.DB.Create(category).Error; err != nil {
		return nil, err
	}

	return category, nil
}

// UpdateCategory updates a budget category
func (s *BudgetService) UpdateCategory(categoryID, userID uint, categoryName string, limit float64) error {
	var category models.BudgetCategory
	if err := database.DB.First(&category, categoryID).Error; err != nil {
		return ErrCategoryNotFound
	}

	// Get budget to check authorization
	budget, err := s.GetBudgetByID(category.BudgetID, userID)
	if err != nil {
		return err
	}

	// Only budget creator or group admin can modify
	if !s.canModifyBudget(budget, userID) {
		return ErrUnauthorized
	}

	return database.DB.Model(&category).Updates(map[string]interface{}{
		"category": categoryName,
		"limit":    limit,
	}).Error
}

// DeleteCategory deletes a budget category
func (s *BudgetService) DeleteCategory(categoryID, userID uint) error {
	var category models.BudgetCategory
	if err := database.DB.First(&category, categoryID).Error; err != nil {
		return ErrCategoryNotFound
	}

	// Get budget to check authorization
	budget, err := s.GetBudgetByID(category.BudgetID, userID)
	if err != nil {
		return err
	}

	// Only budget creator or group admin can modify
	if !s.canModifyBudget(budget, userID) {
		return ErrUnauthorized
	}

	return database.DB.Delete(&category).Error
}

// UpdateCategorySpending updates the spent amount for a category and checks for notifications
func (s *BudgetService) UpdateCategorySpending(budgetID uint, categoryName string, spent float64) error {
	var category models.BudgetCategory
	if err := database.DB.Where("budget_id = ? AND category = ?", budgetID, categoryName).First(&category).Error; err != nil {
		return ErrCategoryNotFound
	}

	category.Spent = spent

	// Check if we need to send notifications
	shouldNotify80 := category.ShouldNotifyAt80()
	shouldNotify100 := category.ShouldNotifyAt100()

	if shouldNotify80 {
		now := time.Now()
		category.NotifiedAt80 = &now
		// Send 80% notification
		s.sendCategoryNotification(&category, 80)
	}

	if shouldNotify100 {
		now := time.Now()
		category.NotifiedAt100 = &now
		// Send 100% notification
		s.sendCategoryNotification(&category, 100)
	}

	return database.DB.Save(&category).Error
}

// CopyFromPreviousMonth copies a budget from the previous month
func (s *BudgetService) CopyFromPreviousMonth(userID uint, groupID *uint, targetYear, targetMonth int) (*models.Budget, error) {
	// Calculate previous month
	prevYear := targetYear
	prevMonth := targetMonth - 1
	if prevMonth < 1 {
		prevMonth = 12
		prevYear--
	}

	// Get previous month's budget
	prevBudget, err := s.GetActiveBudget(userID, groupID, prevYear, prevMonth)
	if err != nil {
		return nil, err
	}

	// Create categories array
	var categories []struct {
		Category string
		Limit    float64
	}
	for _, cat := range prevBudget.Categories {
		categories = append(categories, struct {
			Category string
			Limit    float64
		}{
			Category: cat.Category,
			Limit:    cat.Limit,
		})
	}

	// Create new budget
	return s.CreateBudget(userID, groupID, targetYear, targetMonth, prevBudget.Name, categories)
}

// Helper: canAccessBudget checks if user can view a budget
func (s *BudgetService) canAccessBudget(budget *models.Budget, userID uint) bool {
	// Personal budget - only owner can access
	if budget.GroupID == nil {
		return budget.UserID == userID
	}

	// Group budget - any group member can access
	return s.groupService.IsGroupMember(*budget.GroupID, userID)
}

// Helper: canModifyBudget checks if user can modify a budget
func (s *BudgetService) canModifyBudget(budget *models.Budget, userID uint) bool {
	// Personal budget - only owner can modify
	if budget.GroupID == nil {
		return budget.UserID == userID
	}

	// Group budget - creator or group admin can modify
	return budget.UserID == userID || s.groupService.IsGroupAdmin(*budget.GroupID, userID)
}

// Helper: sendCategoryNotification sends a notification when category reaches threshold
func (s *BudgetService) sendCategoryNotification(category *models.BudgetCategory, percentage int) {
	// Load budget with associations
	var budget models.Budget
	if err := database.DB.Preload("User").Preload("Group").First(&budget, category.BudgetID).Error; err != nil {
		return
	}

	var members []models.User
	if budget.GroupID != nil {
		// Group budget - notify all members
		groupMembers, err := s.groupService.GetGroupMembers(*budget.GroupID)
		if err == nil {
			members = groupMembers
		}
	} else {
		// Personal budget - notify only the user
		members = []models.User{budget.User}
	}

	// Create notification for each member
	for _, member := range members {
		var title, message string
		var notifType models.NotificationType

		if percentage == 80 {
			title = "Alerta de orçamento"
			message = formatBudgetMessage(&budget, category, "próximo do limite", percentage)
			notifType = models.NotificationTypeBudgetAlert
		} else {
			title = "Orçamento atingido"
			message = formatBudgetMessage(&budget, category, "atingiu o limite", percentage)
			notifType = models.NotificationTypeBudgetAlert
		}

		notification := &models.Notification{
			UserID:  member.ID,
			Type:    notifType,
			Title:   title,
			Message: message,
			Link:    "/budgets",
			GroupID: budget.GroupID,
		}
		s.notificationService.Create(notification)
	}
}

// Helper: formatBudgetMessage formats the notification message
func formatBudgetMessage(budget *models.Budget, category *models.BudgetCategory, status string, percentage int) string {
	if budget.GroupID != nil {
		return formatMessage("A categoria \"%s\" do orçamento \"%s\" do grupo %s (R$ %.2f / R$ %.2f - %d%%)",
			category.Category, budget.Name, status, category.Spent, category.Limit, percentage)
	}
	return formatMessage("A categoria \"%s\" do seu orçamento \"%s\" %s (R$ %.2f / R$ %.2f - %d%%)",
		category.Category, budget.Name, status, category.Spent, category.Limit, percentage)
}

// Helper: formatMessage is a simple wrapper for fmt.Sprintf
func formatMessage(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

// UpdateCategorySpent recalculates spending from expense payments for a specific category/month/year
// This method is called when expense payments are created, updated, or deleted
func (s *BudgetService) UpdateCategorySpent(userID uint, category string, year, month int) error {
	// Find all active budgets for this user and period (individual and group budgets)
	var budgets []models.Budget
	database.DB.Where("user_id = ? AND year = ? AND month = ? AND status = ?",
		userID, year, month, models.BudgetStatusActive).
		Preload("Group").
		Preload("Categories").
		Find(&budgets)

	for _, budget := range budgets {
		// Find the category in this budget
		var budgetCategory models.BudgetCategory
		if err := database.DB.Where("budget_id = ? AND category = ?", budget.ID, category).
			First(&budgetCategory).Error; err != nil {
			continue // Category doesn't exist in this budget, skip
		}

		// Recalculate spent amount from expense payments
		var totalSpent float64
		paymentQuery := database.DB.Table("expense_payments").
			Joins("JOIN expenses ON expenses.id = expense_payments.expense_id").
			Where("expenses.category = ? AND expense_payments.year = ? AND expense_payments.month = ?", category, year, month)

		if budget.GroupID != nil {
			// For group budgets, sum all payments from group members' accounts
			paymentQuery = paymentQuery.
				Joins("JOIN accounts ON accounts.id = expenses.account_id").
				Where("accounts.group_id = ?", *budget.GroupID)
		} else {
			// For individual budgets, only count user's account expenses
			paymentQuery = paymentQuery.
				Joins("JOIN accounts ON accounts.id = expenses.account_id").
				Where("accounts.user_id = ? AND (accounts.group_id IS NULL OR accounts.type = ?)", userID, models.AccountTypeIndividual)
		}

		paymentQuery.Select("COALESCE(SUM(expense_payments.amount), 0)").Row().Scan(&totalSpent)

		// Store old spent value to check if thresholds crossed
		oldSpent := budgetCategory.Spent

		// Update spent amount
		database.DB.Model(&budgetCategory).Update("spent", totalSpent)

		// Reload to get updated values
		database.DB.First(&budgetCategory, budgetCategory.ID)

		// Check if thresholds were crossed and send notifications
		oldPercentage := 0.0
		if budgetCategory.Limit > 0 {
			oldPercentage = (oldSpent / budgetCategory.Limit) * 100
		}
		newPercentage := budgetCategory.ProgressPercentage()

		// Send 80% notification if crossed from below to above
		if oldPercentage < 80 && newPercentage >= 80 && budgetCategory.NotifiedAt80 == nil {
			s.sendCategoryNotification(&budgetCategory, 80)
			now := time.Now()
			database.DB.Model(&budgetCategory).Update("notified_at_80", now)
		}

		// Send 100% notification if crossed from below to above
		if oldPercentage < 100 && newPercentage >= 100 && budgetCategory.NotifiedAt100 == nil {
			s.sendCategoryNotification(&budgetCategory, 100)
			now := time.Now()
			database.DB.Model(&budgetCategory).Update("notified_at_100", now)
		}
	}

	return nil
}

// RecalculateBudgetSpent recalculates all category spending for a budget from expense payments
func (s *BudgetService) RecalculateBudgetSpent(budgetID uint) error {
	// Get budget with categories
	var budget models.Budget
	if err := database.DB.Preload("Categories").First(&budget, budgetID).Error; err != nil {
		return ErrBudgetNotFound
	}

	// Recalculate each category
	for _, category := range budget.Categories {
		var totalSpent float64
		paymentQuery := database.DB.Table("expense_payments").
			Joins("JOIN expenses ON expenses.id = expense_payments.expense_id").
			Where("expenses.category = ? AND expense_payments.year = ? AND expense_payments.month = ?",
				category.Category, budget.Year, budget.Month)

		if budget.GroupID != nil {
			// For group budgets, sum all payments from group members' accounts
			paymentQuery = paymentQuery.
				Joins("JOIN accounts ON accounts.id = expenses.account_id").
				Where("accounts.group_id = ?", *budget.GroupID)
		} else {
			// For individual budgets, only count user's account expenses
			paymentQuery = paymentQuery.
				Joins("JOIN accounts ON accounts.id = expenses.account_id").
				Where("accounts.user_id = ? AND (accounts.group_id IS NULL OR accounts.type = ?)", budget.UserID, models.AccountTypeIndividual)
		}

		paymentQuery.Select("COALESCE(SUM(expense_payments.amount), 0)").Row().Scan(&totalSpent)

		// Update spent amount
		database.DB.Model(&category).Update("spent", totalSpent)
	}

	return nil
}
