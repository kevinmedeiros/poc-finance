package handlers

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
	"poc-finance/internal/services"
)

type DashboardHandler struct{}

func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{}
}

type UpcomingBill struct {
	Name    string
	Amount  float64
	DueDate time.Time
	DueIn   int // dias até o vencimento
	Type    string // "expense", "bill", "card"
}

func (h *DashboardHandler) Index(c echo.Context) error {
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	// Resumo do mês atual
	currentSummary := services.GetMonthlySummary(database.DB, year, month)

	// Projeção dos próximos 6 meses
	var monthSummaries []services.MonthlySummary
	for i := 0; i < 6; i++ {
		m := month + i
		y := year
		if m > 12 {
			m -= 12
			y++
		}
		monthSummaries = append(monthSummaries, services.GetMonthlySummary(database.DB, y, m))
	}

	// Faturamento 12 meses e faixa atual
	revenue12M := services.GetRevenue12Months(database.DB)
	bracket, rate, _ := services.GetBracketInfo(revenue12M)

	// Busca configurações de INSS
	settingsData := GetSettingsData()

	// Calcula totais
	totalImpostos := currentSummary.TotalTax + settingsData.INSSAmount
	liquidoAposImpostos := currentSummary.TotalIncomeGross - totalImpostos
	totalSaidas := totalImpostos + currentSummary.TotalExpenses
	saldoFinal := currentSummary.TotalIncomeGross - totalSaidas

	// Próximos vencimentos
	upcomingBills := getUpcomingBills(now)

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
	}

	return c.Render(http.StatusOK, "dashboard.html", data)
}

func getUpcomingBills(now time.Time) []UpcomingBill {
	var upcoming []UpcomingBill
	endDate := now.AddDate(0, 0, 30) // próximos 30 dias

	// Despesas fixas ativas
	var expenses []models.Expense
	database.DB.Where("type = ? AND active = ?", models.ExpenseTypeFixed, true).Find(&expenses)

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
	database.DB.Where("paid = ? AND due_date BETWEEN ? AND ?", false, now, endDate).Find(&bills)

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
