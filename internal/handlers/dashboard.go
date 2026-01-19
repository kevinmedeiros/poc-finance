package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/database"
	"poc-finance/internal/middleware"
	"poc-finance/internal/models"
	"poc-finance/internal/services"
)

type DashboardHandler struct {
	accountService *services.AccountService
}

func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{
		accountService: services.NewAccountService(),
	}
}

type UpcomingBill struct {
	Name    string
	Amount  float64
	DueDate time.Time
	DueIn   int    // dias até o vencimento
	Type    string // "expense", "bill", "card"
}

func (h *DashboardHandler) Index(c echo.Context) error {
	log.Println("[Dashboard] Loading dashboard - query optimization enabled")

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

	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	// Resumo do mês atual
	currentSummary := services.GetMonthlySummaryForAccounts(database.DB, year, month, accountIDs)

	// Projeção dos próximos 6 meses - usando batch query para melhor performance
	// Calculate end month/year for 6-month range
	endMonth := month + 5
	endYear := year
	if endMonth > 12 {
		endMonth -= 12
		endYear++
	}
	// Fetch all 6 months in a single batch call (5 queries total instead of 6x5=30)
	log.Printf("[Dashboard] Fetching 6-month projections using batch query (5 queries instead of 30)")
	monthSummaries := services.GetBatchMonthlySummariesForAccounts(database.DB, year, month, endYear, endMonth, accountIDs)
	log.Printf("[Dashboard] Batch query completed - retrieved %d month summaries", len(monthSummaries))

	// Faturamento 12 meses e faixa atual
	revenue12M := services.GetRevenue12MonthsForAccounts(database.DB, accountIDs)
	bracket, rate, _ := services.GetBracketInfo(revenue12M)

	// Busca configurações de INSS
	settingsData := GetSettingsData()

	// Calcula totais
	totalImpostos := currentSummary.TotalTax + settingsData.INSSAmount
	liquidoAposImpostos := currentSummary.TotalIncomeGross - totalImpostos
	totalSaidas := totalImpostos + currentSummary.TotalExpenses
	saldoFinal := currentSummary.TotalIncomeGross - totalSaidas

	// Próximos vencimentos
	log.Println("[Dashboard] Fetching upcoming bills")
	upcomingBills := getUpcomingBillsForAccounts(now, accountIDs)

	log.Println("[Dashboard] Dashboard data loaded successfully - rendering template")
	data := map[string]interface{}{
		"currentMonth":        currentSummary,
		"monthSummaries":      monthSummaries,
		"revenue12m":          revenue12M,
		"currentBracket":      bracket,
		"effectiveRate":       rate,
		"upcomingBills":       upcomingBills,
		"now":                 now,
		"inssAmount":          settingsData.INSSAmount,
		"proLabore":           settingsData.ProLabore,
		"totalImpostos":       totalImpostos,
		"liquidoAposImpostos": liquidoAposImpostos,
		"totalSaidas":         totalSaidas,
		"saldoFinal":          saldoFinal,
		"accounts":            allAccounts,
		"selectedAccountID":   selectedAccountID,
	}

	return c.Render(http.StatusOK, "dashboard.html", data)
}

func getUpcomingBillsForAccounts(now time.Time, accountIDs []uint) []UpcomingBill {
	var upcoming []UpcomingBill

	if len(accountIDs) == 0 {
		return upcoming
	}

	endDate := now.AddDate(0, 0, 30) // próximos 30 dias

	// Despesas fixas ativas
	var expenses []models.Expense
	database.DB.Where("type = ? AND active = ? AND account_id IN ?", models.ExpenseTypeFixed, true, accountIDs).Find(&expenses)

	for _, e := range expenses {
		dueDate := time.Date(now.Year(), now.Month(), e.DueDay, 0, 0, 0, 0, time.Local)
		if dueDate.Before(now) {
			dueDate = dueDate.AddDate(0, 1, 0)
		}
		if dueDate.Before(endDate) {
			dueIn := int(dueDate.Sub(now).Hours() / 24)
			upcoming = append(upcoming, UpcomingBill{
				Name:    e.Name,
				Amount:  e.Amount,
				DueDate: dueDate,
				DueIn:   dueIn,
				Type:    "expense",
			})
		}
	}

	// Contas a pagar não pagas
	var bills []models.Bill
	database.DB.Where("paid = ? AND due_date BETWEEN ? AND ? AND account_id IN ?", false, now, endDate, accountIDs).Find(&bills)

	for _, b := range bills {
		dueIn := int(b.DueDate.Sub(now).Hours() / 24)
		upcoming = append(upcoming, UpcomingBill{
			Name:    b.Name,
			Amount:  b.Amount,
			DueDate: b.DueDate,
			DueIn:   dueIn,
			Type:    "bill",
		})
	}

	// Ordena por data de vencimento
	for i := 0; i < len(upcoming)-1; i++ {
		for j := i + 1; j < len(upcoming); j++ {
			if upcoming[j].DueDate.Before(upcoming[i].DueDate) {
				upcoming[i], upcoming[j] = upcoming[j], upcoming[i]
			}
		}
	}

	// Limita a 10 itens
	if len(upcoming) > 10 {
		upcoming = upcoming[:10]
	}

	return upcoming
}
