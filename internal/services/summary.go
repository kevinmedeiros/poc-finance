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

// GetRevenue12MonthsForAccounts retorna o faturamento bruto dos últimos 12 meses para contas específicas
func GetRevenue12MonthsForAccounts(db *gorm.DB, accountIDs []uint) float64 {
	if len(accountIDs) == 0 {
		return 0
	}

	endDate := time.Now()
	startDate := endDate.AddDate(-1, 0, 0)

	var total float64
	db.Model(&models.Income{}).
		Where("date BETWEEN ? AND ? AND account_id IN ?", startDate, endDate, accountIDs).
		Select("COALESCE(SUM(gross_amount), 0)").
		Scan(&total)

	return total
}

// GetMonthlySummaryForAccounts retorna o resumo mensal para contas específicas
// MemberContribution representa a contribuição de um membro do grupo
type MemberContribution struct {
	User           models.User
	TotalIncome    float64 // Receitas nas contas conjuntas
	TotalExpenses  float64 // Despesas atribuídas ao membro (splits)
	NetBalance     float64 // income - expenses
	ExpensePercent float64 // Porcentagem das despesas totais
}

// GetMemberContributions retorna as contribuições de cada membro do grupo
func GetMemberContributions(db *gorm.DB, groupID uint, accountIDs []uint) []MemberContribution {
	if len(accountIDs) == 0 {
		return nil
	}

	// Get all group members
	var members []models.GroupMember
	db.Preload("User").Where("group_id = ?", groupID).Find(&members)

	if len(members) == 0 {
		return nil
	}

	// Calculate expense totals from splits for each member
	type ExpenseResult struct {
		UserID uint
		Total  float64
	}
	var expenseResults []ExpenseResult
	db.Model(&models.ExpenseSplit{}).
		Select("expense_splits.user_id, COALESCE(SUM(expense_splits.amount), 0) as total").
		Joins("JOIN expenses ON expenses.id = expense_splits.expense_id").
		Where("expenses.account_id IN ? AND expenses.deleted_at IS NULL AND expense_splits.deleted_at IS NULL", accountIDs).
		Group("expense_splits.user_id").
		Scan(&expenseResults)

	expenseByUser := make(map[uint]float64)
	var totalGroupExpenses float64
	for _, r := range expenseResults {
		expenseByUser[r.UserID] = r.Total
		totalGroupExpenses += r.Total
	}

	// Also get non-split expenses for the accounts and distribute equally
	var nonSplitExpenses []models.Expense
	db.Where("account_id IN ? AND is_split = ?", accountIDs, false).Find(&nonSplitExpenses)
	var totalNonSplit float64
	for _, e := range nonSplitExpenses {
		totalNonSplit += e.Amount
	}
	// Distribute non-split expenses equally among members
	perMemberNonSplit := totalNonSplit / float64(len(members))
	totalGroupExpenses += totalNonSplit

	// Calculate income totals for each member (from accounts they own or group shared)
	// For joint accounts, income is shared equally among members
	var totalIncome float64
	db.Model(&models.Income{}).
		Select("COALESCE(SUM(gross_amount), 0)").
		Where("account_id IN ?", accountIDs).
		Scan(&totalIncome)
	perMemberIncome := totalIncome / float64(len(members))

	// Build contribution list
	contributions := make([]MemberContribution, 0, len(members))
	for _, m := range members {
		userExpenses := expenseByUser[m.UserID] + perMemberNonSplit
		contribution := MemberContribution{
			User:          m.User,
			TotalIncome:   perMemberIncome,
			TotalExpenses: userExpenses,
			NetBalance:    perMemberIncome - userExpenses,
		}
		if totalGroupExpenses > 0 {
			contribution.ExpensePercent = (userExpenses / totalGroupExpenses) * 100
		}
		contributions = append(contributions, contribution)
	}

	return contributions
}

// GetBatchMonthlySummariesForAccounts retorna resumos mensais para múltiplos meses em uma única operação batch
// Esta função otimiza o número de queries ao banco de dados ao buscar todos os dados de uma vez
// e distribuí-los nos meses apropriados, ao invés de fazer queries separadas por mês
func GetBatchMonthlySummariesForAccounts(db *gorm.DB, startYear, startMonth, endYear, endMonth int, accountIDs []uint) []MonthlySummary {
	// Create date range
	rangeStartDate := time.Date(startYear, time.Month(startMonth), 1, 0, 0, 0, 0, time.Local)
	rangeEndDate := time.Date(endYear, time.Month(endMonth), 1, 0, 0, 0, 0, time.Local)
	rangeEndDate = rangeEndDate.AddDate(0, 1, 0).Add(-time.Second)

	// Initialize map to hold summaries by month key (YYYY-MM)
	summaryMap := make(map[string]*MonthlySummary)

	// Pre-populate summaries for all months in range
	currentDate := rangeStartDate
	for currentDate.Before(rangeEndDate) || currentDate.Equal(rangeStartDate) {
		key := currentDate.Format("2006-01")
		summaryMap[key] = &MonthlySummary{
			Month:     time.Date(currentDate.Year(), currentDate.Month(), 1, 0, 0, 0, 0, time.Local),
			MonthName: monthNames[currentDate.Month()] + " " + string(rune(currentDate.Year())),
		}
		currentDate = currentDate.AddDate(0, 1, 0)

		// Break if we've passed the end date
		if currentDate.Year() > endYear || (currentDate.Year() == endYear && int(currentDate.Month()) > endMonth) {
			break
		}
	}

	if len(accountIDs) == 0 {
		// Return empty summaries
		result := make([]MonthlySummary, 0, len(summaryMap))
		for _, summary := range summaryMap {
			result = append(result, *summary)
		}
		return result
	}

	// Batch query 1: Fetch ALL incomes for date range
	var incomes []models.Income
	db.Where("date BETWEEN ? AND ? AND account_id IN ?", rangeStartDate, rangeEndDate, accountIDs).Find(&incomes)
	for _, i := range incomes {
		key := i.Date.Format("2006-01")
		if summary, exists := summaryMap[key]; exists {
			summary.TotalIncomeGross += i.GrossAmount
			summary.TotalIncomeNet += i.NetAmount
			summary.TotalTax += i.TaxAmount
		}
	}

	// Batch query 2: Fetch ALL fixed expenses for accounts
	var fixedExpenses []models.Expense
	db.Where("type = ? AND active = ? AND account_id IN ?", models.ExpenseTypeFixed, true, accountIDs).Find(&fixedExpenses)
	// Fixed expenses apply to all months
	for key := range summaryMap {
		for _, e := range fixedExpenses {
			summaryMap[key].TotalFixed += e.Amount
		}
	}

	// Batch query 3: Fetch ALL variable expenses for date range and accounts
	var variableExpenses []models.Expense
	db.Where("type = ? AND active = ? AND created_at BETWEEN ? AND ? AND account_id IN ?",
		models.ExpenseTypeVariable, true, rangeStartDate, rangeEndDate, accountIDs).Find(&variableExpenses)
	for _, e := range variableExpenses {
		key := e.CreatedAt.Format("2006-01")
		if summary, exists := summaryMap[key]; exists {
			summary.TotalVariable += e.Amount
		}
	}

	// Batch query 4: Fetch ALL credit cards with installments for accounts
	var creditCards []models.CreditCard
	db.Where("account_id IN ?", accountIDs).Preload("Installments").Find(&creditCards)
	for _, card := range creditCards {
		for _, inst := range card.Installments {
			// Calculate which months this installment affects
			installmentMonth := inst.StartDate
			for i := 1; i <= inst.TotalInstallments; i++ {
				key := installmentMonth.Format("2006-01")
				if summary, exists := summaryMap[key]; exists {
					summary.TotalCards += inst.InstallmentAmount
				}
				installmentMonth = installmentMonth.AddDate(0, 1, 0)
			}
		}
	}

	// Batch query 5: Fetch ALL bills for date range and accounts
	var bills []models.Bill
	db.Where("due_date BETWEEN ? AND ? AND account_id IN ?", rangeStartDate, rangeEndDate, accountIDs).Find(&bills)
	for _, b := range bills {
		key := b.DueDate.Format("2006-01")
		if summary, exists := summaryMap[key]; exists {
			summary.TotalBills += b.Amount
		}
	}

	// Calculate totals and build result slice
	result := make([]MonthlySummary, 0, len(summaryMap))
	for _, summary := range summaryMap {
		summary.TotalExpenses = summary.TotalFixed + summary.TotalVariable + summary.TotalCards + summary.TotalBills
		summary.Balance = summary.TotalIncomeNet - summary.TotalExpenses
		result = append(result, *summary)
	}

	// Sort results by month
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].Month.After(result[j].Month) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// GetMonthlySummaryForAccounts retorna o resumo mensal para contas específicas.
// Esta função executa 5 queries separadas (incomes, fixed expenses, variable expenses, credit cards, bills).
//
// Deprecated: Para buscar múltiplos meses, use GetBatchMonthlySummariesForAccounts que é 2-3x mais rápido
// e reduz significativamente o número de queries (5 queries totais vs 5 queries por mês).
// Esta função ainda é útil quando você precisa de apenas um único mês específico.
func GetMonthlySummaryForAccounts(db *gorm.DB, year int, month int, accountIDs []uint) MonthlySummary {
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	endDate := startDate.AddDate(0, 1, 0).Add(-time.Second)

	summary := MonthlySummary{
		Month:     startDate,
		MonthName: monthNames[time.Month(month)] + " " + string(rune(year)),
	}

	if len(accountIDs) == 0 {
		return summary
	}

	// Total de recebimentos
	var incomes []models.Income
	db.Where("date BETWEEN ? AND ? AND account_id IN ?", startDate, endDate, accountIDs).Find(&incomes)
	for _, i := range incomes {
		summary.TotalIncomeGross += i.GrossAmount
		summary.TotalIncomeNet += i.NetAmount
		summary.TotalTax += i.TaxAmount
	}

	// Total de despesas fixas (ativas)
	var fixedExpenses []models.Expense
	db.Where("type = ? AND active = ? AND account_id IN ?", models.ExpenseTypeFixed, true, accountIDs).Find(&fixedExpenses)
	for _, e := range fixedExpenses {
		summary.TotalFixed += e.Amount
	}

	// Total de despesas variáveis do mês
	var variableExpenses []models.Expense
	db.Where("type = ? AND active = ? AND created_at BETWEEN ? AND ? AND account_id IN ?",
		models.ExpenseTypeVariable, true, startDate, endDate, accountIDs).Find(&variableExpenses)
	for _, e := range variableExpenses {
		summary.TotalVariable += e.Amount
	}

	// Total de parcelas de cartão no mês
	var creditCards []models.CreditCard
	db.Where("account_id IN ?", accountIDs).Preload("Installments").Find(&creditCards)
	for _, card := range creditCards {
		for _, inst := range card.Installments {
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
	}

	// Total de contas a pagar do mês
	var bills []models.Bill
	db.Where("due_date BETWEEN ? AND ? AND account_id IN ?", startDate, endDate, accountIDs).Find(&bills)
	for _, b := range bills {
		summary.TotalBills += b.Amount
	}

	summary.TotalExpenses = summary.TotalFixed + summary.TotalVariable + summary.TotalCards + summary.TotalBills
	summary.Balance = summary.TotalIncomeNet - summary.TotalExpenses

	return summary
}
