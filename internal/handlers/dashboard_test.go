package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/database"
	"poc-finance/internal/middleware"
	"poc-finance/internal/models"
	"poc-finance/internal/services"
	"poc-finance/internal/testutil"
)

func setupDashboardTestHandler() (*DashboardHandler, *echo.Echo, uint, uint) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create test user
	authService := services.NewAuthService()
	user, _ := authService.Register("dashboard@example.com", "Password123", "Dashboard User")

	// Create test account for the user
	account := models.Account{
		Name:   "Test Account",
		Type:   models.AccountTypeIndividual,
		UserID: user.ID,
	}
	database.DB.Create(&account)

	// Initialize settings cache
	cacheService := services.NewSettingsCacheService()

	e := echo.New()
	e.Renderer = &testutil.MockRenderer{}
	handler := NewDashboardHandler(cacheService)
	return handler, e, user.ID, account.ID
}

func TestDashboardHandler_Index_Success(t *testing.T) {
	handler, e, userID, accountID := setupDashboardTestHandler()

	// Create some test data
	now := time.Now()
	income := models.Income{
		AccountID:   accountID,
		Date:        time.Date(now.Year(), now.Month(), 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 10000.00,
		NetAmount:   8500.00,
		TaxAmount:   1500.00,
		Description: "Test Income",
	}
	database.DB.Create(&income)

	expense := models.Expense{
		AccountID: accountID,
		Name:      "Test Expense",
		Amount:    500.00,
		Type:      models.ExpenseTypeFixed,
		DueDay:    10,
		Active:    true,
	}
	database.DB.Create(&expense)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.Index(c)
	if err != nil {
		t.Fatalf("Index() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestDashboardHandler_Index_WithAccountFilter(t *testing.T) {
	handler, e, userID, accountID := setupDashboardTestHandler()

	// Create income for the account
	now := time.Now()
	income := models.Income{
		AccountID:   accountID,
		Date:        time.Date(now.Year(), now.Month(), 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 5000.00,
		NetAmount:   4250.00,
		TaxAmount:   750.00,
		Description: "Filtered Income",
	}
	database.DB.Create(&income)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/?account_id=%d", accountID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.Index(c)
	if err != nil {
		t.Fatalf("Index() with account filter returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestDashboardHandler_Index_WithAllAccountsFilter(t *testing.T) {
	handler, e, userID, accountID := setupDashboardTestHandler()

	// Create second account
	account2 := models.Account{
		Name:   "Second Account",
		Type:   models.AccountTypeIndividual,
		UserID: userID,
	}
	database.DB.Create(&account2)

	// Create income for both accounts
	now := time.Now()
	income1 := models.Income{
		AccountID:   accountID,
		Date:        time.Date(now.Year(), now.Month(), 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 5000.00,
		NetAmount:   4250.00,
		TaxAmount:   750.00,
		Description: "Income 1",
	}
	database.DB.Create(&income1)

	income2 := models.Income{
		AccountID:   account2.ID,
		Date:        time.Date(now.Year(), now.Month(), 20, 0, 0, 0, 0, time.Local),
		GrossAmount: 3000.00,
		NetAmount:   2550.00,
		TaxAmount:   450.00,
		Description: "Income 2",
	}
	database.DB.Create(&income2)

	req := httptest.NewRequest(http.MethodGet, "/?account_id=all", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.Index(c)
	if err != nil {
		t.Fatalf("Index() with all accounts filter returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestDashboardHandler_Index_WithInvalidAccountID(t *testing.T) {
	handler, e, userID, _ := setupDashboardTestHandler()

	// Try to access an account that doesn't belong to the user
	invalidAccountID := uint(99999)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/?account_id=%d", invalidAccountID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.Index(c)
	if err != nil {
		t.Fatalf("Index() with invalid account ID returned error: %v", err)
	}

	// Should fallback to all accounts (status OK)
	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestDashboardHandler_Index_WithInvalidAccountIDFormat(t *testing.T) {
	handler, e, userID, _ := setupDashboardTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/?account_id=invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.Index(c)
	if err != nil {
		t.Fatalf("Index() with invalid account ID format returned error: %v", err)
	}

	// Should fallback to all accounts (status OK)
	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestDashboardHandler_Index_NoAccounts(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create user without any accounts
	authService := services.NewAuthService()
	user, _ := authService.Register("noaccounts@example.com", "Password123", "No Accounts User")

	cacheService := services.NewSettingsCacheService()
	e := echo.New()
	e.Renderer = &testutil.MockRenderer{}
	handler := NewDashboardHandler(cacheService)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, user.ID)

	err := handler.Index(c)
	if err != nil {
		t.Fatalf("Index() with no accounts returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestDashboardHandler_Index_WithUpcomingBills(t *testing.T) {
	handler, e, userID, accountID := setupDashboardTestHandler()

	now := time.Now()

	// Create fixed expense with due date in the future
	expense := models.Expense{
		AccountID: accountID,
		Name:      "Monthly Rent",
		Amount:    1500.00,
		Type:      models.ExpenseTypeFixed,
		DueDay:    25,
		Active:    true,
		Category:  "Housing",
	}
	database.DB.Create(&expense)

	// Create bill due in 5 days
	bill := models.Bill{
		AccountID: accountID,
		Name:      "Electric Bill",
		Amount:    200.00,
		DueDate:   now.AddDate(0, 0, 5),
		Paid:      false,
		Category:  "Utilities",
	}
	database.DB.Create(&bill)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.Index(c)
	if err != nil {
		t.Fatalf("Index() with upcoming bills returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestDashboardHandler_Index_WithPaidBill(t *testing.T) {
	handler, e, userID, accountID := setupDashboardTestHandler()

	now := time.Now()

	// Create paid bill (should not appear in upcoming bills)
	bill := models.Bill{
		AccountID: accountID,
		Name:      "Paid Bill",
		Amount:    100.00,
		DueDate:   now.AddDate(0, 0, 5),
		Paid:      true,
		Category:  "Utilities",
	}
	database.DB.Create(&bill)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.Index(c)
	if err != nil {
		t.Fatalf("Index() with paid bill returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestDashboardHandler_Index_WithInactiveExpense(t *testing.T) {
	handler, e, userID, accountID := setupDashboardTestHandler()

	// Create inactive expense (should not appear in upcoming bills)
	expense := models.Expense{
		AccountID: accountID,
		Name:      "Inactive Expense",
		Amount:    500.00,
		Type:      models.ExpenseTypeFixed,
		DueDay:    10,
		Active:    false,
	}
	database.DB.Create(&expense)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.Index(c)
	if err != nil {
		t.Fatalf("Index() with inactive expense returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestDashboardHandler_Index_WithMultipleAccounts(t *testing.T) {
	handler, e, userID, accountID := setupDashboardTestHandler()

	// Create second account
	account2 := models.Account{
		Name:   "Business Account",
		Type:   models.AccountTypeIndividual,
		UserID: userID,
	}
	database.DB.Create(&account2)

	now := time.Now()

	// Create data for both accounts
	income1 := models.Income{
		AccountID:   accountID,
		Date:        time.Date(now.Year(), now.Month(), 10, 0, 0, 0, 0, time.Local),
		GrossAmount: 5000.00,
		NetAmount:   4250.00,
		TaxAmount:   750.00,
		Description: "Personal Income",
	}
	database.DB.Create(&income1)

	income2 := models.Income{
		AccountID:   account2.ID,
		Date:        time.Date(now.Year(), now.Month(), 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 8000.00,
		NetAmount:   6800.00,
		TaxAmount:   1200.00,
		Description: "Business Income",
	}
	database.DB.Create(&income2)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.Index(c)
	if err != nil {
		t.Fatalf("Index() with multiple accounts returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestDashboardHandler_Index_With6MonthProjections(t *testing.T) {
	handler, e, userID, accountID := setupDashboardTestHandler()

	now := time.Now()

	// Create income for current month
	income := models.Income{
		AccountID:   accountID,
		Date:        time.Date(now.Year(), now.Month(), 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 10000.00,
		NetAmount:   8500.00,
		TaxAmount:   1500.00,
		Description: "Monthly Income",
	}
	database.DB.Create(&income)

	// Create fixed expense (should appear in all 6 months)
	expense := models.Expense{
		AccountID: accountID,
		Name:      "Fixed Monthly Expense",
		Amount:    1000.00,
		Type:      models.ExpenseTypeFixed,
		DueDay:    5,
		Active:    true,
	}
	database.DB.Create(&expense)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.Index(c)
	if err != nil {
		t.Fatalf("Index() with 6-month projections returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestGetUpcomingBillsForAccounts_EmptyAccountIDs(t *testing.T) {
	now := time.Now()
	bills := getUpcomingBillsForAccounts(now, []uint{})

	if len(bills) != 0 {
		t.Errorf("Expected 0 bills for empty account IDs, got %d", len(bills))
	}
}

func TestGetUpcomingBillsForAccounts_WithFixedExpenses(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create test account
	account := models.Account{
		Name:   "Test Account",
		Type:   models.AccountTypeIndividual,
		UserID: 1,
	}
	database.DB.Create(&account)

	now := time.Now()

	// Create fixed expense with due date in the future
	expense := models.Expense{
		AccountID: account.ID,
		Name:      "Upcoming Fixed Expense",
		Amount:    500.00,
		Type:      models.ExpenseTypeFixed,
		DueDay:    25,
		Active:    true,
	}
	database.DB.Create(&expense)

	bills := getUpcomingBillsForAccounts(now, []uint{account.ID})

	// Should have at least one upcoming bill
	if len(bills) == 0 {
		t.Error("Expected at least 1 upcoming bill from fixed expense")
	}
}

func TestGetUpcomingBillsForAccounts_WithBills(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create test account
	account := models.Account{
		Name:   "Test Account",
		Type:   models.AccountTypeIndividual,
		UserID: 1,
	}
	database.DB.Create(&account)

	now := time.Now()

	// Create unpaid bill due in 10 days
	bill := models.Bill{
		AccountID: account.ID,
		Name:      "Upcoming Bill",
		Amount:    300.00,
		DueDate:   now.AddDate(0, 0, 10),
		Paid:      false,
	}
	database.DB.Create(&bill)

	bills := getUpcomingBillsForAccounts(now, []uint{account.ID})

	if len(bills) == 0 {
		t.Error("Expected at least 1 upcoming bill")
	}

	// Verify bill details
	found := false
	for _, b := range bills {
		if b.Name == "Upcoming Bill" && b.Amount == 300.00 {
			found = true
			if b.Type != "bill" {
				t.Errorf("Bill type = %s, want 'bill'", b.Type)
			}
			if b.DueIn != 10 {
				t.Errorf("DueIn = %d, want 10", b.DueIn)
			}
		}
	}

	if !found {
		t.Error("Expected to find 'Upcoming Bill' in results")
	}
}

func TestGetUpcomingBillsForAccounts_PastDueBillsNotIncluded(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create test account
	account := models.Account{
		Name:   "Test Account",
		Type:   models.AccountTypeIndividual,
		UserID: 1,
	}
	database.DB.Create(&account)

	now := time.Now()

	// Create bill that's past the 30-day window
	bill := models.Bill{
		AccountID: account.ID,
		Name:      "Far Future Bill",
		Amount:    200.00,
		DueDate:   now.AddDate(0, 0, 45), // 45 days in the future
		Paid:      false,
	}
	database.DB.Create(&bill)

	bills := getUpcomingBillsForAccounts(now, []uint{account.ID})

	// Should not include bills beyond 30 days
	for _, b := range bills {
		if b.Name == "Far Future Bill" {
			t.Error("Should not include bills beyond 30 days")
		}
	}
}

func TestGetUpcomingBillsForAccounts_PaidBillsNotIncluded(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create test account
	account := models.Account{
		Name:   "Test Account",
		Type:   models.AccountTypeIndividual,
		UserID: 1,
	}
	database.DB.Create(&account)

	now := time.Now()

	// Create paid bill
	bill := models.Bill{
		AccountID: account.ID,
		Name:      "Paid Bill",
		Amount:    150.00,
		DueDate:   now.AddDate(0, 0, 5),
		Paid:      true,
	}
	database.DB.Create(&bill)

	bills := getUpcomingBillsForAccounts(now, []uint{account.ID})

	// Should not include paid bills
	for _, b := range bills {
		if b.Name == "Paid Bill" {
			t.Error("Should not include paid bills")
		}
	}
}

func TestGetUpcomingBillsForAccounts_InactiveExpensesNotIncluded(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create test account
	account := models.Account{
		Name:   "Test Account",
		Type:   models.AccountTypeIndividual,
		UserID: 1,
	}
	database.DB.Create(&account)

	now := time.Now()

	// Create inactive expense
	expense := models.Expense{
		AccountID: account.ID,
		Name:      "Inactive Expense",
		Amount:    400.00,
		Type:      models.ExpenseTypeFixed,
		DueDay:    15,
		Active:    true, // Create as active first
	}
	database.DB.Create(&expense)

	// Then update to inactive (workaround for GORM default value issue)
	database.DB.Model(&expense).Update("active", false)

	bills := getUpcomingBillsForAccounts(now, []uint{account.ID})

	// Should not include inactive expenses
	for _, b := range bills {
		if b.Name == "Inactive Expense" {
			t.Error("Should not include inactive expenses")
		}
	}
}

func TestGetUpcomingBillsForAccounts_VariableExpensesNotIncluded(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create test account
	account := models.Account{
		Name:   "Test Account",
		Type:   models.AccountTypeIndividual,
		UserID: 1,
	}
	database.DB.Create(&account)

	now := time.Now()

	// Create variable expense (should not be in upcoming bills)
	expense := models.Expense{
		AccountID: account.ID,
		Name:      "Variable Expense",
		Amount:    250.00,
		Type:      models.ExpenseTypeVariable,
		DueDay:    10,
		Active:    true,
	}
	database.DB.Create(&expense)

	bills := getUpcomingBillsForAccounts(now, []uint{account.ID})

	// Should not include variable expenses
	for _, b := range bills {
		if b.Name == "Variable Expense" {
			t.Error("Should not include variable expenses in upcoming bills")
		}
	}
}

func TestGetUpcomingBillsForAccounts_SortedByDueDate(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create test account
	account := models.Account{
		Name:   "Test Account",
		Type:   models.AccountTypeIndividual,
		UserID: 1,
	}
	database.DB.Create(&account)

	now := time.Now()

	// Create bills with different due dates
	bill1 := models.Bill{
		AccountID: account.ID,
		Name:      "Bill 1",
		Amount:    100.00,
		DueDate:   now.AddDate(0, 0, 20),
		Paid:      false,
	}
	database.DB.Create(&bill1)

	bill2 := models.Bill{
		AccountID: account.ID,
		Name:      "Bill 2",
		Amount:    200.00,
		DueDate:   now.AddDate(0, 0, 5),
		Paid:      false,
	}
	database.DB.Create(&bill2)

	bill3 := models.Bill{
		AccountID: account.ID,
		Name:      "Bill 3",
		Amount:    150.00,
		DueDate:   now.AddDate(0, 0, 15),
		Paid:      false,
	}
	database.DB.Create(&bill3)

	bills := getUpcomingBillsForAccounts(now, []uint{account.ID})

	if len(bills) < 2 {
		t.Fatal("Expected at least 2 bills to test sorting")
	}

	// Verify bills are sorted by due date
	for i := 0; i < len(bills)-1; i++ {
		if bills[i].DueDate.After(bills[i+1].DueDate) {
			t.Errorf("Bills not sorted by due date: bills[%d].DueDate = %v is after bills[%d].DueDate = %v",
				i, bills[i].DueDate, i+1, bills[i+1].DueDate)
		}
	}
}

func TestGetUpcomingBillsForAccounts_LimitTo10Items(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create test account
	account := models.Account{
		Name:   "Test Account",
		Type:   models.AccountTypeIndividual,
		UserID: 1,
	}
	database.DB.Create(&account)

	now := time.Now()

	// Create more than 10 bills
	for i := 1; i <= 15; i++ {
		bill := models.Bill{
			AccountID: account.ID,
			Name:      fmt.Sprintf("Bill %d", i),
			Amount:    float64(i * 100),
			DueDate:   now.AddDate(0, 0, i),
			Paid:      false,
		}
		database.DB.Create(&bill)
	}

	bills := getUpcomingBillsForAccounts(now, []uint{account.ID})

	// Should limit to 10 items
	if len(bills) > 10 {
		t.Errorf("Expected max 10 upcoming bills, got %d", len(bills))
	}
}

func TestGetUpcomingBillsForAccounts_MultipleAccounts(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create two test accounts
	account1 := models.Account{
		Name:   "Account 1",
		Type:   models.AccountTypeIndividual,
		UserID: 1,
	}
	database.DB.Create(&account1)

	account2 := models.Account{
		Name:   "Account 2",
		Type:   models.AccountTypeIndividual,
		UserID: 1,
	}
	database.DB.Create(&account2)

	now := time.Now()

	// Create bill for account 1
	bill1 := models.Bill{
		AccountID: account1.ID,
		Name:      "Account 1 Bill",
		Amount:    100.00,
		DueDate:   now.AddDate(0, 0, 5),
		Paid:      false,
	}
	database.DB.Create(&bill1)

	// Create bill for account 2
	bill2 := models.Bill{
		AccountID: account2.ID,
		Name:      "Account 2 Bill",
		Amount:    200.00,
		DueDate:   now.AddDate(0, 0, 10),
		Paid:      false,
	}
	database.DB.Create(&bill2)

	// Get bills for both accounts
	bills := getUpcomingBillsForAccounts(now, []uint{account1.ID, account2.ID})

	if len(bills) < 2 {
		t.Errorf("Expected at least 2 bills from both accounts, got %d", len(bills))
	}

	// Verify both bills are present
	foundAccount1 := false
	foundAccount2 := false
	for _, b := range bills {
		if b.Name == "Account 1 Bill" {
			foundAccount1 = true
		}
		if b.Name == "Account 2 Bill" {
			foundAccount2 = true
		}
	}

	if !foundAccount1 {
		t.Error("Expected to find bill from account 1")
	}
	if !foundAccount2 {
		t.Error("Expected to find bill from account 2")
	}
}

func TestGetUpcomingBillsForAccounts_ExpenseDueDateCalculation(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create test account
	account := models.Account{
		Name:   "Test Account",
		Type:   models.AccountTypeIndividual,
		UserID: 1,
	}
	database.DB.Create(&account)

	// Use a specific date for predictable testing
	now := time.Date(2024, 1, 10, 0, 0, 0, 0, time.Local)

	// Create expense with due day on 25th (should be this month)
	expense1 := models.Expense{
		AccountID: account.ID,
		Name:      "Expense Due 25th",
		Amount:    500.00,
		Type:      models.ExpenseTypeFixed,
		DueDay:    25,
		Active:    true,
	}
	database.DB.Create(&expense1)

	// Create expense with due day on 5th (already passed, should be next month)
	expense2 := models.Expense{
		AccountID: account.ID,
		Name:      "Expense Due 5th",
		Amount:    300.00,
		Type:      models.ExpenseTypeFixed,
		DueDay:    5,
		Active:    true,
	}
	database.DB.Create(&expense2)

	bills := getUpcomingBillsForAccounts(now, []uint{account.ID})

	if len(bills) == 0 {
		t.Fatal("Expected at least 1 upcoming bill from expenses")
	}

	// Verify expense with due day 25th is this month
	for _, b := range bills {
		if b.Name == "Expense Due 25th" {
			expectedDate := time.Date(2024, 1, 25, 0, 0, 0, 0, time.Local)
			if !b.DueDate.Equal(expectedDate) {
				t.Errorf("Expense Due 25th: DueDate = %v, want %v", b.DueDate, expectedDate)
			}
			if b.Type != "expense" {
				t.Errorf("Type = %s, want 'expense'", b.Type)
			}
		}
		if b.Name == "Expense Due 5th" {
			// Should be February 5th since January 5th already passed
			expectedDate := time.Date(2024, 2, 5, 0, 0, 0, 0, time.Local)
			if !b.DueDate.Equal(expectedDate) {
				t.Errorf("Expense Due 5th: DueDate = %v, want %v", b.DueDate, expectedDate)
			}
		}
	}
}
