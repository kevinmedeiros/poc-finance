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

type AnalyticsHandler struct {
	accountService *services.AccountService
}

func NewAnalyticsHandler() *AnalyticsHandler {
	return &AnalyticsHandler{
		accountService: services.NewAccountService(),
	}
}

// GetTrends returns analytics trends data for HTMX dynamic updates
func (h *AnalyticsHandler) GetTrends(c echo.Context) error {
	log.Println("[Analytics] Loading trends data")

	userID := middleware.GetUserID(c)
	allAccountIDs, _ := h.accountService.GetUserAccountIDs(userID)

	// Handle account filter from query parameter
	var accountIDs []uint
	accountIDParam := c.QueryParam("account_id")

	if accountIDParam != "" && accountIDParam != "all" {
		parsedID, err := strconv.ParseUint(accountIDParam, 10, 32)
		if err == nil {
			selectedAccountID := uint(parsedID)
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

	// Parse months parameter (default to 6)
	months := 6
	monthsParam := c.QueryParam("months")
	if monthsParam != "" {
		if parsed, err := strconv.Atoi(monthsParam); err == nil && parsed > 0 && parsed <= 24 {
			months = parsed
		}
	}

	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	// Fetch analytics data
	log.Printf("[Analytics] Fetching analytics for %d months", months)
	monthOverMonthComparison := services.GetMonthOverMonthComparison(database.DB, year, month, accountIDs)
	categoryBreakdownWithPercentages := services.GetCategoryBreakdownWithPercentages(database.DB, year, month, accountIDs)
	incomeVsExpenseTrend := services.GetIncomeVsExpenseTrend(database.DB, months, accountIDs)
	log.Printf("[Analytics] Analytics data loaded - comparison: %v categories, trend: %d months",
		len(categoryBreakdownWithPercentages), len(incomeVsExpenseTrend))

	response := map[string]interface{}{
		"monthOverMonthComparison":         monthOverMonthComparison,
		"categoryBreakdownWithPercentages": categoryBreakdownWithPercentages,
		"incomeVsExpenseTrend":             incomeVsExpenseTrend,
		"months":                           months,
	}

	log.Println("[Analytics] Trends data loaded successfully - returning JSON")
	return c.JSON(http.StatusOK, response)
}
