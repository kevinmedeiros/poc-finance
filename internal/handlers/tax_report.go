package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

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

	// Get revenue 12 months and bracket info
	revenue12M := services.GetRevenue12MonthsForAccounts(database.DB, accountIDs)
	bracket, effectiveRate, nextBracketAt := services.GetBracketInfo(revenue12M)

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

// ExportTaxReport exports tax report data to Excel format
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

	// Build INSS config from settings
	settingsData := h.cacheService.GetSettingsData()
	inssConfig := &services.INSSConfig{
		ProLabore: settingsData.ProLabore,
		Ceiling:   settingsData.INSSCeiling,
		Rate:      settingsData.INSSRate / 100,
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

	// Get bracket info
	revenue12M := services.GetRevenue12MonthsForAccounts(database.DB, accountIDs)
	bracket, effectiveRate, nextBracketAt := services.GetBracketInfo(revenue12M)

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
