package services

import (
	"fmt"
	"time"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
)

type NotificationService struct{}

func NewNotificationService() *NotificationService {
	return &NotificationService{}
}

// Create creates a new notification for a user
func (s *NotificationService) Create(notification *models.Notification) error {
	return database.DB.Create(notification).Error
}

// GetUserNotifications retrieves all notifications for a user
func (s *NotificationService) GetUserNotifications(userID uint, limit int) ([]models.Notification, error) {
	var notifications []models.Notification
	query := database.DB.Where("user_id = ?", userID).
		Preload("Group").
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&notifications).Error
	return notifications, err
}

// GetUnreadNotifications retrieves only unread notifications for a user
func (s *NotificationService) GetUnreadNotifications(userID uint) ([]models.Notification, error) {
	var notifications []models.Notification
	err := database.DB.Where("user_id = ? AND read = ?", userID, false).
		Preload("Group").
		Order("created_at DESC").
		Find(&notifications).Error
	return notifications, err
}

// GetUnreadCount returns the count of unread notifications for a user
func (s *NotificationService) GetUnreadCount(userID uint) (int64, error) {
	var count int64
	err := database.DB.Model(&models.Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Count(&count).Error
	return count, err
}

// MarkAsRead marks a single notification as read
func (s *NotificationService) MarkAsRead(notificationID, userID uint) error {
	now := time.Now()
	return database.DB.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", notificationID, userID).
		Updates(map[string]interface{}{
			"read":    true,
			"read_at": now,
		}).Error
}

// MarkAllAsRead marks all notifications as read for a user
func (s *NotificationService) MarkAllAsRead(userID uint) error {
	now := time.Now()
	return database.DB.Model(&models.Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Updates(map[string]interface{}{
			"read":    true,
			"read_at": now,
		}).Error
}

// DeleteNotification deletes a notification
func (s *NotificationService) DeleteNotification(notificationID, userID uint) error {
	return database.DB.Where("id = ? AND user_id = ?", notificationID, userID).
		Delete(&models.Notification{}).Error
}

// NotifyGroupInvite creates a notification when a user is added to a group
func (s *NotificationService) NotifyGroupInvite(userID uint, group *models.FamilyGroup, inviterName string) error {
	notification := &models.Notification{
		UserID:  userID,
		Type:    models.NotificationTypeGroupInvite,
		Title:   "Convite para grupo",
		Message: fmt.Sprintf("Você foi adicionado ao grupo \"%s\" por %s", group.Name, inviterName),
		Link:    fmt.Sprintf("/groups/%d/dashboard", group.ID),
		GroupID: &group.ID,
	}
	return s.Create(notification)
}

// NotifyPartnerExpense creates notifications for group members when a new expense is added to a joint account
func (s *NotificationService) NotifyPartnerExpense(expense *models.Expense, account *models.Account, creatorID uint, creatorName string, groupMembers []models.User) error {
	if account.GroupID == nil {
		return nil
	}

	for _, member := range groupMembers {
		// Don't notify the creator
		if member.ID == creatorID {
			continue
		}

		notification := &models.Notification{
			UserID:  member.ID,
			Type:    models.NotificationTypeExpense,
			Title:   "Novo gasto do parceiro",
			Message: fmt.Sprintf("%s adicionou \"%s\" (R$ %.2f) na conta conjunta \"%s\"", creatorName, expense.Name, expense.Amount, account.Name),
			Link:    "/expenses",
			GroupID: account.GroupID,
		}
		if err := s.Create(notification); err != nil {
			return err
		}
	}
	return nil
}

// NotifyGoalReached creates notifications for all group members when a goal is reached
func (s *NotificationService) NotifyGoalReached(goal *models.GroupGoal, groupMembers []models.User) error {
	for _, member := range groupMembers {
		notification := &models.Notification{
			UserID:  member.ID,
			Type:    models.NotificationTypeGoalReached,
			Title:   "Meta atingida!",
			Message: fmt.Sprintf("A meta \"%s\" do grupo \"%s\" foi alcançada! (R$ %.2f)", goal.Name, goal.Group.Name, goal.TargetAmount),
			Link:    fmt.Sprintf("/groups/%d/goals", goal.GroupID),
			GroupID: &goal.GroupID,
		}
		if err := s.Create(notification); err != nil {
			return err
		}
	}
	return nil
}

// GroupSummaryData holds the data for a periodic group summary
type GroupSummaryData struct {
	GroupName     string
	GroupID       uint
	PeriodType    string // "weekly" or "monthly"
	PeriodLabel   string
	TotalIncome   float64
	TotalExpenses float64
	Balance       float64
	ExpenseCount  int
	IncomeCount   int
	GoalsProgress []GoalProgress
}

// GoalProgress represents progress data for a goal
type GoalProgress struct {
	Name          string
	CurrentAmount float64
	TargetAmount  float64
	Percentage    float64
}

// NotifyWeeklySummary creates a weekly summary notification for all group members
func (s *NotificationService) NotifyWeeklySummary(summaryData GroupSummaryData, groupMembers []models.User) error {
	balanceSign := ""
	if summaryData.Balance >= 0 {
		balanceSign = "+"
	}

	message := fmt.Sprintf("Receitas: R$ %.2f | Despesas: R$ %.2f | Saldo: %sR$ %.2f",
		summaryData.TotalIncome,
		summaryData.TotalExpenses,
		balanceSign,
		summaryData.Balance,
	)

	for _, member := range groupMembers {
		notification := &models.Notification{
			UserID:  member.ID,
			Type:    models.NotificationTypeSummary,
			Title:   fmt.Sprintf("Resumo semanal - %s", summaryData.GroupName),
			Message: message,
			Link:    fmt.Sprintf("/groups/%d/dashboard", summaryData.GroupID),
			GroupID: &summaryData.GroupID,
		}
		if err := s.Create(notification); err != nil {
			return err
		}
	}
	return nil
}

// NotifyMonthlySummary creates a monthly summary notification for all group members
func (s *NotificationService) NotifyMonthlySummary(summaryData GroupSummaryData, groupMembers []models.User) error {
	balanceSign := ""
	if summaryData.Balance >= 0 {
		balanceSign = "+"
	}

	message := fmt.Sprintf("%s: Receitas R$ %.2f | Despesas R$ %.2f | Saldo %sR$ %.2f",
		summaryData.PeriodLabel,
		summaryData.TotalIncome,
		summaryData.TotalExpenses,
		balanceSign,
		summaryData.Balance,
	)

	for _, member := range groupMembers {
		notification := &models.Notification{
			UserID:  member.ID,
			Type:    models.NotificationTypeSummary,
			Title:   fmt.Sprintf("Resumo mensal - %s", summaryData.GroupName),
			Message: message,
			Link:    fmt.Sprintf("/groups/%d/dashboard", summaryData.GroupID),
			GroupID: &summaryData.GroupID,
		}
		if err := s.Create(notification); err != nil {
			return err
		}
	}
	return nil
}

// BudgetAlertData holds data for budget limit alert notifications
type BudgetAlertData struct {
	Account       *models.Account
	TotalExpenses float64
	BudgetLimit   float64
	Percentage    float64
}

// NotifyBudgetLimitReached creates notifications when account budget limit is reached or exceeded
func (s *NotificationService) NotifyBudgetLimitReached(alertData BudgetAlertData, members []models.User) error {
	statusText := "atingiu"
	if alertData.Percentage > 100 {
		statusText = "ultrapassou"
	}

	message := fmt.Sprintf("A conta \"%s\" %s o limite de orçamento! Despesas: R$ %.2f / R$ %.2f (%.0f%%)",
		alertData.Account.Name,
		statusText,
		alertData.TotalExpenses,
		alertData.BudgetLimit,
		alertData.Percentage,
	)

	for _, member := range members {
		notification := &models.Notification{
			UserID:  member.ID,
			Type:    models.NotificationTypeBudgetAlert,
			Title:   "Alerta de orçamento",
			Message: message,
			Link:    "/expenses",
			GroupID: alertData.Account.GroupID,
		}
		if err := s.Create(notification); err != nil {
			return err
		}
	}
	return nil
}

// NotifyUpcomingDueDate creates a notification for an upcoming expense due date
func (s *NotificationService) NotifyUpcomingDueDate(expense *models.Expense, userID uint, daysUntilDue int) error {
	var message string
	if daysUntilDue == 0 {
		message = fmt.Sprintf("A despesa \"%s\" (R$ %.2f) vence hoje!", expense.Name, expense.Amount)
	} else if daysUntilDue == 1 {
		message = fmt.Sprintf("A despesa \"%s\" (R$ %.2f) vence amanhã!", expense.Name, expense.Amount)
	} else {
		message = fmt.Sprintf("A despesa \"%s\" (R$ %.2f) vence em %d dias", expense.Name, expense.Amount, daysUntilDue)
	}

	notification := &models.Notification{
		UserID:  userID,
		Type:    models.NotificationTypeDueDate,
		Title:   "Vencimento próximo",
		Message: message,
		Link:    "/expenses",
	}
	return s.Create(notification)
}

// NotifyBudgetThreshold creates notifications when a budget category reaches spending thresholds (80% or 100%)
func (s *NotificationService) NotifyBudgetThreshold(budget *models.Budget, category *models.BudgetCategory, threshold int, members []models.User) error {
	percentage := category.ProgressPercentage()

	var title, statusText string
	if threshold >= 100 {
		title = "Orçamento excedido"
		statusText = "atingiu ou ultrapassou"
	} else {
		title = "Alerta de orçamento"
		statusText = "atingiu"
	}

	message := fmt.Sprintf("A categoria \"%s\" do orçamento \"%s\" %s %d%% do limite! Gastos: R$ %.2f / R$ %.2f (%.0f%%)",
		category.Category,
		budget.Name,
		statusText,
		threshold,
		category.Spent,
		category.Limit,
		percentage,
	)

	for _, member := range members {
		notification := &models.Notification{
			UserID:  member.ID,
			Type:    models.NotificationTypeBudgetAlert,
			Title:   title,
			Message: message,
			Link:    "/budgets",
			GroupID: budget.GroupID,
		}
		if err := s.Create(notification); err != nil {
			return err
		}
	}
	return nil
}
