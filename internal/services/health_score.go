package services

import (
	"time"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
)

// Recommendation represents a personalized financial health recommendation
type Recommendation struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ActionUrl   string `json:"action_url"`
	ActionText  string `json:"action_text"`
	Priority    string `json:"priority"` // "high", "medium", "low"
	Color       string `json:"color"`    // CSS color class: "danger", "warning", "brand", "success"
}

// HealthScoreService handles financial health score calculations and recommendations
type HealthScoreService struct {
	accountService *AccountService
}

// NewHealthScoreService creates a new HealthScoreService instance
func NewHealthScoreService() *HealthScoreService {
	return &HealthScoreService{
		accountService: NewAccountService(),
	}
}

// getRecordStartDate retrieves the record start date from settings
func (s *HealthScoreService) getRecordStartDate() time.Time {
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

// CalculateUserScore calculates the financial health score for a user
// Scoring formula: 30% savings rate + 25% debt level + 25% goal progress + 20% budget adherence
func (s *HealthScoreService) CalculateUserScore(userID uint, accountIDs []uint) (*models.HealthScore, error) {
	// Get record start date from settings
	recordStartDate := s.getRecordStartDate()

	// Calculate component scores
	savingsScore := s.calculateSavingsScore(accountIDs, recordStartDate)
	debtScore := s.calculateDebtScore(accountIDs, recordStartDate)
	goalScore := s.calculateGoalScore(userID, nil)
	budgetScore := s.calculateBudgetScore(accountIDs, recordStartDate)

	// Calculate weighted overall score
	overallScore := (savingsScore * 0.30) + (debtScore * 0.25) + (goalScore * 0.25) + (budgetScore * 0.20)

	// Create and save health score
	healthScore := &models.HealthScore{
		UserID:       &userID,
		Score:        overallScore,
		SavingsScore: savingsScore,
		DebtScore:    debtScore,
		GoalScore:    goalScore,
		BudgetScore:  budgetScore,
		CalculatedAt: time.Now(),
	}

	if err := database.DB.Create(healthScore).Error; err != nil {
		return nil, err
	}

	return healthScore, nil
}

// CalculateGroupScore calculates the financial health score for a family group
func (s *HealthScoreService) CalculateGroupScore(groupID uint) (*models.HealthScore, error) {
	// Get all accounts for the group (individual + joint)
	accountIDs, err := s.accountService.GetAllGroupAccountIDs(groupID)
	if err != nil {
		return nil, err
	}

	// Get record start date from settings
	recordStartDate := s.getRecordStartDate()

	// Calculate component scores
	savingsScore := s.calculateSavingsScore(accountIDs, recordStartDate)
	debtScore := s.calculateDebtScore(accountIDs, recordStartDate)
	goalScore := s.calculateGoalScore(0, &groupID)
	budgetScore := s.calculateBudgetScore(accountIDs, recordStartDate)

	// Calculate weighted overall score
	overallScore := (savingsScore * 0.30) + (debtScore * 0.25) + (goalScore * 0.25) + (budgetScore * 0.20)

	// Create and save health score
	healthScore := &models.HealthScore{
		GroupID:      &groupID,
		Score:        overallScore,
		SavingsScore: savingsScore,
		DebtScore:    debtScore,
		GoalScore:    goalScore,
		BudgetScore:  budgetScore,
		CalculatedAt: time.Now(),
	}

	if err := database.DB.Create(healthScore).Error; err != nil {
		return nil, err
	}

	return healthScore, nil
}

// GetScoreHistory retrieves historical health scores
func (s *HealthScoreService) GetScoreHistory(userID *uint, groupID *uint, months int) ([]models.HealthScore, error) {
	var scores []models.HealthScore

	query := database.DB.Order("calculated_at DESC").Limit(months)

	// Filter by record start date
	recordStartDate := s.getRecordStartDate()
	if !recordStartDate.IsZero() {
		query = query.Where("calculated_at >= ?", recordStartDate)
	}

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	} else if groupID != nil {
		query = query.Where("group_id = ?", *groupID)
	}

	if err := query.Find(&scores).Error; err != nil {
		return nil, err
	}

	return scores, nil
}

// GetRecommendations generates personalized recommendations based on health score
func (s *HealthScoreService) GetRecommendations(score *models.HealthScore) ([]Recommendation, error) {
	recommendations := []Recommendation{}

	// Identify weakest areas (score < 60)
	type component struct {
		name  string
		score float64
	}

	components := []component{
		{"savings", score.SavingsScore},
		{"debt", score.DebtScore},
		{"goals", score.GoalScore},
		{"budget", score.BudgetScore},
	}

	// Sort components by score to prioritize weakest areas
	weakAreas := []component{}
	for _, c := range components {
		if c.score < 60 {
			weakAreas = append(weakAreas, c)
		}
	}

	// Generate recommendations for weak areas
	if len(weakAreas) > 0 {
		for i, area := range weakAreas {
			if i >= 3 { // Max 3 recommendations for weak areas
				break
			}
			rec := s.getRecommendationForArea(area.name, area.score)
			recommendations = append(recommendations, rec)
		}
	}

	// If score is good, provide encouragement and advanced tips
	if score.Score >= 75 && len(recommendations) == 0 {
		recommendations = append(recommendations, Recommendation{
			Title:       "Excelente Saúde Financeira!",
			Description: "Você está no caminho certo. Continue monitorando suas finanças regularmente.",
			ActionUrl:   "/dashboard",
			ActionText:  "Ver Dashboard",
			Priority:    "low",
			Color:       "success",
		})
	} else if score.Score >= 75 {
		// Add one positive recommendation
		recommendations = append(recommendations, Recommendation{
			Title:       "Continue Assim!",
			Description: "Sua saúde financeira está boa. Pequenos ajustes podem torná-la excelente.",
			ActionUrl:   "/dashboard",
			ActionText:  "Ver Dashboard",
			Priority:    "medium",
			Color:       "brand",
		})
	}

	// Ensure at least one recommendation
	if len(recommendations) == 0 {
		recommendations = append(recommendations, Recommendation{
			Title:       "Comece Pequeno",
			Description: "Estabeleça uma meta financeira simples e trabalhe para alcançá-la este mês.",
			ActionUrl:   "/goals",
			ActionText:  "Ver Metas",
			Priority:    "high",
			Color:       "danger",
		})
	}

	// Limit to 4 recommendations maximum
	if len(recommendations) > 4 {
		recommendations = recommendations[:4]
	}

	return recommendations, nil
}

// calculateSavingsScore calculates savings rate score (0-100)
// Based on (income - expenses) / income ratio
func (s *HealthScoreService) calculateSavingsScore(accountIDs []uint, recordStartDate time.Time) float64 {
	if len(accountIDs) == 0 {
		return 0
	}

	// Get last 3 months of data for more stable calculation
	endDate := time.Now()
	startDate := endDate.AddDate(0, -3, 0)

	// Respect record start date - don't go before it
	if !recordStartDate.IsZero() && startDate.Before(recordStartDate) {
		startDate = recordStartDate
	}

	// If start date is after end date, there's no valid data period yet
	if startDate.After(endDate) {
		return 50 // Neutral score - no data available yet
	}

	// Calculate total income (net)
	var totalIncome float64
	database.DB.Model(&models.Income{}).
		Where("account_id IN ? AND date BETWEEN ? AND ?", accountIDs, startDate, endDate).
		Select("COALESCE(SUM(net_amount), 0)").
		Scan(&totalIncome)

	if totalIncome == 0 {
		return 0
	}

	// Calculate total expenses
	var totalFixed float64
	database.DB.Model(&models.Expense{}).
		Where("account_id IN ? AND type = ? AND active = ?", accountIDs, models.ExpenseTypeFixed, true).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalFixed)

	// Fixed expenses count for all 3 months
	totalFixed = totalFixed * 3

	var totalVariable float64
	database.DB.Model(&models.Expense{}).
		Where("account_id IN ? AND type = ? AND active = ? AND created_at BETWEEN ? AND ?",
			accountIDs, models.ExpenseTypeVariable, true, startDate, endDate).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalVariable)

	var totalBills float64
	database.DB.Model(&models.Bill{}).
		Where("account_id IN ? AND due_date BETWEEN ? AND ?", accountIDs, startDate, endDate).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalBills)

	totalExpenses := totalFixed + totalVariable + totalBills

	// Calculate savings rate
	savingsRate := (totalIncome - totalExpenses) / totalIncome

	// Convert to 0-100 score
	// > 30% savings = 100 score
	// 20-30% = 80-100
	// 10-20% = 60-80
	// 0-10% = 40-60
	// negative = 0-40
	var score float64
	if savingsRate >= 0.30 {
		score = 100
	} else if savingsRate >= 0.20 {
		score = 80 + ((savingsRate - 0.20) / 0.10 * 20)
	} else if savingsRate >= 0.10 {
		score = 60 + ((savingsRate - 0.10) / 0.10 * 20)
	} else if savingsRate >= 0 {
		score = 40 + (savingsRate / 0.10 * 20)
	} else {
		// Negative savings (spending more than earning)
		score = 40 + (savingsRate * 40) // Will give 0-40 range
		if score < 0 {
			score = 0
		}
	}

	return score
}

// calculateDebtScore calculates debt management score (0-100)
// Currently based on expense ratio as inverse (lower expenses = better score)
func (s *HealthScoreService) calculateDebtScore(accountIDs []uint, recordStartDate time.Time) float64 {
	if len(accountIDs) == 0 {
		return 100 // No accounts = no debt
	}

	// Get last month of data
	endDate := time.Now()
	startDate := endDate.AddDate(0, -1, 0)

	// Respect record start date - don't go before it
	if !recordStartDate.IsZero() && startDate.Before(recordStartDate) {
		startDate = recordStartDate
	}

	// If start date is after end date, there's no valid data period yet
	if startDate.After(endDate) {
		return 50 // Neutral score - no data available yet
	}

	var totalIncome float64
	database.DB.Model(&models.Income{}).
		Where("account_id IN ? AND date BETWEEN ? AND ?", accountIDs, startDate, endDate).
		Select("COALESCE(SUM(net_amount), 0)").
		Scan(&totalIncome)

	// Calculate total fixed obligations (rent, bills, etc.)
	var totalFixed float64
	database.DB.Model(&models.Expense{}).
		Where("account_id IN ? AND type = ? AND active = ?", accountIDs, models.ExpenseTypeFixed, true).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalFixed)

	var totalBills float64
	database.DB.Model(&models.Bill{}).
		Where("account_id IN ? AND due_date BETWEEN ? AND ?", accountIDs, startDate, endDate).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalBills)

	totalObligations := totalFixed + totalBills

	// If no income but has obligations, that's a critical situation
	if totalIncome == 0 {
		if totalObligations > 0 {
			return 0 // Critical: expenses but no income
		}
		return 100 // No income and no obligations = neutral
	}

	obligationRatio := totalObligations / totalIncome

	// Convert to score (lower ratio = better score)
	// < 30% = 100 score (excellent)
	// 30-50% = 80-100 (good)
	// 50-70% = 60-80 (fair)
	// 70-90% = 40-60 (poor)
	// > 90% = 0-40 (critical)
	var score float64
	if obligationRatio <= 0.30 {
		score = 100
	} else if obligationRatio <= 0.50 {
		score = 80 + ((0.50 - obligationRatio) / 0.20 * 20)
	} else if obligationRatio <= 0.70 {
		score = 60 + ((0.70 - obligationRatio) / 0.20 * 20)
	} else if obligationRatio <= 0.90 {
		score = 40 + ((0.90 - obligationRatio) / 0.20 * 20)
	} else {
		score = 40 * (1 - (obligationRatio - 0.90))
		if score < 0 {
			score = 0
		}
	}

	return score
}

// calculateGoalScore calculates goal progress score (0-100)
func (s *HealthScoreService) calculateGoalScore(userID uint, groupID *uint) float64 {
	var goals []models.GroupGoal

	if groupID != nil {
		database.DB.Where("group_id = ? AND status = ?", *groupID, models.GoalStatusActive).Find(&goals)
	} else if userID > 0 {
		// Get user's groups
		var groupIDs []uint
		database.DB.Model(&models.GroupMember{}).Where("user_id = ?", userID).Pluck("group_id", &groupIDs)

		if len(groupIDs) > 0 {
			database.DB.Where("group_id IN ? AND status = ?", groupIDs, models.GoalStatusActive).Find(&goals)
		}
	}

	if len(goals) == 0 {
		return 50 // No goals = neutral score (not good, not bad)
	}

	// Calculate average progress across all goals
	totalProgress := 0.0
	for _, goal := range goals {
		progress := goal.ProgressPercentage()
		totalProgress += progress
	}

	avgProgress := totalProgress / float64(len(goals))

	// Convert average progress to score
	// >= 80% average progress = 100 score
	// 60-80% = 80-100
	// 40-60% = 60-80
	// 20-40% = 40-60
	// < 20% = 20-40
	var score float64
	if avgProgress >= 80 {
		score = 100
	} else if avgProgress >= 60 {
		score = 80 + ((avgProgress - 60) / 20 * 20)
	} else if avgProgress >= 40 {
		score = 60 + ((avgProgress - 40) / 20 * 20)
	} else if avgProgress >= 20 {
		score = 40 + ((avgProgress - 20) / 20 * 20)
	} else {
		score = 20 + (avgProgress / 20 * 20)
	}

	return score
}

// calculateBudgetScore calculates budget adherence score (0-100)
// Based on consistency of expenses month-over-month
func (s *HealthScoreService) calculateBudgetScore(accountIDs []uint, recordStartDate time.Time) float64 {
	if len(accountIDs) == 0 {
		return 100
	}

	// Get last 3 months of expense data
	endDate := time.Now()

	month1Start := endDate.AddDate(0, -1, 0)
	month1End := endDate

	month2Start := endDate.AddDate(0, -2, 0)
	month2End := month1Start

	month3Start := endDate.AddDate(0, -3, 0)
	month3End := month2Start

	// Respect record start date for each month range
	if !recordStartDate.IsZero() {
		if month1Start.Before(recordStartDate) {
			month1Start = recordStartDate
		}
		if month2Start.Before(recordStartDate) {
			month2Start = recordStartDate
		}
		if month3Start.Before(recordStartDate) {
			month3Start = recordStartDate
		}
	}

	// Calculate expenses for each month (only if the range is valid)
	var month1Expenses, month2Expenses, month3Expenses float64
	var validMonths int

	if month1End.After(month1Start) {
		month1Expenses = s.calculateMonthExpenses(accountIDs, month1Start, month1End)
		validMonths++
	}
	if month2End.After(month2Start) {
		month2Expenses = s.calculateMonthExpenses(accountIDs, month2Start, month2End)
		validMonths++
	}
	if month3End.After(month3Start) {
		month3Expenses = s.calculateMonthExpenses(accountIDs, month3Start, month3End)
		validMonths++
	}

	// If less than 2 valid months, return high score (not enough data to measure consistency)
	if validMonths < 2 {
		return 85
	}

	// If no expenses, return high score (staying within budget of 0!)
	if month1Expenses == 0 && month2Expenses == 0 && month3Expenses == 0 {
		return 90
	}

	// Calculate average and variance using only valid months
	totalExpenses := month1Expenses + month2Expenses + month3Expenses
	avgExpenses := totalExpenses / float64(validMonths)

	if avgExpenses == 0 {
		return 90
	}

	// Calculate coefficient of variation (std dev / mean) for valid months only
	var variance float64
	if month1End.After(month1Start) {
		variance += (month1Expenses - avgExpenses) * (month1Expenses - avgExpenses)
	}
	if month2End.After(month2Start) {
		variance += (month2Expenses - avgExpenses) * (month2Expenses - avgExpenses)
	}
	if month3End.After(month3Start) {
		variance += (month3Expenses - avgExpenses) * (month3Expenses - avgExpenses)
	}
	variance = variance / float64(validMonths)
	stdDev := 0.0
	if variance > 0 {
		// Simple square root approximation
		stdDev = variance / 2 // Simplified calculation
		for i := 0; i < 10; i++ {
			stdDev = (stdDev + variance/stdDev) / 2
		}
	}

	cv := stdDev / avgExpenses

	// Convert to score (lower variance = better score)
	// CV < 10% = 100 (very consistent)
	// CV 10-20% = 80-100
	// CV 20-30% = 60-80
	// CV 30-50% = 40-60
	// CV > 50% = 0-40
	var score float64
	if cv <= 0.10 {
		score = 100
	} else if cv <= 0.20 {
		score = 80 + ((0.20 - cv) / 0.10 * 20)
	} else if cv <= 0.30 {
		score = 60 + ((0.30 - cv) / 0.10 * 20)
	} else if cv <= 0.50 {
		score = 40 + ((0.50 - cv) / 0.20 * 20)
	} else {
		score = 40 * (1 - (cv - 0.50))
		if score < 0 {
			score = 0
		}
	}

	return score
}

// calculateMonthExpenses calculates total expenses for a given month
func (s *HealthScoreService) calculateMonthExpenses(accountIDs []uint, startDate, endDate time.Time) float64 {
	var totalFixed float64
	database.DB.Model(&models.Expense{}).
		Where("account_id IN ? AND type = ? AND active = ?", accountIDs, models.ExpenseTypeFixed, true).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalFixed)

	var totalVariable float64
	database.DB.Model(&models.Expense{}).
		Where("account_id IN ? AND type = ? AND active = ? AND created_at BETWEEN ? AND ?",
			accountIDs, models.ExpenseTypeVariable, true, startDate, endDate).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalVariable)

	var totalBills float64
	database.DB.Model(&models.Bill{}).
		Where("account_id IN ? AND due_date BETWEEN ? AND ?", accountIDs, startDate, endDate).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalBills)

	return totalFixed + totalVariable + totalBills
}

// getRecommendationForArea returns a recommendation for a specific weak area
func (s *HealthScoreService) getRecommendationForArea(area string, score float64) Recommendation {
	switch area {
	case "savings":
		if score < 30 {
			return Recommendation{
				Title:       "Aumente Sua Taxa de Poupança",
				Description: "Sua taxa de poupança está baixa. Tente reduzir despesas variáveis em 10% este mês.",
				ActionUrl:   "/expenses",
				ActionText:  "Ver Despesas",
				Priority:    "high",
				Color:       "danger",
			}
		}
		return Recommendation{
			Title:       "Melhore Suas Economias",
			Description: "Considere criar uma conta poupança automática de 5-10% do seu salário.",
			ActionUrl:   "/dashboard",
			ActionText:  "Ver Dashboard",
			Priority:    "medium",
			Color:       "warning",
		}
	case "debt":
		if score < 30 {
			return Recommendation{
				Title:       "Reduza Suas Obrigações Fixas",
				Description: "Suas despesas fixas estão muito altas. Revise contratos e busque alternativas mais econômicas.",
				ActionUrl:   "/expenses",
				ActionText:  "Ver Despesas",
				Priority:    "high",
				Color:       "danger",
			}
		}
		return Recommendation{
			Title:       "Gerencie Suas Contas",
			Description: "Configure lembretes para pagamento de contas e evite juros de atraso.",
			ActionUrl:   "/bills",
			ActionText:  "Ver Contas",
			Priority:    "medium",
			Color:       "warning",
		}
	case "goals":
		if score < 30 {
			return Recommendation{
				Title:       "Defina Metas Financeiras",
				Description: "Você não tem metas ativas. Criar objetivos financeiros ajuda a manter o foco.",
				ActionUrl:   "/goals",
				ActionText:  "Ver Metas",
				Priority:    "high",
				Color:       "danger",
			}
		}
		return Recommendation{
			Title:       "Aumente Contribuições para Metas",
			Description: "Suas metas estão progredindo lentamente. Considere aumentar as contribuições mensais.",
			ActionUrl:   "/goals",
			ActionText:  "Ver Metas",
			Priority:    "medium",
			Color:       "warning",
		}
	case "budget":
		if score < 30 {
			return Recommendation{
				Title:       "Controle Seus Gastos Variáveis",
				Description: "Suas despesas variam muito mês a mês. Estabeleça um orçamento mensal e monitore semanalmente.",
				ActionUrl:   "/expenses",
				ActionText:  "Ver Despesas",
				Priority:    "high",
				Color:       "danger",
			}
		}
		return Recommendation{
			Title:       "Mantenha a Consistência",
			Description: "Revise seus gastos regularmente para manter a consistência no orçamento.",
			ActionUrl:   "/dashboard",
			ActionText:  "Ver Dashboard",
			Priority:    "medium",
			Color:       "warning",
		}
	default:
		return Recommendation{
			Title:       "Continue Acompanhando",
			Description: "Monitore suas finanças regularmente para manter a saúde financeira.",
			ActionUrl:   "/dashboard",
			ActionText:  "Ver Dashboard",
			Priority:    "low",
			Color:       "brand",
		}
	}
}

// HealthMetrics contains raw financial metrics (percentages)
type HealthMetrics struct {
	SavingsRate     float64 // Percentage of income saved
	DebtRatio       float64 // Debt/obligations as percentage of income
	GoalProgress    float64 // Average progress across goals
	ActiveGoals     int     // Count of active goals
	BudgetAdherence float64 // Budget adherence percentage (100 = perfect)
}

// GetHealthMetrics calculates raw financial metrics for display
func (s *HealthScoreService) GetHealthMetrics(userID uint, accountIDs []uint) HealthMetrics {
	recordStartDate := s.getRecordStartDate()

	metrics := HealthMetrics{
		SavingsRate:     s.getSavingsRate(accountIDs, recordStartDate),
		DebtRatio:       s.getDebtRatio(accountIDs, recordStartDate),
		BudgetAdherence: s.getBudgetAdherence(accountIDs, recordStartDate),
	}

	// Get goal metrics
	metrics.GoalProgress, metrics.ActiveGoals = s.getGoalMetrics(userID)

	return metrics
}

// getSavingsRate returns the actual savings rate as a percentage
func (s *HealthScoreService) getSavingsRate(accountIDs []uint, recordStartDate time.Time) float64 {
	if len(accountIDs) == 0 {
		return 0
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, -3, 0)

	// Respect record start date
	if !recordStartDate.IsZero() && startDate.Before(recordStartDate) {
		startDate = recordStartDate
	}

	// If start date is after end date, there's no valid data period yet
	if startDate.After(endDate) {
		return 0
	}

	var totalIncome float64
	database.DB.Model(&models.Income{}).
		Where("account_id IN ? AND date BETWEEN ? AND ?", accountIDs, startDate, endDate).
		Select("COALESCE(SUM(net_amount), 0)").
		Scan(&totalIncome)

	if totalIncome == 0 {
		return 0
	}

	totalExpenses := s.calculateMonthExpenses(accountIDs, startDate, endDate)
	savingsRate := ((totalIncome - totalExpenses) / totalIncome) * 100

	if savingsRate < 0 {
		savingsRate = 0
	}

	return savingsRate
}

// getDebtRatio returns the debt/obligation ratio as a percentage
func (s *HealthScoreService) getDebtRatio(accountIDs []uint, recordStartDate time.Time) float64 {
	if len(accountIDs) == 0 {
		return 0
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, -1, 0)

	// Respect record start date
	if !recordStartDate.IsZero() && startDate.Before(recordStartDate) {
		startDate = recordStartDate
	}

	// If start date is after end date, there's no valid data period yet
	if startDate.After(endDate) {
		return 0
	}

	var totalIncome float64
	database.DB.Model(&models.Income{}).
		Where("account_id IN ? AND date BETWEEN ? AND ?", accountIDs, startDate, endDate).
		Select("COALESCE(SUM(net_amount), 0)").
		Scan(&totalIncome)

	if totalIncome == 0 {
		return 0
	}

	var totalFixed float64
	database.DB.Model(&models.Expense{}).
		Where("account_id IN ? AND type = ? AND active = ?", accountIDs, models.ExpenseTypeFixed, true).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalFixed)

	var totalBills float64
	database.DB.Model(&models.Bill{}).
		Where("account_id IN ? AND due_date BETWEEN ? AND ?", accountIDs, startDate, endDate).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalBills)

	totalObligations := totalFixed + totalBills
	debtRatio := (totalObligations / totalIncome) * 100

	return debtRatio
}

// getBudgetAdherence returns budget adherence as percentage (100 = consistent spending)
func (s *HealthScoreService) getBudgetAdherence(accountIDs []uint, recordStartDate time.Time) float64 {
	if len(accountIDs) == 0 {
		return 100
	}

	endDate := time.Now()
	month1Start := endDate.AddDate(0, -1, 0)
	month2Start := endDate.AddDate(0, -2, 0)

	// Respect record start date
	if !recordStartDate.IsZero() {
		if month1Start.Before(recordStartDate) {
			month1Start = recordStartDate
		}
		if month2Start.Before(recordStartDate) {
			return 100 // Not enough months for comparison
		}
	}

	// If start date is after end date, there's no valid data period yet
	if month1Start.After(endDate) {
		return 100
	}

	month1Expenses := s.calculateMonthExpenses(accountIDs, month1Start, endDate)
	month2Expenses := s.calculateMonthExpenses(accountIDs, month2Start, month1Start)

	if month2Expenses == 0 {
		return 100
	}

	// Calculate how close current spending is to previous (100% = same, less = overspending)
	ratio := month1Expenses / month2Expenses
	if ratio > 1 {
		// Overspending - reduce from 100
		adherence := 100 - ((ratio - 1) * 100)
		if adherence < 0 {
			adherence = 0
		}
		return adherence
	}

	return 100 // At or under budget
}

// getGoalMetrics returns goal progress and active goals count
func (s *HealthScoreService) getGoalMetrics(userID uint) (progress float64, count int) {
	var goals []models.GroupGoal

	// Get user's groups
	var groupIDs []uint
	database.DB.Model(&models.GroupMember{}).Where("user_id = ?", userID).Pluck("group_id", &groupIDs)

	if len(groupIDs) > 0 {
		database.DB.Where("group_id IN ? AND status = ?", groupIDs, models.GoalStatusActive).Find(&goals)
	}

	if len(goals) == 0 {
		return 0, 0
	}

	totalProgress := 0.0
	for _, goal := range goals {
		totalProgress += goal.ProgressPercentage()
	}

	return totalProgress / float64(len(goals)), len(goals)
}
