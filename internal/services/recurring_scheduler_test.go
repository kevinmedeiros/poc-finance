package services

import (
	"testing"
	"time"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
	"poc-finance/internal/testutil"
)

func TestRecurringSchedulerService_ProcessDueTransactions_Daily(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Migrate RecurringTransaction model
	db.AutoMigrate(&models.RecurringTransaction{})

	// Create test user and account
	user := testutil.CreateTestUser(db, "user@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Personal", models.AccountTypeIndividual, user.ID, nil)

	// Create a daily recurring transaction that's due today
	today := time.Now()
	recurringTx := &models.RecurringTransaction{
		AccountID:       account.ID,
		TransactionType: models.TransactionTypeExpense,
		Frequency:       models.FrequencyDaily,
		Amount:          100.0,
		Description:     "Daily Expense",
		StartDate:       today.AddDate(0, 0, -1),
		NextRunDate:     today,
		Active:          true,
		Category:        "Test",
	}
	db.Create(recurringTx)

	scheduler := NewRecurringSchedulerService()

	// Process due transactions
	err := scheduler.ProcessDueTransactions()
	if err != nil {
		t.Fatalf("ProcessDueTransactions() error = %v", err)
	}

	// Verify expense was created
	var expenses []models.Expense
	db.Where("account_id = ?", account.ID).Find(&expenses)

	if len(expenses) != 1 {
		t.Errorf("Expected 1 expense, got %d", len(expenses))
	}

	if len(expenses) > 0 {
		if expenses[0].Amount != 100.0 {
			t.Errorf("Expense amount = %.2f, want 100.00", expenses[0].Amount)
		}
		if expenses[0].Name != "Daily Expense" {
			t.Errorf("Expense name = %s, want Daily Expense", expenses[0].Name)
		}
	}

	// Verify next run date was updated (should be tomorrow)
	var updatedTx models.RecurringTransaction
	db.First(&updatedTx, recurringTx.ID)

	tomorrow := today.AddDate(0, 0, 1)
	if updatedTx.NextRunDate.Day() != tomorrow.Day() {
		t.Errorf("NextRunDate day = %d, want %d", updatedTx.NextRunDate.Day(), tomorrow.Day())
	}

	// Verify notification was created
	var notifications []models.Notification
	db.Where("user_id = ?", user.ID).Find(&notifications)

	if len(notifications) != 1 {
		t.Errorf("Expected 1 notification, got %d", len(notifications))
	}
}

func TestRecurringSchedulerService_ProcessDueTransactions_Weekly(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Migrate RecurringTransaction model
	db.AutoMigrate(&models.RecurringTransaction{})

	// Create test user and account
	user := testutil.CreateTestUser(db, "user@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Personal", models.AccountTypeIndividual, user.ID, nil)

	// Create a weekly recurring transaction that's due today
	today := time.Now()
	recurringTx := &models.RecurringTransaction{
		AccountID:       account.ID,
		TransactionType: models.TransactionTypeExpense,
		Frequency:       models.FrequencyWeekly,
		Amount:          200.0,
		Description:     "Weekly Expense",
		StartDate:       today.AddDate(0, 0, -7),
		NextRunDate:     today,
		Active:          true,
		Category:        "Test",
	}
	db.Create(recurringTx)

	scheduler := NewRecurringSchedulerService()

	// Process due transactions
	err := scheduler.ProcessDueTransactions()
	if err != nil {
		t.Fatalf("ProcessDueTransactions() error = %v", err)
	}

	// Verify expense was created
	var expenses []models.Expense
	db.Where("account_id = ?", account.ID).Find(&expenses)

	if len(expenses) != 1 {
		t.Errorf("Expected 1 expense, got %d", len(expenses))
	}

	// Verify next run date was updated (should be next week)
	var updatedTx models.RecurringTransaction
	db.First(&updatedTx, recurringTx.ID)

	nextWeek := today.AddDate(0, 0, 7)
	if updatedTx.NextRunDate.Day() != nextWeek.Day() {
		t.Errorf("NextRunDate day = %d, want %d", updatedTx.NextRunDate.Day(), nextWeek.Day())
	}
}

func TestRecurringSchedulerService_ProcessDueTransactions_Monthly(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Migrate RecurringTransaction model
	db.AutoMigrate(&models.RecurringTransaction{})

	// Create test user and account
	user := testutil.CreateTestUser(db, "user@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Personal", models.AccountTypeIndividual, user.ID, nil)

	// Create a monthly recurring transaction that's due today
	today := time.Now()
	recurringTx := &models.RecurringTransaction{
		AccountID:       account.ID,
		TransactionType: models.TransactionTypeExpense,
		Frequency:       models.FrequencyMonthly,
		Amount:          1500.0,
		Description:     "Monthly Rent",
		StartDate:       today.AddDate(0, -1, 0),
		NextRunDate:     today,
		Active:          true,
		Category:        "Housing",
	}
	db.Create(recurringTx)

	scheduler := NewRecurringSchedulerService()

	// Process due transactions
	err := scheduler.ProcessDueTransactions()
	if err != nil {
		t.Fatalf("ProcessDueTransactions() error = %v", err)
	}

	// Verify expense was created
	var expenses []models.Expense
	db.Where("account_id = ?", account.ID).Find(&expenses)

	if len(expenses) != 1 {
		t.Errorf("Expected 1 expense, got %d", len(expenses))
	}

	// Verify next run date was updated (should be next month)
	var updatedTx models.RecurringTransaction
	db.First(&updatedTx, recurringTx.ID)

	nextMonth := today.AddDate(0, 1, 0)
	if updatedTx.NextRunDate.Month() != nextMonth.Month() {
		t.Errorf("NextRunDate month = %d, want %d", updatedTx.NextRunDate.Month(), nextMonth.Month())
	}
}

func TestRecurringSchedulerService_ProcessDueTransactions_Yearly(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Migrate RecurringTransaction model
	db.AutoMigrate(&models.RecurringTransaction{})

	// Create test user and account
	user := testutil.CreateTestUser(db, "user@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Personal", models.AccountTypeIndividual, user.ID, nil)

	// Create a yearly recurring transaction that's due today
	today := time.Now()
	recurringTx := &models.RecurringTransaction{
		AccountID:       account.ID,
		TransactionType: models.TransactionTypeExpense,
		Frequency:       models.FrequencyYearly,
		Amount:          1200.0,
		Description:     "Annual Subscription",
		StartDate:       today.AddDate(-1, 0, 0),
		NextRunDate:     today,
		Active:          true,
		Category:        "Subscriptions",
	}
	db.Create(recurringTx)

	scheduler := NewRecurringSchedulerService()

	// Process due transactions
	err := scheduler.ProcessDueTransactions()
	if err != nil {
		t.Fatalf("ProcessDueTransactions() error = %v", err)
	}

	// Verify expense was created
	var expenses []models.Expense
	db.Where("account_id = ?", account.ID).Find(&expenses)

	if len(expenses) != 1 {
		t.Errorf("Expected 1 expense, got %d", len(expenses))
	}

	// Verify next run date was updated (should be next year)
	var updatedTx models.RecurringTransaction
	db.First(&updatedTx, recurringTx.ID)

	nextYear := today.AddDate(1, 0, 0)
	if updatedTx.NextRunDate.Year() != nextYear.Year() {
		t.Errorf("NextRunDate year = %d, want %d", updatedTx.NextRunDate.Year(), nextYear.Year())
	}
}

func TestRecurringSchedulerService_ProcessDueTransactions_Income(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Migrate RecurringTransaction model
	db.AutoMigrate(&models.RecurringTransaction{})

	// Create test user and account
	user := testutil.CreateTestUser(db, "user@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Personal", models.AccountTypeIndividual, user.ID, nil)

	// Create a monthly recurring income that's due today
	today := time.Now()
	recurringTx := &models.RecurringTransaction{
		AccountID:       account.ID,
		TransactionType: models.TransactionTypeIncome,
		Frequency:       models.FrequencyMonthly,
		Amount:          5000.0,
		Description:     "Monthly Salary",
		StartDate:       today.AddDate(0, -1, 0),
		NextRunDate:     today,
		Active:          true,
		Category:        "Salary",
	}
	db.Create(recurringTx)

	scheduler := NewRecurringSchedulerService()

	// Process due transactions
	err := scheduler.ProcessDueTransactions()
	if err != nil {
		t.Fatalf("ProcessDueTransactions() error = %v", err)
	}

	// Verify income was created
	var incomes []models.Income
	db.Where("account_id = ?", account.ID).Find(&incomes)

	if len(incomes) != 1 {
		t.Errorf("Expected 1 income, got %d", len(incomes))
	}

	if len(incomes) > 0 {
		if incomes[0].AmountBRL != 5000.0 {
			t.Errorf("Income amount = %.2f, want 5000.00", incomes[0].AmountBRL)
		}
		if incomes[0].Description != "Monthly Salary" {
			t.Errorf("Income description = %s, want Monthly Salary", incomes[0].Description)
		}
	}

	// Verify notification was created
	var notifications []models.Notification
	db.Where("user_id = ?", user.ID).Find(&notifications)

	if len(notifications) != 1 {
		t.Errorf("Expected 1 notification, got %d", len(notifications))
	}

	if len(notifications) > 0 {
		if notifications[0].Title != "Receita recorrente gerada" {
			t.Errorf("Notification title = %s", notifications[0].Title)
		}
	}
}

func TestRecurringSchedulerService_ProcessDueTransactions_Inactive(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Migrate RecurringTransaction model
	db.AutoMigrate(&models.RecurringTransaction{})

	// Create test user and account
	user := testutil.CreateTestUser(db, "user@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Personal", models.AccountTypeIndividual, user.ID, nil)

	// Create an inactive recurring transaction that's due today
	today := time.Now()
	recurringTx := &models.RecurringTransaction{
		AccountID:       account.ID,
		TransactionType: models.TransactionTypeExpense,
		Frequency:       models.FrequencyDaily,
		Amount:          100.0,
		Description:     "Inactive Transaction",
		StartDate:       today.AddDate(0, 0, -1),
		NextRunDate:     today,
		Active:          true,
		Category:        "Test",
	}
	db.Create(recurringTx)
	// Explicitly set Active to false (GORM default value handling)
	db.Model(recurringTx).Update("active", false)

	scheduler := NewRecurringSchedulerService()

	// Process due transactions
	err := scheduler.ProcessDueTransactions()
	if err != nil {
		t.Fatalf("ProcessDueTransactions() error = %v", err)
	}

	// Verify NO expense was created (because it's inactive)
	var expenses []models.Expense
	db.Where("account_id = ?", account.ID).Find(&expenses)

	if len(expenses) != 0 {
		t.Errorf("Expected 0 expenses for inactive transaction, got %d", len(expenses))
	}
}

func TestRecurringSchedulerService_ProcessDueTransactions_NotDueYet(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Migrate RecurringTransaction model
	db.AutoMigrate(&models.RecurringTransaction{})

	// Create test user and account
	user := testutil.CreateTestUser(db, "user@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Personal", models.AccountTypeIndividual, user.ID, nil)

	// Create a recurring transaction that's due tomorrow (not today)
	tomorrow := time.Now().AddDate(0, 0, 1)
	recurringTx := &models.RecurringTransaction{
		AccountID:       account.ID,
		TransactionType: models.TransactionTypeExpense,
		Frequency:       models.FrequencyDaily,
		Amount:          100.0,
		Description:     "Future Transaction",
		StartDate:       time.Now(),
		NextRunDate:     tomorrow,
		Active:          true,
		Category:        "Test",
	}
	db.Create(recurringTx)

	scheduler := NewRecurringSchedulerService()

	// Process due transactions
	err := scheduler.ProcessDueTransactions()
	if err != nil {
		t.Fatalf("ProcessDueTransactions() error = %v", err)
	}

	// Verify NO expense was created (not due yet)
	var expenses []models.Expense
	db.Where("account_id = ?", account.ID).Find(&expenses)

	if len(expenses) != 0 {
		t.Errorf("Expected 0 expenses for not-due transaction, got %d", len(expenses))
	}
}

func TestRecurringSchedulerService_ProcessDueTransactions_EndDateReached(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Migrate RecurringTransaction model
	db.AutoMigrate(&models.RecurringTransaction{})

	// Create test user and account
	user := testutil.CreateTestUser(db, "user@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Personal", models.AccountTypeIndividual, user.ID, nil)

	// Create a recurring transaction with end date in the past
	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)
	recurringTx := &models.RecurringTransaction{
		AccountID:       account.ID,
		TransactionType: models.TransactionTypeExpense,
		Frequency:       models.FrequencyDaily,
		Amount:          100.0,
		Description:     "Ended Transaction",
		StartDate:       today.AddDate(0, 0, -10),
		EndDate:         &yesterday,
		NextRunDate:     today,
		Active:          true,
		Category:        "Test",
	}
	db.Create(recurringTx)

	scheduler := NewRecurringSchedulerService()

	// Process due transactions
	err := scheduler.ProcessDueTransactions()
	if err != nil {
		t.Fatalf("ProcessDueTransactions() error = %v", err)
	}

	// Verify NO expense was created (end date reached)
	var expenses []models.Expense
	db.Where("account_id = ?", account.ID).Find(&expenses)

	if len(expenses) != 0 {
		t.Errorf("Expected 0 expenses for ended transaction, got %d", len(expenses))
	}

	// Verify transaction was deactivated
	var updatedTx models.RecurringTransaction
	db.First(&updatedTx, recurringTx.ID)

	if updatedTx.Active {
		t.Error("Transaction should be deactivated when end date is reached")
	}
}

func TestRecurringSchedulerService_GetDueCount(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Migrate RecurringTransaction model
	db.AutoMigrate(&models.RecurringTransaction{})

	// Create test user and account
	user := testutil.CreateTestUser(db, "user@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Personal", models.AccountTypeIndividual, user.ID, nil)

	today := time.Now()
	tomorrow := today.AddDate(0, 0, 1)

	// Create 2 due transactions
	db.Create(&models.RecurringTransaction{
		AccountID:       account.ID,
		TransactionType: models.TransactionTypeExpense,
		Frequency:       models.FrequencyDaily,
		Amount:          100.0,
		Description:     "Due 1",
		StartDate:       today,
		NextRunDate:     today,
		Active:          true,
	})

	db.Create(&models.RecurringTransaction{
		AccountID:       account.ID,
		TransactionType: models.TransactionTypeExpense,
		Frequency:       models.FrequencyDaily,
		Amount:          200.0,
		Description:     "Due 2",
		StartDate:       today,
		NextRunDate:     today,
		Active:          true,
	})

	// Create 1 not-due transaction
	db.Create(&models.RecurringTransaction{
		AccountID:       account.ID,
		TransactionType: models.TransactionTypeExpense,
		Frequency:       models.FrequencyDaily,
		Amount:          300.0,
		Description:     "Not Due",
		StartDate:       today,
		NextRunDate:     tomorrow,
		Active:          true,
	})

	scheduler := NewRecurringSchedulerService()

	count, err := scheduler.GetDueCount()
	if err != nil {
		t.Fatalf("GetDueCount() error = %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 due transactions, got %d", count)
	}
}

func TestRecurringSchedulerService_CalculateNextRunDate(t *testing.T) {
	scheduler := NewRecurringSchedulerService()
	baseDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		frequency models.Frequency
		expected  time.Time
	}{
		{
			name:      "daily adds 1 day",
			frequency: models.FrequencyDaily,
			expected:  baseDate.AddDate(0, 0, 1),
		},
		{
			name:      "weekly adds 7 days",
			frequency: models.FrequencyWeekly,
			expected:  baseDate.AddDate(0, 0, 7),
		},
		{
			name:      "monthly adds 1 month",
			frequency: models.FrequencyMonthly,
			expected:  baseDate.AddDate(0, 1, 0),
		},
		{
			name:      "yearly adds 1 year",
			frequency: models.FrequencyYearly,
			expected:  baseDate.AddDate(1, 0, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scheduler.calculateNextRunDate(baseDate, tt.frequency)

			if result.Year() != tt.expected.Year() ||
				result.Month() != tt.expected.Month() ||
				result.Day() != tt.expected.Day() {
				t.Errorf("calculateNextRunDate() = %v, want %v", result, tt.expected)
			}
		})
	}
}
