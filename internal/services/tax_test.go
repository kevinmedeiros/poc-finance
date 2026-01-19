package services

import (
	"math"
	"testing"
)

func TestCalculateINSS(t *testing.T) {
	tests := []struct {
		name     string
		config   INSSConfig
		expected float64
	}{
		{
			name: "zero pro-labore returns zero",
			config: INSSConfig{
				ProLabore: 0,
				Ceiling:   7786.02,
				Rate:      0.11,
			},
			expected: 0,
		},
		{
			name: "negative pro-labore returns zero",
			config: INSSConfig{
				ProLabore: -1000,
				Ceiling:   7786.02,
				Rate:      0.11,
			},
			expected: 0,
		},
		{
			name: "pro-labore below ceiling",
			config: INSSConfig{
				ProLabore: 5000,
				Ceiling:   7786.02,
				Rate:      0.11,
			},
			expected: 550, // 5000 * 0.11
		},
		{
			name: "pro-labore at ceiling",
			config: INSSConfig{
				ProLabore: 7786.02,
				Ceiling:   7786.02,
				Rate:      0.11,
			},
			expected: 856.4622, // 7786.02 * 0.11
		},
		{
			name: "pro-labore above ceiling uses ceiling",
			config: INSSConfig{
				ProLabore: 15000,
				Ceiling:   7786.02,
				Rate:      0.11,
			},
			expected: 856.4622, // ceiling * rate
		},
		{
			name: "different rate calculation",
			config: INSSConfig{
				ProLabore: 3000,
				Ceiling:   7786.02,
				Rate:      0.20,
			},
			expected: 600, // 3000 * 0.20
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateINSS(tt.config)
			if math.Abs(result-tt.expected) > 0.01 {
				t.Errorf("CalculateINSS() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCalculateTax(t *testing.T) {
	tests := []struct {
		name              string
		revenue12M        float64
		grossAmount       float64
		expectedRate      float64 // effective rate
		expectedBracket   int
		expectedTaxAmount float64
	}{
		{
			name:              "first bracket - new company",
			revenue12M:        0,
			grossAmount:       10000,
			expectedRate:      0.06,
			expectedBracket:   1,
			expectedTaxAmount: 600,
		},
		{
			name:              "first bracket - low revenue",
			revenue12M:        100000,
			grossAmount:       5000,
			expectedRate:      0.06,
			expectedBracket:   1,
			expectedTaxAmount: 300,
		},
		{
			name:              "first bracket - at max",
			revenue12M:        180000,
			grossAmount:       10000,
			expectedRate:      0.06,
			expectedBracket:   1,
			expectedTaxAmount: 600,
		},
		{
			name:            "second bracket - just above first",
			revenue12M:      180000.01,
			grossAmount:     10000,
			expectedBracket: 2,
		},
		{
			name:            "second bracket - mid range",
			revenue12M:      270000,
			grossAmount:     10000,
			expectedBracket: 2,
		},
		{
			name:            "third bracket",
			revenue12M:      500000,
			grossAmount:     10000,
			expectedBracket: 3,
		},
		{
			name:            "fourth bracket",
			revenue12M:      1000000,
			grossAmount:     10000,
			expectedBracket: 4,
		},
		{
			name:            "fifth bracket",
			revenue12M:      2500000,
			grossAmount:     10000,
			expectedBracket: 5,
		},
		{
			name:            "sixth bracket - max simples",
			revenue12M:      4000000,
			grossAmount:     10000,
			expectedBracket: 6,
		},
		{
			name:            "above simples limit uses last bracket",
			revenue12M:      5000000,
			grossAmount:     10000,
			expectedBracket: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTax(tt.revenue12M, tt.grossAmount)

			if result.BracketApplied != tt.expectedBracket {
				t.Errorf("BracketApplied = %v, want %v", result.BracketApplied, tt.expectedBracket)
			}

			if result.GrossAmount != tt.grossAmount {
				t.Errorf("GrossAmount = %v, want %v", result.GrossAmount, tt.grossAmount)
			}

			// For specific cases where we know the expected rate
			if tt.expectedRate > 0 {
				if math.Abs(result.EffectiveRate-tt.expectedRate) > 0.001 {
					t.Errorf("EffectiveRate = %v, want %v", result.EffectiveRate, tt.expectedRate)
				}
			}

			if tt.expectedTaxAmount > 0 {
				if math.Abs(result.TaxAmount-tt.expectedTaxAmount) > 0.01 {
					t.Errorf("TaxAmount = %v, want %v", result.TaxAmount, tt.expectedTaxAmount)
				}
			}

			// Verify net amount calculation
			expectedNet := tt.grossAmount - result.TaxAmount
			if math.Abs(result.NetAmount-expectedNet) > 0.01 {
				t.Errorf("NetAmount = %v, want %v", result.NetAmount, expectedNet)
			}
		})
	}
}

func TestCalculateTax_EffectiveRateFormula(t *testing.T) {
	// Test the effective rate formula: (RBT12 × Alíquota - Dedução) / RBT12
	// For bracket 2: revenue 270000, rate 0.112, deduction 9360
	// Effective rate = (270000 * 0.112 - 9360) / 270000 = 0.077333...

	result := CalculateTax(270000, 10000)

	expectedEffectiveRate := (270000*0.112 - 9360) / 270000
	if math.Abs(result.EffectiveRate-expectedEffectiveRate) > 0.0001 {
		t.Errorf("EffectiveRate = %v, want %v", result.EffectiveRate, expectedEffectiveRate)
	}

	// Tax amount should be gross * effective rate
	expectedTax := 10000 * expectedEffectiveRate
	if math.Abs(result.TaxAmount-expectedTax) > 0.01 {
		t.Errorf("TaxAmount = %v, want %v", result.TaxAmount, expectedTax)
	}
}

func TestGetBracketInfo(t *testing.T) {
	tests := []struct {
		name           string
		revenue12M     float64
		expectedBrkt   int
		expectedNextAt float64
	}{
		{
			name:           "zero revenue returns first bracket",
			revenue12M:     0,
			expectedBrkt:   1,
			expectedNextAt: 180000,
		},
		{
			name:           "negative revenue returns first bracket",
			revenue12M:     -1000,
			expectedBrkt:   1,
			expectedNextAt: 180000,
		},
		{
			name:           "first bracket",
			revenue12M:     100000,
			expectedBrkt:   1,
			expectedNextAt: 180000.01, // MinRevenue of next bracket
		},
		{
			name:           "second bracket",
			revenue12M:     250000,
			expectedBrkt:   2,
			expectedNextAt: 360000.01,
		},
		{
			name:           "third bracket",
			revenue12M:     500000,
			expectedBrkt:   3,
			expectedNextAt: 720000.01,
		},
		{
			name:           "fourth bracket",
			revenue12M:     1000000,
			expectedBrkt:   4,
			expectedNextAt: 1800000.01,
		},
		{
			name:           "fifth bracket",
			revenue12M:     2500000,
			expectedBrkt:   5,
			expectedNextAt: 3600000.01,
		},
		{
			name:           "sixth bracket - last one",
			revenue12M:     4000000,
			expectedBrkt:   6,
			expectedNextAt: 4800000, // No next bracket
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bracket, rate, nextAt := GetBracketInfo(tt.revenue12M)

			if bracket != tt.expectedBrkt {
				t.Errorf("bracket = %v, want %v", bracket, tt.expectedBrkt)
			}

			if math.Abs(nextAt-tt.expectedNextAt) > 0.01 {
				t.Errorf("nextBracketAt = %v, want %v", nextAt, tt.expectedNextAt)
			}

			// Rate should be positive and less than 100%
			if rate < 0 || rate > 100 {
				t.Errorf("rate = %v, should be between 0 and 100", rate)
			}
		})
	}
}

func TestGetBracketInfo_RateCalculation(t *testing.T) {
	// Test that the rate is correctly calculated as percentage
	// For bracket 1 with revenue 100000: effective rate = 6%
	bracket, rate, _ := GetBracketInfo(100000)

	if bracket != 1 {
		t.Errorf("bracket = %v, want 1", bracket)
	}

	if math.Abs(rate-6.0) > 0.01 {
		t.Errorf("rate = %v, want 6.0", rate)
	}
}

func TestAnexoIII_BracketContinuity(t *testing.T) {
	// Verify that bracket boundaries are contiguous
	for i := 0; i < len(AnexoIII)-1; i++ {
		current := AnexoIII[i]
		next := AnexoIII[i+1]

		// The max of current bracket + 0.01 should equal min of next bracket
		expectedNextMin := current.MaxRevenue + 0.01
		if math.Abs(next.MinRevenue-expectedNextMin) > 0.001 {
			t.Errorf("Gap between bracket %d and %d: max=%v, next min=%v",
				i+1, i+2, current.MaxRevenue, next.MinRevenue)
		}
	}
}

func TestAnexoIII_BracketRatesIncrease(t *testing.T) {
	// Verify that rates increase with each bracket
	for i := 0; i < len(AnexoIII)-1; i++ {
		current := AnexoIII[i]
		next := AnexoIII[i+1]

		if next.Rate <= current.Rate {
			t.Errorf("Rate should increase: bracket %d rate=%v, bracket %d rate=%v",
				i+1, current.Rate, i+2, next.Rate)
		}
	}
}
