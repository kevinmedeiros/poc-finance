package services

import "log"

// SimplesNacionalAnexoIII - Cálculo do Simples Nacional Anexo III
// Usado para serviços de tecnologia/consultoria

type TaxBracket struct {
	MinRevenue float64
	MaxRevenue float64
	Rate       float64 // Alíquota nominal
	Deduction  float64 // Valor a deduzir
}

var AnexoIII = []TaxBracket{
	{MinRevenue: 0, MaxRevenue: 180000, Rate: 0.06, Deduction: 0},
	{MinRevenue: 180000.01, MaxRevenue: 360000, Rate: 0.112, Deduction: 9360},
	{MinRevenue: 360000.01, MaxRevenue: 720000, Rate: 0.135, Deduction: 17640},
	{MinRevenue: 720000.01, MaxRevenue: 1800000, Rate: 0.16, Deduction: 35640},
	{MinRevenue: 1800000.01, MaxRevenue: 3600000, Rate: 0.21, Deduction: 125640},
	{MinRevenue: 3600000.01, MaxRevenue: 4800000, Rate: 0.33, Deduction: 648000},
}

type TaxCalculation struct {
	GrossAmount    float64 `json:"gross_amount"`
	Revenue12M     float64 `json:"revenue_12m"`
	EffectiveRate  float64 `json:"effective_rate"`
	TaxAmount      float64 `json:"tax_amount"`
	INSSAmount     float64 `json:"inss_amount"`
	TotalTax       float64 `json:"total_tax"`
	NetAmount      float64 `json:"net_amount"`
	BracketApplied int     `json:"bracket_applied"`
}

type INSSConfig struct {
	ProLabore float64 // Valor do pró-labore mensal
	Ceiling   float64 // Teto do INSS
	Rate      float64 // Alíquota (ex: 0.11 para 11%)
}

// CalculateINSS calcula o INSS sobre o pró-labore
func CalculateINSS(config INSSConfig) float64 {
	if config.ProLabore <= 0 {
		return 0
	}

	// Base de cálculo é o menor entre pró-labore e teto
	base := config.ProLabore
	if base > config.Ceiling {
		base = config.Ceiling
	}

	return base * config.Rate
}

// CalculateTax calcula o imposto baseado no faturamento dos últimos 12 meses
// revenue12M = Receita Bruta Total dos últimos 12 meses
// grossAmount = Valor bruto do recebimento atual
func CalculateTax(revenue12M, grossAmount float64) TaxCalculation {
	result := TaxCalculation{
		GrossAmount: grossAmount,
		Revenue12M:  revenue12M,
	}

	// Se faturamento for zero ou muito baixo, usa a primeira faixa
	if revenue12M <= 0 {
		revenue12M = grossAmount // Considera apenas o recebimento atual
	}

	// Encontra a faixa de tributação
	var bracket TaxBracket
	for i, b := range AnexoIII {
		if revenue12M >= b.MinRevenue && revenue12M <= b.MaxRevenue {
			bracket = b
			result.BracketApplied = i + 1
			break
		}
	}

	// Se ultrapassou o limite do Simples (4.8M), usa a última faixa
	if revenue12M > 4800000 {
		bracket = AnexoIII[len(AnexoIII)-1]
		result.BracketApplied = len(AnexoIII)
	}

	// Calcula a alíquota efetiva
	// Fórmula: (RBT12 × Alíquota - Dedução) / RBT12
	if revenue12M > 0 {
		result.EffectiveRate = (revenue12M*bracket.Rate - bracket.Deduction) / revenue12M
	}

	// Calcula o imposto
	result.TaxAmount = grossAmount * result.EffectiveRate
	result.NetAmount = grossAmount - result.TaxAmount

	return result
}

// GetBracketInfo retorna informações sobre a faixa atual
func GetBracketInfo(revenue12M float64) (bracket int, rate float64, nextBracketAt float64) {
	// Se não houver faturamento, retorna a primeira faixa com alíquota nominal
	if revenue12M <= 0 {
		return 1, 6.0, 180000
	}

	for i, b := range AnexoIII {
		if revenue12M >= b.MinRevenue && revenue12M <= b.MaxRevenue {
			effectiveRate := (revenue12M*b.Rate - b.Deduction) / revenue12M
			nextAt := b.MaxRevenue
			if i < len(AnexoIII)-1 {
				nextAt = AnexoIII[i+1].MinRevenue
			}
			return i + 1, effectiveRate * 100, nextAt
		}
	}
	return len(AnexoIII), 33, 4800000
}

// GetBracketInfoWithManualOverride retorna informações sobre a faixa considerando o override manual
// Se manualBracket > 0, usa a faixa especificada ao invés de calcular automaticamente
// A aliquota efetiva é calculada usando o faturamento real dentro da faixa selecionada
func GetBracketInfoWithManualOverride(revenue12M float64, manualBracket int) (bracket int, rate float64, nextBracketAt float64) {
	// Se não houver override manual (0 = automático), usa o cálculo padrão
	if manualBracket <= 0 || manualBracket > len(AnexoIII) {
		return GetBracketInfo(revenue12M)
	}

	// Usa a faixa manual especificada
	b := AnexoIII[manualBracket-1]

	// Para cálculo da alíquota efetiva com faixa manual, usamos o ponto médio da faixa
	// se o faturamento real estiver fora da faixa selecionada.
	// Isso dá uma alíquota mais representativa da faixa escolhida.
	effectiveRevenue := revenue12M
	if effectiveRevenue < b.MinRevenue || effectiveRevenue > b.MaxRevenue {
		// Usa o ponto médio da faixa para calcular a alíquota efetiva
		effectiveRevenue = (b.MinRevenue + b.MaxRevenue) / 2
		if effectiveRevenue <= 0 {
			effectiveRevenue = b.MaxRevenue / 2 // Primeira faixa tem mínimo 0
		}
	}

	// Fórmula: (RBT12 × Alíquota - Dedução) / RBT12
	effectiveRate := (effectiveRevenue*b.Rate - b.Deduction) / effectiveRevenue

	// Próxima faixa
	nextAt := b.MaxRevenue
	if manualBracket < len(AnexoIII) {
		nextAt = AnexoIII[manualBracket].MinRevenue
	}

	return manualBracket, effectiveRate * 100, nextAt
}

// CalculateTaxWithManualBracket calcula o imposto usando uma faixa específica
// Se manualBracket > 0, usa a faixa especificada ao invés de calcular automaticamente
func CalculateTaxWithManualBracket(revenue12M, grossAmount float64, manualBracket int) TaxCalculation {
	// Se não houver override manual, usa o cálculo padrão
	if manualBracket <= 0 || manualBracket > len(AnexoIII) {
		return CalculateTax(revenue12M, grossAmount)
	}

	// Log para debug
	log.Printf("[Tax] Using manual bracket %d for calculation (revenue12M: %.2f, gross: %.2f)", manualBracket, revenue12M, grossAmount)

	result := TaxCalculation{
		GrossAmount:    grossAmount,
		Revenue12M:     revenue12M,
		BracketApplied: manualBracket,
	}

	// Usa a faixa manual especificada
	bracket := AnexoIII[manualBracket-1]

	// Para cálculo da alíquota efetiva com faixa manual, usamos o ponto médio da faixa
	// se o faturamento real for menor que o mínimo da faixa selecionada.
	// Isso dá uma alíquota mais representativa da faixa escolhida.
	effectiveRevenue := revenue12M
	if effectiveRevenue < bracket.MinRevenue || effectiveRevenue > bracket.MaxRevenue {
		// Usa o ponto médio da faixa para calcular a alíquota efetiva
		effectiveRevenue = (bracket.MinRevenue + bracket.MaxRevenue) / 2
		if effectiveRevenue <= 0 {
			effectiveRevenue = bracket.MaxRevenue / 2 // Primeira faixa tem mínimo 0
		}
	}

	// Fórmula: (RBT12 × Alíquota - Dedução) / RBT12
	result.EffectiveRate = (effectiveRevenue*bracket.Rate - bracket.Deduction) / effectiveRevenue
	log.Printf("[Tax] Calculated effective rate: %.4f (effectiveRevenue: %.2f, rate: %.4f, deduction: %.2f)",
		result.EffectiveRate, effectiveRevenue, bracket.Rate, bracket.Deduction)

	// Calcula o imposto
	result.TaxAmount = grossAmount * result.EffectiveRate
	result.NetAmount = grossAmount - result.TaxAmount

	return result
}
