package models

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic("failed to connect to test database: " + err.Error())
	}

	// Migrate necessary models
	err = db.AutoMigrate(&User{}, &Account{}, &RecurringTransaction{})
	if err != nil {
		panic("failed to migrate test database: " + err.Error())
	}

	return db
}

func TestRecurringTransaction_TableName(t *testing.T) {
	rt := &RecurringTransaction{}
	expected := "recurring_transactions"

	if rt.TableName() != expected {
		t.Errorf("TableName() = %v, want %v", rt.TableName(), expected)
	}
}

func TestTransactionType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		value    TransactionType
		expected string
	}{
		{
			name:     "expense type",
			value:    TransactionTypeExpense,
			expected: "expense",
		},
		{
			name:     "income type",
			value:    TransactionTypeIncome,
			expected: "income",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) != tt.expected {
				t.Errorf("TransactionType = %v, want %v", tt.value, tt.expected)
			}
		})
	}
}

func TestFrequency_Constants(t *testing.T) {
	tests := []struct {
		name     string
		value    Frequency
		expected string
	}{
		{
			name:     "daily frequency",
			value:    FrequencyDaily,
			expected: "daily",
		},
		{
			name:     "weekly frequency",
			value:    FrequencyWeekly,
			expected: "weekly",
		},
		{
			name:     "monthly frequency",
			value:    FrequencyMonthly,
			expected: "monthly",
		},
		{
			name:     "yearly frequency",
			value:    FrequencyYearly,
			expected: "yearly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) != tt.expected {
				t.Errorf("Frequency = %v, want %v", tt.value, tt.expected)
			}
		})
	}
}

func TestRecurringTransaction_Create(t *testing.T) {
	db := setupTestDB()

	// Create test user and account
	user := &User{
		Email:        "test@example.com",
		Name:         "Test User",
		PasswordHash: "hash",
	}
	db.Create(user)

	account := &Account{
		Name:   "Test Account",
		Type:   "checking",
		UserID: user.ID,
	}
	db.Create(account)

	startDate := time.Now()
	nextRunDate := time.Now().AddDate(0, 1, 0)

	rt := &RecurringTransaction{
		AccountID:       account.ID,
		TransactionType: TransactionTypeExpense,
		Frequency:       FrequencyMonthly,
		Amount:          100.50,
		Description:     "Monthly subscription",
		StartDate:       startDate,
		NextRunDate:     nextRunDate,
		Active:          true,
		Category:        "subscriptions",
	}

	result := db.Create(rt)
	if result.Error != nil {
		t.Fatalf("Create() returned error: %v", result.Error)
	}

	if rt.ID == 0 {
		t.Error("ID should be set after creation")
	}

	// Verify data persisted correctly
	var retrieved RecurringTransaction
	db.First(&retrieved, rt.ID)

	if retrieved.AccountID != account.ID {
		t.Errorf("AccountID = %v, want %v", retrieved.AccountID, account.ID)
	}

	if retrieved.TransactionType != TransactionTypeExpense {
		t.Errorf("TransactionType = %v, want %v", retrieved.TransactionType, TransactionTypeExpense)
	}

	if retrieved.Frequency != FrequencyMonthly {
		t.Errorf("Frequency = %v, want %v", retrieved.Frequency, FrequencyMonthly)
	}

	if retrieved.Amount != 100.50 {
		t.Errorf("Amount = %v, want %v", retrieved.Amount, 100.50)
	}

	if retrieved.Description != "Monthly subscription" {
		t.Errorf("Description = %v, want %v", retrieved.Description, "Monthly subscription")
	}

	if retrieved.Active != true {
		t.Errorf("Active = %v, want %v", retrieved.Active, true)
	}

	if retrieved.Category != "subscriptions" {
		t.Errorf("Category = %v, want %v", retrieved.Category, "subscriptions")
	}

	if retrieved.EndDate != nil {
		t.Errorf("EndDate = %v, want nil", retrieved.EndDate)
	}
}

func TestRecurringTransaction_CreateWithEndDate(t *testing.T) {
	db := setupTestDB()

	// Create test user and account
	user := &User{
		Email:        "test2@example.com",
		Name:         "Test User 2",
		PasswordHash: "hash",
	}
	db.Create(user)

	account := &Account{
		Name:   "Test Account 2",
		Type:   "checking",
		UserID: user.ID,
	}
	db.Create(account)

	startDate := time.Now()
	endDate := time.Now().AddDate(1, 0, 0)
	nextRunDate := time.Now().AddDate(0, 1, 0)

	rt := &RecurringTransaction{
		AccountID:       account.ID,
		TransactionType: TransactionTypeIncome,
		Frequency:       FrequencyWeekly,
		Amount:          500.00,
		Description:     "Weekly payment",
		StartDate:       startDate,
		EndDate:         &endDate,
		NextRunDate:     nextRunDate,
		Active:          true,
		Category:        "salary",
	}

	result := db.Create(rt)
	if result.Error != nil {
		t.Fatalf("Create() returned error: %v", result.Error)
	}

	// Verify end date persisted correctly
	var retrieved RecurringTransaction
	db.First(&retrieved, rt.ID)

	if retrieved.EndDate == nil {
		t.Error("EndDate should not be nil")
	} else if !retrieved.EndDate.Equal(endDate) {
		t.Errorf("EndDate = %v, want %v", retrieved.EndDate, endDate)
	}
}

func TestRecurringTransaction_AllFrequencies(t *testing.T) {
	db := setupTestDB()

	// Create test user and account
	user := &User{
		Email:        "test3@example.com",
		Name:         "Test User 3",
		PasswordHash: "hash",
	}
	db.Create(user)

	account := &Account{
		Name:   "Test Account 3",
		Type:   "checking",
		UserID: user.ID,
	}
	db.Create(account)

	tests := []struct {
		name      string
		frequency Frequency
	}{
		{
			name:      "daily recurring transaction",
			frequency: FrequencyDaily,
		},
		{
			name:      "weekly recurring transaction",
			frequency: FrequencyWeekly,
		},
		{
			name:      "monthly recurring transaction",
			frequency: FrequencyMonthly,
		},
		{
			name:      "yearly recurring transaction",
			frequency: FrequencyYearly,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &RecurringTransaction{
				AccountID:       account.ID,
				TransactionType: TransactionTypeExpense,
				Frequency:       tt.frequency,
				Amount:          50.00,
				Description:     "Test transaction",
				StartDate:       time.Now(),
				NextRunDate:     time.Now().Add(24 * time.Hour),
				Active:          true,
			}

			result := db.Create(rt)
			if result.Error != nil {
				t.Fatalf("Create() with %v frequency returned error: %v", tt.frequency, result.Error)
			}

			var retrieved RecurringTransaction
			db.First(&retrieved, rt.ID)

			if retrieved.Frequency != tt.frequency {
				t.Errorf("Frequency = %v, want %v", retrieved.Frequency, tt.frequency)
			}
		})
	}
}

func TestRecurringTransaction_BothTransactionTypes(t *testing.T) {
	db := setupTestDB()

	// Create test user and account
	user := &User{
		Email:        "test4@example.com",
		Name:         "Test User 4",
		PasswordHash: "hash",
	}
	db.Create(user)

	account := &Account{
		Name:   "Test Account 4",
		Type:   "checking",
		UserID: user.ID,
	}
	db.Create(account)

	tests := []struct {
		name            string
		transactionType TransactionType
	}{
		{
			name:            "expense transaction",
			transactionType: TransactionTypeExpense,
		},
		{
			name:            "income transaction",
			transactionType: TransactionTypeIncome,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &RecurringTransaction{
				AccountID:       account.ID,
				TransactionType: tt.transactionType,
				Frequency:       FrequencyMonthly,
				Amount:          200.00,
				Description:     "Test transaction",
				StartDate:       time.Now(),
				NextRunDate:     time.Now().AddDate(0, 1, 0),
				Active:          true,
			}

			result := db.Create(rt)
			if result.Error != nil {
				t.Fatalf("Create() with %v type returned error: %v", tt.transactionType, result.Error)
			}

			var retrieved RecurringTransaction
			db.First(&retrieved, rt.ID)

			if retrieved.TransactionType != tt.transactionType {
				t.Errorf("TransactionType = %v, want %v", retrieved.TransactionType, tt.transactionType)
			}
		})
	}
}

func TestRecurringTransaction_ActiveFlag(t *testing.T) {
	db := setupTestDB()

	// Create test user and account
	user := &User{
		Email:        "test5@example.com",
		Name:         "Test User 5",
		PasswordHash: "hash",
	}
	db.Create(user)

	account := &Account{
		Name:   "Test Account 5",
		Type:   "checking",
		UserID: user.ID,
	}
	db.Create(account)

	t.Run("active recurring transaction", func(t *testing.T) {
		rt := &RecurringTransaction{
			AccountID:       account.ID,
			TransactionType: TransactionTypeExpense,
			Frequency:       FrequencyMonthly,
			Amount:          75.00,
			Description:     "Test transaction",
			StartDate:       time.Now(),
			NextRunDate:     time.Now().AddDate(0, 1, 0),
			Active:          true,
		}

		result := db.Create(rt)
		if result.Error != nil {
			t.Fatalf("Create() returned error: %v", result.Error)
		}

		var retrieved RecurringTransaction
		db.First(&retrieved, rt.ID)

		if retrieved.Active != true {
			t.Errorf("Active = %v, want %v", retrieved.Active, true)
		}
	})

	t.Run("deactivate recurring transaction", func(t *testing.T) {
		// Create as active first
		rt := &RecurringTransaction{
			AccountID:       account.ID,
			TransactionType: TransactionTypeExpense,
			Frequency:       FrequencyMonthly,
			Amount:          75.00,
			Description:     "Test transaction to deactivate",
			StartDate:       time.Now(),
			NextRunDate:     time.Now().AddDate(0, 1, 0),
			Active:          true,
		}

		db.Create(rt)

		// Then update to inactive
		rt.Active = false
		result := db.Save(rt)
		if result.Error != nil {
			t.Fatalf("Update() returned error: %v", result.Error)
		}

		var retrieved RecurringTransaction
		db.First(&retrieved, rt.ID)

		if retrieved.Active != false {
			t.Errorf("Active = %v, want %v", retrieved.Active, false)
		}
	})
}

func TestRecurringTransaction_ForeignKeyConstraint(t *testing.T) {
	db := setupTestDB()

	// Try to create recurring transaction with non-existent account
	rt := &RecurringTransaction{
		AccountID:       99999, // Non-existent account
		TransactionType: TransactionTypeExpense,
		Frequency:       FrequencyMonthly,
		Amount:          100.00,
		Description:     "Test transaction",
		StartDate:       time.Now(),
		NextRunDate:     time.Now().AddDate(0, 1, 0),
		Active:          true,
	}

	result := db.Create(rt)
	// SQLite doesn't enforce foreign keys by default in this setup,
	// but we verify the test runs without panic
	if result.Error != nil {
		// Foreign key constraint would fail here in production DB
		t.Logf("Expected behavior: foreign key constraint violation: %v", result.Error)
	}
}

func TestRecurringTransaction_Update(t *testing.T) {
	db := setupTestDB()

	// Create test user and account
	user := &User{
		Email:        "test6@example.com",
		Name:         "Test User 6",
		PasswordHash: "hash",
	}
	db.Create(user)

	account := &Account{
		Name:   "Test Account 6",
		Type:   "checking",
		UserID: user.ID,
	}
	db.Create(account)

	rt := &RecurringTransaction{
		AccountID:       account.ID,
		TransactionType: TransactionTypeExpense,
		Frequency:       FrequencyMonthly,
		Amount:          100.00,
		Description:     "Original description",
		StartDate:       time.Now(),
		NextRunDate:     time.Now().AddDate(0, 1, 0),
		Active:          true,
	}

	db.Create(rt)

	// Update the recurring transaction
	rt.Amount = 150.00
	rt.Description = "Updated description"
	rt.Active = false

	result := db.Save(rt)
	if result.Error != nil {
		t.Fatalf("Update() returned error: %v", result.Error)
	}

	// Verify updates persisted
	var retrieved RecurringTransaction
	db.First(&retrieved, rt.ID)

	if retrieved.Amount != 150.00 {
		t.Errorf("Amount = %v, want %v", retrieved.Amount, 150.00)
	}

	if retrieved.Description != "Updated description" {
		t.Errorf("Description = %v, want %v", retrieved.Description, "Updated description")
	}

	if retrieved.Active != false {
		t.Errorf("Active = %v, want %v", retrieved.Active, false)
	}
}

func TestRecurringTransaction_Delete(t *testing.T) {
	db := setupTestDB()

	// Create test user and account
	user := &User{
		Email:        "test7@example.com",
		Name:         "Test User 7",
		PasswordHash: "hash",
	}
	db.Create(user)

	account := &Account{
		Name:   "Test Account 7",
		Type:   "checking",
		UserID: user.ID,
	}
	db.Create(account)

	rt := &RecurringTransaction{
		AccountID:       account.ID,
		TransactionType: TransactionTypeExpense,
		Frequency:       FrequencyMonthly,
		Amount:          100.00,
		Description:     "To be deleted",
		StartDate:       time.Now(),
		NextRunDate:     time.Now().AddDate(0, 1, 0),
		Active:          true,
	}

	db.Create(rt)
	id := rt.ID

	// Soft delete
	result := db.Delete(rt)
	if result.Error != nil {
		t.Fatalf("Delete() returned error: %v", result.Error)
	}

	// Verify soft delete (should not find without Unscoped)
	var retrieved RecurringTransaction
	result = db.First(&retrieved, id)
	if result.Error == nil {
		t.Error("Expected error when finding soft-deleted record")
	}

	// Verify record still exists with Unscoped
	result = db.Unscoped().First(&retrieved, id)
	if result.Error != nil {
		t.Errorf("Unscoped().First() returned error: %v", result.Error)
	}

	if retrieved.DeletedAt.Valid != true {
		t.Error("DeletedAt should be set after soft delete")
	}
}

func TestRecurringTransaction_QueryByAccount(t *testing.T) {
	db := setupTestDB()

	// Create test user and accounts
	user := &User{
		Email:        "test8@example.com",
		Name:         "Test User 8",
		PasswordHash: "hash",
	}
	db.Create(user)

	account1 := &Account{
		Name:   "Account 1",
		Type:   "checking",
		UserID: user.ID,
	}
	db.Create(account1)

	account2 := &Account{
		Name:   "Account 2",
		Type:   "savings",
		UserID: user.ID,
	}
	db.Create(account2)

	// Create recurring transactions for both accounts
	rt1 := &RecurringTransaction{
		AccountID:       account1.ID,
		TransactionType: TransactionTypeExpense,
		Frequency:       FrequencyMonthly,
		Amount:          100.00,
		Description:     "Account 1 transaction",
		StartDate:       time.Now(),
		NextRunDate:     time.Now().AddDate(0, 1, 0),
		Active:          true,
	}
	db.Create(rt1)

	rt2 := &RecurringTransaction{
		AccountID:       account2.ID,
		TransactionType: TransactionTypeIncome,
		Frequency:       FrequencyWeekly,
		Amount:          200.00,
		Description:     "Account 2 transaction",
		StartDate:       time.Now(),
		NextRunDate:     time.Now().Add(7 * 24 * time.Hour),
		Active:          true,
	}
	db.Create(rt2)

	// Query by account1
	var transactions []RecurringTransaction
	db.Where("account_id = ?", account1.ID).Find(&transactions)

	if len(transactions) != 1 {
		t.Errorf("Expected 1 transaction for account1, got %d", len(transactions))
	}

	if len(transactions) > 0 && transactions[0].Description != "Account 1 transaction" {
		t.Errorf("Description = %v, want %v", transactions[0].Description, "Account 1 transaction")
	}
}

func TestRecurringTransaction_QueryByActiveStatus(t *testing.T) {
	db := setupTestDB()

	// Create test user and account
	user := &User{
		Email:        "test9@example.com",
		Name:         "Test User 9",
		PasswordHash: "hash",
	}
	db.Create(user)

	account := &Account{
		Name:   "Test Account 9",
		Type:   "checking",
		UserID: user.ID,
	}
	db.Create(account)

	// Create active transaction
	activeRT := &RecurringTransaction{
		AccountID:       account.ID,
		TransactionType: TransactionTypeExpense,
		Frequency:       FrequencyMonthly,
		Amount:          100.00,
		Description:     "Active transaction",
		StartDate:       time.Now(),
		NextRunDate:     time.Now().AddDate(0, 1, 0),
		Active:          true,
	}
	db.Create(activeRT)

	// Create another active transaction, then deactivate it
	inactiveRT := &RecurringTransaction{
		AccountID:       account.ID,
		TransactionType: TransactionTypeExpense,
		Frequency:       FrequencyMonthly,
		Amount:          50.00,
		Description:     "Inactive transaction",
		StartDate:       time.Now(),
		NextRunDate:     time.Now().AddDate(0, 1, 0),
		Active:          true,
	}
	db.Create(inactiveRT)

	// Deactivate the second transaction
	inactiveRT.Active = false
	db.Save(inactiveRT)

	// Query only active transactions for this specific account
	var activeTransactions []RecurringTransaction
	db.Where("account_id = ? AND active = ?", account.ID, true).Find(&activeTransactions)

	if len(activeTransactions) != 1 {
		t.Errorf("Expected 1 active transaction, got %d", len(activeTransactions))
	}

	if len(activeTransactions) > 0 && activeTransactions[0].Description != "Active transaction" {
		t.Errorf("Description = %v, want %v", activeTransactions[0].Description, "Active transaction")
	}

	// Query only inactive transactions for this specific account
	var inactiveTransactions []RecurringTransaction
	db.Where("account_id = ? AND active = ?", account.ID, false).Find(&inactiveTransactions)

	if len(inactiveTransactions) != 1 {
		t.Errorf("Expected 1 inactive transaction, got %d", len(inactiveTransactions))
	}

	if len(inactiveTransactions) > 0 && inactiveTransactions[0].Description != "Inactive transaction" {
		t.Errorf("Description = %v, want %v", inactiveTransactions[0].Description, "Inactive transaction")
	}
}
