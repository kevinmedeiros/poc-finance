package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/xuri/excelize/v2"

	"poc-finance/internal/database"
	"poc-finance/internal/middleware"
	"poc-finance/internal/models"
	"poc-finance/internal/services"
	"poc-finance/internal/testutil"
)

func setupTaxReportTestHandler() (*TaxReportHandler, *echo.Echo, uint, uint) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create test user
	authService := services.NewAuthService()
	user, _ := authService.Register("taxreport@example.com", "Password123", "Tax Report User")

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
	handler := NewTaxReportHandler(cacheService)
	return handler, e, user.ID, account.ID
}

func TestTaxReportHandler_TaxReportPage_Success(t *testing.T) {
	handler, e, userID, accountID := setupTaxReportTestHandler()

	// Create some test income data
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

	req := httptest.NewRequest(http.MethodGet, "/tax-report", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.TaxReportPage(c)
	if err != nil {
		t.Fatalf("TaxReportPage() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestTaxReportHandler_TaxReportPage_WithAccountFilter(t *testing.T) {
	handler, e, userID, accountID := setupTaxReportTestHandler()

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

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tax-report?account_id=%d", accountID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.TaxReportPage(c)
	if err != nil {
		t.Fatalf("TaxReportPage() with account filter returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestTaxReportHandler_TaxReportPage_WithAllAccountsFilter(t *testing.T) {
	handler, e, userID, accountID := setupTaxReportTestHandler()

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

	req := httptest.NewRequest(http.MethodGet, "/tax-report?account_id=all", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.TaxReportPage(c)
	if err != nil {
		t.Fatalf("TaxReportPage() with all accounts filter returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestTaxReportHandler_TaxReportPage_WithInvalidAccountID(t *testing.T) {
	handler, e, userID, _ := setupTaxReportTestHandler()

	// Try to access an account that doesn't belong to the user
	invalidAccountID := uint(99999)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tax-report?account_id=%d", invalidAccountID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.TaxReportPage(c)
	if err != nil {
		t.Fatalf("TaxReportPage() with invalid account ID returned error: %v", err)
	}

	// Should fallback to all accounts (status OK)
	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestTaxReportHandler_TaxReportPage_WithInvalidAccountIDFormat(t *testing.T) {
	handler, e, userID, _ := setupTaxReportTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/tax-report?account_id=invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.TaxReportPage(c)
	if err != nil {
		t.Fatalf("TaxReportPage() with invalid account ID format returned error: %v", err)
	}

	// Should fallback to all accounts (status OK)
	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestTaxReportHandler_TaxReportPage_WithYearFilter(t *testing.T) {
	handler, e, userID, accountID := setupTaxReportTestHandler()

	testYear := time.Now().Year()

	// Create income for the test year
	income := models.Income{
		AccountID:   accountID,
		Date:        time.Date(testYear, 6, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 8000.00,
		NetAmount:   6800.00,
		TaxAmount:   1200.00,
		Description: "Year Filtered Income",
	}
	database.DB.Create(&income)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tax-report?year=%d", testYear), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.TaxReportPage(c)
	if err != nil {
		t.Fatalf("TaxReportPage() with year filter returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestTaxReportHandler_TaxReportPage_WithPreviousYear(t *testing.T) {
	handler, e, userID, accountID := setupTaxReportTestHandler()

	previousYear := time.Now().Year() - 1

	// Create income for the previous year
	income := models.Income{
		AccountID:   accountID,
		Date:        time.Date(previousYear, 6, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 12000.00,
		NetAmount:   10200.00,
		TaxAmount:   1800.00,
		Description: "Previous Year Income",
	}
	database.DB.Create(&income)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tax-report?year=%d", previousYear), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.TaxReportPage(c)
	if err != nil {
		t.Fatalf("TaxReportPage() with previous year returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestTaxReportHandler_TaxReportPage_WithInvalidYear(t *testing.T) {
	handler, e, userID, _ := setupTaxReportTestHandler()

	tests := []struct {
		name      string
		yearParam string
	}{
		{
			name:      "non-numeric year",
			yearParam: "abc",
		},
		{
			name:      "year too old",
			yearParam: "2010",
		},
		{
			name:      "year too far in future",
			yearParam: "2099",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tax-report?year=%s", tt.yearParam), nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set(middleware.UserIDKey, userID)

			err := handler.TaxReportPage(c)
			if err != nil {
				t.Fatalf("TaxReportPage() returned error: %v", err)
			}

			// Should default to current year and still work
			if rec.Code != http.StatusOK {
				t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
			}
		})
	}
}

func TestTaxReportHandler_TaxReportPage_NoAccounts(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create user without any accounts
	authService := services.NewAuthService()
	user, _ := authService.Register("noaccounts@example.com", "Password123", "No Accounts User")

	cacheService := services.NewSettingsCacheService()
	e := echo.New()
	e.Renderer = &testutil.MockRenderer{}
	handler := NewTaxReportHandler(cacheService)

	req := httptest.NewRequest(http.MethodGet, "/tax-report", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, user.ID)

	err := handler.TaxReportPage(c)
	if err != nil {
		t.Fatalf("TaxReportPage() with no accounts returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestTaxReportHandler_TaxReportPage_MultipleIncomes(t *testing.T) {
	handler, e, userID, accountID := setupTaxReportTestHandler()

	now := time.Now()
	testYear := now.Year()

	// Create multiple incomes in different months
	incomes := []models.Income{
		{
			AccountID:   accountID,
			Date:        time.Date(testYear, 1, 15, 0, 0, 0, 0, time.Local),
			GrossAmount: 5000.00,
			NetAmount:   4250.00,
			TaxAmount:   750.00,
			Description: "January Income",
		},
		{
			AccountID:   accountID,
			Date:        time.Date(testYear, 3, 15, 0, 0, 0, 0, time.Local),
			GrossAmount: 7500.00,
			NetAmount:   6375.00,
			TaxAmount:   1125.00,
			Description: "March Income",
		},
		{
			AccountID:   accountID,
			Date:        time.Date(testYear, 6, 15, 0, 0, 0, 0, time.Local),
			GrossAmount: 10000.00,
			NetAmount:   8500.00,
			TaxAmount:   1500.00,
			Description: "June Income",
		},
	}

	for _, income := range incomes {
		database.DB.Create(&income)
	}

	req := httptest.NewRequest(http.MethodGet, "/tax-report", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.TaxReportPage(c)
	if err != nil {
		t.Fatalf("TaxReportPage() with multiple incomes returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestTaxReportHandler_ExportTaxReport_ExcelSuccess(t *testing.T) {
	handler, e, userID, accountID := setupTaxReportTestHandler()

	// Create test data
	now := time.Now()
	testYear := now.Year()
	testDate := time.Date(testYear, 3, 15, 0, 0, 0, 0, time.Local)

	income := models.Income{
		AccountID:    accountID,
		Date:         testDate,
		AmountUSD:    1000.00,
		ExchangeRate: 5.00,
		AmountBRL:    5000.00,
		GrossAmount:  5000.00,
		TaxAmount:    500.00,
		NetAmount:    4500.00,
		Description:  "Test Income",
	}
	database.DB.Create(&income)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tax-report/export?year=%d", testYear), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.ExportTaxReport(c)
	if err != nil {
		t.Fatalf("ExportTaxReport() returned error: %v", err)
	}

	// Verify response headers
	contentType := rec.Header().Get("Content-Type")
	expectedContentType := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	if contentType != expectedContentType {
		t.Errorf("Content-Type = %s, want %s", contentType, expectedContentType)
	}

	contentDisposition := rec.Header().Get("Content-Disposition")
	expectedDisposition := fmt.Sprintf("attachment; filename=relatorio_fiscal_%d.xlsx", testYear)
	if contentDisposition != expectedDisposition {
		t.Errorf("Content-Disposition = %s, want %s", contentDisposition, expectedDisposition)
	}

	// Verify the response contains Excel data
	if rec.Body.Len() == 0 {
		t.Error("Response body is empty, expected Excel file data")
	}

	// Parse the Excel file to verify it's valid
	f, err := excelize.OpenReader(rec.Body)
	if err != nil {
		t.Fatalf("Failed to parse Excel file: %v", err)
	}
	defer f.Close()

	// Verify sheets exist
	sheets := f.GetSheetList()
	expectedSheets := []string{"Resumo Fiscal", "Impostos Mensais", "Faixas Simples Nacional"}
	if len(sheets) != len(expectedSheets) {
		t.Errorf("Number of sheets = %d, want %d", len(sheets), len(expectedSheets))
	}

	for _, expectedSheet := range expectedSheets {
		found := false
		for _, sheet := range sheets {
			if sheet == expectedSheet {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected sheet %s not found in workbook", expectedSheet)
		}
	}
}

func TestTaxReportHandler_ExportTaxReport_PDFSuccess(t *testing.T) {
	handler, e, userID, accountID := setupTaxReportTestHandler()

	// Create test data
	now := time.Now()
	testYear := now.Year()

	income := models.Income{
		AccountID:   accountID,
		Date:        time.Date(testYear, 3, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 5000.00,
		TaxAmount:   500.00,
		NetAmount:   4500.00,
		Description: "Test Income",
	}
	database.DB.Create(&income)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tax-report/export?year=%d&format=pdf", testYear), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.ExportTaxReport(c)
	if err != nil {
		t.Fatalf("ExportTaxReport() with PDF format returned error: %v", err)
	}

	// Verify response headers
	contentType := rec.Header().Get("Content-Type")
	expectedContentType := "application/pdf"
	if contentType != expectedContentType {
		t.Errorf("Content-Type = %s, want %s", contentType, expectedContentType)
	}

	contentDisposition := rec.Header().Get("Content-Disposition")
	expectedDisposition := fmt.Sprintf("attachment; filename=relatorio_fiscal_%d.pdf", testYear)
	if contentDisposition != expectedDisposition {
		t.Errorf("Content-Disposition = %s, want %s", contentDisposition, expectedDisposition)
	}

	// Verify the response contains PDF data
	if rec.Body.Len() == 0 {
		t.Error("Response body is empty, expected PDF file data")
	}

	// Verify PDF magic bytes (%PDF-)
	body := rec.Body.Bytes()
	if len(body) < 4 || string(body[:4]) != "%PDF" {
		t.Error("Response does not appear to be a valid PDF file")
	}
}

func TestTaxReportHandler_ExportTaxReport_DefaultYear(t *testing.T) {
	handler, e, userID, _ := setupTaxReportTestHandler()

	// Request without year parameter should default to current year
	req := httptest.NewRequest(http.MethodGet, "/tax-report/export", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.ExportTaxReport(c)
	if err != nil {
		t.Fatalf("ExportTaxReport() returned error: %v", err)
	}

	// Verify response headers contain current year
	currentYear := time.Now().Year()
	contentDisposition := rec.Header().Get("Content-Disposition")
	expectedDisposition := fmt.Sprintf("attachment; filename=relatorio_fiscal_%d.xlsx", currentYear)
	if contentDisposition != expectedDisposition {
		t.Errorf("Content-Disposition = %s, want %s", contentDisposition, expectedDisposition)
	}

	// Verify valid Excel file
	if rec.Body.Len() == 0 {
		t.Error("Response body is empty, expected Excel file data")
	}

	f, err := excelize.OpenReader(rec.Body)
	if err != nil {
		t.Fatalf("Failed to parse Excel file: %v", err)
	}
	defer f.Close()
}

func TestTaxReportHandler_ExportTaxReport_EmptyDatabase(t *testing.T) {
	handler, e, userID, _ := setupTaxReportTestHandler()

	testYear := time.Now().Year()

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tax-report/export?year=%d", testYear), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.ExportTaxReport(c)
	if err != nil {
		t.Fatalf("ExportTaxReport() returned error: %v", err)
	}

	// Should still generate a valid Excel file even with no data
	if rec.Body.Len() == 0 {
		t.Error("Response body is empty, expected Excel file data")
	}

	// Parse the Excel file
	f, err := excelize.OpenReader(rec.Body)
	if err != nil {
		t.Fatalf("Failed to parse Excel file: %v", err)
	}
	defer f.Close()

	// Verify sheets exist
	sheets := f.GetSheetList()
	expectedSheets := []string{"Resumo Fiscal", "Impostos Mensais", "Faixas Simples Nacional"}
	if len(sheets) != len(expectedSheets) {
		t.Errorf("Number of sheets = %d, want %d", len(sheets), len(expectedSheets))
	}
}

func TestTaxReportHandler_ExportTaxReport_WithAccountFilter(t *testing.T) {
	handler, e, userID, accountID := setupTaxReportTestHandler()

	// Create second account
	account2 := models.Account{
		Name:   "Second Account",
		Type:   models.AccountTypeIndividual,
		UserID: userID,
	}
	database.DB.Create(&account2)

	now := time.Now()
	testYear := now.Year()

	// Create income for both accounts
	income1 := models.Income{
		AccountID:   accountID,
		Date:        time.Date(testYear, 3, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 5000.00,
		TaxAmount:   500.00,
		NetAmount:   4500.00,
		Description: "Income 1",
	}
	database.DB.Create(&income1)

	income2 := models.Income{
		AccountID:   account2.ID,
		Date:        time.Date(testYear, 4, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 3000.00,
		TaxAmount:   300.00,
		NetAmount:   2700.00,
		Description: "Income 2",
	}
	database.DB.Create(&income2)

	// Export with specific account filter
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tax-report/export?year=%d&account_id=%d", testYear, accountID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.ExportTaxReport(c)
	if err != nil {
		t.Fatalf("ExportTaxReport() with account filter returned error: %v", err)
	}

	// Verify valid Excel file
	if rec.Body.Len() == 0 {
		t.Error("Response body is empty, expected Excel file data")
	}

	f, err := excelize.OpenReader(rec.Body)
	if err != nil {
		t.Fatalf("Failed to parse Excel file: %v", err)
	}
	defer f.Close()
}

func TestTaxReportHandler_ExportTaxReport_InvalidAccountID(t *testing.T) {
	handler, e, userID, _ := setupTaxReportTestHandler()

	testYear := time.Now().Year()
	invalidAccountID := uint(99999)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tax-report/export?year=%d&account_id=%d", testYear, invalidAccountID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.ExportTaxReport(c)
	if err != nil {
		t.Fatalf("ExportTaxReport() with invalid account ID returned error: %v", err)
	}

	// Should fallback to all accounts and still generate valid file
	if rec.Body.Len() == 0 {
		t.Error("Response body is empty, expected Excel file data")
	}

	f, err := excelize.OpenReader(rec.Body)
	if err != nil {
		t.Fatalf("Failed to parse Excel file: %v", err)
	}
	defer f.Close()
}

func TestTaxReportHandler_ExportTaxReport_MonthlyTaxSheet(t *testing.T) {
	handler, e, userID, accountID := setupTaxReportTestHandler()

	now := time.Now()
	testYear := now.Year()

	// Create incomes for multiple months
	incomes := []models.Income{
		{
			AccountID:   accountID,
			Date:        time.Date(testYear, 1, 15, 0, 0, 0, 0, time.Local),
			GrossAmount: 5000.00,
			TaxAmount:   500.00,
			NetAmount:   4500.00,
			Description: "January Income",
		},
		{
			AccountID:   accountID,
			Date:        time.Date(testYear, 2, 15, 0, 0, 0, 0, time.Local),
			GrossAmount: 6000.00,
			TaxAmount:   600.00,
			NetAmount:   5400.00,
			Description: "February Income",
		},
		{
			AccountID:   accountID,
			Date:        time.Date(testYear, 3, 15, 0, 0, 0, 0, time.Local),
			GrossAmount: 7000.00,
			TaxAmount:   700.00,
			NetAmount:   6300.00,
			Description: "March Income",
		},
	}

	for _, income := range incomes {
		database.DB.Create(&income)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tax-report/export?year=%d", testYear), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.ExportTaxReport(c)
	if err != nil {
		t.Fatalf("ExportTaxReport() returned error: %v", err)
	}

	f, err := excelize.OpenReader(rec.Body)
	if err != nil {
		t.Fatalf("Failed to parse Excel file: %v", err)
	}
	defer f.Close()

	// Verify "Impostos Mensais" sheet has headers
	rows, err := f.GetRows("Impostos Mensais")
	if err != nil {
		t.Fatalf("Failed to get rows from Impostos Mensais: %v", err)
	}

	if len(rows) < 1 {
		t.Error("Impostos Mensais sheet has no rows")
	}

	// Verify first row contains headers
	if len(rows) > 0 {
		headers := rows[0]
		expectedHeaders := []string{"Mês", "Receita Bruta", "Imposto", "INSS", "Receita Líquida"}
		if len(headers) != len(expectedHeaders) {
			t.Errorf("Number of headers = %d, want %d", len(headers), len(expectedHeaders))
		}

		for i, expected := range expectedHeaders {
			if i < len(headers) && headers[i] != expected {
				t.Errorf("Header[%d] = %s, want %s", i, headers[i], expected)
			}
		}
	}

	// Verify "Impostos Mensais" has 12 data rows (one per month) + 1 header row + 1 total row
	if len(rows) != 14 {
		t.Errorf("Impostos Mensais rows = %d, want 14 (1 header + 12 months + 1 total)", len(rows))
	}
}

func TestTaxReportHandler_ExportTaxReport_BracketInfoSheet(t *testing.T) {
	handler, e, userID, accountID := setupTaxReportTestHandler()

	now := time.Now()
	testYear := now.Year()

	// Create income to establish revenue
	income := models.Income{
		AccountID:   accountID,
		Date:        time.Date(testYear, 3, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 15000.00,
		TaxAmount:   1500.00,
		NetAmount:   13500.00,
		Description: "Test Income",
	}
	database.DB.Create(&income)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tax-report/export?year=%d", testYear), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.ExportTaxReport(c)
	if err != nil {
		t.Fatalf("ExportTaxReport() returned error: %v", err)
	}

	f, err := excelize.OpenReader(rec.Body)
	if err != nil {
		t.Fatalf("Failed to parse Excel file: %v", err)
	}
	defer f.Close()

	// Verify "Faixas Simples Nacional" sheet has headers
	rows, err := f.GetRows("Faixas Simples Nacional")
	if err != nil {
		t.Fatalf("Failed to get rows from Faixas Simples Nacional: %v", err)
	}

	if len(rows) < 1 {
		t.Error("Faixas Simples Nacional sheet has no rows")
	}

	// Verify first row contains headers
	if len(rows) > 0 {
		headers := rows[0]
		expectedHeaders := []string{"Faixa", "Receita Bruta Anual (até)", "Alíquota Nominal", "Dedução", "Status"}
		if len(headers) != len(expectedHeaders) {
			t.Errorf("Number of headers = %d, want %d", len(headers), len(expectedHeaders))
		}

		for i, expected := range expectedHeaders {
			if i < len(headers) && headers[i] != expected {
				t.Errorf("Header[%d] = %s, want %s", i, headers[i], expected)
			}
		}
	}

	// Verify sheet has 6 bracket rows + 1 header + 1 empty + 1 current revenue info
	if len(rows) < 7 {
		t.Errorf("Faixas Simples Nacional rows = %d, want at least 7 (1 header + 6 brackets)", len(rows))
	}
}

func TestTaxReportHandler_ExportTaxReport_PreviousYear(t *testing.T) {
	handler, e, userID, accountID := setupTaxReportTestHandler()

	previousYear := time.Now().Year() - 1

	// Create income for the previous year
	income := models.Income{
		AccountID:   accountID,
		Date:        time.Date(previousYear, 6, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 12000.00,
		TaxAmount:   1200.00,
		NetAmount:   10800.00,
		Description: "Previous Year Income",
	}
	database.DB.Create(&income)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tax-report/export?year=%d", previousYear), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.ExportTaxReport(c)
	if err != nil {
		t.Fatalf("ExportTaxReport() with previous year returned error: %v", err)
	}

	// Verify response headers contain previous year
	contentDisposition := rec.Header().Get("Content-Disposition")
	expectedDisposition := fmt.Sprintf("attachment; filename=relatorio_fiscal_%d.xlsx", previousYear)
	if contentDisposition != expectedDisposition {
		t.Errorf("Content-Disposition = %s, want %s", contentDisposition, expectedDisposition)
	}

	// Verify valid Excel file
	f, err := excelize.OpenReader(rec.Body)
	if err != nil {
		t.Fatalf("Failed to parse Excel file: %v", err)
	}
	defer f.Close()
}

func TestTaxReportHandler_ExportTaxReport_PDFWithMultipleMonths(t *testing.T) {
	handler, e, userID, accountID := setupTaxReportTestHandler()

	now := time.Now()
	testYear := now.Year()

	// Create incomes for multiple months
	for month := 1; month <= 6; month++ {
		income := models.Income{
			AccountID:   accountID,
			Date:        time.Date(testYear, time.Month(month), 15, 0, 0, 0, 0, time.Local),
			GrossAmount: float64(month * 1000),
			TaxAmount:   float64(month * 100),
			NetAmount:   float64(month * 900),
			Description: fmt.Sprintf("Month %d Income", month),
		}
		database.DB.Create(&income)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tax-report/export?year=%d&format=pdf", testYear), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.ExportTaxReport(c)
	if err != nil {
		t.Fatalf("ExportTaxReport() with PDF format returned error: %v", err)
	}

	// Verify PDF is generated
	if rec.Body.Len() == 0 {
		t.Error("Response body is empty, expected PDF file data")
	}

	// Verify PDF magic bytes
	body := rec.Body.Bytes()
	if len(body) < 4 || string(body[:4]) != "%PDF" {
		t.Error("Response does not appear to be a valid PDF file")
	}
}

func TestTaxReportHandler_ExportTaxReport_DefaultFormat(t *testing.T) {
	handler, e, userID, _ := setupTaxReportTestHandler()

	// Request without format parameter should default to xlsx
	req := httptest.NewRequest(http.MethodGet, "/tax-report/export", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.ExportTaxReport(c)
	if err != nil {
		t.Fatalf("ExportTaxReport() returned error: %v", err)
	}

	// Verify response is Excel format
	contentType := rec.Header().Get("Content-Type")
	expectedContentType := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	if contentType != expectedContentType {
		t.Errorf("Content-Type = %s, want %s", contentType, expectedContentType)
	}
}

func TestGetAvailableYears(t *testing.T) {
	currentYear := time.Now().Year()
	years := getAvailableYears(currentYear)

	// Should have 6 years (current year and 5 previous years, limited by 2020)
	expectedCount := 6
	if currentYear-5 < 2020 {
		expectedCount = currentYear - 2020 + 1
	}

	if len(years) != expectedCount {
		t.Errorf("Number of available years = %d, want %d", len(years), expectedCount)
	}

	// First year should be current year
	if len(years) > 0 && years[0] != currentYear {
		t.Errorf("First year = %d, want %d", years[0], currentYear)
	}

	// Years should be in descending order
	for i := 0; i < len(years)-1; i++ {
		if years[i] <= years[i+1] {
			t.Errorf("Years not in descending order: years[%d]=%d, years[%d]=%d", i, years[i], i+1, years[i+1])
		}
	}

	// No year should be less than 2020
	for _, year := range years {
		if year < 2020 {
			t.Errorf("Year %d is less than 2020", year)
		}
	}
}
