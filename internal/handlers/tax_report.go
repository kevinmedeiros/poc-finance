package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/labstack/echo/v4"
	"github.com/xuri/excelize/v2"

	"poc-finance/internal/database"
	"poc-finance/internal/i18n"
	"poc-finance/internal/middleware"
	"poc-finance/internal/services"
)

// TaxReportHandler handles tax projection and report pages
type TaxReportHandler struct {
	accountService *services.AccountService
	cacheService   *services.SettingsCacheService
}

// NewTaxReportHandler creates a new TaxReportHandler instance
func NewTaxReportHandler(cacheService *services.SettingsCacheService) *TaxReportHandler {
	return &TaxReportHandler{
		accountService: services.NewAccountService(),
		cacheService:   cacheService,
	}
}

// TaxReportPage renders the tax projection and report page
func (h *TaxReportHandler) TaxReportPage(c echo.Context) error {
	log.Println("[TaxReport] Loading tax report page")

	userID := middleware.GetUserID(c)
	allAccounts, _ := h.accountService.GetUserAccounts(userID)
	allAccountIDs, _ := h.accountService.GetUserAccountIDs(userID)

	// Handle account filter from query parameter
	var selectedAccountID uint
	var accountIDs []uint
	accountIDParam := c.QueryParam("account_id")

	if accountIDParam != "" && accountIDParam != "all" {
		parsedID, err := strconv.ParseUint(accountIDParam, 10, 32)
		if err == nil {
			selectedAccountID = uint(parsedID)
			// Validate user has access to this account
			if h.accountService.CanUserAccessAccount(userID, selectedAccountID) {
				accountIDs = []uint{selectedAccountID}
			} else {
				// Fallback to all accounts if invalid
				accountIDs = allAccountIDs
			}
		} else {
			accountIDs = allAccountIDs
		}
	} else {
		accountIDs = allAccountIDs
	}

	// Handle year parameter (default to current year)
	now := time.Now()
	year := now.Year()
	yearParam := c.QueryParam("year")
	if yearParam != "" {
		parsedYear, err := strconv.Atoi(yearParam)
		if err == nil && parsedYear >= 2020 && parsedYear <= year+1 {
			year = parsedYear
		}
	}

	// Build INSS config from settings
	settingsData := h.cacheService.GetSettingsData()
	inssConfig := &services.INSSConfig{
		ProLabore: settingsData.ProLabore,
		Ceiling:   settingsData.INSSCeiling,
		Rate:      settingsData.INSSRate / 100, // Convert % to decimal
	}

	// Get tax projection for the selected year
	log.Printf("[TaxReport] Fetching tax projection for year %d", year)
	var projection services.TaxProjection
	if year == now.Year() {
		projection = services.GetTaxProjection(database.DB, accountIDs, inssConfig)
	} else {
		projection = services.GetTaxProjectionForYear(database.DB, year, accountIDs, inssConfig)
	}

	// Get monthly tax breakdown for detailed view
	log.Printf("[TaxReport] Fetching monthly tax breakdown for year %d", year)
	monthlyBreakdown := services.GetMonthlyTaxBreakdown(database.DB, year, accountIDs, inssConfig)

	// Get revenue 12 months and bracket info (with manual override support)
	revenue12M := services.GetRevenue12MonthsForAccounts(database.DB, accountIDs)
	bracket, effectiveRate, nextBracketAt := services.GetBracketInfoWithManualOverride(revenue12M, settingsData.ManualBracket)

	// Calculate bracket progress (percentage towards next bracket)
	var bracketProgress float64
	if projection.BracketWarning != nil {
		bracketProgress = projection.BracketWarning.PercentToNext
	}

	// Calculate amount remaining in current bracket
	amountToNextBracket := nextBracketAt - revenue12M
	if amountToNextBracket < 0 {
		amountToNextBracket = 0
	}

	// Build available years list for year selector
	availableYears := getAvailableYears(now.Year())

	log.Println("[TaxReport] Tax report data loaded successfully - rendering template")
	data := map[string]interface{}{
		// Projection data
		"projection":       projection,
		"monthlyBreakdown": monthlyBreakdown,

		// Current bracket info
		"revenue12m":          revenue12M,
		"currentBracket":      bracket,
		"effectiveRate":       effectiveRate,
		"nextBracketAt":       nextBracketAt,
		"amountToNextBracket": amountToNextBracket,
		"bracketProgress":     bracketProgress,

		// INSS info
		"inssMonthly":   settingsData.INSSAmount,
		"proLabore":     settingsData.ProLabore,
		"inssRate":      settingsData.INSSRate,
		"inssCeiling":   settingsData.INSSCeiling,

		// Filter/selection state
		"accounts":          allAccounts,
		"selectedAccountID": selectedAccountID,
		"selectedYear":      year,
		"availableYears":    availableYears,
		"currentYear":       now.Year(),
		"now":               now,
	}

	return c.Render(http.StatusOK, "tax-report.html", data)
}

// getAvailableYears returns a list of years available for selection
// Includes current year and up to 5 previous years
func getAvailableYears(currentYear int) []int {
	years := make([]int, 0, 6)
	for y := currentYear; y >= currentYear-5 && y >= 2020; y-- {
		years = append(years, y)
	}
	return years
}

// ExportTaxReport exports tax report data to Excel or PDF format
func (h *TaxReportHandler) ExportTaxReport(c echo.Context) error {
	log.Println("[TaxReport] Exporting tax report")

	userID := middleware.GetUserID(c)
	allAccountIDs, _ := h.accountService.GetUserAccountIDs(userID)

	// Handle account filter from query parameter
	var accountIDs []uint
	accountIDParam := c.QueryParam("account_id")

	if accountIDParam != "" && accountIDParam != "all" {
		parsedID, err := strconv.ParseUint(accountIDParam, 10, 32)
		if err == nil {
			selectedAccountID := uint(parsedID)
			if h.accountService.CanUserAccessAccount(userID, selectedAccountID) {
				accountIDs = []uint{selectedAccountID}
			} else {
				accountIDs = allAccountIDs
			}
		} else {
			accountIDs = allAccountIDs
		}
	} else {
		accountIDs = allAccountIDs
	}

	// Handle year parameter (default to current year)
	now := time.Now()
	year := now.Year()
	yearParam := c.QueryParam("year")
	if yearParam != "" {
		parsedYear, err := strconv.Atoi(yearParam)
		if err == nil && parsedYear >= 2020 && parsedYear <= year+1 {
			year = parsedYear
		}
	}

	// Check format parameter (default to xlsx)
	format := c.QueryParam("format")
	if format == "" {
		format = "xlsx"
	}

	// Build INSS config from settings
	settingsData := h.cacheService.GetSettingsData()
	inssConfig := &services.INSSConfig{
		ProLabore: settingsData.ProLabore,
		Ceiling:   settingsData.INSSCeiling,
		Rate:      settingsData.INSSRate / 100,
	}

	// Export to PDF if requested
	if format == "pdf" {
		return h.exportTaxReportPDF(c, year, accountIDs, inssConfig, settingsData)
	}

	// Get tax projection for the selected year
	var projection services.TaxProjection
	if year == now.Year() {
		projection = services.GetTaxProjection(database.DB, accountIDs, inssConfig)
	} else {
		projection = services.GetTaxProjectionForYear(database.DB, year, accountIDs, inssConfig)
	}

	// Get monthly tax breakdown for detailed view
	monthlyBreakdown := services.GetMonthlyTaxBreakdown(database.DB, year, accountIDs, inssConfig)

	// Get bracket info (with manual override support)
	revenue12M := services.GetRevenue12MonthsForAccounts(database.DB, accountIDs)
	bracket, effectiveRate, nextBracketAt := services.GetBracketInfoWithManualOverride(revenue12M, settingsData.ManualBracket)

	// Create Excel file
	f := excelize.NewFile()
	defer f.Close()

	// Create tax-specific sheets
	h.createTaxSummarySheet(f, year, projection, revenue12M, bracket, effectiveRate, nextBracketAt, settingsData)
	h.createMonthlyTaxSheet(f, year, monthlyBreakdown)
	h.createTaxBracketInfoSheet(f, revenue12M)

	// Remove default sheet
	f.DeleteSheet("Sheet1")

	// Set headers for download
	c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=relatorio_fiscal_%d.xlsx", year))

	log.Printf("[TaxReport] Tax report exported successfully for year %d", year)
	return f.Write(c.Response().Writer)
}

// createTaxSummarySheet creates the tax summary sheet with projection data
func (h *TaxReportHandler) createTaxSummarySheet(f *excelize.File, year int, projection services.TaxProjection, revenue12M float64, bracket int, effectiveRate float64, nextBracketAt float64, settings services.SettingsData) {
	sheet := "Resumo Fiscal"
	f.NewSheet(sheet)

	// Header style (blue background)
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	// Section header style (dark blue)
	sectionStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#2F5496"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left"},
	})

	// Currency style
	currencyStyle, _ := f.NewStyle(&excelize.Style{
		NumFmt: 4, // #,##0.00
	})

	// Percentage style
	percentStyle, _ := f.NewStyle(&excelize.Style{
		NumFmt: 10, // 0.00%
	})

	row := 1

	// Title
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("Relatório Fiscal - %d", year))
	f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row))
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), headerStyle)
	row += 2

	// Year-to-Date Section
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Acumulado no Ano (YTD)")
	f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row))
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), sectionStyle)
	row++

	ytdData := []struct {
		label string
		value float64
	}{
		{"Receita Bruta", projection.YTDIncome},
		{"Imposto Pago", projection.YTDTax},
		{"INSS Pago", projection.YTDINSS},
		{"Receita Líquida", projection.YTDNetIncome},
	}

	for _, d := range ytdData {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), d.label)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), d.value)
		f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), currencyStyle)
		row++
	}
	row++

	// Projections Section
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Projeção Anual")
	f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row))
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), sectionStyle)
	row++

	projectionData := []struct {
		label string
		value float64
	}{
		{"Receita Bruta Projetada", projection.ProjectedAnnualIncome},
		{"Imposto Projetado", projection.ProjectedAnnualTax},
		{"INSS Projetado", projection.ProjectedAnnualINSS},
		{"Receita Líquida Projetada", projection.ProjectedNetIncome},
	}

	for _, d := range projectionData {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), d.label)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), d.value)
		f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), currencyStyle)
		row++
	}
	row++

	// Bracket Info Section
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Informações da Faixa")
	f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row))
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), sectionStyle)
	row++

	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Faturamento 12 Meses")
	f.SetCellValue(sheet, fmt.Sprintf("B%d", row), revenue12M)
	f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), currencyStyle)
	row++

	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Faixa Atual")
	f.SetCellValue(sheet, fmt.Sprintf("B%d", row), bracket)
	row++

	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Alíquota Efetiva")
	f.SetCellValue(sheet, fmt.Sprintf("B%d", row), effectiveRate/100)
	f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), percentStyle)
	row++

	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Próxima Faixa em")
	f.SetCellValue(sheet, fmt.Sprintf("B%d", row), nextBracketAt)
	f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), currencyStyle)
	row++

	amountToNext := nextBracketAt - revenue12M
	if amountToNext < 0 {
		amountToNext = 0
	}
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Valor até Próxima Faixa")
	f.SetCellValue(sheet, fmt.Sprintf("B%d", row), amountToNext)
	f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), currencyStyle)
	row += 2

	// INSS Info Section
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Configuração INSS")
	f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row))
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), sectionStyle)
	row++

	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Pró-Labore")
	f.SetCellValue(sheet, fmt.Sprintf("B%d", row), settings.ProLabore)
	f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), currencyStyle)
	row++

	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Alíquota INSS")
	f.SetCellValue(sheet, fmt.Sprintf("B%d", row), settings.INSSRate/100)
	f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), percentStyle)
	row++

	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Teto INSS")
	f.SetCellValue(sheet, fmt.Sprintf("B%d", row), settings.INSSCeiling)
	f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), currencyStyle)
	row++

	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "INSS Mensal")
	f.SetCellValue(sheet, fmt.Sprintf("B%d", row), settings.INSSAmount)
	f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), currencyStyle)
	row += 2

	// Metadata
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Gerado em")
	f.SetCellValue(sheet, fmt.Sprintf("B%d", row), time.Now().Format("02/01/2006 15:04"))
	row++

	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Meses Decorridos")
	f.SetCellValue(sheet, fmt.Sprintf("B%d", row), projection.MonthsElapsed)

	// Adjust column widths
	f.SetColWidth(sheet, "A", "A", 30)
	f.SetColWidth(sheet, "B", "B", 20)
	f.SetColWidth(sheet, "C", "C", 15)
}

// createMonthlyTaxSheet creates the monthly tax breakdown sheet
func (h *TaxReportHandler) createMonthlyTaxSheet(f *excelize.File, year int, breakdown []services.MonthlyTaxBreakdown) {
	sheet := "Impostos Mensais"
	f.NewSheet(sheet)

	// Headers
	headers := []string{"Mês", "Receita Bruta", "Imposto", "INSS", "Receita Líquida"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, header)
	}

	// Header style
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#70AD47"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetCellStyle(sheet, "A1", "E1", headerStyle)

	// Currency style
	currencyStyle, _ := f.NewStyle(&excelize.Style{
		NumFmt: 4,
	})

	// Data rows
	var totalGross, totalTax, totalINSS, totalNet float64

	for i, m := range breakdown {
		row := i + 2
		monthName := i18n.MonthNamesSlice[m.Month-1]

		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), monthName)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), m.GrossIncome)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), m.TaxPaid)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), m.INSSPaid)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), m.NetIncome)

		f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("E%d", row), currencyStyle)

		totalGross += m.GrossIncome
		totalTax += m.TaxPaid
		totalINSS += m.INSSPaid
		totalNet += m.NetIncome
	}

	// Totals row
	totalRow := len(breakdown) + 2
	totalStyle, _ := f.NewStyle(&excelize.Style{
		Font:   &excelize.Font{Bold: true},
		Fill:   excelize.Fill{Type: "pattern", Color: []string{"#E2EFDA"}, Pattern: 1},
		NumFmt: 4,
	})

	f.SetCellValue(sheet, fmt.Sprintf("A%d", totalRow), "TOTAL")
	f.SetCellValue(sheet, fmt.Sprintf("B%d", totalRow), totalGross)
	f.SetCellValue(sheet, fmt.Sprintf("C%d", totalRow), totalTax)
	f.SetCellValue(sheet, fmt.Sprintf("D%d", totalRow), totalINSS)
	f.SetCellValue(sheet, fmt.Sprintf("E%d", totalRow), totalNet)
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", totalRow), fmt.Sprintf("E%d", totalRow), totalStyle)

	// Adjust column widths
	f.SetColWidth(sheet, "A", "A", 15)
	f.SetColWidth(sheet, "B", "E", 18)
}

// createTaxBracketInfoSheet creates a sheet with Simples Nacional bracket information
func (h *TaxReportHandler) createTaxBracketInfoSheet(f *excelize.File, currentRevenue float64) {
	sheet := "Faixas Simples Nacional"
	f.NewSheet(sheet)

	// Headers
	headers := []string{"Faixa", "Receita Bruta Anual (até)", "Alíquota Nominal", "Dedução", "Status"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, header)
	}

	// Header style
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#5B9BD5"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetCellStyle(sheet, "A1", "E1", headerStyle)

	// Bracket data (Simples Nacional Anexo III)
	brackets := []struct {
		bracket   int
		limit     float64
		rate      float64
		deduction float64
	}{
		{1, 180000, 6.00, 0},
		{2, 360000, 11.20, 9360},
		{3, 720000, 13.50, 17640},
		{4, 1800000, 16.00, 35640},
		{5, 3600000, 21.00, 125640},
		{6, 4800000, 33.00, 648000},
	}

	// Currency style
	currencyStyle, _ := f.NewStyle(&excelize.Style{
		NumFmt: 4,
	})

	// Percentage style
	percentStyle, _ := f.NewStyle(&excelize.Style{
		NumFmt: 10,
	})

	// Current bracket style (highlight)
	currentStyle, _ := f.NewStyle(&excelize.Style{
		Fill:   excelize.Fill{Type: "pattern", Color: []string{"#FFF2CC"}, Pattern: 1},
		NumFmt: 4,
	})

	for i, b := range brackets {
		row := i + 2

		// Determine status
		status := ""
		isCurrentBracket := false
		if i == 0 && currentRevenue <= b.limit {
			status = "← Faixa Atual"
			isCurrentBracket = true
		} else if i > 0 && currentRevenue > brackets[i-1].limit && currentRevenue <= b.limit {
			status = "← Faixa Atual"
			isCurrentBracket = true
		} else if currentRevenue > b.limit {
			status = "Ultrapassada"
		} else {
			status = "Próximas"
		}

		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), b.bracket)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), b.limit)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), b.rate/100)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), b.deduction)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), status)

		f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), currencyStyle)
		f.SetCellStyle(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), percentStyle)
		f.SetCellStyle(sheet, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), currencyStyle)

		if isCurrentBracket {
			f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row), currentStyle)
		}
	}

	// Add current revenue info
	row := len(brackets) + 3
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Seu Faturamento 12 Meses:")
	f.SetCellValue(sheet, fmt.Sprintf("B%d", row), currentRevenue)
	f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), currencyStyle)

	// Adjust column widths
	f.SetColWidth(sheet, "A", "A", 10)
	f.SetColWidth(sheet, "B", "B", 25)
	f.SetColWidth(sheet, "C", "C", 18)
	f.SetColWidth(sheet, "D", "D", 15)
	f.SetColWidth(sheet, "E", "E", 15)
}

// exportTaxReportPDF exports tax report data to PDF format
func (h *TaxReportHandler) exportTaxReportPDF(c echo.Context, year int, accountIDs []uint, inssConfig *services.INSSConfig, settings services.SettingsData) error {
	// Get tax projection for the selected year
	now := time.Now()
	var projection services.TaxProjection
	if year == now.Year() {
		projection = services.GetTaxProjection(database.DB, accountIDs, inssConfig)
	} else {
		projection = services.GetTaxProjectionForYear(database.DB, year, accountIDs, inssConfig)
	}

	// Get monthly tax breakdown
	monthlyBreakdown := services.GetMonthlyTaxBreakdown(database.DB, year, accountIDs, inssConfig)

	// Get bracket info (with manual override support)
	revenue12M := services.GetRevenue12MonthsForAccounts(database.DB, accountIDs)
	bracket, effectiveRate, nextBracketAt := services.GetBracketInfoWithManualOverride(revenue12M, settings.ManualBracket)

	// Create PDF document
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	// Title
	pdf.SetFont("Arial", "B", 18)
	pdf.SetTextColor(44, 62, 80) // Dark blue
	pdf.CellFormat(0, 12, fmt.Sprintf("Relatorio Fiscal - %d", year), "", 1, "C", false, 0, "")
	pdf.Ln(5)

	// Generation date
	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(128, 128, 128) // Gray
	pdf.CellFormat(0, 6, fmt.Sprintf("Gerado em: %s", time.Now().Format("02/01/2006 15:04")), "", 1, "C", false, 0, "")
	pdf.Ln(10)

	// Year-to-Date Section
	h.addPDFSectionHeader(pdf, "Acumulado no Ano (YTD)")

	ytdData := [][]string{
		{"Receita Bruta", h.formatCurrency(projection.YTDIncome)},
		{"Imposto Pago", h.formatCurrency(projection.YTDTax)},
		{"INSS Pago", h.formatCurrency(projection.YTDINSS)},
		{"Receita Liquida", h.formatCurrency(projection.YTDNetIncome)},
	}
	h.addPDFTable(pdf, ytdData)
	pdf.Ln(8)

	// Annual Projections Section
	h.addPDFSectionHeader(pdf, "Projecao Anual")

	projectionData := [][]string{
		{"Receita Bruta Projetada", h.formatCurrency(projection.ProjectedAnnualIncome)},
		{"Imposto Projetado", h.formatCurrency(projection.ProjectedAnnualTax)},
		{"INSS Projetado", h.formatCurrency(projection.ProjectedAnnualINSS)},
		{"Receita Liquida Projetada", h.formatCurrency(projection.ProjectedNetIncome)},
	}
	h.addPDFTable(pdf, projectionData)
	pdf.Ln(8)

	// Bracket Info Section
	h.addPDFSectionHeader(pdf, "Informacoes da Faixa")

	amountToNext := nextBracketAt - revenue12M
	if amountToNext < 0 {
		amountToNext = 0
	}

	bracketData := [][]string{
		{"Faturamento 12 Meses", h.formatCurrency(revenue12M)},
		{"Faixa Atual", fmt.Sprintf("%d", bracket)},
		{"Aliquota Efetiva", fmt.Sprintf("%.2f%%", effectiveRate)},
		{"Proxima Faixa em", h.formatCurrency(nextBracketAt)},
		{"Valor ate Proxima Faixa", h.formatCurrency(amountToNext)},
	}
	h.addPDFTable(pdf, bracketData)
	pdf.Ln(8)

	// INSS Config Section
	h.addPDFSectionHeader(pdf, "Configuracao INSS")

	inssData := [][]string{
		{"Pro-Labore", h.formatCurrency(settings.ProLabore)},
		{"Aliquota INSS", fmt.Sprintf("%.2f%%", settings.INSSRate)},
		{"Teto INSS", h.formatCurrency(settings.INSSCeiling)},
		{"INSS Mensal", h.formatCurrency(settings.INSSAmount)},
	}
	h.addPDFTable(pdf, inssData)

	// Add new page for monthly breakdown
	pdf.AddPage()

	// Monthly Tax Breakdown Section
	h.addPDFSectionHeader(pdf, "Impostos Mensais")

	// Monthly table headers
	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(70, 130, 180) // Steel blue
	pdf.SetTextColor(255, 255, 255)
	colWidths := []float64{30, 40, 35, 35, 40}
	headers := []string{"Mes", "Receita Bruta", "Imposto", "INSS", "Receita Liquida"}

	for i, header := range headers {
		pdf.CellFormat(colWidths[i], 8, header, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// Monthly table data
	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(0, 0, 0)

	var totalGross, totalTax, totalINSS, totalNet float64
	fill := false

	for _, m := range monthlyBreakdown {
		monthName := i18n.MonthNamesSlice[m.Month-1]

		if fill {
			pdf.SetFillColor(240, 240, 240) // Light gray
		} else {
			pdf.SetFillColor(255, 255, 255) // White
		}

		pdf.CellFormat(colWidths[0], 7, monthName, "1", 0, "L", fill, 0, "")
		pdf.CellFormat(colWidths[1], 7, h.formatCurrency(m.GrossIncome), "1", 0, "R", fill, 0, "")
		pdf.CellFormat(colWidths[2], 7, h.formatCurrency(m.TaxPaid), "1", 0, "R", fill, 0, "")
		pdf.CellFormat(colWidths[3], 7, h.formatCurrency(m.INSSPaid), "1", 0, "R", fill, 0, "")
		pdf.CellFormat(colWidths[4], 7, h.formatCurrency(m.NetIncome), "1", 0, "R", fill, 0, "")
		pdf.Ln(-1)

		totalGross += m.GrossIncome
		totalTax += m.TaxPaid
		totalINSS += m.INSSPaid
		totalNet += m.NetIncome
		fill = !fill
	}

	// Totals row
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(226, 239, 218) // Light green
	pdf.CellFormat(colWidths[0], 8, "TOTAL", "1", 0, "L", true, 0, "")
	pdf.CellFormat(colWidths[1], 8, h.formatCurrency(totalGross), "1", 0, "R", true, 0, "")
	pdf.CellFormat(colWidths[2], 8, h.formatCurrency(totalTax), "1", 0, "R", true, 0, "")
	pdf.CellFormat(colWidths[3], 8, h.formatCurrency(totalINSS), "1", 0, "R", true, 0, "")
	pdf.CellFormat(colWidths[4], 8, h.formatCurrency(totalNet), "1", 0, "R", true, 0, "")
	pdf.Ln(12)

	// Simples Nacional Brackets Section
	h.addPDFSectionHeader(pdf, "Faixas Simples Nacional")

	// Bracket table headers
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(91, 155, 213) // Blue
	pdf.SetTextColor(255, 255, 255)
	bracketColWidths := []float64{20, 50, 35, 30, 45}
	bracketHeaders := []string{"Faixa", "Receita Anual (ate)", "Aliquota", "Deducao", "Status"}

	for i, header := range bracketHeaders {
		pdf.CellFormat(bracketColWidths[i], 8, header, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// Bracket data
	brackets := []struct {
		bracket   int
		limit     float64
		rate      float64
		deduction float64
	}{
		{1, 180000, 6.00, 0},
		{2, 360000, 11.20, 9360},
		{3, 720000, 13.50, 17640},
		{4, 1800000, 16.00, 35640},
		{5, 3600000, 21.00, 125640},
		{6, 4800000, 33.00, 648000},
	}

	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(0, 0, 0)

	for i, b := range brackets {
		// Determine status and styling
		status := ""
		isCurrentBracket := false
		if i == 0 && revenue12M <= b.limit {
			status = "Faixa Atual"
			isCurrentBracket = true
		} else if i > 0 && revenue12M > brackets[i-1].limit && revenue12M <= b.limit {
			status = "Faixa Atual"
			isCurrentBracket = true
		} else if revenue12M > b.limit {
			status = "Ultrapassada"
		} else {
			status = "Proxima"
		}

		if isCurrentBracket {
			pdf.SetFillColor(255, 242, 204) // Light yellow
		} else {
			pdf.SetFillColor(255, 255, 255) // White
		}

		pdf.CellFormat(bracketColWidths[0], 7, fmt.Sprintf("%d", b.bracket), "1", 0, "C", true, 0, "")
		pdf.CellFormat(bracketColWidths[1], 7, h.formatCurrency(b.limit), "1", 0, "R", true, 0, "")
		pdf.CellFormat(bracketColWidths[2], 7, fmt.Sprintf("%.2f%%", b.rate), "1", 0, "C", true, 0, "")
		pdf.CellFormat(bracketColWidths[3], 7, h.formatCurrency(b.deduction), "1", 0, "R", true, 0, "")
		pdf.CellFormat(bracketColWidths[4], 7, status, "1", 0, "C", true, 0, "")
		pdf.Ln(-1)
	}

	// Set headers for download
	c.Response().Header().Set("Content-Type", "application/pdf")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=relatorio_fiscal_%d.pdf", year))

	log.Printf("[TaxReport] Tax report PDF exported successfully for year %d", year)
	return pdf.Output(c.Response().Writer)
}

// addPDFSectionHeader adds a styled section header to the PDF
func (h *TaxReportHandler) addPDFSectionHeader(pdf *fpdf.Fpdf, title string) {
	pdf.SetFont("Arial", "B", 12)
	pdf.SetFillColor(47, 84, 150) // Dark blue
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(0, 8, title, "", 1, "L", true, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(2)
}

// addPDFTable adds a two-column table to the PDF
func (h *TaxReportHandler) addPDFTable(pdf *fpdf.Fpdf, data [][]string) {
	pdf.SetFont("Arial", "", 10)
	fill := false

	for _, row := range data {
		if fill {
			pdf.SetFillColor(240, 240, 240)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		pdf.CellFormat(80, 7, row[0], "1", 0, "L", fill, 0, "")
		pdf.CellFormat(60, 7, row[1], "1", 1, "R", fill, 0, "")
		fill = !fill
	}
}

// formatCurrency formats a float as Brazilian currency (R$)
func (h *TaxReportHandler) formatCurrency(value float64) string {
	return fmt.Sprintf("R$ %.2f", value)
}
