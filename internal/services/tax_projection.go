package services

import (
	"time"

	"gorm.io/gorm"

	"poc-finance/internal/models"
)

// BracketWarning contains information about approaching the next tax bracket
type BracketWarning struct {
	IsApproaching     bool    `json:"is_approaching"`       // Indica se está se aproximando da próxima faixa
	AmountUntilNext   float64 `json:"amount_until_next"`    // Valor restante até a próxima faixa
	PercentToNext     float64 `json:"percent_to_next"`      // Percentual do caminho até a próxima faixa (0-100)
	WarningLevel      string  `json:"warning_level"`        // Nível do aviso: "none", "low", "medium", "high", "critical"
	WarningMessage    string  `json:"warning_message"`      // Mensagem de aviso para o usuário
	NextBracketRate   float64 `json:"next_bracket_rate"`    // Alíquota nominal da próxima faixa (percentual)
	ProjectedBracket  int     `json:"projected_bracket"`    // Faixa projetada se mantiver ritmo atual
}

// BracketWarningThresholds define os limites para cada nível de aviso
// Valores representam a percentagem do caminho até a próxima faixa
const (
	WarningThresholdLow      = 70.0  // 70% do caminho - aviso baixo
	WarningThresholdMedium   = 85.0  // 85% do caminho - aviso médio
	WarningThresholdHigh     = 95.0  // 95% do caminho - aviso alto
	WarningThresholdCritical = 100.0 // 100% ou mais - vai ultrapassar
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

	// Bracket warning
	BracketWarning *BracketWarning `json:"bracket_warning,omitempty"` // Aviso sobre aproximação da próxima faixa

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

// GetBracketWarning calcula o aviso de aproximação da próxima faixa de imposto
// currentRevenue = Faturamento atual (últimos 12 meses ou projeção)
// projectedRevenue = Faturamento projetado para o período completo
// currentBracket = Faixa atual do Simples Nacional (1-6)
func GetBracketWarning(currentRevenue, projectedRevenue float64, currentBracket int) *BracketWarning {
	warning := &BracketWarning{
		IsApproaching:    false,
		WarningLevel:     "none",
		WarningMessage:   "",
		ProjectedBracket: currentBracket,
	}

	// Se já está na última faixa (6), não há próxima faixa para se aproximar
	if currentBracket >= len(AnexoIII) {
		warning.AmountUntilNext = 0
		warning.PercentToNext = 100
		warning.WarningLevel = "none"
		warning.WarningMessage = "Você já está na última faixa do Simples Nacional."
		return warning
	}

	// Obtém a faixa atual e a próxima
	currentBracketData := AnexoIII[currentBracket-1]
	nextBracketData := AnexoIII[currentBracket]

	// Limites da faixa atual
	bracketMin := currentBracketData.MinRevenue
	bracketMax := currentBracketData.MaxRevenue

	// Calcula o quanto falta até a próxima faixa
	warning.AmountUntilNext = bracketMax - currentRevenue
	if warning.AmountUntilNext < 0 {
		warning.AmountUntilNext = 0
	}

	// Calcula a alíquota nominal da próxima faixa (em percentual)
	warning.NextBracketRate = nextBracketData.Rate * 100

	// Calcula o percentual do caminho percorrido na faixa atual
	bracketRange := bracketMax - bracketMin
	if bracketRange > 0 {
		positionInBracket := currentRevenue - bracketMin
		warning.PercentToNext = (positionInBracket / bracketRange) * 100
		if warning.PercentToNext < 0 {
			warning.PercentToNext = 0
		}
		if warning.PercentToNext > 100 {
			warning.PercentToNext = 100
		}
	}

	// Determina a faixa projetada baseada na receita projetada
	if projectedRevenue > 0 {
		for i, b := range AnexoIII {
			if projectedRevenue >= b.MinRevenue && projectedRevenue <= b.MaxRevenue {
				warning.ProjectedBracket = i + 1
				break
			}
		}
		// Se ultrapassou o limite do Simples
		if projectedRevenue > 4800000 {
			warning.ProjectedBracket = len(AnexoIII)
		}
	}

	// Determina o nível de aviso baseado no percentual e na projeção
	warning.IsApproaching = warning.PercentToNext >= WarningThresholdLow || warning.ProjectedBracket > currentBracket

	if warning.ProjectedBracket > currentBracket {
		// A projeção indica que vai mudar de faixa
		warning.WarningLevel = "critical"
		warning.WarningMessage = formatCriticalMessage(currentBracket, warning.ProjectedBracket, warning.AmountUntilNext, warning.NextBracketRate)
	} else if warning.PercentToNext >= WarningThresholdHigh {
		warning.WarningLevel = "high"
		warning.WarningMessage = formatHighMessage(warning.AmountUntilNext, warning.NextBracketRate)
	} else if warning.PercentToNext >= WarningThresholdMedium {
		warning.WarningLevel = "medium"
		warning.WarningMessage = formatMediumMessage(warning.AmountUntilNext, warning.NextBracketRate)
	} else if warning.PercentToNext >= WarningThresholdLow {
		warning.WarningLevel = "low"
		warning.WarningMessage = formatLowMessage(warning.AmountUntilNext, warning.NextBracketRate)
	} else {
		warning.WarningLevel = "none"
		warning.WarningMessage = ""
	}

	return warning
}

// formatCriticalMessage gera mensagem para aviso crítico (vai ultrapassar faixa)
func formatCriticalMessage(currentBracket, projectedBracket int, amountUntil, nextRate float64) string {
	if amountUntil <= 0 {
		return "Atenção! Sua projeção indica mudança para a faixa " +
			formatBracketNumber(projectedBracket) +
			" com alíquota nominal de " +
			formatPercent(nextRate) + "."
	}
	return "Atenção! Faltam apenas R$ " +
		formatCurrency(amountUntil) +
		" para a próxima faixa (alíquota nominal de " +
		formatPercent(nextRate) +
		"). Sua projeção indica que você passará para a faixa " +
		formatBracketNumber(projectedBracket) + "."
}

// formatHighMessage gera mensagem para aviso alto
func formatHighMessage(amountUntil, nextRate float64) string {
	return "Alerta: Você está muito próximo da próxima faixa! Faltam R$ " +
		formatCurrency(amountUntil) +
		" para a faixa com alíquota nominal de " +
		formatPercent(nextRate) + "."
}

// formatMediumMessage gera mensagem para aviso médio
func formatMediumMessage(amountUntil, nextRate float64) string {
	return "Aviso: Você está se aproximando da próxima faixa. Faltam R$ " +
		formatCurrency(amountUntil) +
		" para a faixa com alíquota nominal de " +
		formatPercent(nextRate) + "."
}

// formatLowMessage gera mensagem para aviso baixo
func formatLowMessage(amountUntil, nextRate float64) string {
	return "Informativo: Faltam R$ " +
		formatCurrency(amountUntil) +
		" para a próxima faixa de tributação (alíquota nominal de " +
		formatPercent(nextRate) + ")."
}

// formatCurrency formata valor monetário em formato brasileiro simplificado
func formatCurrency(value float64) string {
	// Formata com duas casas decimais e separador de milhares
	intPart := int(value)
	decPart := int((value - float64(intPart)) * 100)

	// Formatação simples para milhares
	str := ""
	remaining := intPart
	for remaining > 0 {
		if str != "" {
			str = "." + str
		}
		part := remaining % 1000
		remaining = remaining / 1000
		if remaining > 0 {
			str = padLeft(part, 3) + str
		} else {
			str = formatInt(part) + str
		}
	}
	if str == "" {
		str = "0"
	}

	return str + "," + padLeft(decPart, 2)
}

// formatPercent formata percentual
func formatPercent(value float64) string {
	intPart := int(value)
	decPart := int((value - float64(intPart)) * 10)
	if decPart == 0 {
		return formatInt(intPart) + "%"
	}
	return formatInt(intPart) + "," + formatInt(decPart) + "%"
}

// formatBracketNumber formata número da faixa por extenso
func formatBracketNumber(bracket int) string {
	numbers := []string{"", "1ª", "2ª", "3ª", "4ª", "5ª", "6ª"}
	if bracket >= 1 && bracket <= 6 {
		return numbers[bracket]
	}
	return formatInt(bracket) + "ª"
}

// formatInt converte int para string
func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	negative := n < 0
	if negative {
		n = -n
	}
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n = n / 10
	}
	if negative {
		result = "-" + result
	}
	return result
}

// padLeft adiciona zeros à esquerda
func padLeft(n int, width int) string {
	str := formatInt(n)
	for len(str) < width {
		str = "0" + str
	}
	return str
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

	// Calcular aviso de aproximação da próxima faixa
	projection.BracketWarning = GetBracketWarning(revenue12M, projection.ProjectedAnnualIncome, bracket)

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

	// Calcular aviso de aproximação da próxima faixa (apenas para ano atual)
	if year == currentYear {
		projection.BracketWarning = GetBracketWarning(revenue12M, projection.ProjectedAnnualIncome, bracket)
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
