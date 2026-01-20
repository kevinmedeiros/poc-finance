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

func setupIncomeTestHandler() (*IncomeHandler, *echo.Echo, uint, uint) {
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
	handler := NewIncomeHandler()
	return handler, e, user.ID, account.ID
}

func TestIncomeHandler_Create_Success(t *testing.T) {
	handler, e, userID, accountID := setupIncomeTestHandler()

	form := url.Values{}
	form.Set("account_id", fmt.Sprintf("%d", accountID))
	form.Set("date", "2024-01-15")
	form.Set("amount_usd", "1000.00")
	form.Set("exchange_rate", "5.00")
	form.Set("description", "Test income")

	req := httptest.NewRequest(http.MethodPost, "/income", strings.NewReader(form.Encode()))
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

	// Verify income was created
	var income models.Income
	if err := database.DB.Where("account_id = ?", accountID).First(&income).Error; err != nil {
		t.Fatalf("Failed to find created income: %v", err)
	}

	if income.AmountUSD != 1000.00 {
		t.Errorf("AmountUSD = %f, want %f", income.AmountUSD, 1000.00)
	}

	if income.ExchangeRate != 5.00 {
		t.Errorf("ExchangeRate = %f, want %f", income.ExchangeRate, 5.00)
	}

	expectedAmountBRL := 5000.00
	if income.AmountBRL != expectedAmountBRL {
		t.Errorf("AmountBRL = %f, want %f", income.AmountBRL, expectedAmountBRL)
	}

	if income.Description != "Test income" {
		t.Errorf("Description = %s, want %s", income.Description, "Test income")
	}

	expectedDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	if !income.Date.Equal(expectedDate) {
		t.Errorf("Date = %v, want %v", income.Date, expectedDate)
	}
}

func TestIncomeHandler_Create_InvalidDate(t *testing.T) {
	handler, e, userID, accountID := setupIncomeTestHandler()

	form := url.Values{}
	form.Set("account_id", fmt.Sprintf("%d", accountID))
	form.Set("date", "invalid-date")
	form.Set("amount_usd", "1000.00")
	form.Set("exchange_rate", "5.00")

	req := httptest.NewRequest(http.MethodPost, "/income", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
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

	if !strings.Contains(rec.Body.String(), "Data inválida") {
		t.Errorf("Response body = %s, want to contain 'Data inválida'", rec.Body.String())
	}
}

func TestIncomeHandler_Create_FallbackToIndividualAccount(t *testing.T) {
	handler, e, userID, _ := setupIncomeTestHandler()

	form := url.Values{}
	form.Set("account_id", "0") // No account specified
	form.Set("date", "2024-01-15")
	form.Set("amount_usd", "500.00")
	form.Set("exchange_rate", "5.00")
	form.Set("description", "Fallback test")

	req := httptest.NewRequest(http.MethodPost, "/income", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	e.Renderer = &testutil.MockRenderer{}

	err := handler.Create(c)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	// Verify income was created using individual account
	var income models.Income
	if err := database.DB.Where("description = ?", "Fallback test").First(&income).Error; err != nil {
		t.Fatalf("Failed to find created income: %v", err)
	}

	// Verify it was created with the individual account
	var account models.Account
	database.DB.First(&account, income.AccountID)
	if account.Type != models.AccountTypeIndividual {
		t.Errorf("Account type = %s, want %s", account.Type, models.AccountTypeIndividual)
	}
}

func TestIncomeHandler_Create_UnauthorizedAccount(t *testing.T) {
	handler, e, userID, _ := setupIncomeTestHandler()

	// Create another user and their account
	authService := services.NewAuthService()
	otherUser, _ := authService.Register("other@example.com", "Password123", "Other User")
	otherAccount := models.Account{
		Name:   "Other Account",
		Type:   models.AccountTypeIndividual,
		UserID: otherUser.ID,
	}
	database.DB.Create(&otherAccount)

	// Try to create income for the other user's account
	form := url.Values{}
	form.Set("account_id", fmt.Sprintf("%d", otherAccount.ID))
	form.Set("date", "2024-01-15")
	form.Set("amount_usd", "1000.00")
	form.Set("exchange_rate", "5.00")

	req := httptest.NewRequest(http.MethodPost, "/income", strings.NewReader(form.Encode()))
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

	if !strings.Contains(rec.Body.String(), "Acesso negado") {
		t.Errorf("Response body = %s, want to contain 'Acesso negado'", rec.Body.String())
	}
}

func TestIncomeHandler_Create_MissingIndividualAccount(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create test user WITHOUT an individual account
	authService := services.NewAuthService()
	user, _ := authService.Register("test@example.com", "Password123", "Test User")

	// Delete the individual account that was auto-created
	database.DB.Where("user_id = ? AND type = ?", user.ID, models.AccountTypeIndividual).Delete(&models.Account{})

	e := echo.New()
	handler := NewIncomeHandler()

	form := url.Values{}
	form.Set("account_id", "0") // Fallback to individual account
	form.Set("date", "2024-01-15")
	form.Set("amount_usd", "1000.00")
	form.Set("exchange_rate", "5.00")

	req := httptest.NewRequest(http.MethodPost, "/income", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, user.ID)

	err := handler.Create(c)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	if !strings.Contains(rec.Body.String(), "Conta não encontrada") {
		t.Errorf("Response body = %s, want to contain 'Conta não encontrada'", rec.Body.String())
	}
}

func TestIncomeHandler_Delete_Success(t *testing.T) {
	handler, e, userID, accountID := setupIncomeTestHandler()

	// Create an income to delete
	income := models.Income{
		AccountID:    accountID,
		Date:         time.Now(),
		AmountUSD:    1000.00,
		ExchangeRate: 5.00,
		AmountBRL:    5000.00,
		GrossAmount:  5000.00,
		TaxAmount:    500.00,
		NetAmount:    4500.00,
		Description:  "To be deleted",
	}
	database.DB.Create(&income)

	req := httptest.NewRequest(http.MethodDelete, "/income/:id", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", income.ID))
	c.Set(middleware.UserIDKey, userID)

	e.Renderer = &testutil.MockRenderer{}

	err := handler.Delete(c)
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	// Verify income was deleted
	var deletedIncome models.Income
	result := database.DB.First(&deletedIncome, income.ID)
	if result.Error == nil {
		t.Errorf("Income was not deleted, still found in database")
	}
}

func TestIncomeHandler_Delete_NotFound(t *testing.T) {
	handler, e, userID, _ := setupIncomeTestHandler()

	req := httptest.NewRequest(http.MethodDelete, "/income/:id", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999") // Non-existent ID
	c.Set(middleware.UserIDKey, userID)

	err := handler.Delete(c)
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	if !strings.Contains(rec.Body.String(), "Recebimento não encontrado") {
		t.Errorf("Response body = %s, want to contain 'Recebimento não encontrado'", rec.Body.String())
	}
}

func TestIncomeHandler_Delete_UnauthorizedAccount(t *testing.T) {
	handler, e, userID, _ := setupIncomeTestHandler()

	// Create another user and their income
	authService := services.NewAuthService()
	otherUser, _ := authService.Register("other@example.com", "Password123", "Other User")
	otherAccount := models.Account{
		Name:   "Other Account",
		Type:   models.AccountTypeIndividual,
		UserID: otherUser.ID,
	}
	database.DB.Create(&otherAccount)

	otherIncome := models.Income{
		AccountID:    otherAccount.ID,
		Date:         time.Now(),
		AmountUSD:    1000.00,
		ExchangeRate: 5.00,
		AmountBRL:    5000.00,
		GrossAmount:  5000.00,
		TaxAmount:    500.00,
		NetAmount:    4500.00,
		Description:  "Other user's income",
	}
	database.DB.Create(&otherIncome)

	// Try to delete other user's income
	req := httptest.NewRequest(http.MethodDelete, "/income/:id", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", otherIncome.ID))
	c.Set(middleware.UserIDKey, userID)

	err := handler.Delete(c)
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	// Verify income still exists
	var stillExists models.Income
	result := database.DB.First(&stillExists, otherIncome.ID)
	if result.Error != nil {
		t.Errorf("Income was deleted when it shouldn't have been")
	}
}

func TestIncomeHandler_CalculatePreview_Success(t *testing.T) {
	handler, e, userID, _ := setupIncomeTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/income/calculate-preview?amount_usd=1000&exchange_rate=5.0", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.CalculatePreview(c)
	if err != nil {
		t.Fatalf("CalculatePreview() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Should return JSON with calculated values
	body := rec.Body.String()
	if !strings.Contains(body, "amount_brl") {
		t.Errorf("Response body missing 'amount_brl': %s", body)
	}
	if !strings.Contains(body, "tax") {
		t.Errorf("Response body missing 'tax': %s", body)
	}
	if !strings.Contains(body, "net") {
		t.Errorf("Response body missing 'net': %s", body)
	}
	if !strings.Contains(body, "effective_rate") {
		t.Errorf("Response body missing 'effective_rate': %s", body)
	}
}

func TestIncomeHandler_CalculatePreview_ZeroAmount(t *testing.T) {
	handler, e, userID, _ := setupIncomeTestHandler()

	tests := []struct {
		name         string
		amountUSD    string
		exchangeRate string
	}{
		{
			name:         "zero amount USD",
			amountUSD:    "0",
			exchangeRate: "5.0",
		},
		{
			name:         "zero exchange rate",
			amountUSD:    "1000",
			exchangeRate: "0",
		},
		{
			name:         "negative amount USD",
			amountUSD:    "-1000",
			exchangeRate: "5.0",
		},
		{
			name:         "negative exchange rate",
			amountUSD:    "1000",
			exchangeRate: "-5.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("/income/calculate-preview?amount_usd=%s&exchange_rate=%s", tt.amountUSD, tt.exchangeRate)
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set(middleware.UserIDKey, userID)

			err := handler.CalculatePreview(c)
			if err != nil {
				t.Fatalf("CalculatePreview() returned error: %v", err)
			}

			if rec.Code != http.StatusOK {
				t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
			}

			// Should return zeros for invalid amounts
			body := rec.Body.String()
			if !strings.Contains(body, `"amount_brl":0`) {
				t.Errorf("Response should have amount_brl:0, got: %s", body)
			}
		})
	}
}

func TestIncomeHandler_List_Success(t *testing.T) {
	handler, e, userID, accountID := setupIncomeTestHandler()

	// Create some test incomes
	incomes := []models.Income{
		{
			AccountID:    accountID,
			Date:         time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			AmountUSD:    1000.00,
			ExchangeRate: 5.00,
			AmountBRL:    5000.00,
			GrossAmount:  5000.00,
			TaxAmount:    500.00,
			NetAmount:    4500.00,
			Description:  "Income 1",
		},
		{
			AccountID:    accountID,
			Date:         time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC),
			AmountUSD:    2000.00,
			ExchangeRate: 5.00,
			AmountBRL:    10000.00,
			GrossAmount:  10000.00,
			TaxAmount:    1000.00,
			NetAmount:    9000.00,
			Description:  "Income 2",
		},
	}
	for _, income := range incomes {
		database.DB.Create(&income)
	}

	req := httptest.NewRequest(http.MethodGet, "/income", nil)
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

func TestIncomeHandler_List_MultipleAccounts(t *testing.T) {
	handler, e, userID, accountID := setupIncomeTestHandler()

	// Create another account for the same user
	secondAccount := models.Account{
		Name:   "Second Account",
		Type:   models.AccountTypeJoint,
		UserID: userID,
	}
	database.DB.Create(&secondAccount)

	// Create incomes in both accounts
	income1 := models.Income{
		AccountID:    accountID,
		Date:         time.Now(),
		AmountUSD:    1000.00,
		ExchangeRate: 5.00,
		AmountBRL:    5000.00,
		GrossAmount:  5000.00,
		TaxAmount:    500.00,
		NetAmount:    4500.00,
		Description:  "Account 1 Income",
	}
	database.DB.Create(&income1)

	income2 := models.Income{
		AccountID:    secondAccount.ID,
		Date:         time.Now(),
		AmountUSD:    2000.00,
		ExchangeRate: 5.00,
		AmountBRL:    10000.00,
		GrossAmount:  10000.00,
		TaxAmount:    1000.00,
		NetAmount:    9000.00,
		Description:  "Account 2 Income",
	}
	database.DB.Create(&income2)

	req := httptest.NewRequest(http.MethodGet, "/income", nil)
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

	// Verify both incomes are in the response
	// The mock renderer captures the data, but for this test we just verify no error
}
