package services

import (
	"fmt"
	"log"
	"time"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
)

type DueDateSchedulerService struct {
	notificationService *NotificationService
}

func NewDueDateSchedulerService() *DueDateSchedulerService {
	return &DueDateSchedulerService{
		notificationService: NewNotificationService(),
	}
}

// CheckUpcomingDueDates checks all active fixed expenses and sends notifications for those due in 3 days
func (s *DueDateSchedulerService) CheckUpcomingDueDates() error {
	now := time.Now()
	targetDate := now.AddDate(0, 0, 3) // 3 days from now
	targetDay := targetDate.Day()
	currentMonth := int(now.Month())
	currentYear := now.Year()

	// Find all active fixed expenses
	var expenses []models.Expense
	err := database.DB.Preload("Account").
		Where("type = ? AND active = ?", models.ExpenseTypeFixed, true).
		Find(&expenses).Error

	if err != nil {
		return fmt.Errorf("failed to fetch fixed expenses: %w", err)
	}

	log.Printf("Found %d active fixed expenses to check for upcoming due dates", len(expenses))

	for _, expense := range expenses {
		// Check if the due day matches our target (3 days from now)
		if expense.DueDay != targetDay {
			continue
		}

		// Check if this expense has already been paid for the current month
		var payment models.ExpensePayment
		err := database.DB.Where("expense_id = ? AND month = ? AND year = ?",
			expense.ID, currentMonth, currentYear).
			First(&payment).Error

		if err == nil {
			// Payment record exists, expense is already paid - skip notification
			log.Printf("Expense %d (%s) is already paid for %d/%d, skipping notification",
				expense.ID, expense.Name, currentMonth, currentYear)
			continue
		}

		// No payment record found - expense is unpaid, send notification
		if err := s.notifyUpcomingDueDate(&expense, targetDate); err != nil {
			log.Printf("Error notifying for expense %d: %v", expense.ID, err)
			// Don't fail if notification fails, continue processing other expenses
		}
	}

	return nil
}

// notifyUpcomingDueDate sends notifications to account members about an upcoming due date
func (s *DueDateSchedulerService) notifyUpcomingDueDate(expense *models.Expense, dueDate time.Time) error {
	// Get the account to find group members
	var account models.Account
	if err := database.DB.First(&account, expense.AccountID).Error; err != nil {
		return fmt.Errorf("failed to fetch account: %w", err)
	}

	// For individual accounts, notify only the owner
	if account.IsIndividual() {
		notification := &models.Notification{
			UserID:  account.UserID,
			Type:    models.NotificationTypeDueDate,
			Title:   "Despesa próxima do vencimento",
			Message: fmt.Sprintf("A despesa \"%s\" vence em 3 dias (R$ %.2f)", expense.Name, expense.Amount),
			Link:    "/expenses",
			GroupID: nil,
		}
		if err := s.notificationService.Create(notification); err != nil {
			return err
		}
		log.Printf("Sent due date notification for expense %d (%s) to user %d",
			expense.ID, expense.Name, account.UserID)
		return nil
	}

	// For joint accounts, notify all group members
	if account.GroupID == nil {
		return fmt.Errorf("joint account %d has no group ID", account.ID)
	}

	// Get group members
	var members []models.User
	err := database.DB.Table("users").
		Joins("JOIN group_members ON users.id = group_members.user_id").
		Where("group_members.group_id = ?", *account.GroupID).
		Find(&members).Error

	if err != nil {
		return fmt.Errorf("failed to fetch group members: %w", err)
	}

	// Send notification to each member
	for _, member := range members {
		notification := &models.Notification{
			UserID:  member.ID,
			Type:    models.NotificationTypeDueDate,
			Title:   "Despesa próxima do vencimento",
			Message: fmt.Sprintf("A despesa \"%s\" da conta \"%s\" vence em 3 dias (R$ %.2f)",
				expense.Name, account.Name, expense.Amount),
			Link:    "/expenses",
			GroupID: account.GroupID,
		}
		if err := s.notificationService.Create(notification); err != nil {
			return err
		}
	}

	log.Printf("Sent due date notification for expense %d (%s) to %d group members",
		expense.ID, expense.Name, len(members))
	return nil
}

// GetUpcomingDueDatesCount returns the count of unpaid fixed expenses due in 3 days
func (s *DueDateSchedulerService) GetUpcomingDueDatesCount() (int64, error) {
	now := time.Now()
	targetDate := now.AddDate(0, 0, 3)
	targetDay := targetDate.Day()
	currentMonth := int(now.Month())
	currentYear := now.Year()

	// Find all active fixed expenses with the target due day
	var expenses []models.Expense
	err := database.DB.Where("type = ? AND active = ? AND due_day = ?",
		models.ExpenseTypeFixed, true, targetDay).
		Find(&expenses).Error

	if err != nil {
		return 0, err
	}

	// Count how many are unpaid
	var count int64
	for _, expense := range expenses {
		var payment models.ExpensePayment
		err := database.DB.Where("expense_id = ? AND month = ? AND year = ?",
			expense.ID, currentMonth, currentYear).
			First(&payment).Error

		if err != nil {
			// No payment record found, this expense is unpaid
			count++
		}
	}

	return count, nil
}
