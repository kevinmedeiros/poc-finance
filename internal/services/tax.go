package services

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
