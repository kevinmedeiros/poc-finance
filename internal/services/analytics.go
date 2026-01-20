package services

import (
	"time"

	"gorm.io/gorm"

	"poc-finance/internal/models"
)

// MonthOverMonthComparison represents a comparison between current and previous month spending
type MonthOverMonthComparison struct {
	CurrentMonth         MonthlySummary `json:"current_month"`
	PreviousMonth        MonthlySummary `json:"previous_month"`
	IncomeChange         float64        `json:"income_change"`          // Percentage change in income
	IncomeChangePercent  float64        `json:"income_change_percent"`  // Percentage change
	ExpenseChange        float64        `json:"expense_change"`         // Percentage change in expenses
	ExpenseChangePercent float64        `json:"expense_change_percent"` // Percentage change
	BalanceChange        float64        `json:"balance_change"`         // Absolute change in balance
	BalanceChangePercent float64        `json:"balance_change_percent"` // Percentage change
}

// GetMonthOverMonthComparison retorna comparação entre o mês atual e o mês anterior
// com mudanças percentuais para receitas, despesas e saldo.
//
// Esta função usa GetBatchMonthlySummariesForAccounts para buscar ambos os meses
// de forma eficiente (2 queries batch ao invés de 10 queries separadas).
//
// Parâmetros:
//   - db: Conexão com banco de dados
//   - year: Ano do mês atual
//   - month: Mês atual (1-12)
//   - accountIDs: IDs das contas a incluir no cálculo (pode ser vazio para todas as contas)
//
// Retorna:
//   - MonthOverMonthComparison com dados do mês atual, mês anterior e mudanças percentuais
func GetMonthOverMonthComparison(db *gorm.DB, year int, month int, accountIDs []uint) MonthOverMonthComparison {
	// Calculate previous month
	prevMonth := month - 1
	prevYear := year
	if prevMonth == 0 {
		prevMonth = 12
		prevYear = year - 1
	}

	// Use batch function to fetch both months efficiently (2 queries instead of 10)
	summaries := GetBatchMonthlySummariesForAccounts(db, prevYear, prevMonth, year, month, accountIDs)

	// Initialize with empty summaries
	comparison := MonthOverMonthComparison{
		CurrentMonth:  MonthlySummary{},
		PreviousMonth: MonthlySummary{},
	}

	// Extract current and previous month from batch results
	for _, summary := range summaries {
		if summary.Month.Year() == year && int(summary.Month.Month()) == month {
			comparison.CurrentMonth = summary
		} else if summary.Month.Year() == prevYear && int(summary.Month.Month()) == prevMonth {
			comparison.PreviousMonth = summary
		}
	}

	// Calculate percentage changes
	// Income change
	if comparison.PreviousMonth.TotalIncomeGross > 0 {
		comparison.IncomeChange = comparison.CurrentMonth.TotalIncomeGross - comparison.PreviousMonth.TotalIncomeGross
		comparison.IncomeChangePercent = (comparison.IncomeChange / comparison.PreviousMonth.TotalIncomeGross) * 100
	} else if comparison.CurrentMonth.TotalIncomeGross > 0 {
		// Previous month was 0, current month has income = 100% increase
		comparison.IncomeChange = comparison.CurrentMonth.TotalIncomeGross
		comparison.IncomeChangePercent = 100
	}

	// Expense change
	if comparison.PreviousMonth.TotalExpenses > 0 {
		comparison.ExpenseChange = comparison.CurrentMonth.TotalExpenses - comparison.PreviousMonth.TotalExpenses
		comparison.ExpenseChangePercent = (comparison.ExpenseChange / comparison.PreviousMonth.TotalExpenses) * 100
	} else if comparison.CurrentMonth.TotalExpenses > 0 {
		// Previous month was 0, current month has expenses = 100% increase
		comparison.ExpenseChange = comparison.CurrentMonth.TotalExpenses
		comparison.ExpenseChangePercent = 100
	}

	// Balance change
	if comparison.PreviousMonth.Balance > 0 {
		comparison.BalanceChange = comparison.CurrentMonth.Balance - comparison.PreviousMonth.Balance
		comparison.BalanceChangePercent = (comparison.BalanceChange / comparison.PreviousMonth.Balance) * 100
	} else if comparison.PreviousMonth.Balance < 0 {
		// Previous balance was negative, calculate based on absolute value
		comparison.BalanceChange = comparison.CurrentMonth.Balance - comparison.PreviousMonth.Balance
		prevBalanceAbs := comparison.PreviousMonth.Balance
		if prevBalanceAbs < 0 {
			prevBalanceAbs = -prevBalanceAbs
		}
		if prevBalanceAbs > 0 {
			comparison.BalanceChangePercent = (comparison.BalanceChange / prevBalanceAbs) * 100
		}
	} else {
		// Previous balance was 0, just set the absolute change
		comparison.BalanceChange = comparison.CurrentMonth.Balance
		if comparison.CurrentMonth.Balance > 0 {
			comparison.BalanceChangePercent = 100
		}
	}

	return comparison
}

// CategoryBreakdownWithPercentages represents a category expense breakdown with percentage of total
type CategoryBreakdownWithPercentages struct {
	Category   string  `json:"category"`
	Amount     float64 `json:"amount"`
	Percentage float64 `json:"percentage"` // Percentage of total expenses
}

// GetCategoryBreakdownWithPercentages retorna o breakdown de despesas por categoria
// com percentuais do total para contas específicas.
//
// Esta função estende GetCategoryBreakdownForAccounts adicionando cálculo de percentual
// para cada categoria em relação ao total de despesas.
//
// Parâmetros:
//   - db: Conexão com banco de dados
//   - year: Ano do mês
//   - month: Mês (1-12)
//   - accountIDs: IDs das contas a incluir no cálculo (pode ser vazio para retornar nil)
//
// Retorna:
//   - Slice de CategoryBreakdownWithPercentages com categoria, valor e percentual do total
func GetCategoryBreakdownWithPercentages(db *gorm.DB, year int, month int, accountIDs []uint) []CategoryBreakdownWithPercentages {
	if len(accountIDs) == 0 {
		return nil
	}

	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	endDate := startDate.AddDate(0, 1, 0).Add(-time.Second)

	// Query expenses grouped by category
	type CategoryResult struct {
		Category string
		Total    float64
	}
	var results []CategoryResult
	db.Model(&models.Expense{}).
		Select("category, COALESCE(SUM(amount), 0) as total").
		Where("account_id IN ? AND created_at BETWEEN ? AND ? AND active = ?", accountIDs, startDate, endDate, true).
		Group("category").
		Scan(&results)

	// Calculate total expenses across all categories
	var totalExpenses float64
	for _, r := range results {
		if r.Category != "" {
			totalExpenses += r.Total
		}
	}

	// Build breakdown list with percentages
	breakdown := make([]CategoryBreakdownWithPercentages, 0, len(results))
	for _, r := range results {
		if r.Category != "" { // Only include expenses with a category
			percentage := 0.0
			if totalExpenses > 0 {
				percentage = (r.Total / totalExpenses) * 100
			}

			breakdown = append(breakdown, CategoryBreakdownWithPercentages{
				Category:   r.Category,
				Amount:     r.Total,
				Percentage: percentage,
			})
		}
	}

	return breakdown
}
