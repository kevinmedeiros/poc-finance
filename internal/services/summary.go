package services

import (
	"time"

	"gorm.io/gorm"

	"poc-finance/internal/models"
)

type MonthlySummary struct {
	Month            time.Time `json:"month"`
	MonthName        string    `json:"month_name"`
	TotalIncomeGross float64   `json:"total_income_gross"`
	TotalIncomeNet   float64   `json:"total_income_net"`
	TotalTax         float64   `json:"total_tax"`
	TotalFixed       float64   `json:"total_fixed"`
	TotalVariable    float64   `json:"total_variable"`
	TotalCards       float64   `json:"total_cards"`
	TotalBills       float64   `json:"total_bills"`
	TotalExpenses    float64   `json:"total_expenses"`
	Balance          float64   `json:"balance"`
}

var monthNames = map[time.Month]string{
	time.January:   "Janeiro",
	time.February:  "Fevereiro",
	time.March:     "Março",
	time.April:     "Abril",
	time.May:       "Maio",
	time.June:      "Junho",
	time.July:      "Julho",
	time.August:    "Agosto",
	time.September: "Setembro",
	time.October:   "Outubro",
	time.November:  "Novembro",
	time.December:  "Dezembro",
}

func GetMonthlySummary(db *gorm.DB, year int, month int) MonthlySummary {
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	endDate := startDate.AddDate(0, 1, 0).Add(-time.Second)

	summary := MonthlySummary{
		Month:     startDate,
		MonthName: monthNames[time.Month(month)] + " " + string(rune(year)),
	}

	// Total de recebimentos
	var incomes []models.Income
	db.Where("date BETWEEN ? AND ?", startDate, endDate).Find(&incomes)
	for _, i := range incomes {
		summary.TotalIncomeGross += i.GrossAmount
		summary.TotalIncomeNet += i.NetAmount
		summary.TotalTax += i.TaxAmount
	}

	// Total de despesas fixas (ativas)
	var fixedExpenses []models.Expense
	db.Where("type = ? AND active = ?", models.ExpenseTypeFixed, true).Find(&fixedExpenses)
	for _, e := range fixedExpenses {
		summary.TotalFixed += e.Amount
	}

	// Total de despesas variáveis do mês
	var variableExpenses []models.Expense
	db.Where("type = ? AND active = ? AND created_at BETWEEN ? AND ?",
		models.ExpenseTypeVariable, true, startDate, endDate).Find(&variableExpenses)
	for _, e := range variableExpenses {
		summary.TotalVariable += e.Amount
	}

	// Total de parcelas de cartão no mês
	var installments []models.Installment
	db.Preload("CreditCard").Find(&installments)
	for _, inst := range installments {
		// Calcula se a parcela entra neste mês
		installmentMonth := inst.StartDate
		for i := 1; i <= inst.TotalInstallments; i++ {
			if installmentMonth.Year() == year && int(installmentMonth.Month()) == month {
				summary.TotalCards += inst.InstallmentAmount
				break
			}
			installmentMonth = installmentMonth.AddDate(0, 1, 0)
		}
	}

	// Total de contas a pagar do mês
	var bills []models.Bill
	db.Where("due_date BETWEEN ? AND ?", startDate, endDate).Find(&bills)
	for _, b := range bills {
		summary.TotalBills += b.Amount
	}

	summary.TotalExpenses = summary.TotalFixed + summary.TotalVariable + summary.TotalCards + summary.TotalBills
	summary.Balance = summary.TotalIncomeNet - summary.TotalExpenses

	return summary
}

func GetYearlySummaries(db *gorm.DB, year int) []MonthlySummary {
	var summaries []MonthlySummary
	for month := 1; month <= 12; month++ {
		summaries = append(summaries, GetMonthlySummary(db, year, month))
	}
	return summaries
}

// GetRevenue12Months retorna o faturamento bruto dos últimos 12 meses
func GetRevenue12Months(db *gorm.DB) float64 {
	endDate := time.Now()
	startDate := endDate.AddDate(-1, 0, 0)

	var total float64
	db.Model(&models.Income{}).
		Where("date BETWEEN ? AND ?", startDate, endDate).
		Select("COALESCE(SUM(gross_amount), 0)").
		Scan(&total)

	return total
}
