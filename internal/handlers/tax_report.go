package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/database"
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
