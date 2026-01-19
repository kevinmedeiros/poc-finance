package services

import (
	"fmt"
	"log"
	"time"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
)

type RecurringSchedulerService struct {
	notificationService *NotificationService
}

func NewRecurringSchedulerService() *RecurringSchedulerService {
	return &RecurringSchedulerService{
		notificationService: NewNotificationService(),
	}
}

// ProcessDueTransactions checks all active recurring transactions and generates transactions for those that are due
func (s *RecurringSchedulerService) ProcessDueTransactions() error {
	now := time.Now()

	// Find all active recurring transactions where NextRunDate is today or earlier
	var recurringTransactions []models.RecurringTransaction
	err := database.DB.Preload("Account").
		Where("active = ? AND next_run_date <= ?", true, now).
		Find(&recurringTransactions).Error

	if err != nil {
		return fmt.Errorf("failed to fetch due recurring transactions: %w", err)
	}

	log.Printf("Found %d due recurring transactions to process", len(recurringTransactions))

	for _, rt := range recurringTransactions {
		// Check if end date has passed
		if rt.EndDate != nil && now.After(*rt.EndDate) {
			// Deactivate the recurring transaction
			if err := s.deactivateRecurringTransaction(rt.ID); err != nil {
				log.Printf("Error deactivating recurring transaction %d: %v", rt.ID, err)
			}
			continue
		}

		// Generate the transaction
		if err := s.generateTransaction(&rt); err != nil {
			log.Printf("Error generating transaction for recurring transaction %d: %v", rt.ID, err)
			continue
		}

		// Update NextRunDate
		nextRunDate := s.calculateNextRunDate(rt.NextRunDate, rt.Frequency)
		if err := s.updateNextRunDate(rt.ID, nextRunDate); err != nil {
			log.Printf("Error updating next run date for recurring transaction %d: %v", rt.ID, err)
			continue
		}

		// Notify the user
		if err := s.notifyUser(&rt); err != nil {
			log.Printf("Error notifying user for recurring transaction %d: %v", rt.ID, err)
			// Don't fail if notification fails
		}
	}

	return nil
}

// generateTransaction creates an Expense or Income based on the recurring transaction
func (s *RecurringSchedulerService) generateTransaction(rt *models.RecurringTransaction) error {
	if rt.TransactionType == models.TransactionTypeExpense {
		return s.generateExpense(rt)
	} else if rt.TransactionType == models.TransactionTypeIncome {
		return s.generateIncome(rt)
	}
	return fmt.Errorf("unknown transaction type: %s", rt.TransactionType)
}

// generateExpense creates a new Expense from a recurring transaction
func (s *RecurringSchedulerService) generateExpense(rt *models.RecurringTransaction) error {
	expense := &models.Expense{
		AccountID: rt.AccountID,
		Name:      rt.Description,
		Amount:    rt.Amount,
		Type:      models.ExpenseTypeVariable,
		Category:  rt.Category,
		Active:    true,
		IsSplit:   false,
	}

	if err := database.DB.Create(expense).Error; err != nil {
		return fmt.Errorf("failed to create expense: %w", err)
	}

	log.Printf("Generated expense %d from recurring transaction %d", expense.ID, rt.ID)
	return nil
}

// generateIncome creates a new Income from a recurring transaction
func (s *RecurringSchedulerService) generateIncome(rt *models.RecurringTransaction) error {
	// For income, we need to set USD and BRL amounts
	// Since recurring transactions use BRL, we'll set AmountBRL and use 1.0 as exchange rate
	income := &models.Income{
		AccountID:    rt.AccountID,
		Date:         time.Now(),
		AmountUSD:    rt.Amount, // Using BRL as USD for now
		ExchangeRate: 1.0,
		AmountBRL:    rt.Amount,
		GrossAmount:  rt.Amount,
		TaxAmount:    0, // Tax calculation would be done separately
		NetAmount:    rt.Amount,
		Description:  rt.Description,
	}

	if err := database.DB.Create(income).Error; err != nil {
		return fmt.Errorf("failed to create income: %w", err)
	}

	log.Printf("Generated income %d from recurring transaction %d", income.ID, rt.ID)
	return nil
}

// calculateNextRunDate determines the next run date based on frequency
func (s *RecurringSchedulerService) calculateNextRunDate(currentRunDate time.Time, frequency models.Frequency) time.Time {
	switch frequency {
	case models.FrequencyDaily:
		return currentRunDate.AddDate(0, 0, 1)
	case models.FrequencyWeekly:
		return currentRunDate.AddDate(0, 0, 7)
	case models.FrequencyMonthly:
		return currentRunDate.AddDate(0, 1, 0)
	case models.FrequencyYearly:
		return currentRunDate.AddDate(1, 0, 0)
	default:
		// Default to monthly if unknown
		return currentRunDate.AddDate(0, 1, 0)
	}
}

// updateNextRunDate updates the next run date for a recurring transaction
func (s *RecurringSchedulerService) updateNextRunDate(recurringTransactionID uint, nextRunDate time.Time) error {
	return database.DB.Model(&models.RecurringTransaction{}).
		Where("id = ?", recurringTransactionID).
		Update("next_run_date", nextRunDate).Error
}

// deactivateRecurringTransaction marks a recurring transaction as inactive
func (s *RecurringSchedulerService) deactivateRecurringTransaction(recurringTransactionID uint) error {
	log.Printf("Deactivating recurring transaction %d (end date reached)", recurringTransactionID)
	return database.DB.Model(&models.RecurringTransaction{}).
		Where("id = ?", recurringTransactionID).
		Update("active", false).Error
}

// notifyUser sends a notification when a transaction is generated from a recurring schedule
func (s *RecurringSchedulerService) notifyUser(rt *models.RecurringTransaction) error {
	// Get the user ID from the account
	var account models.Account
	if err := database.DB.First(&account, rt.AccountID).Error; err != nil {
		return fmt.Errorf("failed to fetch account: %w", err)
	}

	var title, message string
	if rt.TransactionType == models.TransactionTypeExpense {
		title = "Despesa recorrente gerada"
		message = fmt.Sprintf("Uma despesa recorrente foi criada: %s (R$ %.2f)", rt.Description, rt.Amount)
	} else {
		title = "Receita recorrente gerada"
		message = fmt.Sprintf("Uma receita recorrente foi criada: %s (R$ %.2f)", rt.Description, rt.Amount)
	}

	notification := &models.Notification{
		UserID:  account.UserID,
		Type:    models.NotificationTypeExpense, // Using expense type for both
		Title:   title,
		Message: message,
		Link:    "/recurring",
		GroupID: account.GroupID,
	}

	return s.notificationService.Create(notification)
}

// GetDueCount returns the count of recurring transactions that are due for processing
func (s *RecurringSchedulerService) GetDueCount() (int64, error) {
	now := time.Now()
	var count int64
	err := database.DB.Model(&models.RecurringTransaction{}).
		Where("active = ? AND next_run_date <= ?", true, now).
		Count(&count).Error
	return count, err
}
