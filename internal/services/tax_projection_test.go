package services

import (
	"math"
	"testing"
	"time"

	"poc-finance/internal/models"
	"poc-finance/internal/testutil"
)

// TestGetYTDINSS tests the INSS YTD calculation
func TestGetYTDINSS(t *testing.T) {
	tests := []struct {
		name     string
		months   int
		config   INSSConfig
		expected float64
	}{
		{
			name:   "zero months returns zero",
			months: 0,
			config: INSSConfig{
				ProLabore: 5000,
				Ceiling:   7786.02,
				Rate:      0.11,
			},
			expected: 0,
		},
		{
			name:   "negative months returns zero",
			months: -1,
			config: INSSConfig{
				ProLabore: 5000,
				Ceiling:   7786.02,
				Rate:      0.11,
			},
			expected: 0,
		},
		{
			name:   "one month calculation",
			months: 1,
			config: INSSConfig{
				ProLabore: 5000,
				Ceiling:   7786.02,
				Rate:      0.11,
			},
			expected: 550, // 5000 * 0.11 * 1
		},
		{
			name:   "six months calculation",
			months: 6,
			config: INSSConfig{
				ProLabore: 5000,
				Ceiling:   7786.02,
				Rate:      0.11,
			},
			expected: 3300, // 5000 * 0.11 * 6
		},
		{
			name:   "twelve months full year",
			months: 12,
			config: INSSConfig{
				ProLabore: 5000,
				Ceiling:   7786.02,
				Rate:      0.11,
			},
			expected: 6600, // 5000 * 0.11 * 12
		},
		{
			name:   "months above 12 are capped",
			months: 15,
			config: INSSConfig{
				ProLabore: 5000,
				Ceiling:   7786.02,
				Rate:      0.11,
			},
			expected: 6600, // Capped at 12 months
		},
		{
			name:   "pro-labore above ceiling uses ceiling",
			months: 3,
			config: INSSConfig{
				ProLabore: 10000,
				Ceiling:   7786.02,
				Rate:      0.11,
			},
			expected: 2569.3866, // 7786.02 * 0.11 * 3
		},
		{
			name:   "zero pro-labore returns zero",
			months: 6,
			config: INSSConfig{
				ProLabore: 0,
				Ceiling:   7786.02,
				Rate:      0.11,
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetYTDINSS(tt.months, tt.config)
			if math.Abs(result-tt.expected) > 0.01 {
				t.Errorf("GetYTDINSS() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestGetMonthsElapsed tests the months elapsed calculation
func TestGetMonthsElapsed(t *testing.T) {
	// This test verifies the function returns the current month number
	// The actual result depends on when the test is run
	result := GetMonthsElapsed()

	// Should return a value between 1 and 12
	if result < 1 || result > 12 {
		t.Errorf("GetMonthsElapsed() = %v, should be between 1 and 12", result)
	}

	// Should match current month
	expectedMonth := int(time.Now().Month())
	if result != expectedMonth {
		t.Errorf("GetMonthsElapsed() = %v, want %v", result, expectedMonth)
	}
}

// TestGetBracketWarning tests bracket warning calculations
func TestGetBracketWarning(t *testing.T) {
	tests := []struct {
		name                  string
		currentRevenue        float64
		projectedRevenue      float64
		currentBracket        int
		expectedIsApproaching bool
		expectedWarningLevel  string
		expectedProjected     int
	}{
		{
			name:                  "last bracket - no warning possible",
			currentRevenue:        4500000,
			projectedRevenue:      4800000,
			currentBracket:        6,
			expectedIsApproaching: false,
			expectedWarningLevel:  "none",
			expectedProjected:     6,
		},
		{
			name:                  "first bracket - low revenue - no warning",
			currentRevenue:        50000,
			projectedRevenue:      100000,
			currentBracket:        1,
			expectedIsApproaching: false,
			expectedWarningLevel:  "none",
			expectedProjected:     1,
		},
		{
			name:                  "first bracket - 70% threshold - low warning",
			currentRevenue:        126000, // 70% of 180000
			projectedRevenue:      150000,
			currentBracket:        1,
			expectedIsApproaching: true,
			expectedWarningLevel:  "low",
			expectedProjected:     1,
		},
		{
			name:                  "first bracket - 85% threshold - medium warning",
			currentRevenue:        153000, // 85% of 180000
			projectedRevenue:      170000,
			currentBracket:        1,
			expectedIsApproaching: true,
			expectedWarningLevel:  "medium",
			expectedProjected:     1,
		},
		{
			name:                  "first bracket - 95% threshold - high warning",
			currentRevenue:        171000, // 95% of 180000
			projectedRevenue:      175000,
			currentBracket:        1,
			expectedIsApproaching: true,
			expectedWarningLevel:  "high",
			expectedProjected:     1,
		},
		{
			name:                  "first bracket - projection exceeds - critical",
			currentRevenue:        150000,
			projectedRevenue:      200000, // Exceeds bracket 1 limit
			currentBracket:        1,
			expectedIsApproaching: true,
			expectedWarningLevel:  "critical",
			expectedProjected:     2,
		},
		{
			name:                  "second bracket - mid range - no warning",
			currentRevenue:        220000,
			projectedRevenue:      250000,
			currentBracket:        2,
			expectedIsApproaching: false,
			expectedWarningLevel:  "none",
			expectedProjected:     2,
		},
		{
			name:                  "projection exceeds simples limit",
			currentRevenue:        4500000,
			projectedRevenue:      5000000, // Exceeds Simples limit
			currentBracket:        6,
			expectedIsApproaching: false,
			expectedWarningLevel:  "none",
			expectedProjected:     6, // Stays at 6 (max)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetBracketWarning(tt.currentRevenue, tt.projectedRevenue, tt.currentBracket)

			if result == nil {
				t.Fatal("GetBracketWarning() returned nil")
			}

			if result.IsApproaching != tt.expectedIsApproaching {
				t.Errorf("IsApproaching = %v, want %v", result.IsApproaching, tt.expectedIsApproaching)
			}

			if result.WarningLevel != tt.expectedWarningLevel {
				t.Errorf("WarningLevel = %v, want %v", result.WarningLevel, tt.expectedWarningLevel)
			}

			if result.ProjectedBracket != tt.expectedProjected {
				t.Errorf("ProjectedBracket = %v, want %v", result.ProjectedBracket, tt.expectedProjected)
			}

			// Verify AmountUntilNext is non-negative
			if result.AmountUntilNext < 0 {
				t.Errorf("AmountUntilNext = %v, should be >= 0", result.AmountUntilNext)
			}

			// Verify PercentToNext is in valid range
			if result.PercentToNext < 0 || result.PercentToNext > 100 {
				t.Errorf("PercentToNext = %v, should be between 0 and 100", result.PercentToNext)
			}
		})
	}
}

// TestGetBracketWarning_AmountUntilNext tests the amount until next bracket calculation
func TestGetBracketWarning_AmountUntilNext(t *testing.T) {
	// Bracket 1: 0 - 180,000
	// If at 150,000, should have 30,000 until next
	result := GetBracketWarning(150000, 150000, 1)

	expectedAmount := 180000.0 - 150000.0
	if math.Abs(result.AmountUntilNext-expectedAmount) > 0.01 {
		t.Errorf("AmountUntilNext = %v, want %v", result.AmountUntilNext, expectedAmount)
	}
}

// TestGetBracketWarning_NextBracketRate tests the next bracket rate is reported correctly
func TestGetBracketWarning_NextBracketRate(t *testing.T) {
	// Test bracket 1 -> bracket 2 transition
	result := GetBracketWarning(150000, 200000, 1)

	// Bracket 2 nominal rate is 11.2%, stored as 0.112
	expectedRate := AnexoIII[1].Rate * 100
	if math.Abs(result.NextBracketRate-expectedRate) > 0.01 {
		t.Errorf("NextBracketRate = %v, want %v", result.NextBracketRate, expectedRate)
	}
}

// TestGetBracketWarning_WarningMessages tests that warning messages are generated
func TestGetBracketWarning_WarningMessages(t *testing.T) {
	tests := []struct {
		name           string
		currentRevenue float64
		projected      float64
		bracket        int
		expectMessage  bool
	}{
		{
			name:           "no warning - no message",
			currentRevenue: 50000,
			projected:      80000,
			bracket:        1,
			expectMessage:  false,
		},
		{
			name:           "low warning - has message",
			currentRevenue: 130000,
			projected:      160000,
			bracket:        1,
			expectMessage:  true,
		},
		{
			name:           "critical warning - has message",
			currentRevenue: 170000,
			projected:      250000,
			bracket:        1,
			expectMessage:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetBracketWarning(tt.currentRevenue, tt.projected, tt.bracket)

			hasMessage := result.WarningMessage != ""
			if hasMessage != tt.expectMessage {
				t.Errorf("WarningMessage present = %v, want %v", hasMessage, tt.expectMessage)
			}
		})
	}
}

// TestFormatCurrency tests the currency formatting function
func TestTaxProjectionFormatCurrency(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		expected string
	}{
		{
			name:     "zero value",
			value:    0,
			expected: "0,00",
		},
		{
			name:     "small value",
			value:    123.45,
			expected: "123,45",
		},
		{
			name:     "thousands - integer",
			value:    1234,
			expected: "1.234,00",
		},
		{
			name:     "whole number",
			value:    5000.00,
			expected: "5.000,00",
		},
		{
			name:     "simple decimal",
			value:    100.50,
			expected: "100,50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCurrency(tt.value)
			if result != tt.expected {
				t.Errorf("formatCurrency(%v) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}

// TestFormatPercent tests the percentage formatting function
func TestTaxProjectionFormatPercent(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		expected string
	}{
		{
			name:     "whole number",
			value:    6.0,
			expected: "6%",
		},
		{
			name:     "decimal 5.5",
			value:    5.5,
			expected: "5,5%",
		},
		{
			name:     "zero",
			value:    0,
			expected: "0%",
		},
		{
			name:     "large whole number",
			value:    33.0,
			expected: "33%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatPercent(tt.value)
			if result != tt.expected {
				t.Errorf("formatPercent(%v) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}

// TestFormatBracketNumber tests the bracket number formatting
func TestTaxProjectionFormatBracketNumber(t *testing.T) {
	tests := []struct {
		name     string
		bracket  int
		expected string
	}{
		{name: "bracket 1", bracket: 1, expected: "1ª"},
		{name: "bracket 2", bracket: 2, expected: "2ª"},
		{name: "bracket 3", bracket: 3, expected: "3ª"},
		{name: "bracket 4", bracket: 4, expected: "4ª"},
		{name: "bracket 5", bracket: 5, expected: "5ª"},
		{name: "bracket 6", bracket: 6, expected: "6ª"},
		{name: "bracket 7 (out of range)", bracket: 7, expected: "7ª"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBracketNumber(tt.bracket)
			if result != tt.expected {
				t.Errorf("formatBracketNumber(%v) = %v, want %v", tt.bracket, result, tt.expected)
			}
		})
	}
}

// TestFormatInt tests the integer formatting
func TestTaxProjectionFormatInt(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		expected string
	}{
		{name: "zero", value: 0, expected: "0"},
		{name: "positive", value: 123, expected: "123"},
		{name: "negative", value: -456, expected: "-456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatInt(tt.value)
			if result != tt.expected {
				t.Errorf("formatInt(%v) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}

// TestPadLeft tests the left padding function
func TestTaxProjectionPadLeft(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		width    int
		expected string
	}{
		{name: "no padding needed", value: 123, width: 3, expected: "123"},
		{name: "one zero pad", value: 12, width: 3, expected: "012"},
		{name: "two zero pad", value: 1, width: 3, expected: "001"},
		{name: "zero with padding", value: 0, width: 2, expected: "00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := padLeft(tt.value, tt.width)
			if result != tt.expected {
				t.Errorf("padLeft(%v, %v) = %v, want %v", tt.value, tt.width, result, tt.expected)
			}
		})
	}
}

// TestGetYTDIncome_EmptyAccounts tests YTD income with empty account list
func TestGetYTDIncome_EmptyAccounts(t *testing.T) {
	db := testutil.SetupTestDB()

	result := GetYTDIncome(db, []uint{})

	if result != 0 {
		t.Errorf("GetYTDIncome() with empty accounts = %v, want 0", result)
	}
}

// TestGetYTDIncome_WithData tests YTD income calculation with test data
func TestGetYTDIncome_WithData(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	now := time.Now()

	// Create income for current year - only use past dates (up to current month)
	db.Create(&models.Income{
		AccountID:   account.ID,
		Date:        time.Date(now.Year(), 1, 5, 0, 0, 0, 0, time.Local),
		GrossAmount: 5000.00,
		TaxAmount:   500.00,
		NetAmount:   4500.00,
	})

	// Create income for previous year (should not be included)
	db.Create(&models.Income{
		AccountID:   account.ID,
		Date:        time.Date(now.Year()-1, 12, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 10000.00,
		TaxAmount:   1000.00,
		NetAmount:   9000.00,
	})

	result := GetYTDIncome(db, []uint{account.ID})

	// Should only include current year incomes (past dates only)
	expected := 5000.00
	if math.Abs(result-expected) > 0.01 {
		t.Errorf("GetYTDIncome() = %v, want %v", result, expected)
	}
}

// TestGetYTDTax_EmptyAccounts tests YTD tax with empty account list
func TestGetYTDTax_EmptyAccounts(t *testing.T) {
	db := testutil.SetupTestDB()

	result := GetYTDTax(db, []uint{})

	if result != 0 {
		t.Errorf("GetYTDTax() with empty accounts = %v, want 0", result)
	}
}

// TestGetYTDTax_WithData tests YTD tax calculation with test data
func TestGetYTDTax_WithData(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	now := time.Now()

	// Create income for current year - only use past dates
	db.Create(&models.Income{
		AccountID:   account.ID,
		Date:        time.Date(now.Year(), 1, 5, 0, 0, 0, 0, time.Local),
		GrossAmount: 5000.00,
		TaxAmount:   500.00,
		NetAmount:   4500.00,
	})

	result := GetYTDTax(db, []uint{account.ID})

	expected := 500.00
	if math.Abs(result-expected) > 0.01 {
		t.Errorf("GetYTDTax() = %v, want %v", result, expected)
	}
}

// TestGetYTDNetIncome_EmptyAccounts tests YTD net income with empty account list
func TestGetYTDNetIncome_EmptyAccounts(t *testing.T) {
	db := testutil.SetupTestDB()

	result := GetYTDNetIncome(db, []uint{})

	if result != 0 {
		t.Errorf("GetYTDNetIncome() with empty accounts = %v, want 0", result)
	}
}

// TestGetYTDNetIncome_WithData tests YTD net income calculation
func TestGetYTDNetIncome_WithData(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	now := time.Now()

	// Create income for current year - only use past dates
	db.Create(&models.Income{
		AccountID:   account.ID,
		Date:        time.Date(now.Year(), 1, 5, 0, 0, 0, 0, time.Local),
		GrossAmount: 5000.00,
		TaxAmount:   500.00,
		NetAmount:   4500.00,
	})

	result := GetYTDNetIncome(db, []uint{account.ID})

	expected := 4500.00
	if math.Abs(result-expected) > 0.01 {
		t.Errorf("GetYTDNetIncome() = %v, want %v", result, expected)
	}
}

// TestGetTaxProjection_EmptyAccounts tests projection with no accounts
func TestGetTaxProjection_EmptyAccounts(t *testing.T) {
	db := testutil.SetupTestDB()

	result := GetTaxProjection(db, []uint{}, nil)

	// Should return projection with zero values
	if result.YTDIncome != 0 {
		t.Errorf("YTDIncome = %v, want 0", result.YTDIncome)
	}

	if result.YTDTax != 0 {
		t.Errorf("YTDTax = %v, want 0", result.YTDTax)
	}

	// Year should be current year
	if result.Year != time.Now().Year() {
		t.Errorf("Year = %v, want %v", result.Year, time.Now().Year())
	}
}

// TestGetTaxProjection_WithINSSConfig tests projection with INSS configuration
func TestGetTaxProjection_WithINSSConfig(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	inssConfig := &INSSConfig{
		ProLabore: 5000,
		Ceiling:   7786.02,
		Rate:      0.11,
	}

	result := GetTaxProjection(db, []uint{account.ID}, inssConfig)

	// Should calculate INSS
	monthsElapsed := GetMonthsElapsed()
	expectedYTDINSS := float64(monthsElapsed) * 550.00 // 5000 * 0.11 per month

	if math.Abs(result.YTDINSS-expectedYTDINSS) > 0.01 {
		t.Errorf("YTDINSS = %v, want %v", result.YTDINSS, expectedYTDINSS)
	}

	// Projected annual INSS should be 12 months
	expectedAnnualINSS := 550.00 * 12
	if math.Abs(result.ProjectedAnnualINSS-expectedAnnualINSS) > 0.01 {
		t.Errorf("ProjectedAnnualINSS = %v, want %v", result.ProjectedAnnualINSS, expectedAnnualINSS)
	}
}

// TestGetTaxProjection_WithIncome tests projection with income data
func TestGetTaxProjection_WithIncome(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	now := time.Now()

	// Create incomes for current year
	for month := 1; month <= int(now.Month()); month++ {
		db.Create(&models.Income{
			AccountID:   account.ID,
			Date:        time.Date(now.Year(), time.Month(month), 15, 0, 0, 0, 0, time.Local),
			GrossAmount: 10000.00,
			TaxAmount:   600.00,
			NetAmount:   9400.00,
		})
	}

	result := GetTaxProjection(db, []uint{account.ID}, nil)

	// Verify YTD values
	monthsElapsed := int(now.Month())
	expectedYTDIncome := float64(monthsElapsed) * 10000.00

	if math.Abs(result.YTDIncome-expectedYTDIncome) > 0.01 {
		t.Errorf("YTDIncome = %v, want %v", result.YTDIncome, expectedYTDIncome)
	}

	// Projected annual income should extrapolate to 12 months
	expectedAnnualIncome := 10000.00 * 12
	if math.Abs(result.ProjectedAnnualIncome-expectedAnnualIncome) > 0.01 {
		t.Errorf("ProjectedAnnualIncome = %v, want %v", result.ProjectedAnnualIncome, expectedAnnualIncome)
	}

	// Should have bracket information
	if result.CurrentBracket < 1 || result.CurrentBracket > 6 {
		t.Errorf("CurrentBracket = %v, should be between 1 and 6", result.CurrentBracket)
	}

	// Should have bracket warning
	if result.BracketWarning == nil {
		t.Error("BracketWarning should not be nil")
	}
}

// TestGetTaxProjection_MetadataFields tests that metadata fields are populated
func TestGetTaxProjection_MetadataFields(t *testing.T) {
	db := testutil.SetupTestDB()

	result := GetTaxProjection(db, []uint{}, nil)

	// Check year
	if result.Year != time.Now().Year() {
		t.Errorf("Year = %v, want %v", result.Year, time.Now().Year())
	}

	// Check months elapsed
	if result.MonthsElapsed < 1 || result.MonthsElapsed > 12 {
		t.Errorf("MonthsElapsed = %v, should be between 1 and 12", result.MonthsElapsed)
	}

	// Check calculated at
	if result.CalculatedAt.IsZero() {
		t.Error("CalculatedAt should not be zero")
	}

	// CalculatedAt should be recent (within last minute)
	if time.Since(result.CalculatedAt) > time.Minute {
		t.Error("CalculatedAt should be recent")
	}
}

// TestGetTaxProjectionForYear_FutureYear tests projection for future year
func TestGetTaxProjectionForYear_FutureYear(t *testing.T) {
	db := testutil.SetupTestDB()

	futureYear := time.Now().Year() + 1
	result := GetTaxProjectionForYear(db, futureYear, []uint{}, nil)

	// Should return empty projection for future year
	if result.Year != futureYear {
		t.Errorf("Year = %v, want %v", result.Year, futureYear)
	}

	if result.YTDIncome != 0 {
		t.Errorf("YTDIncome for future year = %v, want 0", result.YTDIncome)
	}

	if result.MonthsElapsed != 0 {
		t.Errorf("MonthsElapsed for future year = %v, want 0", result.MonthsElapsed)
	}
}

// TestGetTaxProjectionForYear_PastYear tests projection for past year
func TestGetTaxProjectionForYear_PastYear(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	pastYear := time.Now().Year() - 1

	// Create income for past year
	for month := 1; month <= 12; month++ {
		db.Create(&models.Income{
			AccountID:   account.ID,
			Date:        time.Date(pastYear, time.Month(month), 15, 0, 0, 0, 0, time.Local),
			GrossAmount: 8000.00,
			TaxAmount:   480.00,
			NetAmount:   7520.00,
		})
	}

	result := GetTaxProjectionForYear(db, pastYear, []uint{account.ID}, nil)

	// Should have 12 months elapsed for past year
	if result.MonthsElapsed != 12 {
		t.Errorf("MonthsElapsed for past year = %v, want 12", result.MonthsElapsed)
	}

	// For past years, projection equals actual
	expectedIncome := 8000.00 * 12
	if math.Abs(result.YTDIncome-expectedIncome) > 0.01 {
		t.Errorf("YTDIncome = %v, want %v", result.YTDIncome, expectedIncome)
	}

	if math.Abs(result.ProjectedAnnualIncome-result.YTDIncome) > 0.01 {
		t.Errorf("ProjectedAnnualIncome = %v, should equal YTDIncome %v for past year",
			result.ProjectedAnnualIncome, result.YTDIncome)
	}
}

// TestGetTaxProjectionForYear_CurrentYear tests projection for current year
func TestGetTaxProjectionForYear_CurrentYear(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	currentYear := time.Now().Year()

	// Create income for current year up to current month
	for month := 1; month <= int(time.Now().Month()); month++ {
		db.Create(&models.Income{
			AccountID:   account.ID,
			Date:        time.Date(currentYear, time.Month(month), 15, 0, 0, 0, 0, time.Local),
			GrossAmount: 7000.00,
			TaxAmount:   420.00,
			NetAmount:   6580.00,
		})
	}

	result := GetTaxProjectionForYear(db, currentYear, []uint{account.ID}, nil)

	// Should have correct months elapsed
	expectedMonths := int(time.Now().Month())
	if result.MonthsElapsed != expectedMonths {
		t.Errorf("MonthsElapsed = %v, want %v", result.MonthsElapsed, expectedMonths)
	}

	// Should have bracket warning for current year
	if result.BracketWarning == nil {
		t.Error("BracketWarning should not be nil for current year")
	}
}

// TestGetMonthlyTaxBreakdown_EmptyAccounts tests monthly breakdown with no accounts
func TestGetMonthlyTaxBreakdown_EmptyAccounts(t *testing.T) {
	db := testutil.SetupTestDB()

	result := GetMonthlyTaxBreakdown(db, 2024, []uint{}, nil)

	// Should return 12 empty months
	if len(result) != 12 {
		t.Errorf("GetMonthlyTaxBreakdown() returned %d months, want 12", len(result))
	}

	// All months should have zero values
	for i, mb := range result {
		if mb.Month != i+1 {
			t.Errorf("Month %d: Month = %d, want %d", i, mb.Month, i+1)
		}

		if mb.GrossIncome != 0 {
			t.Errorf("Month %d: GrossIncome = %v, want 0", i+1, mb.GrossIncome)
		}
	}
}

// TestGetMonthlyTaxBreakdown_WithData tests monthly breakdown with income data
func TestGetMonthlyTaxBreakdown_WithData(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	// Create income for Q1 2024
	for month := 1; month <= 3; month++ {
		db.Create(&models.Income{
			AccountID:   account.ID,
			Date:        time.Date(2024, time.Month(month), 15, 0, 0, 0, 0, time.Local),
			GrossAmount: float64(5000 + month*1000),
			TaxAmount:   float64(300 + month*60),
			NetAmount:   float64(4700 + month*940),
		})
	}

	result := GetMonthlyTaxBreakdown(db, 2024, []uint{account.ID}, nil)

	// Verify Q1 months have data
	for month := 1; month <= 3; month++ {
		mb := result[month-1]
		expectedGross := float64(5000 + month*1000)

		if math.Abs(mb.GrossIncome-expectedGross) > 0.01 {
			t.Errorf("Month %d: GrossIncome = %v, want %v", month, mb.GrossIncome, expectedGross)
		}
	}

	// Verify Q2-Q4 months have no data
	for month := 4; month <= 12; month++ {
		mb := result[month-1]
		if mb.GrossIncome != 0 {
			t.Errorf("Month %d: GrossIncome = %v, want 0", month, mb.GrossIncome)
		}
	}
}

// TestGetMonthlyTaxBreakdown_WithINSS tests monthly breakdown includes INSS
func TestGetMonthlyTaxBreakdown_WithINSS(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	inssConfig := &INSSConfig{
		ProLabore: 5000,
		Ceiling:   7786.02,
		Rate:      0.11,
	}

	result := GetMonthlyTaxBreakdown(db, 2024, []uint{account.ID}, inssConfig)

	// All months should have INSS calculated
	expectedINSS := 5000 * 0.11
	for i, mb := range result {
		if math.Abs(mb.INSSPaid-expectedINSS) > 0.01 {
			t.Errorf("Month %d: INSSPaid = %v, want %v", i+1, mb.INSSPaid, expectedINSS)
		}
	}
}

// TestGetMonthlyTaxBreakdown_MonthNames tests that month names are correct
func TestGetMonthlyTaxBreakdown_MonthNames(t *testing.T) {
	db := testutil.SetupTestDB()

	result := GetMonthlyTaxBreakdown(db, 2024, []uint{}, nil)

	expectedNames := []string{
		"Janeiro", "Fevereiro", "Março", "Abril",
		"Maio", "Junho", "Julho", "Agosto",
		"Setembro", "Outubro", "Novembro", "Dezembro",
	}

	for i, mb := range result {
		if mb.MonthName != expectedNames[i] {
			t.Errorf("Month %d: MonthName = %s, want %s", i+1, mb.MonthName, expectedNames[i])
		}
	}
}

// TestGetMonthlyTaxBreakdown_MultipleAccounts tests breakdown with multiple accounts
func TestGetMonthlyTaxBreakdown_MultipleAccounts(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account1 := testutil.CreateTestAccount(db, "Account 1", models.AccountTypeIndividual, user.ID, nil)
	account2 := testutil.CreateTestAccount(db, "Account 2", models.AccountTypeIndividual, user.ID, nil)

	// Create income for both accounts in January 2024
	db.Create(&models.Income{
		AccountID:   account1.ID,
		Date:        time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 5000.00,
		TaxAmount:   500.00,
		NetAmount:   4500.00,
	})

	db.Create(&models.Income{
		AccountID:   account2.ID,
		Date:        time.Date(2024, 1, 20, 0, 0, 0, 0, time.Local),
		GrossAmount: 3000.00,
		TaxAmount:   300.00,
		NetAmount:   2700.00,
	})

	result := GetMonthlyTaxBreakdown(db, 2024, []uint{account1.ID, account2.ID}, nil)

	// January should have combined income
	january := result[0]
	expectedGross := 8000.00 // 5000 + 3000
	expectedTax := 800.00    // 500 + 300
	expectedNet := 7200.00   // 4500 + 2700

	if math.Abs(january.GrossIncome-expectedGross) > 0.01 {
		t.Errorf("January GrossIncome = %v, want %v", january.GrossIncome, expectedGross)
	}

	if math.Abs(january.TaxPaid-expectedTax) > 0.01 {
		t.Errorf("January TaxPaid = %v, want %v", january.TaxPaid, expectedTax)
	}

	if math.Abs(january.NetIncome-expectedNet) > 0.01 {
		t.Errorf("January NetIncome = %v, want %v", january.NetIncome, expectedNet)
	}
}

// TestBracketWarningThresholds tests that warning thresholds are correctly defined
func TestTaxProjectionBracketWarningThresholds(t *testing.T) {
	// Verify threshold constants are in ascending order
	if WarningThresholdLow >= WarningThresholdMedium {
		t.Error("WarningThresholdLow should be less than WarningThresholdMedium")
	}

	if WarningThresholdMedium >= WarningThresholdHigh {
		t.Error("WarningThresholdMedium should be less than WarningThresholdHigh")
	}

	if WarningThresholdHigh >= WarningThresholdCritical {
		t.Error("WarningThresholdHigh should be less than WarningThresholdCritical")
	}

	// Verify thresholds are within valid percentage range
	if WarningThresholdLow < 0 || WarningThresholdLow > 100 {
		t.Errorf("WarningThresholdLow = %v, should be between 0 and 100", WarningThresholdLow)
	}

	if WarningThresholdCritical < 0 || WarningThresholdCritical > 100 {
		t.Errorf("WarningThresholdCritical = %v, should be between 0 and 100", WarningThresholdCritical)
	}
}

// TestTaxProjectionStruct_Fields tests that TaxProjection struct has required fields
func TestTaxProjectionStruct_Fields(t *testing.T) {
	projection := TaxProjection{
		YTDIncome:             1000,
		YTDTax:                100,
		YTDINSS:               50,
		YTDNetIncome:          850,
		ProjectedAnnualIncome: 12000,
		ProjectedAnnualTax:    1200,
		ProjectedAnnualINSS:   600,
		ProjectedNetIncome:    10200,
		CurrentBracket:        1,
		EffectiveRate:         6.0,
		NextBracketAt:         180000,
		MonthsElapsed:         1,
		Year:                  2024,
		CalculatedAt:          time.Now(),
	}

	// Verify all fields are accessible
	if projection.YTDIncome != 1000 {
		t.Errorf("YTDIncome = %v, want 1000", projection.YTDIncome)
	}

	if projection.ProjectedNetIncome != 10200 {
		t.Errorf("ProjectedNetIncome = %v, want 10200", projection.ProjectedNetIncome)
	}
}

// TestBracketWarningStruct_Fields tests that BracketWarning struct has required fields
func TestBracketWarningStruct_Fields(t *testing.T) {
	warning := BracketWarning{
		IsApproaching:    true,
		AmountUntilNext:  30000,
		PercentToNext:    85.0,
		WarningLevel:     "medium",
		WarningMessage:   "Test message",
		NextBracketRate:  11.2,
		ProjectedBracket: 2,
	}

	// Verify all fields are accessible
	if !warning.IsApproaching {
		t.Error("IsApproaching should be true")
	}

	if warning.WarningLevel != "medium" {
		t.Errorf("WarningLevel = %v, want medium", warning.WarningLevel)
	}
}

// TestMonthlyTaxBreakdownStruct_Fields tests that MonthlyTaxBreakdown struct has required fields
func TestMonthlyTaxBreakdownStruct_Fields(t *testing.T) {
	breakdown := MonthlyTaxBreakdown{
		Month:       1,
		MonthName:   "Janeiro",
		GrossIncome: 5000,
		TaxPaid:     500,
		NetIncome:   4500,
		INSSPaid:    550,
	}

	// Verify all fields are accessible
	if breakdown.Month != 1 {
		t.Errorf("Month = %v, want 1", breakdown.Month)
	}

	if breakdown.MonthName != "Janeiro" {
		t.Errorf("MonthName = %v, want Janeiro", breakdown.MonthName)
	}

	if breakdown.INSSPaid != 550 {
		t.Errorf("INSSPaid = %v, want 550", breakdown.INSSPaid)
	}
}
