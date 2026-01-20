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
	"poc-finance/internal/models"
	"poc-finance/internal/testutil"
)

func setupExportTestHandler() (*ExportHandler, *echo.Echo, uint) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create test account
	account := models.Account{
		Name: "Test Export Account",
		Type: models.AccountTypeIndividual,
	}
	database.DB.Create(&account)

	e := echo.New()
	handler := NewExportHandler()
	return handler, e, account.ID
}

func TestExportHandler_ExportYear_Success(t *testing.T) {
	handler, e, accountID := setupExportTestHandler()

	// Create test data - use current year for realistic testing
	now := time.Now()
	testYear := now.Year()
	testDate := time.Date(testYear, 3, 15, 0, 0, 0, 0, time.Local)

	// Create income
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

	// Create expense
	expense := models.Expense{
		AccountID: accountID,
		Name:      "Test Expense",
		Amount:    150.00,
		Type:      models.ExpenseTypeFixed,
		DueDay:    10,
		Category:  "Bills",
		Active:    true,
	}
	database.DB.Create(&expense)

	// Create credit card
	creditCard := models.CreditCard{
		AccountID:   accountID,
		Name:        "Test Card",
		ClosingDay:  15,
		DueDay:      20,
		LimitAmount: 5000.00,
	}
	database.DB.Create(&creditCard)

	// Create installment that started recently and is still active
	installment := models.Installment{
		CreditCardID:      creditCard.ID,
		Description:       "Test Purchase",
		TotalAmount:       1200.00,
		InstallmentAmount: 100.00,
		TotalInstallments: 24,
		StartDate:         now.AddDate(0, -2, 0), // Started 2 months ago
		Category:          "Shopping",
	}
	database.DB.Create(&installment)

	// Test export
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/export?year=%d", testYear), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.ExportYear(c)
	if err != nil {
		t.Fatalf("ExportYear() returned error: %v", err)
	}

	// Verify response headers
	contentType := rec.Header().Get("Content-Type")
	expectedContentType := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	if contentType != expectedContentType {
		t.Errorf("Content-Type = %s, want %s", contentType, expectedContentType)
	}

	contentDisposition := rec.Header().Get("Content-Disposition")
	expectedDisposition := fmt.Sprintf("attachment; filename=financeiro_%d.xlsx", testYear)
	if contentDisposition != expectedDisposition {
		t.Errorf("Content-Disposition = %s, want %s", contentDisposition, expectedDisposition)
	}

	// Verify the response contains Excel data by trying to open it
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
	expectedSheets := []string{"Resumo Mensal", "Recebimentos", "Despesas", "Parcelamentos"}
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

	// Verify "Resumo Mensal" sheet has headers
	rows, err := f.GetRows("Resumo Mensal")
	if err != nil {
		t.Fatalf("Failed to get rows from Resumo Mensal: %v", err)
	}

	if len(rows) < 1 {
		t.Error("Resumo Mensal sheet has no rows")
	}

	// Verify first row contains headers
	if len(rows) > 0 {
		headers := rows[0]
		expectedHeaders := []string{"Mês", "Receita Bruta", "Imposto", "Receita Líquida", "Despesas Fixas", "Despesas Variáveis", "Cartões", "Contas", "Total Despesas", "Saldo"}
		if len(headers) != len(expectedHeaders) {
			t.Errorf("Number of headers = %d, want %d", len(headers), len(expectedHeaders))
		}

		for i, expected := range expectedHeaders {
			if i < len(headers) && headers[i] != expected {
				t.Errorf("Header[%d] = %s, want %s", i, headers[i], expected)
			}
		}
	}

	// Verify "Resumo Mensal" has 12 data rows (one per month) + 1 header row
	if len(rows) != 13 {
		t.Errorf("Resumo Mensal rows = %d, want 13 (1 header + 12 months)", len(rows))
	}

	// Verify "Recebimentos" sheet has data
	incomesRows, err := f.GetRows("Recebimentos")
	if err != nil {
		t.Fatalf("Failed to get rows from Recebimentos: %v", err)
	}

	if len(incomesRows) < 2 {
		t.Error("Recebimentos sheet should have at least 2 rows (header + 1 income)")
	}

	// Verify "Despesas" sheet has data
	expensesRows, err := f.GetRows("Despesas")
	if err != nil {
		t.Fatalf("Failed to get rows from Despesas: %v", err)
	}

	if len(expensesRows) < 2 {
		t.Error("Despesas sheet should have at least 2 rows (header + 1 expense)")
	}

	// Verify "Parcelamentos" sheet has data
	installmentsRows, err := f.GetRows("Parcelamentos")
	if err != nil {
		t.Fatalf("Failed to get rows from Parcelamentos: %v", err)
	}

	if len(installmentsRows) < 2 {
		t.Error("Parcelamentos sheet should have at least 2 rows (header + 1 installment)")
	}
}

func TestExportHandler_ExportYear_EmptyDatabase(t *testing.T) {
	handler, e, _ := setupExportTestHandler()

	testYear := 2024

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/export?year=%d", testYear), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.ExportYear(c)
	if err != nil {
		t.Fatalf("ExportYear() returned error: %v", err)
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
	expectedSheets := []string{"Resumo Mensal", "Recebimentos", "Despesas", "Parcelamentos"}
	if len(sheets) != len(expectedSheets) {
		t.Errorf("Number of sheets = %d, want %d", len(sheets), len(expectedSheets))
	}

	// Verify "Resumo Mensal" still has 12 months of data (with zeros)
	rows, err := f.GetRows("Resumo Mensal")
	if err != nil {
		t.Fatalf("Failed to get rows from Resumo Mensal: %v", err)
	}

	if len(rows) != 13 {
		t.Errorf("Resumo Mensal rows = %d, want 13 (1 header + 12 months)", len(rows))
	}
}

func TestExportHandler_ExportYear_DefaultYear(t *testing.T) {
	handler, e, _ := setupExportTestHandler()

	// Request without year parameter should default to current year
	req := httptest.NewRequest(http.MethodGet, "/export", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.ExportYear(c)
	if err != nil {
		t.Fatalf("ExportYear() returned error: %v", err)
	}

	// Verify response headers contain current year
	currentYear := time.Now().Year()
	contentDisposition := rec.Header().Get("Content-Disposition")
	expectedDisposition := fmt.Sprintf("attachment; filename=financeiro_%d.xlsx", currentYear)
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

func TestExportHandler_ExportYear_InvalidYear(t *testing.T) {
	handler, e, _ := setupExportTestHandler()

	tests := []struct {
		name      string
		yearParam string
	}{
		{
			name:      "non-numeric year",
			yearParam: "abc",
		},
		{
			name:      "empty year",
			yearParam: "",
		},
		{
			name:      "special characters",
			yearParam: "20@4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var url string
			if tt.yearParam == "" {
				url = "/export"
			} else {
				url = fmt.Sprintf("/export?year=%s", tt.yearParam)
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.ExportYear(c)
			if err != nil {
				t.Fatalf("ExportYear() returned error: %v", err)
			}

			// Should default to current year and still work
			currentYear := time.Now().Year()
			contentDisposition := rec.Header().Get("Content-Disposition")
			expectedDisposition := fmt.Sprintf("attachment; filename=financeiro_%d.xlsx", currentYear)
			if contentDisposition != expectedDisposition {
				t.Errorf("Content-Disposition = %s, want %s", contentDisposition, expectedDisposition)
			}

			// Verify valid Excel file
			if rec.Body.Len() == 0 {
				t.Error("Response body is empty, expected Excel file data")
			}
		})
	}
}

func TestExportHandler_ExportYear_MultipleIncomes(t *testing.T) {
	handler, e, accountID := setupExportTestHandler()

	testYear := 2024

	// Create multiple incomes in different months
	incomes := []models.Income{
		{
			AccountID:    accountID,
			Date:         time.Date(testYear, 1, 15, 0, 0, 0, 0, time.Local),
			AmountUSD:    1000.00,
			ExchangeRate: 5.00,
			AmountBRL:    5000.00,
			GrossAmount:  5000.00,
			TaxAmount:    500.00,
			NetAmount:    4500.00,
			Description:  "January Income",
		},
		{
			AccountID:    accountID,
			Date:         time.Date(testYear, 3, 20, 0, 0, 0, 0, time.Local),
			AmountUSD:    1500.00,
			ExchangeRate: 5.10,
			AmountBRL:    7650.00,
			GrossAmount:  7650.00,
			TaxAmount:    765.00,
			NetAmount:    6885.00,
			Description:  "March Income",
		},
		{
			AccountID:    accountID,
			Date:         time.Date(testYear, 6, 10, 0, 0, 0, 0, time.Local),
			AmountUSD:    2000.00,
			ExchangeRate: 4.90,
			AmountBRL:    9800.00,
			GrossAmount:  9800.00,
			TaxAmount:    980.00,
			NetAmount:    8820.00,
			Description:  "June Income",
		},
	}

	for _, income := range incomes {
		database.DB.Create(&income)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/export?year=%d", testYear), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.ExportYear(c)
	if err != nil {
		t.Fatalf("ExportYear() returned error: %v", err)
	}

	// Parse Excel file
	f, err := excelize.OpenReader(rec.Body)
	if err != nil {
		t.Fatalf("Failed to parse Excel file: %v", err)
	}
	defer f.Close()

	// Verify "Recebimentos" sheet has all incomes
	rows, err := f.GetRows("Recebimentos")
	if err != nil {
		t.Fatalf("Failed to get rows from Recebimentos: %v", err)
	}

	// Should have 1 header + 3 income rows
	if len(rows) != 4 {
		t.Errorf("Recebimentos rows = %d, want 4 (1 header + 3 incomes)", len(rows))
	}

	// Verify incomes are in chronological order (after header row)
	if len(rows) >= 4 {
		// Check first income is January (row index 1, after header)
		if !contains(rows[1], "January Income") {
			t.Errorf("First income row should contain 'January Income', got: %v", rows[1])
		}

		// Check second income is March (row index 2)
		if !contains(rows[2], "March Income") {
			t.Errorf("Second income row should contain 'March Income', got: %v", rows[2])
		}

		// Check third income is June (row index 3)
		if !contains(rows[3], "June Income") {
			t.Errorf("Third income row should contain 'June Income', got: %v", rows[3])
		}
	}
}

func TestExportHandler_ExportYear_MultipleExpenses(t *testing.T) {
	handler, e, accountID := setupExportTestHandler()

	testYear := 2024

	// Create multiple expenses
	expenses := []models.Expense{
		{
			AccountID: accountID,
			Name:      "Rent",
			Amount:    1500.00,
			Type:      models.ExpenseTypeFixed,
			DueDay:    5,
			Category:  "Housing",
			Active:    true,
		},
		{
			AccountID: accountID,
			Name:      "Internet",
			Amount:    100.00,
			Type:      models.ExpenseTypeFixed,
			DueDay:    10,
			Category:  "Bills",
			Active:    true,
		},
		{
			AccountID: accountID,
			Name:      "Groceries",
			Amount:    300.00,
			Type:      models.ExpenseTypeVariable,
			DueDay:    1,
			Category:  "Food",
			Active:    true,
		},
	}

	for _, expense := range expenses {
		database.DB.Create(&expense)
	}

	// Create inactive expense separately and update it to ensure Active is false
	inactiveExpense := models.Expense{
		AccountID: accountID,
		Name:      "Old Subscription",
		Amount:    50.00,
		Type:      models.ExpenseTypeFixed,
		DueDay:    15,
		Category:  "Services",
		Active:    true, // Create as active first
	}
	database.DB.Create(&inactiveExpense)
	// Now update to inactive
	database.DB.Model(&inactiveExpense).Update("active", false)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/export?year=%d", testYear), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.ExportYear(c)
	if err != nil {
		t.Fatalf("ExportYear() returned error: %v", err)
	}

	// Parse Excel file
	f, err := excelize.OpenReader(rec.Body)
	if err != nil {
		t.Fatalf("Failed to parse Excel file: %v", err)
	}
	defer f.Close()

	// Verify "Despesas" sheet has all expenses
	rows, err := f.GetRows("Despesas")
	if err != nil {
		t.Fatalf("Failed to get rows from Despesas: %v", err)
	}

	// Should have 1 header + 4 expense rows
	if len(rows) != 5 {
		t.Errorf("Despesas rows = %d, want 5 (1 header + 4 expenses)", len(rows))
	}

	// Verify expense types are correctly labeled
	foundRent := false
	foundInternet := false
	foundGroceries := false
	foundOldSub := false

	for i, row := range rows {
		if i == 0 {
			continue // Skip header
		}

		if contains(row, "Rent") {
			foundRent = true
			if !contains(row, "Fixa") {
				t.Errorf("Rent should be labeled as 'Fixa', row: %v", row)
			}
			if !contains(row, "Sim") {
				t.Errorf("Rent should be active ('Sim'), row: %v", row)
			}
		}

		if contains(row, "Internet") {
			foundInternet = true
			if !contains(row, "Fixa") {
				t.Errorf("Internet should be labeled as 'Fixa', row: %v", row)
			}
		}

		if contains(row, "Groceries") {
			foundGroceries = true
			if !contains(row, "Variável") {
				t.Errorf("Groceries should be labeled as 'Variável', row: %v", row)
			}
		}

		if contains(row, "Old Subscription") {
			foundOldSub = true
			// Check that the active column (index 5) is "Não"
			if len(row) > 5 && row[5] != "Não" {
				t.Errorf("Old Subscription should be inactive ('Não'), got '%s', full row: %v", row[5], row)
			}
		}
	}

	if !foundRent || !foundInternet || !foundGroceries || !foundOldSub {
		t.Error("Not all expenses found in export")
	}
}

func TestExportHandler_ExportYear_MultipleInstallments(t *testing.T) {
	handler, e, accountID := setupExportTestHandler()

	now := time.Now()
	testYear := now.Year()

	// Create credit cards
	card1 := models.CreditCard{
		AccountID:   accountID,
		Name:        "Card 1",
		ClosingDay:  15,
		DueDay:      20,
		LimitAmount: 5000.00,
	}
	database.DB.Create(&card1)

	card2 := models.CreditCard{
		AccountID:   accountID,
		Name:        "Card 2",
		ClosingDay:  10,
		DueDay:      15,
		LimitAmount: 3000.00,
	}
	database.DB.Create(&card2)

	// Create installments
	installments := []models.Installment{
		{
			CreditCardID:      card1.ID,
			Description:       "Laptop",
			TotalAmount:       2400.00,
			InstallmentAmount: 200.00,
			TotalInstallments: 24,
			StartDate:         now.AddDate(0, -2, 0), // Started 2 months ago
			Category:          "Electronics",
		},
		{
			CreditCardID:      card2.ID,
			Description:       "Phone",
			TotalAmount:       1200.00,
			InstallmentAmount: 100.00,
			TotalInstallments: 18,
			StartDate:         now.AddDate(0, -1, 0), // Started 1 month ago
			Category:          "Electronics",
		},
		{
			CreditCardID:      card1.ID,
			Description:       "Furniture",
			TotalAmount:       3000.00,
			InstallmentAmount: 500.00,
			TotalInstallments: 6,
			StartDate:         now.AddDate(0, -10, 0), // Started 10 months ago, finished
			Category:          "Home",
		},
	}

	for _, inst := range installments {
		database.DB.Create(&inst)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/export?year=%d", testYear), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.ExportYear(c)
	if err != nil {
		t.Fatalf("ExportYear() returned error: %v", err)
	}

	// Parse Excel file
	f, err := excelize.OpenReader(rec.Body)
	if err != nil {
		t.Fatalf("Failed to parse Excel file: %v", err)
	}
	defer f.Close()

	// Verify "Parcelamentos" sheet
	rows, err := f.GetRows("Parcelamentos")
	if err != nil {
		t.Fatalf("Failed to get rows from Parcelamentos: %v", err)
	}

	// Should include only active installments (not fully paid)
	// Furniture started 10 months ago with 6 installments, so it has finished
	// Only Laptop and Phone should be present (header + 2 active installments = 3 rows)
	if len(rows) < 3 {
		t.Errorf("Parcelamentos rows = %d, want at least 3 (header + 2 active installments)", len(rows))
	}

	// Verify active installments are present
	foundLaptop := false
	foundPhone := false
	foundFurniture := false

	for i, row := range rows {
		if i == 0 {
			continue // Skip header
		}

		if contains(row, "Laptop") {
			foundLaptop = true
		}
		if contains(row, "Phone") {
			foundPhone = true
		}
		if contains(row, "Furniture") {
			foundFurniture = true
		}
	}

	if !foundLaptop {
		t.Error("Active Laptop installment should be in export")
	}
	if !foundPhone {
		t.Error("Active Phone installment should be in export")
	}
	if foundFurniture {
		t.Error("Completed Furniture installment should NOT be in export")
	}
}

func TestExportHandler_ExportYear_DifferentYears(t *testing.T) {
	handler, e, accountID := setupExportTestHandler()

	// Create incomes in different years
	income2023 := models.Income{
		AccountID:    accountID,
		Date:         time.Date(2023, 6, 15, 0, 0, 0, 0, time.Local),
		AmountUSD:    1000.00,
		ExchangeRate: 5.00,
		AmountBRL:    5000.00,
		GrossAmount:  5000.00,
		TaxAmount:    500.00,
		NetAmount:    4500.00,
		Description:  "2023 Income",
	}
	database.DB.Create(&income2023)

	income2024 := models.Income{
		AccountID:    accountID,
		Date:         time.Date(2024, 6, 15, 0, 0, 0, 0, time.Local),
		AmountUSD:    1500.00,
		ExchangeRate: 5.10,
		AmountBRL:    7650.00,
		GrossAmount:  7650.00,
		TaxAmount:    765.00,
		NetAmount:    6885.00,
		Description:  "2024 Income",
	}
	database.DB.Create(&income2024)

	// Export 2024
	req := httptest.NewRequest(http.MethodGet, "/export?year=2024", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.ExportYear(c)
	if err != nil {
		t.Fatalf("ExportYear() returned error: %v", err)
	}

	// Parse Excel file
	f, err := excelize.OpenReader(rec.Body)
	if err != nil {
		t.Fatalf("Failed to parse Excel file: %v", err)
	}
	defer f.Close()

	// Verify only 2024 income is in the export
	rows, err := f.GetRows("Recebimentos")
	if err != nil {
		t.Fatalf("Failed to get rows from Recebimentos: %v", err)
	}

	// Should have 1 header + 1 income (only 2024)
	if len(rows) != 2 {
		t.Errorf("Recebimentos rows = %d, want 2 (1 header + 1 income from 2024)", len(rows))
	}

	// Verify it's the 2024 income
	if len(rows) >= 2 {
		if !contains(rows[1], "2024 Income") {
			t.Error("Export should only contain 2024 income")
		}
		if contains(rows[1], "2023 Income") {
			t.Error("Export should not contain 2023 income")
		}
	}
}

func TestExportHandler_ExportYear_SpecialCharactersInData(t *testing.T) {
	handler, e, accountID := setupExportTestHandler()

	testYear := 2024

	// Create income with special characters
	income := models.Income{
		AccountID:    accountID,
		Date:         time.Date(testYear, 3, 15, 0, 0, 0, 0, time.Local),
		AmountUSD:    1000.00,
		ExchangeRate: 5.00,
		AmountBRL:    5000.00,
		GrossAmount:  5000.00,
		TaxAmount:    500.00,
		NetAmount:    4500.00,
		Description:  "Test & Special <Characters> \"Income\"",
	}
	database.DB.Create(&income)

	// Create expense with special characters
	expense := models.Expense{
		AccountID: accountID,
		Name:      "Expense with émojis 🎉 and symbols: @#$%",
		Amount:    150.00,
		Type:      models.ExpenseTypeFixed,
		DueDay:    10,
		Category:  "Special & Category",
		Active:    true,
	}
	database.DB.Create(&expense)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/export?year=%d", testYear), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.ExportYear(c)
	if err != nil {
		t.Fatalf("ExportYear() returned error: %v", err)
	}

	// Should still generate valid Excel file
	if rec.Body.Len() == 0 {
		t.Error("Response body is empty, expected Excel file data")
	}

	// Parse Excel file to verify it's valid
	f, err := excelize.OpenReader(rec.Body)
	if err != nil {
		t.Fatalf("Failed to parse Excel file with special characters: %v", err)
	}
	defer f.Close()

	// Verify data is present
	incomesRows, err := f.GetRows("Recebimentos")
	if err != nil {
		t.Fatalf("Failed to get rows from Recebimentos: %v", err)
	}

	if len(incomesRows) < 2 {
		t.Error("Recebimentos should have at least 2 rows")
	}

	expensesRows, err := f.GetRows("Despesas")
	if err != nil {
		t.Fatalf("Failed to get rows from Despesas: %v", err)
	}

	if len(expensesRows) < 2 {
		t.Error("Despesas should have at least 2 rows")
	}
}

// Helper function to check if a slice contains a string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
