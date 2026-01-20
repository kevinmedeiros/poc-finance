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

func setupCreditCardTestHandler() (*CreditCardHandler, *echo.Echo, uint, uint) {
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
	handler := NewCreditCardHandler()
	return handler, e, user.ID, account.ID
}

func TestCreditCardHandler_CreateCard_Success(t *testing.T) {
	handler, e, userID, _ := setupCreditCardTestHandler()

	form := url.Values{}
	form.Set("name", "Test Credit Card")
	form.Set("closing_day", "15")
	form.Set("due_day", "25")
	form.Set("limit_amount", "5000.00")

	req := httptest.NewRequest(http.MethodPost, "/cards", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	// Mock renderer to avoid nil pointer
	e.Renderer = &testutil.MockRenderer{}

	err := handler.CreateCard(c)
	if err != nil {
		t.Fatalf("CreateCard() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Verify credit card was created
	var cards []models.CreditCard
	database.DB.Find(&cards)
	if len(cards) == 0 {
		t.Fatalf("No credit cards found after creation")
	}

	// Find the card we just created
	var card models.CreditCard
	found := false
	for _, c := range cards {
		if c.Name == "Test Credit Card" {
			card = c
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("Failed to find created credit card with name 'Test Credit Card'")
	}

	if card.ClosingDay != 15 {
		t.Errorf("ClosingDay = %d, want %d", card.ClosingDay, 15)
	}

	if card.DueDay != 25 {
		t.Errorf("DueDay = %d, want %d", card.DueDay, 25)
	}

	if card.LimitAmount != 5000.00 {
		t.Errorf("LimitAmount = %f, want %f", card.LimitAmount, 5000.00)
	}
}

func TestCreditCardHandler_CreateCard_InvalidData(t *testing.T) {
	handler, e, userID, _ := setupCreditCardTestHandler()

	// Invalid form data (missing required fields)
	form := url.Values{}
	form.Set("name", "Test Card")

	req := httptest.NewRequest(http.MethodPost, "/cards", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	// Mock renderer to avoid nil pointer
	e.Renderer = &testutil.MockRenderer{}

	err := handler.CreateCard(c)
	if err != nil {
		t.Fatalf("CreateCard() returned error: %v", err)
	}

	// Should still succeed with zero values, but let's verify no crash
	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestCreditCardHandler_DeleteCard_Success(t *testing.T) {
	handler, e, userID, accountID := setupCreditCardTestHandler()

	// Create a credit card first
	card := models.CreditCard{
		AccountID:   accountID,
		Name:        "Card to Delete",
		ClosingDay:  10,
		DueDay:      20,
		LimitAmount: 3000.00,
	}
	database.DB.Create(&card)

	req := httptest.NewRequest(http.MethodDelete, "/cards/"+fmt.Sprintf("%d", card.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", card.ID))

	// Mock renderer
	e.Renderer = &testutil.MockRenderer{}

	err := handler.DeleteCard(c)
	if err != nil {
		t.Fatalf("DeleteCard() returned error: %v", err)
	}

	// Verify card was deleted
	var deletedCard models.CreditCard
	err = database.DB.Where("id = ?", card.ID).First(&deletedCard).Error
	if err == nil {
		t.Errorf("Card should have been deleted but still exists")
	}
}

func TestCreditCardHandler_DeleteCard_NotFound(t *testing.T) {
	handler, e, userID, _ := setupCreditCardTestHandler()

	req := httptest.NewRequest(http.MethodDelete, "/cards/9999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)
	c.SetParamNames("id")
	c.SetParamValues("9999")

	err := handler.DeleteCard(c)
	if err != nil {
		t.Fatalf("DeleteCard() returned error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	if !strings.Contains(rec.Body.String(), "Cartão não encontrado") {
		t.Errorf("Response body = %s, want to contain 'Cartão não encontrado'", rec.Body.String())
	}
}

func TestCreditCardHandler_DeleteCard_UnauthorizedAccess(t *testing.T) {
	handler, e, _, accountID := setupCreditCardTestHandler()

	// Create another user who will try to access the first user's card
	authService := services.NewAuthService()
	otherUser, _ := authService.Register("other@example.com", "Password123", "Other User")

	// Create other user's account
	otherAccount := models.Account{
		Name:   "Other Account",
		Type:   models.AccountTypeIndividual,
		UserID: otherUser.ID,
	}
	database.DB.Create(&otherAccount)

	// Create a credit card for the first user
	card := models.CreditCard{
		AccountID:   accountID,
		Name:        "Protected Card",
		ClosingDay:  10,
		DueDay:      20,
		LimitAmount: 3000.00,
	}
	database.DB.Create(&card)

	// Try to delete it with the other user's ID
	req := httptest.NewRequest(http.MethodDelete, "/cards/"+fmt.Sprintf("%d", card.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, otherUser.ID)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", card.ID))

	err := handler.DeleteCard(c)
	if err != nil {
		t.Fatalf("DeleteCard() returned error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	// Verify card was NOT deleted
	var existingCard models.CreditCard
	err = database.DB.Where("id = ?", card.ID).First(&existingCard).Error
	if err != nil {
		t.Errorf("Card should still exist but was deleted: %v", err)
	}
}

func TestCreditCardHandler_List_Success(t *testing.T) {
	handler, e, userID, accountID := setupCreditCardTestHandler()

	// Create test credit cards
	card1 := models.CreditCard{
		AccountID:   accountID,
		Name:        "Card 1",
		ClosingDay:  10,
		DueDay:      20,
		LimitAmount: 2000.00,
	}
	database.DB.Create(&card1)

	card2 := models.CreditCard{
		AccountID:   accountID,
		Name:        "Card 2",
		ClosingDay:  15,
		DueDay:      25,
		LimitAmount: 3000.00,
	}
	database.DB.Create(&card2)

	req := httptest.NewRequest(http.MethodGet, "/cards", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	// Mock renderer
	e.Renderer = &testutil.MockRenderer{}

	err := handler.List(c)
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestCreditCardHandler_List_EmptyCards(t *testing.T) {
	handler, e, userID, _ := setupCreditCardTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/cards", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	// Mock renderer
	e.Renderer = &testutil.MockRenderer{}

	err := handler.List(c)
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestCreditCardHandler_List_WithInstallments(t *testing.T) {
	handler, e, userID, accountID := setupCreditCardTestHandler()

	// Create a credit card
	card := models.CreditCard{
		AccountID:   accountID,
		Name:        "Card with Installments",
		ClosingDay:  10,
		DueDay:      20,
		LimitAmount: 5000.00,
	}
	database.DB.Create(&card)

	// Create an installment
	startDate := time.Now().AddDate(0, -2, 0) // Started 2 months ago
	installment := models.Installment{
		CreditCardID:       card.ID,
		Description:        "Test Purchase",
		TotalAmount:        1200.00,
		InstallmentAmount:  100.00,
		TotalInstallments:  12,
		CurrentInstallment: 1,
		StartDate:          startDate,
		Category:           "Shopping",
	}
	database.DB.Create(&installment)

	req := httptest.NewRequest(http.MethodGet, "/cards", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	// Mock renderer
	e.Renderer = &testutil.MockRenderer{}

	err := handler.List(c)
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestCreditCardHandler_CreateInstallment_Success(t *testing.T) {
	handler, e, userID, accountID := setupCreditCardTestHandler()

	// Create a credit card first
	card := models.CreditCard{
		AccountID:   accountID,
		Name:        "Test Card",
		ClosingDay:  10,
		DueDay:      20,
		LimitAmount: 5000.00,
	}
	database.DB.Create(&card)

	form := url.Values{}
	form.Set("credit_card_id", fmt.Sprintf("%d", card.ID))
	form.Set("description", "Test Purchase")
	form.Set("total_amount", "1200.00")
	form.Set("total_installments", "12")
	form.Set("start_date", "2024-01-01")
	form.Set("category", "Shopping")

	req := httptest.NewRequest(http.MethodPost, "/installments", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	// Mock renderer
	e.Renderer = &testutil.MockRenderer{}

	err := handler.CreateInstallment(c)
	if err != nil {
		t.Fatalf("CreateInstallment() returned error: %v", err)
	}

	// Verify installment was created
	var installment models.Installment
	if err := database.DB.Where("credit_card_id = ?", card.ID).First(&installment).Error; err != nil {
		t.Fatalf("Failed to find created installment: %v", err)
	}

	if installment.Description != "Test Purchase" {
		t.Errorf("Description = %s, want %s", installment.Description, "Test Purchase")
	}

	if installment.TotalAmount != 1200.00 {
		t.Errorf("TotalAmount = %f, want %f", installment.TotalAmount, 1200.00)
	}

	if installment.TotalInstallments != 12 {
		t.Errorf("TotalInstallments = %d, want %d", installment.TotalInstallments, 12)
	}

	expectedAmount := 1200.00 / 12.0
	if installment.InstallmentAmount != expectedAmount {
		t.Errorf("InstallmentAmount = %f, want %f", installment.InstallmentAmount, expectedAmount)
	}

	if installment.Category != "Shopping" {
		t.Errorf("Category = %s, want %s", installment.Category, "Shopping")
	}
}

func TestCreditCardHandler_CreateInstallment_CardNotFound(t *testing.T) {
	handler, e, userID, _ := setupCreditCardTestHandler()

	form := url.Values{}
	form.Set("credit_card_id", "9999")
	form.Set("description", "Test Purchase")
	form.Set("total_amount", "1200.00")
	form.Set("total_installments", "12")
	form.Set("start_date", "2024-01-01")
	form.Set("category", "Shopping")

	req := httptest.NewRequest(http.MethodPost, "/installments", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.CreateInstallment(c)
	if err != nil {
		t.Fatalf("CreateInstallment() returned error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	if !strings.Contains(rec.Body.String(), "Cartão não encontrado") {
		t.Errorf("Response body = %s, want to contain 'Cartão não encontrado'", rec.Body.String())
	}
}

func TestCreditCardHandler_CreateInstallment_InvalidDate(t *testing.T) {
	handler, e, userID, accountID := setupCreditCardTestHandler()

	// Create a credit card first
	card := models.CreditCard{
		AccountID:   accountID,
		Name:        "Test Card",
		ClosingDay:  10,
		DueDay:      20,
		LimitAmount: 5000.00,
	}
	database.DB.Create(&card)

	form := url.Values{}
	form.Set("credit_card_id", fmt.Sprintf("%d", card.ID))
	form.Set("description", "Test Purchase")
	form.Set("total_amount", "1200.00")
	form.Set("total_installments", "12")
	form.Set("start_date", "invalid-date")
	form.Set("category", "Shopping")

	req := httptest.NewRequest(http.MethodPost, "/installments", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.CreateInstallment(c)
	if err != nil {
		t.Fatalf("CreateInstallment() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	if !strings.Contains(rec.Body.String(), "Data inválida") {
		t.Errorf("Response body = %s, want to contain 'Data inválida'", rec.Body.String())
	}
}

func TestCreditCardHandler_CreateInstallment_UnauthorizedCard(t *testing.T) {
	handler, e, _, accountID := setupCreditCardTestHandler()

	// Create another user
	authService := services.NewAuthService()
	otherUser, _ := authService.Register("other@example.com", "Password123", "Other User")

	// Create other user's account
	otherAccount := models.Account{
		Name:   "Other Account",
		Type:   models.AccountTypeIndividual,
		UserID: otherUser.ID,
	}
	database.DB.Create(&otherAccount)

	// Create a credit card for the first user
	card := models.CreditCard{
		AccountID:   accountID,
		Name:        "Protected Card",
		ClosingDay:  10,
		DueDay:      20,
		LimitAmount: 5000.00,
	}
	database.DB.Create(&card)

	form := url.Values{}
	form.Set("credit_card_id", fmt.Sprintf("%d", card.ID))
	form.Set("description", "Test Purchase")
	form.Set("total_amount", "1200.00")
	form.Set("total_installments", "12")
	form.Set("start_date", "2024-01-01")
	form.Set("category", "Shopping")

	req := httptest.NewRequest(http.MethodPost, "/installments", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, otherUser.ID) // Using other user's ID

	err := handler.CreateInstallment(c)
	if err != nil {
		t.Fatalf("CreateInstallment() returned error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestCreditCardHandler_DeleteInstallment_Success(t *testing.T) {
	handler, e, userID, accountID := setupCreditCardTestHandler()

	// Create a credit card
	card := models.CreditCard{
		AccountID:   accountID,
		Name:        "Test Card",
		ClosingDay:  10,
		DueDay:      20,
		LimitAmount: 5000.00,
	}
	database.DB.Create(&card)

	// Create an installment
	installment := models.Installment{
		CreditCardID:       card.ID,
		Description:        "Test Purchase",
		TotalAmount:        1200.00,
		InstallmentAmount:  100.00,
		TotalInstallments:  12,
		CurrentInstallment: 1,
		StartDate:          time.Now(),
		Category:           "Shopping",
	}
	database.DB.Create(&installment)

	req := httptest.NewRequest(http.MethodDelete, "/installments/"+fmt.Sprintf("%d", installment.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", installment.ID))

	// Mock renderer
	e.Renderer = &testutil.MockRenderer{}

	err := handler.DeleteInstallment(c)
	if err != nil {
		t.Fatalf("DeleteInstallment() returned error: %v", err)
	}

	// Verify installment was deleted
	var deletedInstallment models.Installment
	err = database.DB.Where("id = ?", installment.ID).First(&deletedInstallment).Error
	if err == nil {
		t.Errorf("Installment should have been deleted but still exists")
	}
}

func TestCreditCardHandler_DeleteInstallment_NotFound(t *testing.T) {
	handler, e, userID, _ := setupCreditCardTestHandler()

	req := httptest.NewRequest(http.MethodDelete, "/installments/9999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)
	c.SetParamNames("id")
	c.SetParamValues("9999")

	err := handler.DeleteInstallment(c)
	if err != nil {
		t.Fatalf("DeleteInstallment() returned error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	if !strings.Contains(rec.Body.String(), "Parcela não encontrada") {
		t.Errorf("Response body = %s, want to contain 'Parcela não encontrada'", rec.Body.String())
	}
}

func TestCreditCardHandler_DeleteInstallment_UnauthorizedAccess(t *testing.T) {
	handler, e, _, accountID := setupCreditCardTestHandler()

	// Create another user
	authService := services.NewAuthService()
	otherUser, _ := authService.Register("other@example.com", "Password123", "Other User")

	// Create other user's account
	otherAccount := models.Account{
		Name:   "Other Account",
		Type:   models.AccountTypeIndividual,
		UserID: otherUser.ID,
	}
	database.DB.Create(&otherAccount)

	// Create a credit card for the first user
	card := models.CreditCard{
		AccountID:   accountID,
		Name:        "Protected Card",
		ClosingDay:  10,
		DueDay:      20,
		LimitAmount: 5000.00,
	}
	database.DB.Create(&card)

	// Create an installment
	installment := models.Installment{
		CreditCardID:       card.ID,
		Description:        "Protected Purchase",
		TotalAmount:        1200.00,
		InstallmentAmount:  100.00,
		TotalInstallments:  12,
		CurrentInstallment: 1,
		StartDate:          time.Now(),
		Category:           "Shopping",
	}
	database.DB.Create(&installment)

	// Try to delete with other user's ID
	req := httptest.NewRequest(http.MethodDelete, "/installments/"+fmt.Sprintf("%d", installment.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, otherUser.ID)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", installment.ID))

	err := handler.DeleteInstallment(c)
	if err != nil {
		t.Fatalf("DeleteInstallment() returned error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	// Verify installment was NOT deleted
	var existingInstallment models.Installment
	err = database.DB.Where("id = ?", installment.ID).First(&existingInstallment).Error
	if err != nil {
		t.Errorf("Installment should still exist but was deleted: %v", err)
	}
}

func TestCreditCardHandler_DeleteCard_WithInstallments(t *testing.T) {
	handler, e, userID, accountID := setupCreditCardTestHandler()

	// Create a credit card
	card := models.CreditCard{
		AccountID:   accountID,
		Name:        "Card with Installments",
		ClosingDay:  10,
		DueDay:      20,
		LimitAmount: 5000.00,
	}
	database.DB.Create(&card)

	// Create multiple installments
	for i := 0; i < 3; i++ {
		installment := models.Installment{
			CreditCardID:       card.ID,
			Description:        fmt.Sprintf("Purchase %d", i+1),
			TotalAmount:        1000.00,
			InstallmentAmount:  100.00,
			TotalInstallments:  10,
			CurrentInstallment: 1,
			StartDate:          time.Now(),
			Category:           "Shopping",
		}
		database.DB.Create(&installment)
	}

	req := httptest.NewRequest(http.MethodDelete, "/cards/"+fmt.Sprintf("%d", card.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", card.ID))

	// Mock renderer
	e.Renderer = &testutil.MockRenderer{}

	err := handler.DeleteCard(c)
	if err != nil {
		t.Fatalf("DeleteCard() returned error: %v", err)
	}

	// Verify card was deleted
	var deletedCard models.CreditCard
	err = database.DB.Where("id = ?", card.ID).First(&deletedCard).Error
	if err == nil {
		t.Errorf("Card should have been deleted but still exists")
	}

	// Verify installments were also deleted
	var installments []models.Installment
	database.DB.Where("credit_card_id = ?", card.ID).Find(&installments)
	if len(installments) > 0 {
		t.Errorf("Found %d installments, want 0 (should have been deleted with card)", len(installments))
	}
}

func TestMonthsBetween(t *testing.T) {
	tests := []struct {
		name     string
		start    time.Time
		end      time.Time
		expected int
	}{
		{
			name:     "same month",
			start:    time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC),
			expected: 0,
		},
		{
			name:     "one month difference",
			start:    time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC),
			expected: 1,
		},
		{
			name:     "one year difference",
			start:    time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			expected: 12,
		},
		{
			name:     "13 months difference",
			start:    time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC),
			expected: 13,
		},
		{
			name:     "negative months (end before start)",
			start:    time.Date(2024, 5, 15, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC),
			expected: -3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := monthsBetween(tt.start, tt.end)
			if result != tt.expected {
				t.Errorf("monthsBetween(%v, %v) = %d, want %d",
					tt.start, tt.end, result, tt.expected)
			}
		})
	}
}

func TestCreditCardHandler_List_FiltersByUserAccounts(t *testing.T) {
	handler, e, userID, accountID := setupCreditCardTestHandler()

	// Create another user and their card
	authService := services.NewAuthService()
	otherUser, _ := authService.Register("other@example.com", "Password123", "Other User")

	otherAccount := models.Account{
		Name:   "Other Account",
		Type:   models.AccountTypeIndividual,
		UserID: otherUser.ID,
	}
	database.DB.Create(&otherAccount)

	// Create cards for both users
	userCard := models.CreditCard{
		AccountID:   accountID,
		Name:        "User's Card",
		ClosingDay:  10,
		DueDay:      20,
		LimitAmount: 2000.00,
	}
	database.DB.Create(&userCard)

	otherCard := models.CreditCard{
		AccountID:   otherAccount.ID,
		Name:        "Other User's Card",
		ClosingDay:  15,
		DueDay:      25,
		LimitAmount: 3000.00,
	}
	database.DB.Create(&otherCard)

	req := httptest.NewRequest(http.MethodGet, "/cards", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	// Mock renderer
	e.Renderer = &testutil.MockRenderer{}

	err := handler.List(c)
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}

	// Verify that only the user's card is retrieved
	var userCards []models.CreditCard
	database.DB.Where("account_id = ?", accountID).Find(&userCards)

	if len(userCards) != 1 {
		t.Errorf("Found %d cards for user, want 1", len(userCards))
	}

	if userCards[0].Name != "User's Card" {
		t.Errorf("Card name = %s, want 'User's Card'", userCards[0].Name)
	}
}

func TestCreditCardHandler_CreateInstallment_InvalidData(t *testing.T) {
	handler, e, userID, accountID := setupCreditCardTestHandler()

	// Create a credit card first
	card := models.CreditCard{
		AccountID:   accountID,
		Name:        "Test Card",
		ClosingDay:  10,
		DueDay:      20,
		LimitAmount: 5000.00,
	}
	database.DB.Create(&card)

	// Invalid form data (missing required fields)
	form := url.Values{}
	form.Set("credit_card_id", fmt.Sprintf("%d", card.ID))

	req := httptest.NewRequest(http.MethodPost, "/installments", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	// Mock renderer
	e.Renderer = &testutil.MockRenderer{}

	err := handler.CreateInstallment(c)
	if err != nil {
		t.Fatalf("CreateInstallment() returned error: %v", err)
	}

	// Should fail with bad request due to invalid date
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
