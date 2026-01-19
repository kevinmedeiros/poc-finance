package services

import (
	"testing"
	"time"

	"poc-finance/internal/models"
	"poc-finance/internal/testutil"
)

func TestGetBatchMonthlySummariesForAccounts_EmptyAccounts(t *testing.T) {
	db := testutil.SetupTestDB()

	summaries := GetBatchMonthlySummariesForAccounts(db, 2024, 1, 2024, 6, []uint{})

	if len(summaries) == 0 {
		t.Error("GetBatchMonthlySummariesForAccounts() should return empty summaries for 6 months, got 0")
	}

	// Should still return 6 months of empty summaries
	expectedMonths := 6
	if len(summaries) != expectedMonths {
		t.Errorf("GetBatchMonthlySummariesForAccounts() returned %d months, want %d", len(summaries), expectedMonths)
	}

	// All summaries should have zero values
	for _, summary := range summaries {
		if summary.TotalIncomeGross != 0 || summary.TotalExpenses != 0 {
			t.Errorf("Empty account summary should have zero values, got income=%.2f, expenses=%.2f",
				summary.TotalIncomeGross, summary.TotalExpenses)
		}
	}
}

func TestGetBatchMonthlySummariesForAccounts_SingleMonth(t *testing.T) {
	db := testutil.SetupTestDB()

	// Create test data
	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	// Create income for January 2024
	db.Create(&models.Income{
		AccountID:   account.ID,
		Date:        time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 5000.00,
		TaxAmount:   500.00,
		NetAmount:   4500.00,
		Description: "January Income",
	})

	// Create fixed expense
	db.Create(&models.Expense{
		AccountID: account.ID,
		Name:      "Rent",
		Amount:    1000.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	})

	summaries := GetBatchMonthlySummariesForAccounts(db, 2024, 1, 2024, 1, []uint{account.ID})

	if len(summaries) != 1 {
		t.Fatalf("GetBatchMonthlySummariesForAccounts() returned %d months, want 1", len(summaries))
	}

	summary := summaries[0]

	if summary.TotalIncomeGross != 5000.00 {
		t.Errorf("TotalIncomeGross = %.2f, want 5000.00", summary.TotalIncomeGross)
	}

	if summary.TotalIncomeNet != 4500.00 {
		t.Errorf("TotalIncomeNet = %.2f, want 4500.00", summary.TotalIncomeNet)
	}

	if summary.TotalTax != 500.00 {
		t.Errorf("TotalTax = %.2f, want 500.00", summary.TotalTax)
	}

	if summary.TotalFixed != 1000.00 {
		t.Errorf("TotalFixed = %.2f, want 1000.00", summary.TotalFixed)
	}

	if summary.Balance != 3500.00 {
		t.Errorf("Balance = %.2f, want 3500.00", summary.Balance)
	}
}

func TestGetBatchMonthlySummariesForAccounts_MultipleMonths(t *testing.T) {
	db := testutil.SetupTestDB()

	// Create test data
	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	// Create incomes for different months
	db.Create(&models.Income{
		AccountID:   account.ID,
		Date:        time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 5000.00,
		TaxAmount:   500.00,
		NetAmount:   4500.00,
		Description: "January Income",
	})

	db.Create(&models.Income{
		AccountID:   account.ID,
		Date:        time.Date(2024, 2, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 6000.00,
		TaxAmount:   600.00,
		NetAmount:   5400.00,
		Description: "February Income",
	})

	db.Create(&models.Income{
		AccountID:   account.ID,
		Date:        time.Date(2024, 3, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 5500.00,
		TaxAmount:   550.00,
		NetAmount:   4950.00,
		Description: "March Income",
	})

	// Create fixed expense (applies to all months)
	db.Create(&models.Expense{
		AccountID: account.ID,
		Name:      "Rent",
		Amount:    1000.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	})

	// Create variable expenses for specific months
	expenseJan := &models.Expense{
		AccountID: account.ID,
		Name:      "Groceries Jan",
		Amount:    500.00,
		Type:      models.ExpenseTypeVariable,
		Active:    true,
	}
	db.Create(expenseJan)
	// Update CreatedAt to specific date
	db.Model(expenseJan).Update("created_at", time.Date(2024, 1, 20, 0, 0, 0, 0, time.Local))

	expenseFeb := &models.Expense{
		AccountID: account.ID,
		Name:      "Groceries Feb",
		Amount:    600.00,
		Type:      models.ExpenseTypeVariable,
		Active:    true,
	}
	db.Create(expenseFeb)
	// Update CreatedAt to specific date
	db.Model(expenseFeb).Update("created_at", time.Date(2024, 2, 20, 0, 0, 0, 0, time.Local))

	summaries := GetBatchMonthlySummariesForAccounts(db, 2024, 1, 2024, 3, []uint{account.ID})

	if len(summaries) != 3 {
		t.Fatalf("GetBatchMonthlySummariesForAccounts() returned %d months, want 3", len(summaries))
	}

	// Verify January
	if summaries[0].Month.Month() != time.January {
		t.Errorf("First summary month = %v, want January", summaries[0].Month.Month())
	}
	if summaries[0].TotalIncomeGross != 5000.00 {
		t.Errorf("January TotalIncomeGross = %.2f, want 5000.00", summaries[0].TotalIncomeGross)
	}
	if summaries[0].TotalFixed != 1000.00 {
		t.Errorf("January TotalFixed = %.2f, want 1000.00", summaries[0].TotalFixed)
	}
	if summaries[0].TotalVariable != 500.00 {
		t.Errorf("January TotalVariable = %.2f, want 500.00", summaries[0].TotalVariable)
	}

	// Verify February
	if summaries[1].Month.Month() != time.February {
		t.Errorf("Second summary month = %v, want February", summaries[1].Month.Month())
	}
	if summaries[1].TotalIncomeGross != 6000.00 {
		t.Errorf("February TotalIncomeGross = %.2f, want 6000.00", summaries[1].TotalIncomeGross)
	}
	if summaries[1].TotalFixed != 1000.00 {
		t.Errorf("February TotalFixed = %.2f, want 1000.00", summaries[1].TotalFixed)
	}
	if summaries[1].TotalVariable != 600.00 {
		t.Errorf("February TotalVariable = %.2f, want 600.00", summaries[1].TotalVariable)
	}

	// Verify March (no variable expenses)
	if summaries[2].Month.Month() != time.March {
		t.Errorf("Third summary month = %v, want March", summaries[2].Month.Month())
	}
	if summaries[2].TotalIncomeGross != 5500.00 {
		t.Errorf("March TotalIncomeGross = %.2f, want 5500.00", summaries[2].TotalIncomeGross)
	}
	if summaries[2].TotalFixed != 1000.00 {
		t.Errorf("March TotalFixed = %.2f, want 1000.00", summaries[2].TotalFixed)
	}
	if summaries[2].TotalVariable != 0.00 {
		t.Errorf("March TotalVariable = %.2f, want 0.00", summaries[2].TotalVariable)
	}
}

func TestGetBatchMonthlySummariesForAccounts_WithBills(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	// Create bills for different months
	db.Create(&models.Bill{
		AccountID: account.ID,
		Name:      "Electric Bill Jan",
		Amount:    150.00,
		DueDate:   time.Date(2024, 1, 10, 0, 0, 0, 0, time.Local),
	})

	db.Create(&models.Bill{
		AccountID: account.ID,
		Name:      "Water Bill Jan",
		Amount:    80.00,
		DueDate:   time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local),
	})

	db.Create(&models.Bill{
		AccountID: account.ID,
		Name:      "Electric Bill Feb",
		Amount:    160.00,
		DueDate:   time.Date(2024, 2, 10, 0, 0, 0, 0, time.Local),
	})

	summaries := GetBatchMonthlySummariesForAccounts(db, 2024, 1, 2024, 2, []uint{account.ID})

	if len(summaries) != 2 {
		t.Fatalf("GetBatchMonthlySummariesForAccounts() returned %d months, want 2", len(summaries))
	}

	// January should have 2 bills
	if summaries[0].TotalBills != 230.00 {
		t.Errorf("January TotalBills = %.2f, want 230.00", summaries[0].TotalBills)
	}

	// February should have 1 bill
	if summaries[1].TotalBills != 160.00 {
		t.Errorf("February TotalBills = %.2f, want 160.00", summaries[1].TotalBills)
	}
}

func TestGetBatchMonthlySummariesForAccounts_WithCreditCardInstallments(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	// Create credit card
	creditCard := &models.CreditCard{
		AccountID:  account.ID,
		Name:       "Test Card",
		ClosingDay: 15,
		DueDay:     25,
	}
	db.Create(creditCard)

	// Create installment starting in January 2024, 3 installments of 100.00 each
	db.Create(&models.Installment{
		CreditCardID:      creditCard.ID,
		Description:       "TV Purchase",
		TotalAmount:       300.00,
		InstallmentAmount: 100.00,
		TotalInstallments: 3,
		StartDate:         time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local),
	})

	summaries := GetBatchMonthlySummariesForAccounts(db, 2024, 1, 2024, 4, []uint{account.ID})

	if len(summaries) != 4 {
		t.Fatalf("GetBatchMonthlySummariesForAccounts() returned %d months, want 4", len(summaries))
	}

	// January should have installment
	if summaries[0].TotalCards != 100.00 {
		t.Errorf("January TotalCards = %.2f, want 100.00", summaries[0].TotalCards)
	}

	// February should have installment
	if summaries[1].TotalCards != 100.00 {
		t.Errorf("February TotalCards = %.2f, want 100.00", summaries[1].TotalCards)
	}

	// March should have installment
	if summaries[2].TotalCards != 100.00 {
		t.Errorf("March TotalCards = %.2f, want 100.00", summaries[2].TotalCards)
	}

	// April should have no installment (only 3 total)
	if summaries[3].TotalCards != 0.00 {
		t.Errorf("April TotalCards = %.2f, want 0.00", summaries[3].TotalCards)
	}
}

func TestGetBatchMonthlySummariesForAccounts_MultipleAccounts(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account1 := testutil.CreateTestAccount(db, "Account 1", models.AccountTypeIndividual, user.ID, nil)
	account2 := testutil.CreateTestAccount(db, "Account 2", models.AccountTypeIndividual, user.ID, nil)

	// Create incomes for both accounts
	db.Create(&models.Income{
		AccountID:   account1.ID,
		Date:        time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 3000.00,
		TaxAmount:   300.00,
		NetAmount:   2700.00,
	})

	db.Create(&models.Income{
		AccountID:   account2.ID,
		Date:        time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 2000.00,
		TaxAmount:   200.00,
		NetAmount:   1800.00,
	})

	// Create fixed expense for account 1
	db.Create(&models.Expense{
		AccountID: account1.ID,
		Name:      "Rent",
		Amount:    1000.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	})

	// Create fixed expense for account 2
	db.Create(&models.Expense{
		AccountID: account2.ID,
		Name:      "Utilities",
		Amount:    500.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	})

	summaries := GetBatchMonthlySummariesForAccounts(db, 2024, 1, 2024, 1, []uint{account1.ID, account2.ID})

	if len(summaries) != 1 {
		t.Fatalf("GetBatchMonthlySummariesForAccounts() returned %d months, want 1", len(summaries))
	}

	summary := summaries[0]

	// Should combine income from both accounts
	if summary.TotalIncomeGross != 5000.00 {
		t.Errorf("TotalIncomeGross = %.2f, want 5000.00 (3000 + 2000)", summary.TotalIncomeGross)
	}

	if summary.TotalIncomeNet != 4500.00 {
		t.Errorf("TotalIncomeNet = %.2f, want 4500.00 (2700 + 1800)", summary.TotalIncomeNet)
	}

	// Should combine expenses from both accounts
	if summary.TotalFixed != 1500.00 {
		t.Errorf("TotalFixed = %.2f, want 1500.00 (1000 + 500)", summary.TotalFixed)
	}
}

func TestGetBatchMonthlySummariesForAccounts_SixMonthRange(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	// Create income for each month
	for month := 1; month <= 6; month++ {
		db.Create(&models.Income{
			AccountID:   account.ID,
			Date:        time.Date(2024, time.Month(month), 15, 0, 0, 0, 0, time.Local),
			GrossAmount: float64(5000 + month*100),
			TaxAmount:   float64(500 + month*10),
			NetAmount:   float64(4500 + month*90),
			Description: "Monthly Income",
		})
	}

	// Create fixed expense
	db.Create(&models.Expense{
		AccountID: account.ID,
		Name:      "Fixed Expense",
		Amount:    1000.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	})

	// Create variable expenses for each month
	for month := 1; month <= 6; month++ {
		expense := &models.Expense{
			AccountID: account.ID,
			Name:      "Variable Expense",
			Amount:    float64(300 + month*50),
			Type:      models.ExpenseTypeVariable,
			Active:    true,
		}
		db.Create(expense)
		// Update CreatedAt to specific date
		db.Model(expense).Update("created_at", time.Date(2024, time.Month(month), 20, 0, 0, 0, 0, time.Local))
	}

	summaries := GetBatchMonthlySummariesForAccounts(db, 2024, 1, 2024, 6, []uint{account.ID})

	if len(summaries) != 6 {
		t.Fatalf("GetBatchMonthlySummariesForAccounts() returned %d months, want 6", len(summaries))
	}

	// Verify each month has correct data
	for i, summary := range summaries {
		month := i + 1
		expectedGross := float64(5000 + month*100)
		expectedVariable := float64(300 + month*50)

		if summary.Month.Month() != time.Month(month) {
			t.Errorf("Summary %d: month = %v, want %v", i, summary.Month.Month(), time.Month(month))
		}

		if summary.TotalIncomeGross != expectedGross {
			t.Errorf("Month %d: TotalIncomeGross = %.2f, want %.2f", month, summary.TotalIncomeGross, expectedGross)
		}

		if summary.TotalFixed != 1000.00 {
			t.Errorf("Month %d: TotalFixed = %.2f, want 1000.00", month, summary.TotalFixed)
		}

		if summary.TotalVariable != expectedVariable {
			t.Errorf("Month %d: TotalVariable = %.2f, want %.2f", month, summary.TotalVariable, expectedVariable)
		}

		// Verify totals and balance are calculated correctly
		expectedExpenses := summary.TotalFixed + summary.TotalVariable + summary.TotalCards + summary.TotalBills
		if summary.TotalExpenses != expectedExpenses {
			t.Errorf("Month %d: TotalExpenses = %.2f, want %.2f", month, summary.TotalExpenses, expectedExpenses)
		}

		expectedBalance := summary.TotalIncomeNet - summary.TotalExpenses
		if summary.Balance != expectedBalance {
			t.Errorf("Month %d: Balance = %.2f, want %.2f", month, summary.Balance, expectedBalance)
		}
	}
}

func TestGetBatchMonthlySummariesForAccounts_YearBoundary(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	// Create income in December 2023
	db.Create(&models.Income{
		AccountID:   account.ID,
		Date:        time.Date(2023, 12, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 5000.00,
		TaxAmount:   500.00,
		NetAmount:   4500.00,
	})

	// Create income in January 2024
	db.Create(&models.Income{
		AccountID:   account.ID,
		Date:        time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 6000.00,
		TaxAmount:   600.00,
		NetAmount:   5400.00,
	})

	summaries := GetBatchMonthlySummariesForAccounts(db, 2023, 12, 2024, 1, []uint{account.ID})

	if len(summaries) != 2 {
		t.Fatalf("GetBatchMonthlySummariesForAccounts() returned %d months, want 2", len(summaries))
	}

	// December 2023
	if summaries[0].Month.Year() != 2023 || summaries[0].Month.Month() != time.December {
		t.Errorf("First summary = %v, want December 2023", summaries[0].Month)
	}
	if summaries[0].TotalIncomeGross != 5000.00 {
		t.Errorf("December TotalIncomeGross = %.2f, want 5000.00", summaries[0].TotalIncomeGross)
	}

	// January 2024
	if summaries[1].Month.Year() != 2024 || summaries[1].Month.Month() != time.January {
		t.Errorf("Second summary = %v, want January 2024", summaries[1].Month)
	}
	if summaries[1].TotalIncomeGross != 6000.00 {
		t.Errorf("January TotalIncomeGross = %.2f, want 6000.00", summaries[1].TotalIncomeGross)
	}
}

func TestGetBatchMonthlySummariesForAccounts_ComplexScenario(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	// Create multiple incomes in January
	db.Create(&models.Income{
		AccountID:   account.ID,
		Date:        time.Date(2024, 1, 5, 0, 0, 0, 0, time.Local),
		GrossAmount: 3000.00,
		TaxAmount:   300.00,
		NetAmount:   2700.00,
		Description: "Income 1",
	})
	db.Create(&models.Income{
		AccountID:   account.ID,
		Date:        time.Date(2024, 1, 20, 0, 0, 0, 0, time.Local),
		GrossAmount: 2000.00,
		TaxAmount:   200.00,
		NetAmount:   1800.00,
		Description: "Income 2",
	})

	// Create fixed expense
	db.Create(&models.Expense{
		AccountID: account.ID,
		Name:      "Rent",
		Amount:    1200.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	})

	// Create multiple variable expenses
	groceries := &models.Expense{
		AccountID: account.ID,
		Name:      "Groceries",
		Amount:    400.00,
		Type:      models.ExpenseTypeVariable,
		Active:    true,
	}
	db.Create(groceries)
	db.Model(groceries).Update("created_at", time.Date(2024, 1, 10, 0, 0, 0, 0, time.Local))

	transport := &models.Expense{
		AccountID: account.ID,
		Name:      "Transport",
		Amount:    150.00,
		Type:      models.ExpenseTypeVariable,
		Active:    true,
	}
	db.Create(transport)
	db.Model(transport).Update("created_at", time.Date(2024, 1, 25, 0, 0, 0, 0, time.Local))

	// Create credit card with installments
	creditCard := &models.CreditCard{
		AccountID:  account.ID,
		Name:       "Credit Card",
		ClosingDay: 15,
		DueDay:     25,
	}
	db.Create(creditCard)

	db.Create(&models.Installment{
		CreditCardID:      creditCard.ID,
		Description:       "Laptop",
		TotalAmount:       1200.00,
		InstallmentAmount: 400.00,
		TotalInstallments: 3,
		StartDate:         time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local),
	})

	// Create bills
	db.Create(&models.Bill{
		AccountID: account.ID,
		Name:      "Electric",
		Amount:    180.00,
		DueDate:   time.Date(2024, 1, 10, 0, 0, 0, 0, time.Local),
	})
	db.Create(&models.Bill{
		AccountID: account.ID,
		Name:      "Internet",
		Amount:    120.00,
		DueDate:   time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local),
	})

	summaries := GetBatchMonthlySummariesForAccounts(db, 2024, 1, 2024, 1, []uint{account.ID})

	if len(summaries) != 1 {
		t.Fatalf("GetBatchMonthlySummariesForAccounts() returned %d months, want 1", len(summaries))
	}

	summary := summaries[0]

	// Verify all components
	if summary.TotalIncomeGross != 5000.00 {
		t.Errorf("TotalIncomeGross = %.2f, want 5000.00 (3000 + 2000)", summary.TotalIncomeGross)
	}

	if summary.TotalIncomeNet != 4500.00 {
		t.Errorf("TotalIncomeNet = %.2f, want 4500.00 (2700 + 1800)", summary.TotalIncomeNet)
	}

	if summary.TotalTax != 500.00 {
		t.Errorf("TotalTax = %.2f, want 500.00 (300 + 200)", summary.TotalTax)
	}

	if summary.TotalFixed != 1200.00 {
		t.Errorf("TotalFixed = %.2f, want 1200.00", summary.TotalFixed)
	}

	if summary.TotalVariable != 550.00 {
		t.Errorf("TotalVariable = %.2f, want 550.00 (400 + 150)", summary.TotalVariable)
	}

	if summary.TotalCards != 400.00 {
		t.Errorf("TotalCards = %.2f, want 400.00", summary.TotalCards)
	}

	if summary.TotalBills != 300.00 {
		t.Errorf("TotalBills = %.2f, want 300.00 (180 + 120)", summary.TotalBills)
	}

	// Total expenses = 1200 + 550 + 400 + 300 = 2450
	if summary.TotalExpenses != 2450.00 {
		t.Errorf("TotalExpenses = %.2f, want 2450.00", summary.TotalExpenses)
	}

	// Balance = 4500 - 2450 = 2050
	if summary.Balance != 2050.00 {
		t.Errorf("Balance = %.2f, want 2050.00", summary.Balance)
	}
}

func TestGetBatchMonthlySummariesForAccounts_InactiveExpenses(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	// Create active fixed expense
	activeExpense := &models.Expense{
		AccountID: account.ID,
		Name:      "Active Rent",
		Amount:    1000.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	}
	db.Create(activeExpense)

	// Create inactive fixed expense (should be ignored)
	inactiveExpense := &models.Expense{
		AccountID: account.ID,
		Name:      "Old Rent",
		Amount:    800.00,
		Type:      models.ExpenseTypeFixed,
	}
	db.Create(inactiveExpense)
	// Explicitly set active to false after creation
	db.Model(inactiveExpense).Update("active", false)

	// Verify the expenses were created with correct active status
	var allExpenses []models.Expense
	db.Where("account_id = ?", account.ID).Find(&allExpenses)
	if len(allExpenses) != 2 {
		t.Fatalf("Expected 2 expenses in database, got %d", len(allExpenses))
	}

	// Verify query filtering works
	var activeExpenses []models.Expense
	db.Where("type = ? AND active = ? AND account_id = ?", models.ExpenseTypeFixed, true, account.ID).Find(&activeExpenses)
	if len(activeExpenses) != 1 {
		t.Fatalf("Expected 1 active expense, got %d", len(activeExpenses))
	}
	if activeExpenses[0].Amount != 1000.00 {
		t.Fatalf("Expected active expense amount 1000.00, got %.2f", activeExpenses[0].Amount)
	}

	summaries := GetBatchMonthlySummariesForAccounts(db, 2024, 1, 2024, 1, []uint{account.ID})

	if len(summaries) != 1 {
		t.Fatalf("GetBatchMonthlySummariesForAccounts() returned %d months, want 1", len(summaries))
	}

	// Should only include active expense
	if summaries[0].TotalFixed != 1000.00 {
		t.Errorf("TotalFixed = %.2f, want 1000.00 (inactive should be excluded)", summaries[0].TotalFixed)
	}
}

func TestGetBatchMonthlySummariesForAccounts_MonthNameGeneration(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	summaries := GetBatchMonthlySummariesForAccounts(db, 2024, 1, 2024, 3, []uint{account.ID})

	if len(summaries) != 3 {
		t.Fatalf("GetBatchMonthlySummariesForAccounts() returned %d months, want 3", len(summaries))
	}

	// Verify month names are generated correctly (Portuguese month names)
	expectedMonths := []string{"Janeiro", "Fevereiro", "Março"}
	for i, summary := range summaries {
		if summary.MonthName == "" {
			t.Errorf("Month %d: MonthName is empty", i+1)
		}
		// Check if the month name contains the expected Portuguese month name
		if summary.MonthName[:len(expectedMonths[i])] != expectedMonths[i] {
			t.Errorf("Month %d: MonthName = %s, want to start with %s", i+1, summary.MonthName, expectedMonths[i])
		}
	}
}

// TestCompareBatchVsLoopImplementation verifies that the new batch implementation
// produces identical results to the old loop-based implementation
func TestCompareBatchVsLoopImplementation(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account1 := testutil.CreateTestAccount(db, "Account 1", models.AccountTypeIndividual, user.ID, nil)
	account2 := testutil.CreateTestAccount(db, "Account 2", models.AccountTypeIndividual, user.ID, nil)
	accountIDs := []uint{account1.ID, account2.ID}

	// Create comprehensive test data across 6 months (simulating real dashboard usage)

	// Incomes for different months and accounts
	incomeData := []struct {
		accountID uint
		month     int
		gross     float64
		tax       float64
		net       float64
	}{
		{account1.ID, 1, 5000.00, 500.00, 4500.00},
		{account2.ID, 1, 3000.00, 300.00, 2700.00},
		{account1.ID, 2, 5200.00, 520.00, 4680.00},
		{account2.ID, 2, 3100.00, 310.00, 2790.00},
		{account1.ID, 3, 5400.00, 540.00, 4860.00},
		{account1.ID, 4, 5100.00, 510.00, 4590.00},
		{account2.ID, 5, 3200.00, 320.00, 2880.00},
		{account1.ID, 6, 5500.00, 550.00, 4950.00},
	}

	for _, income := range incomeData {
		db.Create(&models.Income{
			AccountID:   income.accountID,
			Date:        time.Date(2024, time.Month(income.month), 15, 0, 0, 0, 0, time.Local),
			GrossAmount: income.gross,
			TaxAmount:   income.tax,
			NetAmount:   income.net,
			Description: "Test Income",
		})
	}

	// Fixed expenses (apply to all months)
	db.Create(&models.Expense{
		AccountID: account1.ID,
		Name:      "Rent Account 1",
		Amount:    1200.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	})
	db.Create(&models.Expense{
		AccountID: account2.ID,
		Name:      "Utilities Account 2",
		Amount:    800.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	})

	// Inactive expense (should be excluded)
	inactiveExpense := &models.Expense{
		AccountID: account1.ID,
		Name:      "Old Expense",
		Amount:    500.00,
		Type:      models.ExpenseTypeFixed,
	}
	db.Create(inactiveExpense)
	db.Model(inactiveExpense).Update("active", false)

	// Variable expenses for different months
	variableExpensesData := []struct {
		accountID uint
		month     int
		amount    float64
	}{
		{account1.ID, 1, 350.00},
		{account2.ID, 1, 250.00},
		{account1.ID, 2, 400.00},
		{account1.ID, 3, 380.00},
		{account2.ID, 4, 300.00},
		{account1.ID, 5, 420.00},
	}

	for _, ve := range variableExpensesData {
		expense := &models.Expense{
			AccountID: ve.accountID,
			Name:      "Variable Expense",
			Amount:    ve.amount,
			Type:      models.ExpenseTypeVariable,
			Active:    true,
		}
		db.Create(expense)
		db.Model(expense).Update("created_at", time.Date(2024, time.Month(ve.month), 20, 0, 0, 0, 0, time.Local))
	}

	// Credit card installments
	creditCard1 := &models.CreditCard{
		AccountID:  account1.ID,
		Name:       "Card 1",
		ClosingDay: 15,
		DueDay:     25,
	}
	db.Create(creditCard1)

	// 3 installments starting in month 2
	db.Create(&models.Installment{
		CreditCardID:      creditCard1.ID,
		Description:       "Purchase 1",
		TotalAmount:       900.00,
		InstallmentAmount: 300.00,
		TotalInstallments: 3,
		StartDate:         time.Date(2024, 2, 1, 0, 0, 0, 0, time.Local),
	})

	creditCard2 := &models.CreditCard{
		AccountID:  account2.ID,
		Name:       "Card 2",
		ClosingDay: 10,
		DueDay:     20,
	}
	db.Create(creditCard2)

	// 2 installments starting in month 4
	db.Create(&models.Installment{
		CreditCardID:      creditCard2.ID,
		Description:       "Purchase 2",
		TotalAmount:       500.00,
		InstallmentAmount: 250.00,
		TotalInstallments: 2,
		StartDate:         time.Date(2024, 4, 1, 0, 0, 0, 0, time.Local),
	})

	// Bills for different months
	billsData := []struct {
		accountID uint
		month     int
		amount    float64
	}{
		{account1.ID, 1, 150.00},
		{account2.ID, 1, 100.00},
		{account1.ID, 2, 180.00},
		{account1.ID, 3, 160.00},
		{account2.ID, 4, 120.00},
		{account1.ID, 5, 170.00},
		{account2.ID, 6, 140.00},
	}

	for _, bill := range billsData {
		db.Create(&models.Bill{
			AccountID: bill.accountID,
			Name:      "Test Bill",
			Amount:    bill.amount,
			DueDate:   time.Date(2024, time.Month(bill.month), 10, 0, 0, 0, 0, time.Local),
		})
	}

	// OLD IMPLEMENTATION: Loop through each month calling GetMonthlySummaryForAccounts
	loopResults := make([]MonthlySummary, 0, 6)
	for month := 1; month <= 6; month++ {
		summary := GetMonthlySummaryForAccounts(db, 2024, month, accountIDs)
		loopResults = append(loopResults, summary)
	}

	// NEW IMPLEMENTATION: Single batch call
	batchResults := GetBatchMonthlySummariesForAccounts(db, 2024, 1, 2024, 6, accountIDs)

	// COMPARISON: Verify both implementations produce identical results
	if len(loopResults) != len(batchResults) {
		t.Fatalf("Length mismatch: loop=%d, batch=%d", len(loopResults), len(batchResults))
	}

	if len(loopResults) != 6 {
		t.Fatalf("Expected 6 months, got %d", len(loopResults))
	}

	// Compare each month's summary
	for i := 0; i < 6; i++ {
		month := i + 1
		loop := loopResults[i]
		batch := batchResults[i]

		// Compare month
		if loop.Month.Year() != batch.Month.Year() || loop.Month.Month() != batch.Month.Month() {
			t.Errorf("Month %d: Date mismatch - loop=%v, batch=%v", month, loop.Month, batch.Month)
		}

		// Compare month name
		if loop.MonthName != batch.MonthName {
			t.Errorf("Month %d: MonthName mismatch - loop=%s, batch=%s", month, loop.MonthName, batch.MonthName)
		}

		// Compare all financial fields (must match exactly)
		if loop.TotalIncomeGross != batch.TotalIncomeGross {
			t.Errorf("Month %d: TotalIncomeGross mismatch - loop=%.2f, batch=%.2f",
				month, loop.TotalIncomeGross, batch.TotalIncomeGross)
		}

		if loop.TotalIncomeNet != batch.TotalIncomeNet {
			t.Errorf("Month %d: TotalIncomeNet mismatch - loop=%.2f, batch=%.2f",
				month, loop.TotalIncomeNet, batch.TotalIncomeNet)
		}

		if loop.TotalTax != batch.TotalTax {
			t.Errorf("Month %d: TotalTax mismatch - loop=%.2f, batch=%.2f",
				month, loop.TotalTax, batch.TotalTax)
		}

		if loop.TotalFixed != batch.TotalFixed {
			t.Errorf("Month %d: TotalFixed mismatch - loop=%.2f, batch=%.2f",
				month, loop.TotalFixed, batch.TotalFixed)
		}

		if loop.TotalVariable != batch.TotalVariable {
			t.Errorf("Month %d: TotalVariable mismatch - loop=%.2f, batch=%.2f",
				month, loop.TotalVariable, batch.TotalVariable)
		}

		if loop.TotalCards != batch.TotalCards {
			t.Errorf("Month %d: TotalCards mismatch - loop=%.2f, batch=%.2f",
				month, loop.TotalCards, batch.TotalCards)
		}

		if loop.TotalBills != batch.TotalBills {
			t.Errorf("Month %d: TotalBills mismatch - loop=%.2f, batch=%.2f",
				month, loop.TotalBills, batch.TotalBills)
		}

		if loop.TotalExpenses != batch.TotalExpenses {
			t.Errorf("Month %d: TotalExpenses mismatch - loop=%.2f, batch=%.2f",
				month, loop.TotalExpenses, batch.TotalExpenses)
		}

		if loop.Balance != batch.Balance {
			t.Errorf("Month %d: Balance mismatch - loop=%.2f, batch=%.2f",
				month, loop.Balance, batch.Balance)
		}
	}

	// Additional verification: Spot check specific expected values
	// Month 1: Should have income from both accounts, bills from both, variable expenses from both
	if batchResults[0].TotalIncomeGross != 8000.00 { // 5000 + 3000
		t.Errorf("Month 1: Expected TotalIncomeGross=8000.00, got %.2f", batchResults[0].TotalIncomeGross)
	}

	if batchResults[0].TotalFixed != 2000.00 { // 1200 + 800
		t.Errorf("Month 1: Expected TotalFixed=2000.00, got %.2f", batchResults[0].TotalFixed)
	}

	if batchResults[0].TotalVariable != 600.00 { // 350 + 250
		t.Errorf("Month 1: Expected TotalVariable=600.00, got %.2f", batchResults[0].TotalVariable)
	}

	if batchResults[0].TotalCards != 0.00 { // No installments in month 1
		t.Errorf("Month 1: Expected TotalCards=0.00, got %.2f", batchResults[0].TotalCards)
	}

	if batchResults[0].TotalBills != 250.00 { // 150 + 100
		t.Errorf("Month 1: Expected TotalBills=250.00, got %.2f", batchResults[0].TotalBills)
	}

	// Month 2: Should have installments starting
	if batchResults[1].TotalCards != 300.00 { // First installment
		t.Errorf("Month 2: Expected TotalCards=300.00, got %.2f", batchResults[1].TotalCards)
	}

	// Month 4: Should have installments from both cards
	if batchResults[3].TotalCards != 550.00 { // 300 (card1 3rd) + 250 (card2 1st)
		t.Errorf("Month 4: Expected TotalCards=550.00, got %.2f", batchResults[3].TotalCards)
	}

	// Month 5: Should only have card2's second installment (card1 ended in month 4)
	if batchResults[4].TotalCards != 250.00 { // 250 (card2 2nd, last)
		t.Errorf("Month 5: Expected TotalCards=250.00, got %.2f", batchResults[4].TotalCards)
	}

	// Month 6: No installments (both cards finished)
	if batchResults[5].TotalCards != 0.00 {
		t.Errorf("Month 6: Expected TotalCards=0.00, got %.2f", batchResults[5].TotalCards)
	}
}
