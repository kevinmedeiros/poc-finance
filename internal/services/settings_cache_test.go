package services

import (
	"sync"
	"testing"
	"time"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
	"poc-finance/internal/testutil"
)

// createTestSettings creates test settings in the database
func createTestSettings(t *testing.T) {
	settings := []models.Settings{
		{Key: models.SettingProLabore, Value: "5000.00"},
		{Key: models.SettingINSSCeiling, Value: "7786.02"},
		{Key: models.SettingINSSRate, Value: "11.00"},
		{Key: models.SettingBudgetWarningThreshold, Value: "80.00"},
	}

	for _, setting := range settings {
		if err := database.DB.Create(&setting).Error; err != nil {
			t.Fatalf("Failed to create test setting %s: %v", setting.Key, err)
		}
	}
}

func TestNewSettingsCacheService(t *testing.T) {
	service := NewSettingsCacheService()

	if service == nil {
		t.Fatal("NewSettingsCacheService() returned nil")
	}

	if service.ttl != 5*time.Minute {
		t.Errorf("TTL = %v, want %v", service.ttl, 5*time.Minute)
	}

	if !service.lastFetch.IsZero() {
		t.Error("lastFetch should be zero value for new service")
	}
}

func TestSettingsCacheService_GetSettingsData_InitialFetch(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	createTestSettings(t)

	service := NewSettingsCacheService()

	// First call should fetch from database
	data := service.GetSettingsData()

	if data.ProLabore != 5000.00 {
		t.Errorf("ProLabore = %v, want %v", data.ProLabore, 5000.00)
	}

	if data.INSSCeiling != 7786.02 {
		t.Errorf("INSSCeiling = %v, want %v", data.INSSCeiling, 7786.02)
	}

	if data.INSSRate != 11.00 {
		t.Errorf("INSSRate = %v, want %v", data.INSSRate, 11.00)
	}

	// Verify INSS amount is calculated correctly
	expectedINSS := 5000.00 * 0.11 // ProLabore * (Rate/100)
	if data.INSSAmount != expectedINSS {
		t.Errorf("INSSAmount = %v, want %v", data.INSSAmount, expectedINSS)
	}

	// Verify lastFetch was set
	if service.lastFetch.IsZero() {
		t.Error("lastFetch should be set after initial fetch")
	}
}

func TestSettingsCacheService_GetSettingsData_CacheHit(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	createTestSettings(t)

	service := NewSettingsCacheService()

	// First call - fetch from database
	data1 := service.GetSettingsData()
	firstFetch := service.lastFetch

	// Immediate second call - should return cached data
	time.Sleep(10 * time.Millisecond) // Small delay to ensure time passes
	data2 := service.GetSettingsData()

	// Data should be identical
	if data1 != data2 {
		t.Error("Cache hit should return identical data")
	}

	// lastFetch should not change (cache was used)
	if service.lastFetch != firstFetch {
		t.Error("lastFetch should not change on cache hit")
	}
}

func TestSettingsCacheService_GetSettingsData_CacheExpired(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	createTestSettings(t)

	service := NewSettingsCacheService()
	service.ttl = 50 * time.Millisecond // Short TTL for testing

	// First call - fetch from database
	data1 := service.GetSettingsData()
	firstFetch := service.lastFetch

	// Wait for cache to expire
	time.Sleep(100 * time.Millisecond)

	// Update settings in database
	database.DB.Model(&models.Settings{}).
		Where("key = ?", models.SettingProLabore).
		Update("value", "6000.00")

	// Second call after expiration - should fetch fresh data
	data2 := service.GetSettingsData()

	// Data should be different (new value from DB)
	if data2.ProLabore == data1.ProLabore {
		t.Error("Expired cache should fetch fresh data from database")
	}

	if data2.ProLabore != 6000.00 {
		t.Errorf("ProLabore = %v, want %v", data2.ProLabore, 6000.00)
	}

	// lastFetch should be updated
	if !service.lastFetch.After(firstFetch) {
		t.Error("lastFetch should be updated after cache expiration")
	}
}

func TestSettingsCacheService_InvalidateCache(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	createTestSettings(t)

	service := NewSettingsCacheService()

	// Fetch initial data
	data1 := service.GetSettingsData()

	// Verify cache is populated
	if service.lastFetch.IsZero() {
		t.Fatal("lastFetch should be set after initial fetch")
	}

	// Update settings in database
	database.DB.Model(&models.Settings{}).
		Where("key = ?", models.SettingProLabore).
		Update("value", "7000.00")

	// Invalidate cache
	service.InvalidateCache()

	// Verify lastFetch was reset
	if !service.lastFetch.IsZero() {
		t.Error("InvalidateCache() should reset lastFetch to zero")
	}

	// Next fetch should get fresh data
	data2 := service.GetSettingsData()

	if data2.ProLabore == data1.ProLabore {
		t.Error("After invalidation, should fetch fresh data from database")
	}

	if data2.ProLabore != 7000.00 {
		t.Errorf("ProLabore = %v, want %v", data2.ProLabore, 7000.00)
	}
}

func TestSettingsCacheService_GetSettingsData_EmptyDatabase(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Don't create any settings - database is empty

	service := NewSettingsCacheService()
	data := service.GetSettingsData()

	// Should return zero values
	if data.ProLabore != 0 {
		t.Errorf("ProLabore = %v, want 0", data.ProLabore)
	}

	if data.INSSCeiling != 0 {
		t.Errorf("INSSCeiling = %v, want 0", data.INSSCeiling)
	}

	if data.INSSRate != 0 {
		t.Errorf("INSSRate = %v, want 0", data.INSSRate)
	}

	if data.INSSAmount != 0 {
		t.Errorf("INSSAmount = %v, want 0", data.INSSAmount)
	}
}

func TestSettingsCacheService_GetSettingsData_INSSCalculation(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	tests := []struct {
		name             string
		proLabore        string
		inssCeiling      string
		inssRate         string
		expectedINSS     float64
		description      string
	}{
		{
			name:         "normal case",
			proLabore:    "5000.00",
			inssCeiling:  "7786.02",
			inssRate:     "11.00",
			expectedINSS: 550.00, // 5000 * 0.11
			description:  "ProLabore below ceiling",
		},
		{
			name:         "at ceiling",
			proLabore:    "7786.02",
			inssCeiling:  "7786.02",
			inssRate:     "11.00",
			expectedINSS: 856.46, // 7786.02 * 0.11 (rounded)
			description:  "ProLabore equals ceiling",
		},
		{
			name:         "above ceiling",
			proLabore:    "10000.00",
			inssCeiling:  "7786.02",
			inssRate:     "11.00",
			expectedINSS: 856.46, // 7786.02 * 0.11 (capped at ceiling)
			description:  "ProLabore above ceiling should be capped",
		},
		{
			name:         "zero pro-labore",
			proLabore:    "0",
			inssCeiling:  "7786.02",
			inssRate:     "11.00",
			expectedINSS: 0,
			description:  "Zero ProLabore should result in zero INSS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear and recreate settings for each test
			database.DB.Exec("DELETE FROM settings")

			settings := []models.Settings{
				{Key: models.SettingProLabore, Value: tt.proLabore},
				{Key: models.SettingINSSCeiling, Value: tt.inssCeiling},
				{Key: models.SettingINSSRate, Value: tt.inssRate},
			}

			for _, setting := range settings {
				database.DB.Create(&setting)
			}

			service := NewSettingsCacheService()
			data := service.GetSettingsData()

			// Allow small floating point differences
			diff := data.INSSAmount - tt.expectedINSS
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.02 { // Allow 2 cents difference for floating point
				t.Errorf("%s: INSSAmount = %v, want %v", tt.description, data.INSSAmount, tt.expectedINSS)
			}
		})
	}
}

func TestSettingsCacheService_GetSettingsData_ConcurrentAccess(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	createTestSettings(t)

	service := NewSettingsCacheService()
	service.ttl = 100 * time.Millisecond

	// Launch multiple goroutines to access cache concurrently
	var wg sync.WaitGroup
	results := make([]SettingsData, 20)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index] = service.GetSettingsData()
		}(i)
	}

	wg.Wait()

	// All results should be identical (same cached data)
	firstResult := results[0]
	for i, result := range results {
		if result != firstResult {
			t.Errorf("Result %d differs from first result", i)
		}
	}
}

func TestSettingsCacheService_GetSettingsData_ConcurrentInvalidation(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	createTestSettings(t)

	service := NewSettingsCacheService()

	// Fetch initial data to populate cache
	service.GetSettingsData()

	// Launch concurrent invalidations and fetches
	var wg sync.WaitGroup

	// 10 goroutines invalidating
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			service.InvalidateCache()
		}()
	}

	// 10 goroutines fetching
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			service.GetSettingsData()
		}()
	}

	wg.Wait()

	// Should complete without deadlock or panic
	// Final fetch should succeed
	data := service.GetSettingsData()
	if data.ProLabore != 5000.00 {
		t.Errorf("After concurrent operations, ProLabore = %v, want %v", data.ProLabore, 5000.00)
	}
}

func TestSettingsCacheService_DoubleCheckedLocking(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	createTestSettings(t)

	service := NewSettingsCacheService()
	service.ttl = 50 * time.Millisecond

	// Fetch initial data
	service.GetSettingsData()

	// Wait for cache to expire
	time.Sleep(100 * time.Millisecond)

	// Launch multiple goroutines that will all see expired cache
	var wg sync.WaitGroup
	fetchCount := 0
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// All goroutines check for expiration
			service.mu.RLock()
			expired := time.Since(service.lastFetch) >= service.ttl
			service.mu.RUnlock()

			if expired {
				mu.Lock()
				fetchCount++
				mu.Unlock()
			}

			// Fetch data (double-checked locking should prevent multiple fetches)
			service.GetSettingsData()
		}()
	}

	wg.Wait()

	// Multiple goroutines saw expiration, but double-checked locking
	// should have prevented multiple fetches
	if fetchCount == 0 {
		t.Error("Expected at least some goroutines to see expired cache")
	}

	// The service should still have valid data
	data := service.GetSettingsData()
	if data.ProLabore != 5000.00 {
		t.Error("Data should be valid after concurrent expiration handling")
	}
}

func TestGetSettingFloat(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	tests := []struct {
		name     string
		key      string
		value    string
		expected float64
	}{
		{
			name:     "valid float",
			key:      "test_key_1",
			value:    "123.45",
			expected: 123.45,
		},
		{
			name:     "integer value",
			key:      "test_key_2",
			value:    "100",
			expected: 100.0,
		},
		{
			name:     "zero value",
			key:      "test_key_3",
			value:    "0",
			expected: 0.0,
		},
		{
			name:     "negative value",
			key:      "test_key_4",
			value:    "-50.25",
			expected: -50.25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create setting
			setting := models.Settings{
				Key:   tt.key,
				Value: tt.value,
			}
			database.DB.Create(&setting)

			// Get value
			result := getSettingFloat(tt.key)

			if result != tt.expected {
				t.Errorf("getSettingFloat(%s) = %v, want %v", tt.key, result, tt.expected)
			}
		})
	}
}

func TestGetSettingFloat_NotFound(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	result := getSettingFloat("nonexistent_key")

	if result != 0 {
		t.Errorf("getSettingFloat(nonexistent) = %v, want 0", result)
	}
}

func TestGetSettingFloat_InvalidValue(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create setting with invalid float value
	setting := models.Settings{
		Key:   "invalid_float",
		Value: "not-a-number",
	}
	database.DB.Create(&setting)

	result := getSettingFloat("invalid_float")

	// Should return 0 for invalid values
	if result != 0 {
		t.Errorf("getSettingFloat(invalid) = %v, want 0", result)
	}
}

func TestSettingsCacheService_BudgetThreshold_FetchFromDatabase(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	createTestSettings(t)

	service := NewSettingsCacheService()

	// Fetch settings data
	data := service.GetSettingsData()

	// Verify budget threshold is loaded correctly
	if data.BudgetWarningThreshold != 80.00 {
		t.Errorf("BudgetWarningThreshold = %v, want %v", data.BudgetWarningThreshold, 80.00)
	}
}

func TestSettingsCacheService_BudgetThreshold_DefaultValue(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create settings without budget threshold
	settings := []models.Settings{
		{Key: models.SettingProLabore, Value: "5000.00"},
		{Key: models.SettingINSSCeiling, Value: "7786.02"},
		{Key: models.SettingINSSRate, Value: "11.00"},
		// Intentionally omitting SettingBudgetWarningThreshold
	}

	for _, setting := range settings {
		database.DB.Create(&setting)
	}

	service := NewSettingsCacheService()

	// Fetch settings data
	data := service.GetSettingsData()

	// Should default to 100
	if data.BudgetWarningThreshold != 100.00 {
		t.Errorf("BudgetWarningThreshold = %v, want %v (default)", data.BudgetWarningThreshold, 100.00)
	}
}

func TestSettingsCacheService_BudgetThreshold_ZeroDefaultsTo100(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create settings with zero budget threshold
	settings := []models.Settings{
		{Key: models.SettingProLabore, Value: "5000.00"},
		{Key: models.SettingINSSCeiling, Value: "7786.02"},
		{Key: models.SettingINSSRate, Value: "11.00"},
		{Key: models.SettingBudgetWarningThreshold, Value: "0"},
	}

	for _, setting := range settings {
		database.DB.Create(&setting)
	}

	service := NewSettingsCacheService()

	// Fetch settings data
	data := service.GetSettingsData()

	// Zero should default to 100
	if data.BudgetWarningThreshold != 100.00 {
		t.Errorf("BudgetWarningThreshold = %v, want %v (zero should default to 100)", data.BudgetWarningThreshold, 100.00)
	}
}

func TestSettingsCacheService_BudgetThreshold_VariousValues(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	tests := []struct {
		name      string
		value     string
		expected  float64
		description string
	}{
		{
			name:        "50 percent",
			value:       "50.00",
			expected:    50.00,
			description: "50% threshold",
		},
		{
			name:        "75 percent",
			value:       "75.00",
			expected:    75.00,
			description: "75% threshold",
		},
		{
			name:        "90 percent",
			value:       "90.00",
			expected:    90.00,
			description: "90% threshold",
		},
		{
			name:        "100 percent",
			value:       "100.00",
			expected:    100.00,
			description: "100% threshold (no warning)",
		},
		{
			name:        "decimal value",
			value:       "85.50",
			expected:    85.50,
			description: "Threshold with decimal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear and recreate settings for each test
			database.DB.Exec("DELETE FROM settings")

			settings := []models.Settings{
				{Key: models.SettingProLabore, Value: "5000.00"},
				{Key: models.SettingINSSCeiling, Value: "7786.02"},
				{Key: models.SettingINSSRate, Value: "11.00"},
				{Key: models.SettingBudgetWarningThreshold, Value: tt.value},
			}

			for _, setting := range settings {
				database.DB.Create(&setting)
			}

			service := NewSettingsCacheService()
			data := service.GetSettingsData()

			if data.BudgetWarningThreshold != tt.expected {
				t.Errorf("%s: BudgetWarningThreshold = %v, want %v", tt.description, data.BudgetWarningThreshold, tt.expected)
			}
		})
	}
}

func TestSettingsCacheService_BudgetThreshold_CacheInvalidation(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	createTestSettings(t)

	service := NewSettingsCacheService()

	// Fetch initial data
	data1 := service.GetSettingsData()

	if data1.BudgetWarningThreshold != 80.00 {
		t.Fatalf("Initial BudgetWarningThreshold = %v, want %v", data1.BudgetWarningThreshold, 80.00)
	}

	// Update budget threshold in database
	database.DB.Model(&models.Settings{}).
		Where("key = ?", models.SettingBudgetWarningThreshold).
		Update("value", "70.00")

	// Invalidate cache
	service.InvalidateCache()

	// Fetch fresh data
	data2 := service.GetSettingsData()

	// Should have new value
	if data2.BudgetWarningThreshold != 70.00 {
		t.Errorf("After invalidation, BudgetWarningThreshold = %v, want %v", data2.BudgetWarningThreshold, 70.00)
	}

	// Should be different from initial
	if data2.BudgetWarningThreshold == data1.BudgetWarningThreshold {
		t.Error("After invalidation, should fetch fresh budget threshold from database")
	}
}

func TestSettingsCacheService_BudgetThreshold_CacheExpiration(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	createTestSettings(t)

	service := NewSettingsCacheService()
	service.ttl = 50 * time.Millisecond // Short TTL for testing

	// First call - fetch from database
	data1 := service.GetSettingsData()

	if data1.BudgetWarningThreshold != 80.00 {
		t.Fatalf("Initial BudgetWarningThreshold = %v, want %v", data1.BudgetWarningThreshold, 80.00)
	}

	// Wait for cache to expire
	time.Sleep(100 * time.Millisecond)

	// Update budget threshold in database
	database.DB.Model(&models.Settings{}).
		Where("key = ?", models.SettingBudgetWarningThreshold).
		Update("value", "65.00")

	// Second call after expiration - should fetch fresh data
	data2 := service.GetSettingsData()

	// Data should be different (new value from DB)
	if data2.BudgetWarningThreshold == data1.BudgetWarningThreshold {
		t.Error("Expired cache should fetch fresh budget threshold from database")
	}

	if data2.BudgetWarningThreshold != 65.00 {
		t.Errorf("After cache expiration, BudgetWarningThreshold = %v, want %v", data2.BudgetWarningThreshold, 65.00)
	}
}

func TestSettingsCacheService_BudgetThreshold_AllFieldsPresent(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	createTestSettings(t)

	service := NewSettingsCacheService()
	data := service.GetSettingsData()

	// Verify all fields are populated correctly
	if data.ProLabore != 5000.00 {
		t.Errorf("ProLabore = %v, want %v", data.ProLabore, 5000.00)
	}

	if data.INSSCeiling != 7786.02 {
		t.Errorf("INSSCeiling = %v, want %v", data.INSSCeiling, 7786.02)
	}

	if data.INSSRate != 11.00 {
		t.Errorf("INSSRate = %v, want %v", data.INSSRate, 11.00)
	}

	if data.BudgetWarningThreshold != 80.00 {
		t.Errorf("BudgetWarningThreshold = %v, want %v", data.BudgetWarningThreshold, 80.00)
	}

	// Verify INSS amount is calculated correctly
	expectedINSS := 5000.00 * 0.11
	if data.INSSAmount != expectedINSS {
		t.Errorf("INSSAmount = %v, want %v", data.INSSAmount, expectedINSS)
	}
}
