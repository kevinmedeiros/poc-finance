package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/database"
	"poc-finance/internal/middleware"
	"poc-finance/internal/models"
	"poc-finance/internal/services"
	"poc-finance/internal/testutil"
)

func setupExpenseTestHandler() (*ExpenseHandler, *echo.Echo, uint, uint) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create test user
	authService := services.NewAuthService()
	user, _ := authService.Register("test@example.com", "Password123", "Test User")

	// Create test account for the user
	account := models.Account{
		Name:   "Test Account",
		Type:   models.AccountTypeIndividual,
		UserID: user.ID,
	}
	database.DB.Create(&account)

	e := echo.New()
	handler := NewExpenseHandler()
	return handler, e, user.ID, account.ID
}

func TestExpenseHandler_Create_Success_Fixed(t *testing.T) {
	handler, e, userID, accountID := setupExpenseTestHandler()

	form := url.Values{}
	form.Set("account_id", fmt.Sprintf("%d", accountID))
	form.Set("name", "Test Fixed Expense")
	form.Set("amount", "100.00")
	form.Set("type", "fixed")
	form.Set("due_day", "15")
	form.Set("category", "Moradia")

	req := httptest.NewRequest(http.MethodPost, "/expenses", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	// Mock renderer to avoid nil pointer
	e.Renderer = &testutil.MockRenderer{}

	err := handler.Create(c)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	// Verify expense was created
	var expense models.Expense
	if err := database.DB.Where("account_id = ?", accountID).First(&expense).Error; err != nil {
		t.Fatalf("Failed to find created expense: %v", err)
	}

	if expense.Name != "Test Fixed Expense" {
		t.Errorf("Name = %s, want %s", expense.Name, "Test Fixed Expense")
	}

	if expense.Amount != 100.00 {
		t.Errorf("Amount = %f, want %f", expense.Amount, 100.00)
	}

	if expense.Type != models.ExpenseTypeFixed {
		t.Errorf("Type = %s, want %s", expense.Type, models.ExpenseTypeFixed)
	}

	if expense.DueDay != 15 {
		t.Errorf("DueDay = %d, want %d", expense.DueDay, 15)
	}

	if expense.Category != "Moradia" {
		t.Errorf("Category = %s, want %s", expense.Category, "Moradia")
	}

	if !expense.Active {
		t.Errorf("Active = %v, want %v", expense.Active, true)
	}

	if expense.IsSplit {
		t.Errorf("IsSplit = %v, want %v", expense.IsSplit, false)
	}
}

func TestExpenseHandler_Create_Success_Variable(t *testing.T) {
	handler, e, userID, accountID := setupExpenseTestHandler()

	form := url.Values{}
	form.Set("account_id", fmt.Sprintf("%d", accountID))
	form.Set("name", "Test Variable Expense")
	form.Set("amount", "50.00")
	form.Set("type", "variable")
	form.Set("category", "Lazer")

	req := httptest.NewRequest(http.MethodPost, "/expenses", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	e.Renderer = &testutil.MockRenderer{}

	err := handler.Create(c)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	// Verify expense was created
	var expense models.Expense
	if err := database.DB.Where("account_id = ?", accountID).First(&expense).Error; err != nil {
		t.Fatalf("Failed to find created expense: %v", err)
	}

	if expense.Type != models.ExpenseTypeVariable {
		t.Errorf("Type = %s, want %s", expense.Type, models.ExpenseTypeVariable)
	}

	if expense.Amount != 50.00 {
		t.Errorf("Amount = %f, want %f", expense.Amount, 50.00)
	}
}

func TestExpenseHandler_Create_WithSplit(t *testing.T) {
	handler, e, userID, accountID := setupExpenseTestHandler()

	// Create additional user for split
	authService := services.NewAuthService()
	user2, _ := authService.Register("test2@example.com", "Password123", "Test User 2")

	form := url.Values{}
	form.Set("account_id", fmt.Sprintf("%d", accountID))
	form.Set("name", "Split Expense")
	form.Set("amount", "200.00")
	form.Set("type", "fixed")
	form.Set("due_day", "10")
	form.Set("category", "Alimentação")
	form.Set("is_split", "true")
	form.Add("split_user_ids", fmt.Sprintf("%d", userID))
	form.Add("split_user_ids", fmt.Sprintf("%d", user2.ID))
	form.Add("split_percentages", "60")
	form.Add("split_percentages", "40")

	req := httptest.NewRequest(http.MethodPost, "/expenses", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	// Parse the form to populate req.Form
	req.ParseForm()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	e.Renderer = &testutil.MockRenderer{}

	err := handler.Create(c)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	// Verify expense was created
	var expense models.Expense
	if err := database.DB.Preload("Splits").Where("account_id = ? AND name = ?", accountID, "Split Expense").First(&expense).Error; err != nil {
		t.Fatalf("Failed to find created expense: %v", err)
	}

	if !expense.IsSplit {
		t.Errorf("IsSplit = %v, want %v", expense.IsSplit, true)
	}

	if len(expense.Splits) != 2 {
		t.Fatalf("Number of splits = %d, want %d", len(expense.Splits), 2)
	}

	// Verify split amounts
	split1 := expense.Splits[0]
	split2 := expense.Splits[1]

	if split1.Percentage != 60 {
		t.Errorf("Split1 Percentage = %f, want %f", split1.Percentage, 60.0)
	}

	if split1.Amount != 120.00 {
		t.Errorf("Split1 Amount = %f, want %f", split1.Amount, 120.00)
	}

	if split2.Percentage != 40 {
		t.Errorf("Split2 Percentage = %f, want %f", split2.Percentage, 40.0)
	}

	if split2.Amount != 80.00 {
		t.Errorf("Split2 Amount = %f, want %f", split2.Amount, 80.00)
	}
}

func TestExpenseHandler_Create_WithSplit_InvalidPercentage(t *testing.T) {
	handler, e, userID, accountID := setupExpenseTestHandler()

	authService := services.NewAuthService()
	user2, _ := authService.Register("test2@example.com", "Password123", "Test User 2")

	form := url.Values{}
	form.Set("account_id", fmt.Sprintf("%d", accountID))
	form.Set("name", "Split Expense Bad")
	form.Set("amount", "200.00")
	form.Set("type", "fixed")
	form.Set("is_split", "true")
	form.Add("split_user_ids", fmt.Sprintf("%d", userID))
	form.Add("split_user_ids", fmt.Sprintf("%d", user2.ID))
	form.Add("split_percentages", "60")
	form.Add("split_percentages", "30") // Total = 90%, not 100%

	req := httptest.NewRequest(http.MethodPost, "/expenses", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	// Parse the form to populate req.Form
	req.ParseForm()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.Create(c)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	if !strings.Contains(rec.Body.String(), "A soma dos percentuais deve ser 100%") {
		t.Errorf("Response body = %s, want to contain 'A soma dos percentuais deve ser 100%%'", rec.Body.String())
	}

	// Verify expense was NOT created
	var count int64
	database.DB.Model(&models.Expense{}).Where("account_id = ?", accountID).Count(&count)
	if count != 0 {
		t.Errorf("Expense count = %d, want %d (should be rolled back)", count, 0)
	}
}

func TestExpenseHandler_Create_Unauthorized_Account(t *testing.T) {
	handler, e, userID, _ := setupExpenseTestHandler()

	// Create another user with their own account
	authService := services.NewAuthService()
	user2, _ := authService.Register("test2@example.com", "Password123", "Test User 2")
	account2 := models.Account{
		Name:   "Other User Account",
		Type:   models.AccountTypeIndividual,
		UserID: user2.ID,
	}
	database.DB.Create(&account2)

	form := url.Values{}
	form.Set("account_id", fmt.Sprintf("%d", account2.ID)) // Try to create expense in another user's account
	form.Set("name", "Unauthorized Expense")
	form.Set("amount", "100.00")
	form.Set("type", "fixed")

	req := httptest.NewRequest(http.MethodPost, "/expenses", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.Create(c)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if rec.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	if !strings.Contains(rec.Body.String(), "Acesso negado à conta selecionada") {
		t.Errorf("Response body = %s, want to contain 'Acesso negado à conta selecionada'", rec.Body.String())
	}
}

func TestExpenseHandler_Create_NoAccountSpecified_UsesIndividual(t *testing.T) {
	handler, e, userID, _ := setupExpenseTestHandler()

	form := url.Values{}
	// No account_id specified
	form.Set("name", "Default Account Expense")
	form.Set("amount", "100.00")
	form.Set("type", "fixed")

	req := httptest.NewRequest(http.MethodPost, "/expenses", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	e.Renderer = &testutil.MockRenderer{}

	err := handler.Create(c)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	// Verify expense was created in user's individual account
	var expense models.Expense
	if err := database.DB.Preload("Account").Where("name = ?", "Default Account Expense").First(&expense).Error; err != nil {
		t.Fatalf("Failed to find created expense: %v", err)
	}

	if expense.Account.Type != models.AccountTypeIndividual {
		t.Errorf("Account Type = %s, want %s", expense.Account.Type, models.AccountTypeIndividual)
	}

	if expense.Account.UserID != userID {
		t.Errorf("Account UserID = %d, want %d", expense.Account.UserID, userID)
	}
}

func TestExpenseHandler_Toggle_Success(t *testing.T) {
	handler, e, userID, accountID := setupExpenseTestHandler()

	// Create an expense
	expense := models.Expense{
		AccountID: accountID,
		Name:      "Toggle Test",
		Amount:    100.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	}
	database.DB.Create(&expense)

	req := httptest.NewRequest(http.MethodPost, "/expenses/"+fmt.Sprintf("%d", expense.ID)+"/toggle", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", expense.ID))
	c.Set(middleware.UserIDKey, userID)

	e.Renderer = &testutil.MockRenderer{}

	err := handler.Toggle(c)
	if err != nil {
		t.Fatalf("Toggle() returned error: %v", err)
	}

	// Verify expense was toggled
	var updated models.Expense
	database.DB.First(&updated, expense.ID)
	if updated.Active {
		t.Errorf("Active = %v, want %v", updated.Active, false)
	}

	// Toggle again
	req = httptest.NewRequest(http.MethodPost, "/expenses/"+fmt.Sprintf("%d", expense.ID)+"/toggle", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", expense.ID))
	c.Set(middleware.UserIDKey, userID)

	err = handler.Toggle(c)
	if err != nil {
		t.Fatalf("Toggle() returned error: %v", err)
	}

	// Verify expense was toggled back
	database.DB.First(&updated, expense.ID)
	if !updated.Active {
		t.Errorf("Active = %v, want %v", updated.Active, true)
	}
}

func TestExpenseHandler_Toggle_NotFound(t *testing.T) {
	handler, e, userID, _ := setupExpenseTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/expenses/99999/toggle", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999")
	c.Set(middleware.UserIDKey, userID)

	err := handler.Toggle(c)
	if err != nil {
		t.Fatalf("Toggle() returned error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	if !strings.Contains(rec.Body.String(), "Despesa não encontrada") {
		t.Errorf("Response body = %s, want to contain 'Despesa não encontrada'", rec.Body.String())
	}
}

func TestExpenseHandler_Delete_Success(t *testing.T) {
	handler, e, userID, accountID := setupExpenseTestHandler()

	// Create an expense
	expense := models.Expense{
		AccountID: accountID,
		Name:      "Delete Test",
		Amount:    100.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	}
	database.DB.Create(&expense)

	req := httptest.NewRequest(http.MethodDelete, "/expenses/"+fmt.Sprintf("%d", expense.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", expense.ID))
	c.Set(middleware.UserIDKey, userID)

	e.Renderer = &testutil.MockRenderer{}

	err := handler.Delete(c)
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	// Verify expense was deleted
	var count int64
	database.DB.Model(&models.Expense{}).Where("id = ?", expense.ID).Count(&count)
	if count != 0 {
		t.Errorf("Expense count = %d, want %d (should be deleted)", count, 0)
	}
}

func TestExpenseHandler_Delete_NotFound(t *testing.T) {
	handler, e, userID, _ := setupExpenseTestHandler()

	req := httptest.NewRequest(http.MethodDelete, "/expenses/99999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999")
	c.Set(middleware.UserIDKey, userID)

	err := handler.Delete(c)
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	if !strings.Contains(rec.Body.String(), "Despesa não encontrada") {
		t.Errorf("Response body = %s, want to contain 'Despesa não encontrada'", rec.Body.String())
	}
}

func TestExpenseHandler_Delete_Unauthorized(t *testing.T) {
	handler, e, userID, _ := setupExpenseTestHandler()

	// Create another user with their own account and expense
	authService := services.NewAuthService()
	user2, _ := authService.Register("test2@example.com", "Password123", "Test User 2")
	account2 := models.Account{
		Name:   "Other User Account",
		Type:   models.AccountTypeIndividual,
		UserID: user2.ID,
	}
	database.DB.Create(&account2)

	expense := models.Expense{
		AccountID: account2.ID,
		Name:      "Other User Expense",
		Amount:    100.00,
		Type:      models.ExpenseTypeFixed,
	}
	database.DB.Create(&expense)

	req := httptest.NewRequest(http.MethodDelete, "/expenses/"+fmt.Sprintf("%d", expense.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", expense.ID))
	c.Set(middleware.UserIDKey, userID)

	err := handler.Delete(c)
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	// Verify expense was NOT deleted
	var count int64
	database.DB.Model(&models.Expense{}).Where("id = ?", expense.ID).Count(&count)
	if count == 0 {
		t.Errorf("Expense count = %d, want %d (should not be deleted)", count, 1)
	}
}

func TestExpenseHandler_MarkPaid_Success(t *testing.T) {
	handler, e, userID, accountID := setupExpenseTestHandler()

	// Create an expense
	expense := models.Expense{
		AccountID: accountID,
		Name:      "Mark Paid Test",
		Amount:    100.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	}
	database.DB.Create(&expense)

	req := httptest.NewRequest(http.MethodPost, "/expenses/"+fmt.Sprintf("%d", expense.ID)+"/mark-paid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", expense.ID))
	c.Set(middleware.UserIDKey, userID)

	e.Renderer = &testutil.MockRenderer{}

	err := handler.MarkPaid(c)
	if err != nil {
		t.Fatalf("MarkPaid() returned error: %v", err)
	}

	// Verify payment was created
	now := time.Now()
	month := int(now.Month())
	year := now.Year()

	var payment models.ExpensePayment
	if err := database.DB.Where("expense_id = ? AND month = ? AND year = ?", expense.ID, month, year).First(&payment).Error; err != nil {
		t.Fatalf("Failed to find payment: %v", err)
	}

	if payment.Amount != expense.Amount {
		t.Errorf("Payment Amount = %f, want %f", payment.Amount, expense.Amount)
	}
}

func TestExpenseHandler_MarkPaid_AlreadyPaid(t *testing.T) {
	handler, e, userID, accountID := setupExpenseTestHandler()

	// Create an expense
	expense := models.Expense{
		AccountID: accountID,
		Name:      "Already Paid Test",
		Amount:    100.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	}
	database.DB.Create(&expense)

	now := time.Now()
	month := int(now.Month())
	year := now.Year()

	// Create existing payment
	payment := models.ExpensePayment{
		ExpenseID: expense.ID,
		Month:     month,
		Year:      year,
		PaidAt:    now,
		Amount:    expense.Amount,
	}
	database.DB.Create(&payment)

	req := httptest.NewRequest(http.MethodPost, "/expenses/"+fmt.Sprintf("%d", expense.ID)+"/mark-paid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", expense.ID))
	c.Set(middleware.UserIDKey, userID)

	e.Renderer = &testutil.MockRenderer{}

	err := handler.MarkPaid(c)
	if err != nil {
		t.Fatalf("MarkPaid() returned error: %v", err)
	}

	// Verify only one payment exists (no duplicate)
	var count int64
	database.DB.Model(&models.ExpensePayment{}).Where("expense_id = ? AND month = ? AND year = ?", expense.ID, month, year).Count(&count)
	if count != 1 {
		t.Errorf("Payment count = %d, want %d (should not create duplicate)", count, 1)
	}
}

func TestExpenseHandler_MarkPaid_NotFound(t *testing.T) {
	handler, e, userID, _ := setupExpenseTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/expenses/99999/mark-paid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999")
	c.Set(middleware.UserIDKey, userID)

	err := handler.MarkPaid(c)
	if err != nil {
		t.Fatalf("MarkPaid() returned error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	if !strings.Contains(rec.Body.String(), "Despesa não encontrada") {
		t.Errorf("Response body = %s, want to contain 'Despesa não encontrada'", rec.Body.String())
	}
}

func TestExpenseHandler_MarkUnpaid_Success(t *testing.T) {
	handler, e, userID, accountID := setupExpenseTestHandler()

	// Create an expense
	expense := models.Expense{
		AccountID: accountID,
		Name:      "Mark Unpaid Test",
		Amount:    100.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	}
	database.DB.Create(&expense)

	now := time.Now()
	month := int(now.Month())
	year := now.Year()

	// Create payment
	payment := models.ExpensePayment{
		ExpenseID: expense.ID,
		Month:     month,
		Year:      year,
		PaidAt:    now,
		Amount:    expense.Amount,
	}
	database.DB.Create(&payment)

	req := httptest.NewRequest(http.MethodPost, "/expenses/"+fmt.Sprintf("%d", expense.ID)+"/mark-unpaid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", expense.ID))
	c.Set(middleware.UserIDKey, userID)

	e.Renderer = &testutil.MockRenderer{}

	err := handler.MarkUnpaid(c)
	if err != nil {
		t.Fatalf("MarkUnpaid() returned error: %v", err)
	}

	// Verify payment was deleted
	var count int64
	database.DB.Model(&models.ExpensePayment{}).Where("expense_id = ? AND month = ? AND year = ?", expense.ID, month, year).Count(&count)
	if count != 0 {
		t.Errorf("Payment count = %d, want %d (should be deleted)", count, 0)
	}
}

func TestExpenseHandler_MarkUnpaid_NotFound(t *testing.T) {
	handler, e, userID, _ := setupExpenseTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/expenses/99999/mark-unpaid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999")
	c.Set(middleware.UserIDKey, userID)

	err := handler.MarkUnpaid(c)
	if err != nil {
		t.Fatalf("MarkUnpaid() returned error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	if !strings.Contains(rec.Body.String(), "Despesa não encontrada") {
		t.Errorf("Response body = %s, want to contain 'Despesa não encontrada'", rec.Body.String())
	}
}

func TestExpenseHandler_GetAccountMembers_Success(t *testing.T) {
	handler, e, userID, _ := setupExpenseTestHandler()

	// Create a joint account with group
	authService := services.NewAuthService()
	user2, _ := authService.Register("test2@example.com", "Password123", "Test User 2")

	group := models.FamilyGroup{
		Name:        "Test Group",
		CreatedByID: userID,
	}
	database.DB.Create(&group)

	// Add members to group
	database.DB.Create(&models.GroupMember{GroupID: group.ID, UserID: userID, Role: "admin"})
	database.DB.Create(&models.GroupMember{GroupID: group.ID, UserID: user2.ID, Role: "member"})

	// Create joint account
	jointAccount := models.Account{
		Name:    "Joint Account",
		Type:    models.AccountTypeJoint,
		GroupID: &group.ID,
	}
	database.DB.Create(&jointAccount)

	req := httptest.NewRequest(http.MethodGet, "/accounts/"+fmt.Sprintf("%d", jointAccount.ID)+"/members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("accountId")
	c.SetParamValues(fmt.Sprintf("%d", jointAccount.ID))
	c.Set(middleware.UserIDKey, userID)

	e.Renderer = &testutil.MockRenderer{}

	err := handler.GetAccountMembers(c)
	if err != nil {
		t.Fatalf("GetAccountMembers() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestExpenseHandler_GetAccountMembers_Unauthorized(t *testing.T) {
	handler, e, userID, _ := setupExpenseTestHandler()

	// Create another user with their own account
	authService := services.NewAuthService()
	user2, _ := authService.Register("test2@example.com", "Password123", "Test User 2")
	account2 := models.Account{
		Name:   "Other User Account",
		Type:   models.AccountTypeIndividual,
		UserID: user2.ID,
	}
	database.DB.Create(&account2)

	req := httptest.NewRequest(http.MethodGet, "/accounts/"+fmt.Sprintf("%d", account2.ID)+"/members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("accountId")
	c.SetParamValues(fmt.Sprintf("%d", account2.ID))
	c.Set(middleware.UserIDKey, userID)

	err := handler.GetAccountMembers(c)
	if err != nil {
		t.Fatalf("GetAccountMembers() returned error: %v", err)
	}

	if rec.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	if !strings.Contains(rec.Body.String(), "Acesso negado") {
		t.Errorf("Response body = %s, want to contain 'Acesso negado'", rec.Body.String())
	}
}

func TestExpenseHandler_GetAccountMembers_NotFound(t *testing.T) {
	handler, e, userID, _ := setupExpenseTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/accounts/99999/members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("accountId")
	c.SetParamValues("99999")
	c.Set(middleware.UserIDKey, userID)

	err := handler.GetAccountMembers(c)
	if err != nil {
		t.Fatalf("GetAccountMembers() returned error: %v", err)
	}

	if rec.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestExpenseHandler_List_Success(t *testing.T) {
	handler, e, userID, accountID := setupExpenseTestHandler()

	// Create some test expenses
	fixedExpense := models.Expense{
		AccountID: accountID,
		Name:      "Fixed Expense",
		Amount:    100.00,
		Type:      models.ExpenseTypeFixed,
		DueDay:    15,
		Category:  "Moradia",
		Active:    true,
	}
	database.DB.Create(&fixedExpense)

	variableExpense := models.Expense{
		AccountID: accountID,
		Name:      "Variable Expense",
		Amount:    50.00,
		Type:      models.ExpenseTypeVariable,
		Category:  "Lazer",
		Active:    true,
	}
	database.DB.Create(&variableExpense)

	req := httptest.NewRequest(http.MethodGet, "/expenses", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	e.Renderer = &testutil.MockRenderer{}

	err := handler.List(c)
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func Test_isExpensePaid(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create test data
	authService := services.NewAuthService()
	user, _ := authService.Register("test@example.com", "Password123", "Test User")

	account := models.Account{
		Name:   "Test Account",
		Type:   models.AccountTypeIndividual,
		UserID: user.ID,
	}
	database.DB.Create(&account)

	expense := models.Expense{
		AccountID: account.ID,
		Name:      "Test Expense",
		Amount:    100.00,
		Type:      models.ExpenseTypeFixed,
	}
	database.DB.Create(&expense)

	now := time.Now()
	month := int(now.Month())
	year := now.Year()

	// Initially should not be paid
	if isExpensePaid(expense.ID, month, year) {
		t.Error("isExpensePaid() = true, want false (no payment exists)")
	}

	// Create payment
	payment := models.ExpensePayment{
		ExpenseID: expense.ID,
		Month:     month,
		Year:      year,
		PaidAt:    now,
		Amount:    expense.Amount,
	}
	database.DB.Create(&payment)

	// Now should be paid
	if !isExpensePaid(expense.ID, month, year) {
		t.Error("isExpensePaid() = false, want true (payment exists)")
	}

	// Different month should not be paid
	if isExpensePaid(expense.ID, month+1, year) {
		t.Error("isExpensePaid() = true, want false (different month)")
	}

	// Different year should not be paid
	if isExpensePaid(expense.ID, month, year+1) {
		t.Error("isExpensePaid() = true, want false (different year)")
	}
}

func Test_getExpenseCategories(t *testing.T) {
	categories := getExpenseCategories()

	expectedCategories := []string{
		"Moradia",
		"Alimentação",
		"Transporte",
		"Saúde",
		"Educação",
		"Lazer",
		"Serviços",
		"Impostos",
		"Outros",
	}

	if len(categories) != len(expectedCategories) {
		t.Errorf("Number of categories = %d, want %d", len(categories), len(expectedCategories))
	}

	for i, cat := range expectedCategories {
		if i >= len(categories) || categories[i] != cat {
			t.Errorf("Category[%d] = %s, want %s", i, categories[i], cat)
		}
	}
}
