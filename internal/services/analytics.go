package services

import (
	"time"

	"gorm.io/gorm"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
)

// getRecordStartDateSetting retrieves the record start date from settings
func getRecordStartDateSetting() time.Time {
	var setting models.Settings
	if err := database.DB.Where("key = ?", models.SettingRecordStartDate).First(&setting).Error; err != nil {
		return time.Time{}
	}
	parsed, err := time.Parse("2006-01-02", setting.Value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

// MonthOverMonthComparison represents a comparison between current and previous month spending
type MonthOverMonthComparison struct {
	CurrentMonth         MonthlySummary `json:"current_month"`
	PreviousMonth        MonthlySummary `json:"previous_month"`
	HasPreviousMonth     bool           `json:"has_previous_month"`     // Whether previous month data exists
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
// GROUP ANALYTICS SUPPORT:
// Esta função funciona com contas individuais, múltiplas contas, e contas conjuntas (joint accounts).
// Para análises de grupos familiares, passe os IDs das contas conjuntas do grupo.
// Exemplo: accountIDs = GetGroupJointAccountIDs(groupID)
//
// Parâmetros:
//   - db: Conexão com banco de dados
//   - year: Ano do mês atual
//   - month: Mês atual (1-12)
//   - accountIDs: IDs das contas a incluir no cálculo (pode ser vazio para todas as contas,
//     pode incluir contas individuais e/ou conjuntas)
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

	// Check record start date from settings
	recordStartDate := getRecordStartDateSetting()
	prevMonthDate := time.Date(prevYear, time.Month(prevMonth), 1, 0, 0, 0, 0, time.Local)
	includePrevMonth := recordStartDate.IsZero() || !prevMonthDate.Before(recordStartDate)

	var summaries []MonthlySummary
	if includePrevMonth {
		// Use batch function to fetch both months efficiently (2 queries instead of 10)
		summaries = GetBatchMonthlySummariesForAccounts(db, prevYear, prevMonth, year, month, accountIDs)
	} else {
		// Only fetch current month since previous is before start date
		summaries = GetBatchMonthlySummariesForAccounts(db, year, month, year, month, accountIDs)
	}

	// Initialize with empty summaries
	comparison := MonthOverMonthComparison{
		CurrentMonth:     MonthlySummary{},
		PreviousMonth:    MonthlySummary{},
		HasPreviousMonth: includePrevMonth,
	}

	// Extract current and previous month from batch results
	for _, summary := range summaries {
		if summary.Month.Year() == year && int(summary.Month.Month()) == month {
			comparison.CurrentMonth = summary
		} else if includePrevMonth && summary.Month.Year() == prevYear && int(summary.Month.Month()) == prevMonth {
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
// GROUP ANALYTICS SUPPORT:
// Esta função funciona com contas individuais, múltiplas contas, e contas conjuntas (joint accounts).
// Para análises de grupos familiares, passe os IDs das contas conjuntas do grupo.
// Exemplo: accountIDs = GetGroupJointAccountIDs(groupID)
//
// Parâmetros:
//   - db: Conexão com banco de dados
//   - year: Ano do mês
//   - month: Mês (1-12)
//   - accountIDs: IDs das contas a incluir no cálculo (pode ser vazio para retornar nil,
//     pode incluir contas individuais e/ou conjuntas)
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

// IncomeVsExpenseTrendPoint represents a single data point in the income vs expense trend
type IncomeVsExpenseTrendPoint struct {
	Month        time.Time `json:"month"`
	MonthName    string    `json:"month_name"`
	TotalIncome  float64   `json:"total_income"`  // Total income (gross) for the month
	TotalExpense float64   `json:"total_expense"` // Total expenses for the month
	NetBalance   float64   `json:"net_balance"`   // income - expense
}

// GetIncomeVsExpenseTrend retorna dados de tendência de receitas vs despesas para múltiplos meses,
// otimizado para uso em gráficos Chart.js de linha.
//
// Esta função utiliza o padrão batch de GetBatchMonthlySummariesForAccounts para buscar
// dados de múltiplos meses de forma eficiente (5 queries batch ao invés de N*5 queries).
//
// GROUP ANALYTICS SUPPORT:
// Esta função funciona com contas individuais, múltiplas contas, e contas conjuntas (joint accounts).
// Para análises de grupos familiares, passe os IDs das contas conjuntas do grupo.
// Exemplo: accountIDs = GetGroupJointAccountIDs(groupID)
//
// Parâmetros:
//   - db: Conexão com banco de dados
//   - months: Número de meses anteriores a incluir (ex: 6 para últimos 6 meses)
//   - accountIDs: IDs das contas a incluir no cálculo (pode ser vazio para retornar dados zerados,
//     pode incluir contas individuais e/ou conjuntas)
//
// Retorna:
//   - Slice de IncomeVsExpenseTrendPoint ordenado cronologicamente (mais antigo para mais recente)
//
// Exemplo de uso:
//
//	trend := GetIncomeVsExpenseTrend(db, 6, []uint{1, 2}) // Últimos 6 meses para contas 1 e 2
//	// Retorna: [{Jan 2024, 5000, 3000, 2000}, {Feb 2024, 6000, 3500, 2500}, ...]
func GetIncomeVsExpenseTrend(db *gorm.DB, months int, accountIDs []uint) []IncomeVsExpenseTrendPoint {
	if months <= 0 {
		return nil
	}

	// Calculate date range: last N months from current month
	now := time.Now()
	endYear := now.Year()
	endMonth := int(now.Month())

	// Calculate start date (N months ago)
	startDate := time.Date(endYear, time.Month(endMonth), 1, 0, 0, 0, 0, time.Local).AddDate(0, -(months - 1), 0)
	startYear := startDate.Year()
	startMonth := int(startDate.Month())

	// Use batch function to fetch all months efficiently (5 queries instead of N*5)
	summaries := GetBatchMonthlySummariesForAccounts(db, startYear, startMonth, endYear, endMonth, accountIDs)

	// Convert MonthlySummary to IncomeVsExpenseTrendPoint
	// MonthlySummary has all the fields we need, we just extract the relevant ones
	trendPoints := make([]IncomeVsExpenseTrendPoint, 0, len(summaries))
	for _, summary := range summaries {
		trendPoints = append(trendPoints, IncomeVsExpenseTrendPoint{
			Month:        summary.Month,
			MonthName:    summary.MonthName,
			TotalIncome:  summary.TotalIncomeGross,
			TotalExpense: summary.TotalExpenses,
			NetBalance:   summary.Balance,
		})
	}

	// Sort by month (chronological order: oldest to newest)
	// This is important for Chart.js line charts which expect data in order
	for i := 0; i < len(trendPoints)-1; i++ {
		for j := i + 1; j < len(trendPoints); j++ {
			if trendPoints[i].Month.After(trendPoints[j].Month) {
				trendPoints[i], trendPoints[j] = trendPoints[j], trendPoints[i]
			}
		}
	}

	return trendPoints
}
