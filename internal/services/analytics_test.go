package services

import (
	"testing"
	"time"

	"poc-finance/internal/models"
	"poc-finance/internal/testutil"
)

func TestGetMonthOverMonthComparison(t *testing.T) {
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

	// Create income for February 2024 (20% increase)
	db.Create(&models.Income{
		AccountID:   account.ID,
		Date:        time.Date(2024, 2, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 6000.00,
		TaxAmount:   600.00,
		NetAmount:   5400.00,
		Description: "February Income",
	})

	// Create fixed expense (applies to both months)
	db.Create(&models.Expense{
		AccountID: account.ID,
		Name:      "Rent",
		Amount:    1000.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	})

	// Create variable expense for January
	expenseJan := &models.Expense{
		AccountID: account.ID,
		Name:      "Groceries Jan",
		Amount:    500.00,
		Type:      models.ExpenseTypeVariable,
		Active:    true,
	}
	db.Create(expenseJan)
	db.Model(expenseJan).Update("created_at", time.Date(2024, 1, 20, 0, 0, 0, 0, time.Local))

	// Create variable expense for February (higher amount)
	expenseFeb := &models.Expense{
		AccountID: account.ID,
		Name:      "Groceries Feb",
		Amount:    800.00,
		Type:      models.ExpenseTypeVariable,
		Active:    true,
	}
	db.Create(expenseFeb)
	db.Model(expenseFeb).Update("created_at", time.Date(2024, 2, 20, 0, 0, 0, 0, time.Local))

	comparison := GetMonthOverMonthComparison(db, 2024, 2, []uint{account.ID})

	// Verify current month (February)
	if comparison.CurrentMonth.Month.Year() != 2024 || comparison.CurrentMonth.Month.Month() != time.February {
		t.Errorf("CurrentMonth = %v, want February 2024", comparison.CurrentMonth.Month)
	}

	if comparison.CurrentMonth.TotalIncomeGross != 6000.00 {
		t.Errorf("CurrentMonth TotalIncomeGross = %.2f, want 6000.00", comparison.CurrentMonth.TotalIncomeGross)
	}

	// February expenses: 1000 (fixed) + 800 (variable) = 1800
	if comparison.CurrentMonth.TotalExpenses != 1800.00 {
		t.Errorf("CurrentMonth TotalExpenses = %.2f, want 1800.00", comparison.CurrentMonth.TotalExpenses)
	}

	// Verify previous month (January)
	if comparison.PreviousMonth.Month.Year() != 2024 || comparison.PreviousMonth.Month.Month() != time.January {
		t.Errorf("PreviousMonth = %v, want January 2024", comparison.PreviousMonth.Month)
	}

	if comparison.PreviousMonth.TotalIncomeGross != 5000.00 {
		t.Errorf("PreviousMonth TotalIncomeGross = %.2f, want 5000.00", comparison.PreviousMonth.TotalIncomeGross)
	}

	// January expenses: 1000 (fixed) + 500 (variable) = 1500
	if comparison.PreviousMonth.TotalExpenses != 1500.00 {
		t.Errorf("PreviousMonth TotalExpenses = %.2f, want 1500.00", comparison.PreviousMonth.TotalExpenses)
	}

	// Verify percentage changes
	// Income: (6000 - 5000) / 5000 * 100 = 20%
	expectedIncomeChangePercent := 20.0
	if comparison.IncomeChangePercent != expectedIncomeChangePercent {
		t.Errorf("IncomeChangePercent = %.2f, want %.2f", comparison.IncomeChangePercent, expectedIncomeChangePercent)
	}

	// Expense: (1800 - 1500) / 1500 * 100 = 20%
	expectedExpenseChangePercent := 20.0
	if comparison.ExpenseChangePercent != expectedExpenseChangePercent {
		t.Errorf("ExpenseChangePercent = %.2f, want %.2f", comparison.ExpenseChangePercent, expectedExpenseChangePercent)
	}

	// Verify absolute changes
	if comparison.IncomeChange != 1000.00 {
		t.Errorf("IncomeChange = %.2f, want 1000.00", comparison.IncomeChange)
	}

	if comparison.ExpenseChange != 300.00 {
		t.Errorf("ExpenseChange = %.2f, want 300.00", comparison.ExpenseChange)
	}
}

func TestGetMonthOverMonthComparison_YearBoundary(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	// Create income for December 2023
	db.Create(&models.Income{
		AccountID:   account.ID,
		Date:        time.Date(2023, 12, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 4000.00,
		TaxAmount:   400.00,
		NetAmount:   3600.00,
	})

	// Create income for January 2024
	db.Create(&models.Income{
		AccountID:   account.ID,
		Date:        time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 5000.00,
		TaxAmount:   500.00,
		NetAmount:   4500.00,
	})

	// Create fixed expense
	db.Create(&models.Expense{
		AccountID: account.ID,
		Name:      "Rent",
		Amount:    1000.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	})

	comparison := GetMonthOverMonthComparison(db, 2024, 1, []uint{account.ID})

	// Verify current month (January 2024)
	if comparison.CurrentMonth.Month.Year() != 2024 || comparison.CurrentMonth.Month.Month() != time.January {
		t.Errorf("CurrentMonth = %v, want January 2024", comparison.CurrentMonth.Month)
	}

	// Verify previous month (December 2023)
	if comparison.PreviousMonth.Month.Year() != 2023 || comparison.PreviousMonth.Month.Month() != time.December {
		t.Errorf("PreviousMonth = %v, want December 2023", comparison.PreviousMonth.Month)
	}

	// Verify income change across year boundary
	// Income: (5000 - 4000) / 4000 * 100 = 25%
	expectedIncomeChangePercent := 25.0
	if comparison.IncomeChangePercent != expectedIncomeChangePercent {
		t.Errorf("IncomeChangePercent = %.2f, want %.2f", comparison.IncomeChangePercent, expectedIncomeChangePercent)
	}
}

func TestGetMonthOverMonthComparison_EmptyAccounts(t *testing.T) {
	db := testutil.SetupTestDB()

	comparison := GetMonthOverMonthComparison(db, 2024, 2, []uint{})

	// Should return empty comparison with zero values
	if comparison.CurrentMonth.TotalIncomeGross != 0 {
		t.Errorf("CurrentMonth TotalIncomeGross = %.2f, want 0.00", comparison.CurrentMonth.TotalIncomeGross)
	}

	if comparison.PreviousMonth.TotalIncomeGross != 0 {
		t.Errorf("PreviousMonth TotalIncomeGross = %.2f, want 0.00", comparison.PreviousMonth.TotalIncomeGross)
	}

	if comparison.IncomeChangePercent != 0 {
		t.Errorf("IncomeChangePercent = %.2f, want 0.00", comparison.IncomeChangePercent)
	}
}

func TestGetMonthOverMonthComparison_ZeroPreviousMonth(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	// Only create income for February (no income in January)
	db.Create(&models.Income{
		AccountID:   account.ID,
		Date:        time.Date(2024, 2, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 5000.00,
		TaxAmount:   500.00,
		NetAmount:   4500.00,
	})

	comparison := GetMonthOverMonthComparison(db, 2024, 2, []uint{account.ID})

	// Previous month should be zero
	if comparison.PreviousMonth.TotalIncomeGross != 0 {
		t.Errorf("PreviousMonth TotalIncomeGross = %.2f, want 0.00", comparison.PreviousMonth.TotalIncomeGross)
	}

	// When previous is zero and current is positive, should show 100% increase
	if comparison.IncomeChangePercent != 100.0 {
		t.Errorf("IncomeChangePercent = %.2f, want 100.00 (going from 0 to positive)", comparison.IncomeChangePercent)
	}

	if comparison.IncomeChange != 5000.00 {
		t.Errorf("IncomeChange = %.2f, want 5000.00", comparison.IncomeChange)
	}
}

func TestGetMonthOverMonthComparison_MultipleAccounts(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account1 := testutil.CreateTestAccount(db, "Account 1", models.AccountTypeIndividual, user.ID, nil)
	account2 := testutil.CreateTestAccount(db, "Account 2", models.AccountTypeIndividual, user.ID, nil)

	// Create incomes for both accounts in January
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

	// Create incomes for both accounts in February
	db.Create(&models.Income{
		AccountID:   account1.ID,
		Date:        time.Date(2024, 2, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 3500.00,
		TaxAmount:   350.00,
		NetAmount:   3150.00,
	})
	db.Create(&models.Income{
		AccountID:   account2.ID,
		Date:        time.Date(2024, 2, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 2500.00,
		TaxAmount:   250.00,
		NetAmount:   2250.00,
	})

	// Create fixed expenses for both accounts
	db.Create(&models.Expense{
		AccountID: account1.ID,
		Name:      "Rent",
		Amount:    1000.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	})
	db.Create(&models.Expense{
		AccountID: account2.ID,
		Name:      "Utilities",
		Amount:    500.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	})

	comparison := GetMonthOverMonthComparison(db, 2024, 2, []uint{account1.ID, account2.ID})

	// February total income: 3500 + 2500 = 6000
	if comparison.CurrentMonth.TotalIncomeGross != 6000.00 {
		t.Errorf("CurrentMonth TotalIncomeGross = %.2f, want 6000.00", comparison.CurrentMonth.TotalIncomeGross)
	}

	// January total income: 3000 + 2000 = 5000
	if comparison.PreviousMonth.TotalIncomeGross != 5000.00 {
		t.Errorf("PreviousMonth TotalIncomeGross = %.2f, want 5000.00", comparison.PreviousMonth.TotalIncomeGross)
	}

	// Income change: (6000 - 5000) / 5000 * 100 = 20%
	expectedIncomeChangePercent := 20.0
	if comparison.IncomeChangePercent != expectedIncomeChangePercent {
		t.Errorf("IncomeChangePercent = %.2f, want %.2f", comparison.IncomeChangePercent, expectedIncomeChangePercent)
	}

	// Total expenses for both accounts: 1000 + 500 = 1500
	if comparison.CurrentMonth.TotalExpenses != 1500.00 {
		t.Errorf("CurrentMonth TotalExpenses = %.2f, want 1500.00", comparison.CurrentMonth.TotalExpenses)
	}
}

func TestGetMonthOverMonthComparison_WithCreditCardAndBills(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	// Create income for both months
	db.Create(&models.Income{
		AccountID:   account.ID,
		Date:        time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 5000.00,
		TaxAmount:   500.00,
		NetAmount:   4500.00,
	})
	db.Create(&models.Income{
		AccountID:   account.ID,
		Date:        time.Date(2024, 2, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 5000.00,
		TaxAmount:   500.00,
		NetAmount:   4500.00,
	})

	// Create fixed expense
	db.Create(&models.Expense{
		AccountID: account.ID,
		Name:      "Rent",
		Amount:    1000.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	})

	// Create credit card with installments starting in January
	creditCard := &models.CreditCard{
		AccountID:  account.ID,
		Name:       "Test Card",
		ClosingDay: 15,
		DueDay:     25,
	}
	db.Create(creditCard)

	// 3 installments of 200.00 each
	db.Create(&models.Installment{
		CreditCardID:      creditCard.ID,
		Description:       "Purchase",
		TotalAmount:       600.00,
		InstallmentAmount: 200.00,
		TotalInstallments: 3,
		StartDate:         time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local),
	})

	// Create bills
	db.Create(&models.Bill{
		AccountID: account.ID,
		Name:      "Electric Jan",
		Amount:    150.00,
		DueDate:   time.Date(2024, 1, 10, 0, 0, 0, 0, time.Local),
	})
	db.Create(&models.Bill{
		AccountID: account.ID,
		Name:      "Electric Feb",
		Amount:    180.00,
		DueDate:   time.Date(2024, 2, 10, 0, 0, 0, 0, time.Local),
	})

	comparison := GetMonthOverMonthComparison(db, 2024, 2, []uint{account.ID})

	// January expenses: 1000 (fixed) + 200 (card) + 150 (bill) = 1350
	if comparison.PreviousMonth.TotalExpenses != 1350.00 {
		t.Errorf("PreviousMonth TotalExpenses = %.2f, want 1350.00", comparison.PreviousMonth.TotalExpenses)
	}

	// February expenses: 1000 (fixed) + 200 (card) + 180 (bill) = 1380
	if comparison.CurrentMonth.TotalExpenses != 1380.00 {
		t.Errorf("CurrentMonth TotalExpenses = %.2f, want 1380.00", comparison.CurrentMonth.TotalExpenses)
	}

	// Expense change: (1380 - 1350) / 1350 * 100 = 2.22%
	expectedExpenseChangePercent := (30.0 / 1350.0) * 100
	tolerance := 0.01
	if comparison.ExpenseChangePercent < expectedExpenseChangePercent-tolerance ||
		comparison.ExpenseChangePercent > expectedExpenseChangePercent+tolerance {
		t.Errorf("ExpenseChangePercent = %.2f, want %.2f", comparison.ExpenseChangePercent, expectedExpenseChangePercent)
	}
}

func TestGetMonthOverMonthComparison_DecreaseScenario(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	// January has higher income
	db.Create(&models.Income{
		AccountID:   account.ID,
		Date:        time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 6000.00,
		TaxAmount:   600.00,
		NetAmount:   5400.00,
	})

	// February has lower income (33% decrease)
	db.Create(&models.Income{
		AccountID:   account.ID,
		Date:        time.Date(2024, 2, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 4000.00,
		TaxAmount:   400.00,
		NetAmount:   3600.00,
	})

	// Fixed expense
	db.Create(&models.Expense{
		AccountID: account.ID,
		Name:      "Rent",
		Amount:    1000.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	})

	comparison := GetMonthOverMonthComparison(db, 2024, 2, []uint{account.ID})

	// Income should show negative change
	// (4000 - 6000) / 6000 * 100 = -33.33%
	expectedIncomeChangePercent := -33.333333333333336
	tolerance := 0.01
	if comparison.IncomeChangePercent < expectedIncomeChangePercent-tolerance ||
		comparison.IncomeChangePercent > expectedIncomeChangePercent+tolerance {
		t.Errorf("IncomeChangePercent = %.2f, want %.2f (negative change)", comparison.IncomeChangePercent, expectedIncomeChangePercent)
	}

	if comparison.IncomeChange != -2000.00 {
		t.Errorf("IncomeChange = %.2f, want -2000.00", comparison.IncomeChange)
	}
}

func TestGetCategoryBreakdownWithPercentages(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	// Create expenses in different categories for January 2024
	// Total: 1000 + 500 + 300 = 1800
	expenses := []struct {
		name     string
		category string
		amount   float64
	}{
		{"Groceries", "Food", 1000.00},
		{"Gas", "Transportation", 500.00},
		{"Movie", "Entertainment", 300.00},
	}

	for _, exp := range expenses {
		expense := &models.Expense{
			AccountID: account.ID,
			Name:      exp.name,
			Category:  exp.category,
			Amount:    exp.amount,
			Type:      models.ExpenseTypeVariable,
			Active:    true,
		}
		db.Create(expense)
		// Set created_at to January 2024
		db.Model(expense).Update("created_at", time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local))
	}

	breakdown := GetCategoryBreakdownWithPercentages(db, 2024, 1, []uint{account.ID})

	// Should have 3 categories
	if len(breakdown) != 3 {
		t.Errorf("len(breakdown) = %d, want 3", len(breakdown))
	}

	// Verify each category has correct amount and percentage
	expectedBreakdown := map[string]struct {
		amount     float64
		percentage float64
	}{
		"Food":           {1000.00, 55.555555555555557}, // 1000/1800 * 100
		"Transportation": {500.00, 27.777777777777779},  // 500/1800 * 100
		"Entertainment":  {300.00, 16.666666666666668},  // 300/1800 * 100
	}

	tolerance := 0.01
	for _, item := range breakdown {
		expected, ok := expectedBreakdown[item.Category]
		if !ok {
			t.Errorf("Unexpected category: %s", item.Category)
			continue
		}

		if item.Amount != expected.amount {
			t.Errorf("Category %s: Amount = %.2f, want %.2f", item.Category, item.Amount, expected.amount)
		}

		if item.Percentage < expected.percentage-tolerance ||
			item.Percentage > expected.percentage+tolerance {
			t.Errorf("Category %s: Percentage = %.2f, want %.2f", item.Category, item.Percentage, expected.percentage)
		}
	}

	// Verify percentages add up to 100%
	var totalPercentage float64
	for _, item := range breakdown {
		totalPercentage += item.Percentage
	}

	if totalPercentage < 99.99 || totalPercentage > 100.01 {
		t.Errorf("Total percentage = %.2f, want ~100.00", totalPercentage)
	}
}

func TestGetCategoryBreakdownWithPercentages_EmptyAccounts(t *testing.T) {
	db := testutil.SetupTestDB()

	breakdown := GetCategoryBreakdownWithPercentages(db, 2024, 1, []uint{})

	// Should return nil for empty account list
	if breakdown != nil {
		t.Errorf("breakdown = %v, want nil for empty accounts", breakdown)
	}
}

func TestGetCategoryBreakdownWithPercentages_NoExpenses(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	breakdown := GetCategoryBreakdownWithPercentages(db, 2024, 1, []uint{account.ID})

	// Should return empty slice when no expenses
	if len(breakdown) != 0 {
		t.Errorf("len(breakdown) = %d, want 0 for no expenses", len(breakdown))
	}
}

func TestGetCategoryBreakdownWithPercentages_SingleCategory(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	// Create single expense
	expense := &models.Expense{
		AccountID: account.ID,
		Name:      "Groceries",
		Category:  "Food",
		Amount:    500.00,
		Type:      models.ExpenseTypeVariable,
		Active:    true,
	}
	db.Create(expense)
	db.Model(expense).Update("created_at", time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local))

	breakdown := GetCategoryBreakdownWithPercentages(db, 2024, 1, []uint{account.ID})

	// Should have 1 category with 100%
	if len(breakdown) != 1 {
		t.Errorf("len(breakdown) = %d, want 1", len(breakdown))
	}

	if breakdown[0].Category != "Food" {
		t.Errorf("Category = %s, want Food", breakdown[0].Category)
	}

	if breakdown[0].Amount != 500.00 {
		t.Errorf("Amount = %.2f, want 500.00", breakdown[0].Amount)
	}

	if breakdown[0].Percentage != 100.00 {
		t.Errorf("Percentage = %.2f, want 100.00 (single category should be 100%%)", breakdown[0].Percentage)
	}
}

func TestGetCategoryBreakdownWithPercentages_MultipleAccounts(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account1 := testutil.CreateTestAccount(db, "Account 1", models.AccountTypeIndividual, user.ID, nil)
	account2 := testutil.CreateTestAccount(db, "Account 2", models.AccountTypeIndividual, user.ID, nil)

	// Create expenses in account1
	expense1 := &models.Expense{
		AccountID: account1.ID,
		Name:      "Groceries",
		Category:  "Food",
		Amount:    600.00,
		Type:      models.ExpenseTypeVariable,
		Active:    true,
	}
	db.Create(expense1)
	db.Model(expense1).Update("created_at", time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local))

	// Create expenses in account2
	expense2 := &models.Expense{
		AccountID: account2.ID,
		Name:      "Gas",
		Category:  "Transportation",
		Amount:    400.00,
		Type:      models.ExpenseTypeVariable,
		Active:    true,
	}
	db.Create(expense2)
	db.Model(expense2).Update("created_at", time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local))

	// Query both accounts together
	breakdown := GetCategoryBreakdownWithPercentages(db, 2024, 1, []uint{account1.ID, account2.ID})

	// Should have 2 categories
	if len(breakdown) != 2 {
		t.Errorf("len(breakdown) = %d, want 2", len(breakdown))
	}

	// Total: 600 + 400 = 1000
	// Food: 600/1000 = 60%
	// Transportation: 400/1000 = 40%
	expectedBreakdown := map[string]struct {
		amount     float64
		percentage float64
	}{
		"Food":           {600.00, 60.0},
		"Transportation": {400.00, 40.0},
	}

	for _, item := range breakdown {
		expected, ok := expectedBreakdown[item.Category]
		if !ok {
			t.Errorf("Unexpected category: %s", item.Category)
			continue
		}

		if item.Amount != expected.amount {
			t.Errorf("Category %s: Amount = %.2f, want %.2f", item.Category, item.Amount, expected.amount)
		}

		if item.Percentage != expected.percentage {
			t.Errorf("Category %s: Percentage = %.2f, want %.2f", item.Category, item.Percentage, expected.percentage)
		}
	}
}
