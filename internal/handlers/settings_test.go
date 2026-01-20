package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
	"poc-finance/internal/services"
	"poc-finance/internal/testutil"
)

func setupSettingsTestHandler() (*SettingsHandler, *echo.Echo) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create initial settings in the database
	database.DB.Create(&models.Settings{Key: models.SettingProLabore, Value: "5000.00"})
	database.DB.Create(&models.Settings{Key: models.SettingINSSCeiling, Value: "7507.49"})
	database.DB.Create(&models.Settings{Key: models.SettingINSSRate, Value: "11.00"})

	cacheService := services.NewSettingsCacheService()
	handler := NewSettingsHandler(cacheService)

	e := echo.New()
	e.Renderer = &testutil.MockRenderer{}

	return handler, e
}

func TestSettingsHandler_Get_Success(t *testing.T) {
	handler, e := setupSettingsTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Get(c)
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestSettingsHandler_Get_ReturnsCorrectValues(t *testing.T) {
	handler, e := setupSettingsTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Get(c)
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}

	// Verify the cached data is returned with correct values
	cachedData := handler.cacheService.GetSettingsData()

	if cachedData.ProLabore != 5000.00 {
		t.Errorf("ProLabore = %f, want %f", cachedData.ProLabore, 5000.00)
	}

	if cachedData.INSSCeiling != 7507.49 {
		t.Errorf("INSSCeiling = %f, want %f", cachedData.INSSCeiling, 7507.49)
	}

	if cachedData.INSSRate != 11.00 {
		t.Errorf("INSSRate = %f, want %f", cachedData.INSSRate, 11.00)
	}

	// INSS amount should be calculated (5000.00 * 0.11 = 550.00)
	expectedINSS := 550.00
	if cachedData.INSSAmount != expectedINSS {
		t.Errorf("INSSAmount = %f, want %f", cachedData.INSSAmount, expectedINSS)
	}
}

func TestSettingsHandler_Update_Success(t *testing.T) {
	handler, e := setupSettingsTestHandler()

	form := url.Values{}
	form.Set("pro_labore", "6000.00")
	form.Set("inss_ceiling", "8000.00")
	form.Set("inss_rate", "12.00")

	req := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Update(c)
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Verify settings were updated in database
	var proLaboreSetting models.Settings
	database.DB.Where("key = ?", models.SettingProLabore).First(&proLaboreSetting)
	if proLaboreSetting.Value != "6000.00" {
		t.Errorf("ProLabore in DB = %s, want %s", proLaboreSetting.Value, "6000.00")
	}

	var inssCeilingSetting models.Settings
	database.DB.Where("key = ?", models.SettingINSSCeiling).First(&inssCeilingSetting)
	if inssCeilingSetting.Value != "8000.00" {
		t.Errorf("INSSCeiling in DB = %s, want %s", inssCeilingSetting.Value, "8000.00")
	}

	var inssRateSetting models.Settings
	database.DB.Where("key = ?", models.SettingINSSRate).First(&inssRateSetting)
	if inssRateSetting.Value != "12.00" {
		t.Errorf("INSSRate in DB = %s, want %s", inssRateSetting.Value, "12.00")
	}
}

func TestSettingsHandler_Update_CacheInvalidation(t *testing.T) {
	handler, e := setupSettingsTestHandler()

	// First, get the settings to populate cache
	handler.cacheService.GetSettingsData()

	// Update settings
	form := url.Values{}
	form.Set("pro_labore", "7000.00")
	form.Set("inss_ceiling", "9000.00")
	form.Set("inss_rate", "13.00")

	req := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Update(c)
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	// Get cached data - should reflect new values
	cachedData := handler.cacheService.GetSettingsData()

	if cachedData.ProLabore != 7000.00 {
		t.Errorf("ProLabore after update = %f, want %f", cachedData.ProLabore, 7000.00)
	}

	if cachedData.INSSCeiling != 9000.00 {
		t.Errorf("INSSCeiling after update = %f, want %f", cachedData.INSSCeiling, 9000.00)
	}

	if cachedData.INSSRate != 13.00 {
		t.Errorf("INSSRate after update = %f, want %f", cachedData.INSSRate, 13.00)
	}

	// INSS amount should be recalculated (7000.00 * 0.13 = 910.00)
	expectedINSS := 910.00
	if cachedData.INSSAmount != expectedINSS {
		t.Errorf("INSSAmount after update = %f, want %f", cachedData.INSSAmount, expectedINSS)
	}
}

func TestSettingsHandler_Update_WithZeroValues(t *testing.T) {
	handler, e := setupSettingsTestHandler()

	form := url.Values{}
	form.Set("pro_labore", "0")
	form.Set("inss_ceiling", "0")
	form.Set("inss_rate", "0")

	req := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Update(c)
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	// Verify zero values were saved
	var proLaboreSetting models.Settings
	database.DB.Where("key = ?", models.SettingProLabore).First(&proLaboreSetting)
	if proLaboreSetting.Value != "0.00" {
		t.Errorf("ProLabore in DB = %s, want %s", proLaboreSetting.Value, "0.00")
	}

	// INSS amount should be zero when pro_labore is zero
	cachedData := handler.cacheService.GetSettingsData()
	if cachedData.INSSAmount != 0 {
		t.Errorf("INSSAmount with zero pro_labore = %f, want %f", cachedData.INSSAmount, 0.0)
	}
}

func TestSettingsHandler_Update_WithDecimalValues(t *testing.T) {
	handler, e := setupSettingsTestHandler()

	form := url.Values{}
	form.Set("pro_labore", "5500.75")
	form.Set("inss_ceiling", "7507.49")
	form.Set("inss_rate", "11.50")

	req := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Update(c)
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	// Verify decimal values were saved correctly
	var proLaboreSetting models.Settings
	database.DB.Where("key = ?", models.SettingProLabore).First(&proLaboreSetting)
	if proLaboreSetting.Value != "5500.75" {
		t.Errorf("ProLabore in DB = %s, want %s", proLaboreSetting.Value, "5500.75")
	}

	var inssRateSetting models.Settings
	database.DB.Where("key = ?", models.SettingINSSRate).First(&inssRateSetting)
	if inssRateSetting.Value != "11.50" {
		t.Errorf("INSSRate in DB = %s, want %s", inssRateSetting.Value, "11.50")
	}
}

func TestSettingsHandler_Update_WithInvalidFormValues(t *testing.T) {
	handler, e := setupSettingsTestHandler()

	form := url.Values{}
	form.Set("pro_labore", "invalid")
	form.Set("inss_ceiling", "also_invalid")
	form.Set("inss_rate", "not_a_number")

	req := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Update(c)
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	// Invalid values should be parsed as 0
	var proLaboreSetting models.Settings
	database.DB.Where("key = ?", models.SettingProLabore).First(&proLaboreSetting)
	if proLaboreSetting.Value != "0.00" {
		t.Errorf("ProLabore with invalid input in DB = %s, want %s", proLaboreSetting.Value, "0.00")
	}
}

func TestSettingsHandler_Update_EmptyFormValues(t *testing.T) {
	handler, e := setupSettingsTestHandler()

	form := url.Values{}
	// Don't set any values

	req := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Update(c)
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	// Empty values should be parsed as 0
	var proLaboreSetting models.Settings
	database.DB.Where("key = ?", models.SettingProLabore).First(&proLaboreSetting)
	if proLaboreSetting.Value != "0.00" {
		t.Errorf("ProLabore with empty input in DB = %s, want %s", proLaboreSetting.Value, "0.00")
	}
}

func TestSettingsHandler_Update_CreatesSettingIfNotExists(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Don't create initial settings - test that Update creates them
	cacheService := services.NewSettingsCacheService()
	handler := NewSettingsHandler(cacheService)

	e := echo.New()
	e.Renderer = &testutil.MockRenderer{}

	form := url.Values{}
	form.Set("pro_labore", "4500.00")
	form.Set("inss_ceiling", "7000.00")
	form.Set("inss_rate", "10.00")

	req := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Update(c)
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	// Verify settings were created
	var proLaboreSetting models.Settings
	result := database.DB.Where("key = ?", models.SettingProLabore).First(&proLaboreSetting)
	if result.Error != nil {
		t.Fatalf("Failed to find created ProLabore setting: %v", result.Error)
	}
	if proLaboreSetting.Value != "4500.00" {
		t.Errorf("ProLabore in DB = %s, want %s", proLaboreSetting.Value, "4500.00")
	}
}

func TestSettingsHandler_Update_INSSCalculation(t *testing.T) {
	handler, e := setupSettingsTestHandler()

	tests := []struct {
		name             string
		proLabore        string
		inssRate         string
		expectedINSS     float64
		expectedCeiling  float64
	}{
		{
			name:            "normal calculation",
			proLabore:       "5000.00",
			inssRate:        "11.00",
			expectedINSS:    550.00, // 5000 * 0.11
			expectedCeiling: 7507.49,
		},
		{
			name:            "above ceiling",
			proLabore:       "10000.00",
			inssRate:        "11.00",
			expectedINSS:    825.8239, // ceiling 7507.49 * 0.11
			expectedCeiling: 7507.49,
		},
		{
			name:            "at ceiling",
			proLabore:       "7507.49",
			inssRate:        "11.00",
			expectedINSS:    825.8239, // 7507.49 * 0.11
			expectedCeiling: 7507.49,
		},
		{
			name:            "below ceiling",
			proLabore:       "3000.00",
			inssRate:        "11.00",
			expectedINSS:    330.00, // 3000 * 0.11
			expectedCeiling: 7507.49,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{}
			form.Set("pro_labore", tt.proLabore)
			form.Set("inss_ceiling", "7507.49")
			form.Set("inss_rate", tt.inssRate)

			req := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(form.Encode()))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.Update(c)
			if err != nil {
				t.Fatalf("Update() returned error: %v", err)
			}

			// Get cached data and verify INSS calculation
			cachedData := handler.cacheService.GetSettingsData()

			if cachedData.INSSAmount != tt.expectedINSS {
				t.Errorf("INSSAmount = %f, want %f", cachedData.INSSAmount, tt.expectedINSS)
			}

			if cachedData.INSSCeiling != tt.expectedCeiling {
				t.Errorf("INSSCeiling = %f, want %f", cachedData.INSSCeiling, tt.expectedCeiling)
			}
		})
	}
}

func TestSettingsHandler_Update_MultipleUpdates(t *testing.T) {
	handler, e := setupSettingsTestHandler()

	// First update
	form1 := url.Values{}
	form1.Set("pro_labore", "5000.00")
	form1.Set("inss_ceiling", "7507.49")
	form1.Set("inss_rate", "11.00")

	req1 := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(form1.Encode()))
	req1.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec1 := httptest.NewRecorder()
	c1 := e.NewContext(req1, rec1)

	err := handler.Update(c1)
	if err != nil {
		t.Fatalf("First Update() returned error: %v", err)
	}

	// Second update
	form2 := url.Values{}
	form2.Set("pro_labore", "6000.00")
	form2.Set("inss_ceiling", "8000.00")
	form2.Set("inss_rate", "12.00")

	req2 := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(form2.Encode()))
	req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)

	err = handler.Update(c2)
	if err != nil {
		t.Fatalf("Second Update() returned error: %v", err)
	}

	// Verify final values
	var proLaboreSetting models.Settings
	database.DB.Where("key = ?", models.SettingProLabore).First(&proLaboreSetting)
	if proLaboreSetting.Value != "6000.00" {
		t.Errorf("ProLabore after multiple updates = %s, want %s", proLaboreSetting.Value, "6000.00")
	}

	cachedData := handler.cacheService.GetSettingsData()
	if cachedData.ProLabore != 6000.00 {
		t.Errorf("Cached ProLabore after multiple updates = %f, want %f", cachedData.ProLabore, 6000.00)
	}

	// INSS should reflect the latest values
	expectedINSS := 720.00 // 6000 * 0.12
	if cachedData.INSSAmount != expectedINSS {
		t.Errorf("INSSAmount after multiple updates = %f, want %f", cachedData.INSSAmount, expectedINSS)
	}
}

func TestUpdateSetting_CreatesIfNotExists(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Update a setting that doesn't exist
	updateSetting("test_key", "test_value")

	// Verify it was created
	var setting models.Settings
	result := database.DB.Where("key = ?", "test_key").First(&setting)
	if result.Error != nil {
		t.Fatalf("Failed to find created setting: %v", result.Error)
	}

	if setting.Value != "test_value" {
		t.Errorf("Setting value = %s, want %s", setting.Value, "test_value")
	}
}

func TestUpdateSetting_UpdatesIfExists(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create initial setting
	database.DB.Create(&models.Settings{Key: "test_key", Value: "initial_value"})

	// Update the setting
	updateSetting("test_key", "updated_value")

	// Verify it was updated
	var setting models.Settings
	database.DB.Where("key = ?", "test_key").First(&setting)

	if setting.Value != "updated_value" {
		t.Errorf("Setting value = %s, want %s", setting.Value, "updated_value")
	}

	// Verify only one record exists
	var count int64
	database.DB.Model(&models.Settings{}).Where("key = ?", "test_key").Count(&count)
	if count != 1 {
		t.Errorf("Setting count = %d, want %d", count, 1)
	}
}
