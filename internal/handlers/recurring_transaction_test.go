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

func setupRecurringTransactionTestHandler() (*RecurringTransactionHandler, *echo.Echo, uint, uint) {
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
	handler := NewRecurringTransactionHandler()
	return handler, e, user.ID, account.ID
}

func TestRecurringTransactionHandler_Create_Success(t *testing.T) {
	handler, e, userID, accountID := setupRecurringTransactionTestHandler()

	form := url.Values{}
	form.Set("account_id", fmt.Sprintf("%d", accountID))
	form.Set("transaction_type", "expense")
	form.Set("frequency", "monthly")
	form.Set("amount", "100.00")
	form.Set("description", "Test recurring expense")
	form.Set("start_date", "2024-01-01")
	form.Set("category", "Bills")

	req := httptest.NewRequest(http.MethodPost, "/recurring-transactions", strings.NewReader(form.Encode()))
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

	// Verify recurring transaction was created
	var rt models.RecurringTransaction
	if err := database.DB.Where("account_id = ?", accountID).First(&rt).Error; err != nil {
		t.Fatalf("Failed to find created recurring transaction: %v", err)
	}

	if rt.TransactionType != models.TransactionTypeExpense {
		t.Errorf("TransactionType = %s, want %s", rt.TransactionType, models.TransactionTypeExpense)
	}

	if rt.Frequency != models.FrequencyMonthly {
		t.Errorf("Frequency = %s, want %s", rt.Frequency, models.FrequencyMonthly)
	}

	if rt.Amount != 100.00 {
		t.Errorf("Amount = %f, want %f", rt.Amount, 100.00)
	}

	if rt.Description != "Test recurring expense" {
		t.Errorf("Description = %s, want %s", rt.Description, "Test recurring expense")
	}

	if !rt.Active {
		t.Errorf("Active = %v, want %v", rt.Active, true)
	}
}

func TestRecurringTransactionHandler_Create_InvalidTransactionType(t *testing.T) {
	handler, e, userID, accountID := setupRecurringTransactionTestHandler()

	form := url.Values{}
	form.Set("account_id", fmt.Sprintf("%d", accountID))
	form.Set("transaction_type", "invalid")
	form.Set("frequency", "monthly")
	form.Set("amount", "100.00")
	form.Set("start_date", "2024-01-01")

	req := httptest.NewRequest(http.MethodPost, "/recurring-transactions", strings.NewReader(form.Encode()))
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

	if !strings.Contains(rec.Body.String(), "Tipo de transação inválido") {
		t.Errorf("Response body = %s, want to contain 'Tipo de transação inválido'", rec.Body.String())
	}
}

func TestRecurringTransactionHandler_Create_InvalidFrequency(t *testing.T) {
	handler, e, userID, accountID := setupRecurringTransactionTestHandler()

	form := url.Values{}
	form.Set("account_id", fmt.Sprintf("%d", accountID))
	form.Set("transaction_type", "expense")
	form.Set("frequency", "invalid")
	form.Set("amount", "100.00")
	form.Set("start_date", "2024-01-01")

	req := httptest.NewRequest(http.MethodPost, "/recurring-transactions", strings.NewReader(form.Encode()))
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

	if !strings.Contains(rec.Body.String(), "Frequência inválida") {
		t.Errorf("Response body = %s, want to contain 'Frequência inválida'", rec.Body.String())
	}
}

func TestRecurringTransactionHandler_Create_InvalidAmount(t *testing.T) {
	handler, e, userID, accountID := setupRecurringTransactionTestHandler()

	tests := []struct {
		name   string
		amount string
	}{
		{
			name:   "zero amount",
			amount: "0",
		},
		{
			name:   "negative amount",
			amount: "-10.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{}
			form.Set("account_id", fmt.Sprintf("%d", accountID))
			form.Set("transaction_type", "expense")
			form.Set("frequency", "monthly")
			form.Set("amount", tt.amount)
			form.Set("start_date", "2024-01-01")

			req := httptest.NewRequest(http.MethodPost, "/recurring-transactions", strings.NewReader(form.Encode()))
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

			if !strings.Contains(rec.Body.String(), "Valor deve ser maior que zero") {
				t.Errorf("Response body = %s, want to contain 'Valor deve ser maior que zero'", rec.Body.String())
			}
		})
	}
}

func TestRecurringTransactionHandler_Create_InvalidStartDate(t *testing.T) {
	handler, e, userID, accountID := setupRecurringTransactionTestHandler()

	form := url.Values{}
	form.Set("account_id", fmt.Sprintf("%d", accountID))
	form.Set("transaction_type", "expense")
	form.Set("frequency", "monthly")
	form.Set("amount", "100.00")
	form.Set("start_date", "invalid-date")

	req := httptest.NewRequest(http.MethodPost, "/recurring-transactions", strings.NewReader(form.Encode()))
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

	if !strings.Contains(rec.Body.String(), "Data de início inválida") {
		t.Errorf("Response body = %s, want to contain 'Data de início inválida'", rec.Body.String())
	}
}

func TestRecurringTransactionHandler_Create_InvalidEndDate(t *testing.T) {
	handler, e, userID, accountID := setupRecurringTransactionTestHandler()

	form := url.Values{}
	form.Set("account_id", fmt.Sprintf("%d", accountID))
	form.Set("transaction_type", "expense")
	form.Set("frequency", "monthly")
	form.Set("amount", "100.00")
	form.Set("start_date", "2024-01-01")
	form.Set("end_date", "invalid-date")

	req := httptest.NewRequest(http.MethodPost, "/recurring-transactions", strings.NewReader(form.Encode()))
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

	if !strings.Contains(rec.Body.String(), "Data de término inválida") {
		t.Errorf("Response body = %s, want to contain 'Data de término inválida'", rec.Body.String())
	}
}

func TestRecurringTransactionHandler_Create_UnauthorizedAccount(t *testing.T) {
	handler, e, userID, _ := setupRecurringTransactionTestHandler()

	// Create another user with their own account
	authService := services.NewAuthService()
	otherUser, _ := authService.Register("other@example.com", "Password123", "Other User")
	otherAccount := models.Account{
		Name:   "Other Account",
		Type:   models.AccountTypeIndividual,
		UserID: otherUser.ID,
	}
	database.DB.Create(&otherAccount)

	// Try to create recurring transaction for other user's account
	form := url.Values{}
	form.Set("account_id", fmt.Sprintf("%d", otherAccount.ID))
	form.Set("transaction_type", "expense")
	form.Set("frequency", "monthly")
	form.Set("amount", "100.00")
	form.Set("start_date", "2024-01-01")

	req := httptest.NewRequest(http.MethodPost, "/recurring-transactions", strings.NewReader(form.Encode()))
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

func TestRecurringTransactionHandler_Create_WithEndDate(t *testing.T) {
	handler, e, userID, accountID := setupRecurringTransactionTestHandler()

	form := url.Values{}
	form.Set("account_id", fmt.Sprintf("%d", accountID))
	form.Set("transaction_type", "income")
	form.Set("frequency", "weekly")
	form.Set("amount", "50.00")
	form.Set("description", "Weekly income")
	form.Set("start_date", "2024-01-01")
	form.Set("end_date", "2024-12-31")
	form.Set("category", "Salary")

	req := httptest.NewRequest(http.MethodPost, "/recurring-transactions", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	e.Renderer = &testutil.MockRenderer{}

	err := handler.Create(c)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	var rt models.RecurringTransaction
	if err := database.DB.Where("account_id = ?", accountID).First(&rt).Error; err != nil {
		t.Fatalf("Failed to find created recurring transaction: %v", err)
	}

	if rt.EndDate == nil {
		t.Errorf("EndDate = nil, want not nil")
	} else {
		expectedEndDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
		if !rt.EndDate.Equal(expectedEndDate) {
			t.Errorf("EndDate = %v, want %v", rt.EndDate, expectedEndDate)
		}
	}
}

func TestRecurringTransactionHandler_List_Success(t *testing.T) {
	handler, e, userID, accountID := setupRecurringTransactionTestHandler()

	// Create active recurring transaction
	activeRT := models.RecurringTransaction{
		AccountID:       accountID,
		TransactionType: models.TransactionTypeExpense,
		Frequency:       models.FrequencyMonthly,
		Amount:          100.00,
		Description:     "Active transaction",
		StartDate:       time.Now(),
		NextRunDate:     time.Now(),
		Active:          true,
	}
	database.DB.Create(&activeRT)

	// Create paused recurring transaction
	pausedRT := models.RecurringTransaction{
		AccountID:       accountID,
		TransactionType: models.TransactionTypeIncome,
		Frequency:       models.FrequencyWeekly,
		Amount:          50.00,
		Description:     "Paused transaction",
		StartDate:       time.Now(),
		NextRunDate:     time.Now(),
		Active:          false,
	}
	database.DB.Create(&pausedRT)

	req := httptest.NewRequest(http.MethodGet, "/recurring-transactions", nil)
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

func TestRecurringTransactionHandler_Update_Success(t *testing.T) {
	handler, e, userID, accountID := setupRecurringTransactionTestHandler()

	// Create recurring transaction
	rt := models.RecurringTransaction{
		AccountID:       accountID,
		TransactionType: models.TransactionTypeExpense,
		Frequency:       models.FrequencyMonthly,
		Amount:          100.00,
		Description:     "Original description",
		StartDate:       time.Now(),
		NextRunDate:     time.Now(),
		Active:          true,
		Category:        "Bills",
	}
	database.DB.Create(&rt)

	// Update the recurring transaction
	form := url.Values{}
	form.Set("amount", "200.00")
	form.Set("description", "Updated description")
	form.Set("category", "Utilities")

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/recurring-transactions/%d", rt.ID), strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", rt.ID))
	c.Set(middleware.UserIDKey, userID)

	e.Renderer = &testutil.MockRenderer{}

	err := handler.Update(c)
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	// Verify updates
	var updatedRT models.RecurringTransaction
	database.DB.First(&updatedRT, rt.ID)

	if updatedRT.Amount != 200.00 {
		t.Errorf("Amount = %f, want %f", updatedRT.Amount, 200.00)
	}

	if updatedRT.Description != "Updated description" {
		t.Errorf("Description = %s, want %s", updatedRT.Description, "Updated description")
	}

	if updatedRT.Category != "Utilities" {
		t.Errorf("Category = %s, want %s", updatedRT.Category, "Utilities")
	}
}

func TestRecurringTransactionHandler_Update_NotFound(t *testing.T) {
	handler, e, userID, _ := setupRecurringTransactionTestHandler()

	form := url.Values{}
	form.Set("amount", "200.00")

	req := httptest.NewRequest(http.MethodPut, "/recurring-transactions/99999", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999")
	c.Set(middleware.UserIDKey, userID)

	err := handler.Update(c)
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	if !strings.Contains(rec.Body.String(), "Transação recorrente não encontrada") {
		t.Errorf("Response body = %s, want to contain 'Transação recorrente não encontrada'", rec.Body.String())
	}
}

func TestRecurringTransactionHandler_Update_UnauthorizedAccount(t *testing.T) {
	handler, e, userID, _ := setupRecurringTransactionTestHandler()

	// Create another user with their own account and recurring transaction
	authService := services.NewAuthService()
	otherUser, _ := authService.Register("other@example.com", "Password123", "Other User")
	otherAccount := models.Account{
		Name:   "Other Account",
		Type:   models.AccountTypeIndividual,
		UserID: otherUser.ID,
	}
	database.DB.Create(&otherAccount)

	otherRT := models.RecurringTransaction{
		AccountID:       otherAccount.ID,
		TransactionType: models.TransactionTypeExpense,
		Frequency:       models.FrequencyMonthly,
		Amount:          100.00,
		Description:     "Other user's transaction",
		StartDate:       time.Now(),
		NextRunDate:     time.Now(),
		Active:          true,
	}
	database.DB.Create(&otherRT)

	// Try to update other user's recurring transaction
	form := url.Values{}
	form.Set("amount", "200.00")

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/recurring-transactions/%d", otherRT.ID), strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", otherRT.ID))
	c.Set(middleware.UserIDKey, userID)

	err := handler.Update(c)
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestRecurringTransactionHandler_Delete_Success(t *testing.T) {
	handler, e, userID, accountID := setupRecurringTransactionTestHandler()

	// Create recurring transaction
	rt := models.RecurringTransaction{
		AccountID:       accountID,
		TransactionType: models.TransactionTypeExpense,
		Frequency:       models.FrequencyMonthly,
		Amount:          100.00,
		Description:     "To be deleted",
		StartDate:       time.Now(),
		NextRunDate:     time.Now(),
		Active:          true,
	}
	database.DB.Create(&rt)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/recurring-transactions/%d", rt.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", rt.ID))
	c.Set(middleware.UserIDKey, userID)

	e.Renderer = &testutil.MockRenderer{}

	err := handler.Delete(c)
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	// Verify deletion
	var deletedRT models.RecurringTransaction
	result := database.DB.First(&deletedRT, rt.ID)
	if result.Error == nil {
		t.Errorf("Recurring transaction still exists after delete")
	}
}

func TestRecurringTransactionHandler_Delete_NotFound(t *testing.T) {
	handler, e, userID, _ := setupRecurringTransactionTestHandler()

	req := httptest.NewRequest(http.MethodDelete, "/recurring-transactions/99999", nil)
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

	if !strings.Contains(rec.Body.String(), "Transação recorrente não encontrada") {
		t.Errorf("Response body = %s, want to contain 'Transação recorrente não encontrada'", rec.Body.String())
	}
}

func TestRecurringTransactionHandler_Delete_UnauthorizedAccount(t *testing.T) {
	handler, e, userID, _ := setupRecurringTransactionTestHandler()

	// Create another user with their own account and recurring transaction
	authService := services.NewAuthService()
	otherUser, _ := authService.Register("other@example.com", "Password123", "Other User")
	otherAccount := models.Account{
		Name:   "Other Account",
		Type:   models.AccountTypeIndividual,
		UserID: otherUser.ID,
	}
	database.DB.Create(&otherAccount)

	otherRT := models.RecurringTransaction{
		AccountID:       otherAccount.ID,
		TransactionType: models.TransactionTypeExpense,
		Frequency:       models.FrequencyMonthly,
		Amount:          100.00,
		Description:     "Other user's transaction",
		StartDate:       time.Now(),
		NextRunDate:     time.Now(),
		Active:          true,
	}
	database.DB.Create(&otherRT)

	// Try to delete other user's recurring transaction
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/recurring-transactions/%d", otherRT.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", otherRT.ID))
	c.Set(middleware.UserIDKey, userID)

	err := handler.Delete(c)
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	// Verify the transaction still exists
	var stillExists models.RecurringTransaction
	if err := database.DB.First(&stillExists, otherRT.ID).Error; err != nil {
		t.Errorf("Other user's transaction was deleted when it shouldn't be")
	}
}

func TestRecurringTransactionHandler_Toggle_Success(t *testing.T) {
	handler, e, userID, accountID := setupRecurringTransactionTestHandler()

	// Create active recurring transaction
	rt := models.RecurringTransaction{
		AccountID:       accountID,
		TransactionType: models.TransactionTypeExpense,
		Frequency:       models.FrequencyMonthly,
		Amount:          100.00,
		Description:     "To be toggled",
		StartDate:       time.Now(),
		NextRunDate:     time.Now(),
		Active:          true,
	}
	database.DB.Create(&rt)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/recurring-transactions/%d/toggle", rt.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", rt.ID))
	c.Set(middleware.UserIDKey, userID)

	e.Renderer = &testutil.MockRenderer{}

	err := handler.Toggle(c)
	if err != nil {
		t.Fatalf("Toggle() returned error: %v", err)
	}

	// Verify toggle
	var toggledRT models.RecurringTransaction
	database.DB.First(&toggledRT, rt.ID)

	if toggledRT.Active {
		t.Errorf("Active = %v, want %v", toggledRT.Active, false)
	}

	// Toggle again
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/recurring-transactions/%d/toggle", rt.ID), nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", rt.ID))
	c.Set(middleware.UserIDKey, userID)

	err = handler.Toggle(c)
	if err != nil {
		t.Fatalf("Toggle() returned error: %v", err)
	}

	// Verify toggle back
	database.DB.First(&toggledRT, rt.ID)

	if !toggledRT.Active {
		t.Errorf("Active = %v, want %v", toggledRT.Active, true)
	}
}

func TestRecurringTransactionHandler_Toggle_NotFound(t *testing.T) {
	handler, e, userID, _ := setupRecurringTransactionTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/recurring-transactions/99999/toggle", nil)
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

	if !strings.Contains(rec.Body.String(), "Transação recorrente não encontrada") {
		t.Errorf("Response body = %s, want to contain 'Transação recorrente não encontrada'", rec.Body.String())
	}
}

func TestRecurringTransactionHandler_Toggle_UnauthorizedAccount(t *testing.T) {
	handler, e, userID, _ := setupRecurringTransactionTestHandler()

	// Create another user with their own account and recurring transaction
	authService := services.NewAuthService()
	otherUser, _ := authService.Register("other@example.com", "Password123", "Other User")
	otherAccount := models.Account{
		Name:   "Other Account",
		Type:   models.AccountTypeIndividual,
		UserID: otherUser.ID,
	}
	database.DB.Create(&otherAccount)

	otherRT := models.RecurringTransaction{
		AccountID:       otherAccount.ID,
		TransactionType: models.TransactionTypeExpense,
		Frequency:       models.FrequencyMonthly,
		Amount:          100.00,
		Description:     "Other user's transaction",
		StartDate:       time.Now(),
		NextRunDate:     time.Now(),
		Active:          true,
	}
	database.DB.Create(&otherRT)

	// Try to toggle other user's recurring transaction
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/recurring-transactions/%d/toggle", otherRT.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", otherRT.ID))
	c.Set(middleware.UserIDKey, userID)

	err := handler.Toggle(c)
	if err != nil {
		t.Fatalf("Toggle() returned error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	// Verify the transaction wasn't toggled
	var unchanged models.RecurringTransaction
	database.DB.First(&unchanged, otherRT.ID)
	if !unchanged.Active {
		t.Errorf("Other user's transaction was toggled when it shouldn't be")
	}
}
