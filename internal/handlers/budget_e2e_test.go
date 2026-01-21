package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"poc-finance/internal/database"
	"poc-finance/internal/middleware"
	"poc-finance/internal/models"
	"poc-finance/internal/services"
	"poc-finance/internal/testutil"
)

// TestBudgetE2E_IndividualBudget tests creating an individual budget, adding categories, and tracking expenses
func TestBudgetE2E_IndividualBudget(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create test user and account
	user := testutil.CreateTestUser(db, "user@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, user.ID, "Test Account", 1000.00)

	// Initialize services and handler
	budgetService := services.NewBudgetService()
	budgetHandler := &BudgetHandler{
		budgetService: budgetService,
		groupService:  services.NewGroupService(),
	}

	e := echo.New()

	t.Run("Create individual budget", func(t *testing.T) {
		formData := url.Values{}
		formData.Set("name", "January Budget")
		formData.Set("year", "2024")
		formData.Set("month", "1")

		req := httptest.NewRequest(http.MethodPost, "/budgets", strings.NewReader(formData.Encode()))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(middleware.UserKey, user.ID)

		if err := budgetHandler.Create(c); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("Status code = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("Add categories to budget", func(t *testing.T) {
		// Get the created budget
		budgets, err := budgetService.GetUserBudgets(user.ID, 2024, 1)
		if err != nil || len(budgets) == 0 {
			t.Fatalf("Failed to get created budget")
		}
		budget := budgets[0]

		// Add Food category
		categories := []struct {
			Category string
			Limit    float64
		}{
			{"Alimentação", 500.00},
			{"Transporte", 200.00},
			{"Lazer", 100.00},
		}

		for _, cat := range categories {
			err := budgetService.AddCategory(user.ID, budget.ID, cat.Category, cat.Limit)
			if err != nil {
				t.Errorf("AddCategory() error = %v", err)
			}
		}

		// Verify categories were added
		updatedBudget, err := budgetService.GetBudgetByID(user.ID, budget.ID)
		if err != nil {
			t.Fatalf("GetBudgetByID() error = %v", err)
		}

		if len(updatedBudget.Categories) != 3 {
			t.Errorf("Categories count = %d, want 3", len(updatedBudget.Categories))
		}

		// Verify total limit
		totalLimit := updatedBudget.TotalLimit()
		if totalLimit != 800.00 {
			t.Errorf("TotalLimit = %.2f, want 800.00", totalLimit)
		}
	})

	t.Run("Track expense and verify budget progress", func(t *testing.T) {
		// Get budget
		budgets, _ := budgetService.GetUserBudgets(user.ID, 2024, 1)
		budget := budgets[0]

		// Create expense
		expense := &models.Expense{
			UserID:      user.ID,
			AccountID:   account.ID,
			Amount:      100.00,
			Category:    "Alimentação",
			Description: "Grocery shopping",
			Date:        "2024-01-15",
		}
		db.Create(expense)

		// Create expense payment (this is what budgets track)
		payment := &models.ExpensePayment{
			ExpenseID:    expense.ID,
			AccountID:    account.ID,
			Amount:       100.00,
			PaymentDate:  "2024-01-15",
			Year:         2024,
			Month:        1,
			IsSplit:      false,
			SplitGroupID: nil,
		}
		db.Create(payment)

		// Update budget spending
		err := budgetService.UpdateCategorySpent(user.ID, "Alimentação", 2024, 1)
		if err != nil {
			t.Fatalf("UpdateCategorySpent() error = %v", err)
		}

		// Verify budget was updated
		updatedBudget, _ := budgetService.GetBudgetByID(user.ID, budget.ID)

		// Find Alimentação category
		var foodCategory *models.BudgetCategory
		for i := range updatedBudget.Categories {
			if updatedBudget.Categories[i].Category == "Alimentação" {
				foodCategory = &updatedBudget.Categories[i]
				break
			}
		}

		if foodCategory == nil {
			t.Fatal("Alimentação category not found")
		}

		if foodCategory.Spent != 100.00 {
			t.Errorf("Spent = %.2f, want 100.00", foodCategory.Spent)
		}

		progress := foodCategory.ProgressPercentage()
		if progress != 20.00 {
			t.Errorf("Progress = %.2f%%, want 20.00%%", progress)
		}

		remaining := foodCategory.RemainingAmount()
		if remaining != 400.00 {
			t.Errorf("Remaining = %.2f, want 400.00", remaining)
		}

		status := foodCategory.GetStatus()
		if status != models.CategoryStatusGood {
			t.Errorf("Status = %s, want %s", status, models.CategoryStatusGood)
		}
	})
}

// TestBudgetE2E_GroupBudget tests creating a group budget and verifying members can see it
func TestBudgetE2E_GroupBudget(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create test users
	user1 := testutil.CreateTestUser(db, "user1@example.com", "User 1", "hash")
	user2 := testutil.CreateTestUser(db, "user2@example.com", "User 2", "hash")

	// Create family group
	group := testutil.CreateTestGroup(db, user1.ID, "Test Family")
	testutil.AddGroupMember(db, group.ID, user2.ID, models.RoleMember)

	// Initialize services
	budgetService := services.NewBudgetService()
	groupService := services.NewGroupService()

	t.Run("Create group budget", func(t *testing.T) {
		categories := []struct {
			Category string
			Limit    float64
		}{
			{"Alimentação", 1000.00},
			{"Contas", 500.00},
		}

		budget, err := budgetService.CreateBudget(user1.ID, &group.ID, 2024, 1, "Family January Budget", categories)
		if err != nil {
			t.Fatalf("CreateBudget() error = %v", err)
		}

		if !budget.IsGroupBudget() {
			t.Error("Budget should be a group budget")
		}

		if *budget.GroupID != group.ID {
			t.Errorf("GroupID = %d, want %d", *budget.GroupID, group.ID)
		}
	})

	t.Run("Verify group member can view budget", func(t *testing.T) {
		// User 2 should be able to get group budgets
		budgets, err := budgetService.GetGroupBudgets(user2.ID, group.ID, 2024, 1)
		if err != nil {
			t.Fatalf("GetGroupBudgets() error = %v", err)
		}

		if len(budgets) != 1 {
			t.Errorf("Budgets count = %d, want 1", len(budgets))
		}

		if budgets[0].Name != "Family January Budget" {
			t.Errorf("Budget name = %s, want Family January Budget", budgets[0].Name)
		}
	})

	t.Run("Verify non-member cannot view budget", func(t *testing.T) {
		user3 := testutil.CreateTestUser(db, "user3@example.com", "User 3", "hash")

		// User 3 is not a group member, should return error
		_, err := budgetService.GetGroupBudgets(user3.ID, group.ID, 2024, 1)
		if err != services.ErrUnauthorized {
			t.Errorf("Expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("Verify IsGroupMember check in GroupService", func(t *testing.T) {
		// Verify user1 is a member
		isMember, err := groupService.IsGroupMember(group.ID, user1.ID)
		if err != nil {
			t.Fatalf("IsGroupMember() error = %v", err)
		}
		if !isMember {
			t.Error("User1 should be a group member")
		}

		// Verify user2 is a member
		isMember, err = groupService.IsGroupMember(group.ID, user2.ID)
		if err != nil {
			t.Fatalf("IsGroupMember() error = %v", err)
		}
		if !isMember {
			t.Error("User2 should be a group member")
		}

		// Verify user3 is not a member
		user3 := testutil.CreateTestUser(db, "user3@example.com", "User 3", "hash")
		isMember, err = groupService.IsGroupMember(group.ID, user3.ID)
		if err != nil {
			t.Fatalf("IsGroupMember() error = %v", err)
		}
		if isMember {
			t.Error("User3 should not be a group member")
		}
	})
}

// TestBudgetE2E_CopyFromPreviousMonth tests copying a budget from previous month
func TestBudgetE2E_CopyFromPreviousMonth(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "Test User", "hash")
	budgetService := services.NewBudgetService()

	t.Run("Copy budget from previous month", func(t *testing.T) {
		// Create December budget
		decemberCategories := []struct {
			Category string
			Limit    float64
		}{
			{"Alimentação", 500.00},
			{"Transporte", 200.00},
			{"Lazer", 150.00},
		}

		decBudget, err := budgetService.CreateBudget(user.ID, nil, 2023, 12, "December Budget", decemberCategories)
		if err != nil {
			t.Fatalf("CreateBudget() error = %v", err)
		}

		if len(decBudget.Categories) != 3 {
			t.Errorf("December categories = %d, want 3", len(decBudget.Categories))
		}

		// Copy to January
		janBudget, err := budgetService.CopyFromPreviousMonth(user.ID, nil, 2024, 1, "January Budget (Copied)")
		if err != nil {
			t.Fatalf("CopyFromPreviousMonth() error = %v", err)
		}

		if janBudget.Year != 2024 {
			t.Errorf("Year = %d, want 2024", janBudget.Year)
		}

		if janBudget.Month != 1 {
			t.Errorf("Month = %d, want 1", janBudget.Month)
		}

		if len(janBudget.Categories) != 3 {
			t.Errorf("January categories = %d, want 3", len(janBudget.Categories))
		}

		// Verify categories were copied correctly
		categoryLimits := make(map[string]float64)
		for _, cat := range janBudget.Categories {
			categoryLimits[cat.Category] = cat.Limit

			// Spent should be reset to 0
			if cat.Spent != 0 {
				t.Errorf("Category %s spent = %.2f, want 0.00", cat.Category, cat.Spent)
			}
		}

		if categoryLimits["Alimentação"] != 500.00 {
			t.Errorf("Alimentação limit = %.2f, want 500.00", categoryLimits["Alimentação"])
		}
		if categoryLimits["Transporte"] != 200.00 {
			t.Errorf("Transporte limit = %.2f, want 200.00", categoryLimits["Transporte"])
		}
		if categoryLimits["Lazer"] != 150.00 {
			t.Errorf("Lazer limit = %.2f, want 150.00", categoryLimits["Lazer"])
		}
	})

	t.Run("Copy without previous month returns error", func(t *testing.T) {
		// Try to copy for a month without previous budget
		_, err := budgetService.CopyFromPreviousMonth(user.ID, nil, 2024, 6, "June Budget")
		if err == nil {
			t.Error("Expected error when copying without previous budget")
		}
	})
}

// TestBudgetE2E_ThresholdNotifications tests 80% and 100% threshold notifications
func TestBudgetE2E_ThresholdNotifications(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, user.ID, "Test Account", 1000.00)

	budgetService := services.NewBudgetService()

	t.Run("Trigger 80% notification", func(t *testing.T) {
		// Create budget with 500 limit
		categories := []struct {
			Category string
			Limit    float64
		}{
			{"Alimentação", 500.00},
		}

		budget, _ := budgetService.CreateBudget(user.ID, nil, 2024, 2, "February Budget", categories)

		// Add expense that crosses 80% threshold (400 = 80% of 500)
		expense := &models.Expense{
			UserID:      user.ID,
			AccountID:   account.ID,
			Amount:      400.00,
			Category:    "Alimentação",
			Description: "Large grocery purchase",
			Date:        "2024-02-10",
		}
		db.Create(expense)

		payment := &models.ExpensePayment{
			ExpenseID:   expense.ID,
			AccountID:   account.ID,
			Amount:      400.00,
			PaymentDate: "2024-02-10",
			Year:        2024,
			Month:       2,
		}
		db.Create(payment)

		// Update budget
		budgetService.UpdateCategorySpent(user.ID, "Alimentação", 2024, 2)

		// Check if notification was created
		var notificationCount int64
		db.Model(&models.Notification{}).Where("user_id = ? AND type = ?", user.ID, services.NotificationTypeBudget80).Count(&notificationCount)

		if notificationCount != 1 {
			t.Errorf("80%% notifications = %d, want 1", notificationCount)
		}

		// Verify category notification flag
		updatedBudget, _ := budgetService.GetBudgetByID(user.ID, budget.ID)
		var category *models.BudgetCategory
		for i := range updatedBudget.Categories {
			if updatedBudget.Categories[i].Category == "Alimentação" {
				category = &updatedBudget.Categories[i]
				break
			}
		}

		if !category.NotifiedAt80 {
			t.Error("NotifiedAt80 should be true")
		}
	})

	t.Run("Trigger 100% notification", func(t *testing.T) {
		// Get budget from previous test
		budgets, _ := budgetService.GetUserBudgets(user.ID, 2024, 2)
		budget := budgets[0]

		// Add another expense that crosses 100% threshold
		expense := &models.Expense{
			UserID:      user.ID,
			AccountID:   account.ID,
			Amount:      150.00,
			Category:    "Alimentação",
			Description: "More food",
			Date:        "2024-02-15",
		}
		db.Create(expense)

		payment := &models.ExpensePayment{
			ExpenseID:   expense.ID,
			AccountID:   account.ID,
			Amount:      150.00,
			PaymentDate: "2024-02-15",
			Year:        2024,
			Month:       2,
		}
		db.Create(payment)

		// Update budget
		budgetService.UpdateCategorySpent(user.ID, "Alimentação", 2024, 2)

		// Check if 100% notification was created
		var notificationCount int64
		db.Model(&models.Notification{}).Where("user_id = ? AND type = ?", user.ID, services.NotificationTypeBudget100).Count(&notificationCount)

		if notificationCount != 1 {
			t.Errorf("100%% notifications = %d, want 1", notificationCount)
		}

		// Verify category notification flags
		updatedBudget, _ := budgetService.GetBudgetByID(user.ID, budget.ID)
		var category *models.BudgetCategory
		for i := range updatedBudget.Categories {
			if updatedBudget.Categories[i].Category == "Alimentação" {
				category = &updatedBudget.Categories[i]
				break
			}
		}

		if !category.NotifiedAt100 {
			t.Error("NotifiedAt100 should be true")
		}

		// Verify spending
		if category.Spent != 550.00 {
			t.Errorf("Spent = %.2f, want 550.00", category.Spent)
		}
	})

	t.Run("No duplicate notifications", func(t *testing.T) {
		// Add another expense, should not create duplicate notifications
		expense := &models.Expense{
			UserID:      user.ID,
			AccountID:   account.ID,
			Amount:      50.00,
			Category:    "Alimentação",
			Description: "Small purchase",
			Date:        "2024-02-20",
		}
		db.Create(expense)

		payment := &models.ExpensePayment{
			ExpenseID:   expense.ID,
			AccountID:   account.ID,
			Amount:      50.00,
			PaymentDate: "2024-02-20",
			Year:        2024,
			Month:       2,
		}
		db.Create(payment)

		// Update budget
		budgetService.UpdateCategorySpent(user.ID, "Alimentação", 2024, 2)

		// Should still have only 1 notification of each type
		var count80 int64
		db.Model(&models.Notification{}).Where("user_id = ? AND type = ?", user.ID, services.NotificationTypeBudget80).Count(&count80)
		if count80 != 1 {
			t.Errorf("80%% notifications = %d, want 1", count80)
		}

		var count100 int64
		db.Model(&models.Notification{}).Where("user_id = ? AND type = ?", user.ID, services.NotificationTypeBudget100).Count(&count100)
		if count100 != 1 {
			t.Errorf("100%% notifications = %d, want 1", count100)
		}
	})
}

// TestBudgetE2E_VisualIndicators tests green/yellow/red color coding
func TestBudgetE2E_VisualIndicators(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "Test User", "hash")
	budgetService := services.NewBudgetService()

	// Create a budget for testing
	categories := []struct {
		Category string
		Limit    float64
	}{
		{"Test Category", 100.00},
	}
	budget, _ := budgetService.CreateBudget(user.ID, nil, 2024, 3, "Test Budget", categories)

	tests := []struct {
		name           string
		spent          float64
		expectedStatus models.CategoryStatus
	}{
		{"Green indicator (50%)", 50.00, models.CategoryStatusGood},
		{"Green indicator (70%)", 70.00, models.CategoryStatusGood},
		{"Yellow indicator (80%)", 80.00, models.CategoryStatusWarning},
		{"Yellow indicator (95%)", 95.00, models.CategoryStatusWarning},
		{"Red indicator (100%)", 100.00, models.CategoryStatusExceeded},
		{"Red indicator (120%)", 120.00, models.CategoryStatusExceeded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Update the category spent amount directly
			var category models.BudgetCategory
			db.First(&category, "budget_id = ?", budget.ID)
			category.Spent = tt.spent
			db.Save(&category)

			// Get status
			status := category.GetStatus()
			if status != tt.expectedStatus {
				t.Errorf("Status = %s, want %s (%.2f%% spent)", status, tt.expectedStatus, category.ProgressPercentage())
			}
		})
	}
}

// TestBudgetE2E_RealTimeUpdates verifies budget updates when expenses are created/deleted
func TestBudgetE2E_RealTimeUpdates(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, user.ID, "Test Account", 1000.00)

	budgetService := services.NewBudgetService()

	// Create budget
	categories := []struct {
		Category string
		Limit    float64
	}{
		{"Alimentação", 500.00},
	}
	budget, _ := budgetService.CreateBudget(user.ID, nil, 2024, 4, "April Budget", categories)

	t.Run("Budget updates when expense is paid", func(t *testing.T) {
		// Create expense
		expense := &models.Expense{
			UserID:      user.ID,
			AccountID:   account.ID,
			Amount:      100.00,
			Category:    "Alimentação",
			Description: "Test expense",
			Date:        "2024-04-10",
		}
		db.Create(expense)

		// Mark as paid (creates expense payment)
		payment := &models.ExpensePayment{
			ExpenseID:   expense.ID,
			AccountID:   account.ID,
			Amount:      100.00,
			PaymentDate: "2024-04-10",
			Year:        2024,
			Month:       4,
		}
		db.Create(payment)

		// Update budget
		budgetService.UpdateCategorySpent(user.ID, "Alimentação", 2024, 4)

		// Verify budget updated
		updatedBudget, _ := budgetService.GetBudgetByID(user.ID, budget.ID)
		var category *models.BudgetCategory
		for i := range updatedBudget.Categories {
			if updatedBudget.Categories[i].Category == "Alimentação" {
				category = &updatedBudget.Categories[i]
				break
			}
		}

		if category.Spent != 100.00 {
			t.Errorf("Spent = %.2f, want 100.00", category.Spent)
		}
	})

	t.Run("Budget updates when expense payment is deleted", func(t *testing.T) {
		// Delete the payment
		db.Where("year = ? AND month = ?", 2024, 4).Delete(&models.ExpensePayment{})

		// Update budget
		budgetService.UpdateCategorySpent(user.ID, "Alimentação", 2024, 4)

		// Verify budget updated
		updatedBudget, _ := budgetService.GetBudgetByID(user.ID, budget.ID)
		var category *models.BudgetCategory
		for i := range updatedBudget.Categories {
			if updatedBudget.Categories[i].Category == "Alimentação" {
				category = &updatedBudget.Categories[i]
				break
			}
		}

		if category.Spent != 0.00 {
			t.Errorf("Spent = %.2f, want 0.00", category.Spent)
		}
	})
}
