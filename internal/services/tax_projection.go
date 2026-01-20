package services

import (
	"time"

	"gorm.io/gorm"

	"poc-finance/internal/models"
)

// TaxProjection contains year-to-date income tracking and annual tax projections
type TaxProjection struct {
	// Year-to-date figures
	YTDIncome     float64 `json:"ytd_income"`      // Receita bruta acumulada no ano
	YTDTax        float64 `json:"ytd_tax"`         // Imposto pago acumulado no ano
	YTDINSS       float64 `json:"ytd_inss"`        // INSS pago acumulado no ano
	YTDNetIncome  float64 `json:"ytd_net_income"`  // Receita líquida acumulada no ano

	// Projections for full year
	ProjectedAnnualIncome float64 `json:"projected_annual_income"` // Receita bruta projetada para o ano
	ProjectedAnnualTax    float64 `json:"projected_annual_tax"`    // Imposto projetado para o ano
	ProjectedAnnualINSS   float64 `json:"projected_annual_inss"`   // INSS projetado para o ano
	ProjectedNetIncome    float64 `json:"projected_net_income"`    // Receita líquida projetada para o ano

	// Bracket information
	CurrentBracket    int     `json:"current_bracket"`     // Faixa atual do Simples Nacional
	EffectiveRate     float64 `json:"effective_rate"`      // Alíquota efetiva atual (percentual)
	NextBracketAt     float64 `json:"next_bracket_at"`     // Faturamento para próxima faixa

	// Metadata
	MonthsElapsed int       `json:"months_elapsed"` // Meses decorridos no ano
	Year          int       `json:"year"`           // Ano da projeção
	CalculatedAt  time.Time `json:"calculated_at"`  // Data/hora do cálculo
}

// GetYTDIncome retorna a receita bruta acumulada no ano para contas específicas
func GetYTDIncome(db *gorm.DB, accountIDs []uint) float64 {
	if len(accountIDs) == 0 {
		return 0
	}

	now := time.Now()
	startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.Local)

	var total float64
	db.Model(&models.Income{}).
		Where("date >= ? AND date <= ? AND account_id IN ?", startOfYear, now, accountIDs).
		Select("COALESCE(SUM(gross_amount), 0)").
		Scan(&total)

	return total
}

// GetYTDTax retorna o imposto pago acumulado no ano para contas específicas
func GetYTDTax(db *gorm.DB, accountIDs []uint) float64 {
	if len(accountIDs) == 0 {
		return 0
	}

	now := time.Now()
	startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.Local)

	var total float64
	db.Model(&models.Income{}).
		Where("date >= ? AND date <= ? AND account_id IN ?", startOfYear, now, accountIDs).
		Select("COALESCE(SUM(tax_amount), 0)").
		Scan(&total)

	return total
}

// GetYTDNetIncome retorna a receita líquida acumulada no ano para contas específicas
func GetYTDNetIncome(db *gorm.DB, accountIDs []uint) float64 {
	if len(accountIDs) == 0 {
		return 0
	}

	now := time.Now()
	startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.Local)

	var total float64
	db.Model(&models.Income{}).
		Where("date >= ? AND date <= ? AND account_id IN ?", startOfYear, now, accountIDs).
		Select("COALESCE(SUM(net_amount), 0)").
		Scan(&total)

	return total
}

// GetYTDINSS calcula o INSS acumulado no ano baseado no número de meses e configuração
// months = número de meses completos decorridos no ano
// config = configuração do INSS (pró-labore, teto e alíquota)
func GetYTDINSS(months int, config INSSConfig) float64 {
	if months <= 0 {
		return 0
	}

	// Limita a 12 meses
	if months > 12 {
		months = 12
	}

	// Calcula o INSS mensal e multiplica pelos meses
	monthlyINSS := CalculateINSS(config)
	return monthlyINSS * float64(months)
}

// GetMonthsElapsed retorna o número de meses completos decorridos no ano atual
// Se estamos em janeiro, retorna 1 (contamos o mês atual como em andamento)
func GetMonthsElapsed() int {
	now := time.Now()
	return int(now.Month())
}

// GetTaxProjection calcula a projeção de impostos para o ano
// db = conexão com banco de dados
// accountIDs = IDs das contas a considerar
// inssConfig = configuração do INSS (pode ser nil para não calcular INSS)
func GetTaxProjection(db *gorm.DB, accountIDs []uint, inssConfig *INSSConfig) TaxProjection {
	now := time.Now()
	monthsElapsed := GetMonthsElapsed()

	projection := TaxProjection{
		Year:          now.Year(),
		MonthsElapsed: monthsElapsed,
		CalculatedAt:  now,
	}

	// Obter valores YTD
	projection.YTDIncome = GetYTDIncome(db, accountIDs)
	projection.YTDTax = GetYTDTax(db, accountIDs)
	projection.YTDNetIncome = GetYTDNetIncome(db, accountIDs)

	// Calcular INSS YTD se config fornecida
	if inssConfig != nil {
		projection.YTDINSS = GetYTDINSS(monthsElapsed, *inssConfig)
		// Projeção anual de INSS (12 meses)
		projection.ProjectedAnnualINSS = CalculateINSS(*inssConfig) * 12
	}

	// Projeção de receita anual baseada na média mensal
	// Evita divisão por zero
	if monthsElapsed > 0 {
		avgMonthlyIncome := projection.YTDIncome / float64(monthsElapsed)
		projection.ProjectedAnnualIncome = avgMonthlyIncome * 12
	}

	// Obter faturamento dos últimos 12 meses para cálculo da alíquota
	revenue12M := GetRevenue12MonthsForAccounts(db, accountIDs)

	// Se não há histórico de 12 meses, usa a projeção anual
	if revenue12M <= 0 {
		revenue12M = projection.ProjectedAnnualIncome
	}

	// Calcular bracket e alíquota efetiva
	bracket, rate, nextAt := GetBracketInfo(revenue12M)
	projection.CurrentBracket = bracket
	projection.EffectiveRate = rate
	projection.NextBracketAt = nextAt

	// Projeção de imposto anual
	// Usa a alíquota efetiva atual sobre a receita projetada
	if projection.ProjectedAnnualIncome > 0 && revenue12M > 0 {
		taxCalc := CalculateTax(revenue12M, projection.ProjectedAnnualIncome)
		projection.ProjectedAnnualTax = taxCalc.TaxAmount
		projection.ProjectedNetIncome = projection.ProjectedAnnualIncome - projection.ProjectedAnnualTax - projection.ProjectedAnnualINSS
	}

	return projection
}

// GetTaxProjectionForYear calcula a projeção de impostos para um ano específico
// Útil para visualizar projeções de anos anteriores ou fazer simulações
func GetTaxProjectionForYear(db *gorm.DB, year int, accountIDs []uint, inssConfig *INSSConfig) TaxProjection {
	now := time.Now()
	currentYear := now.Year()

	// Para anos futuros, não temos dados
	if year > currentYear {
		return TaxProjection{
			Year:         year,
			CalculatedAt: now,
		}
	}

	// Para anos passados, usamos todos os 12 meses
	monthsElapsed := 12
	if year == currentYear {
		monthsElapsed = int(now.Month())
	}

	projection := TaxProjection{
		Year:          year,
		MonthsElapsed: monthsElapsed,
		CalculatedAt:  now,
	}

	// Definir datas do período
	startOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
	endOfPeriod := time.Date(year, time.Month(monthsElapsed), 1, 0, 0, 0, 0, time.Local).AddDate(0, 1, 0).Add(-time.Second)
	if year == currentYear {
		endOfPeriod = now
	}

	if len(accountIDs) > 0 {
		// YTD Income
		db.Model(&models.Income{}).
			Where("date >= ? AND date <= ? AND account_id IN ?", startOfYear, endOfPeriod, accountIDs).
			Select("COALESCE(SUM(gross_amount), 0)").
			Scan(&projection.YTDIncome)

		// YTD Tax
		db.Model(&models.Income{}).
			Where("date >= ? AND date <= ? AND account_id IN ?", startOfYear, endOfPeriod, accountIDs).
			Select("COALESCE(SUM(tax_amount), 0)").
			Scan(&projection.YTDTax)

		// YTD Net Income
		db.Model(&models.Income{}).
			Where("date >= ? AND date <= ? AND account_id IN ?", startOfYear, endOfPeriod, accountIDs).
			Select("COALESCE(SUM(net_amount), 0)").
			Scan(&projection.YTDNetIncome)
	}

	// Calcular INSS YTD
	if inssConfig != nil {
		projection.YTDINSS = GetYTDINSS(monthsElapsed, *inssConfig)
		projection.ProjectedAnnualINSS = CalculateINSS(*inssConfig) * 12
	}

	// Projeção de receita anual
	if monthsElapsed > 0 {
		avgMonthlyIncome := projection.YTDIncome / float64(monthsElapsed)
		projection.ProjectedAnnualIncome = avgMonthlyIncome * 12
	}

	// Para anos passados completos, a projeção é igual ao realizado
	if year < currentYear {
		projection.ProjectedAnnualIncome = projection.YTDIncome
		projection.ProjectedAnnualTax = projection.YTDTax
		projection.ProjectedNetIncome = projection.YTDNetIncome
	}

	// Obter informações do bracket
	revenue12M := GetRevenue12MonthsForAccounts(db, accountIDs)
	if revenue12M <= 0 {
		revenue12M = projection.ProjectedAnnualIncome
	}

	bracket, rate, nextAt := GetBracketInfo(revenue12M)
	projection.CurrentBracket = bracket
	projection.EffectiveRate = rate
	projection.NextBracketAt = nextAt

	// Calcular projeção de imposto para ano atual
	if year == currentYear && projection.ProjectedAnnualIncome > 0 && revenue12M > 0 {
		taxCalc := CalculateTax(revenue12M, projection.ProjectedAnnualIncome)
		projection.ProjectedAnnualTax = taxCalc.TaxAmount
		projection.ProjectedNetIncome = projection.ProjectedAnnualIncome - projection.ProjectedAnnualTax - projection.ProjectedAnnualINSS
	}

	return projection
}

// MonthlyTaxBreakdown represents tax information for a specific month
type MonthlyTaxBreakdown struct {
	Month       int     `json:"month"`
	MonthName   string  `json:"month_name"`
	GrossIncome float64 `json:"gross_income"`
	TaxPaid     float64 `json:"tax_paid"`
	NetIncome   float64 `json:"net_income"`
	INSSPaid    float64 `json:"inss_paid"` // INSS contribution for the month
}

// GetMonthlyTaxBreakdown returns tax breakdown by month for the year
func GetMonthlyTaxBreakdown(db *gorm.DB, year int, accountIDs []uint, inssConfig *INSSConfig) []MonthlyTaxBreakdown {
	monthNames := []string{
		"", // Index 0 não usado
		"Janeiro", "Fevereiro", "Março", "Abril",
		"Maio", "Junho", "Julho", "Agosto",
		"Setembro", "Outubro", "Novembro", "Dezembro",
	}

	breakdown := make([]MonthlyTaxBreakdown, 12)

	for month := 1; month <= 12; month++ {
		startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
		endDate := startDate.AddDate(0, 1, 0).Add(-time.Second)

		mb := MonthlyTaxBreakdown{
			Month:     month,
			MonthName: monthNames[month],
		}

		if len(accountIDs) > 0 {
			// Query income for the month
			db.Model(&models.Income{}).
				Where("date >= ? AND date <= ? AND account_id IN ?", startDate, endDate, accountIDs).
				Select("COALESCE(SUM(gross_amount), 0)").
				Scan(&mb.GrossIncome)

			db.Model(&models.Income{}).
				Where("date >= ? AND date <= ? AND account_id IN ?", startDate, endDate, accountIDs).
				Select("COALESCE(SUM(tax_amount), 0)").
				Scan(&mb.TaxPaid)

			db.Model(&models.Income{}).
				Where("date >= ? AND date <= ? AND account_id IN ?", startDate, endDate, accountIDs).
				Select("COALESCE(SUM(net_amount), 0)").
				Scan(&mb.NetIncome)
		}

		// Calculate INSS for the month
		if inssConfig != nil {
			mb.INSSPaid = CalculateINSS(*inssConfig)
		}

		breakdown[month-1] = mb
	}

	return breakdown
}
