package handlers

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
	"poc-finance/internal/services"
)

type SettingsHandler struct{
	cacheService *services.SettingsCacheService
}

func NewSettingsHandler(cacheService *services.SettingsCacheService) *SettingsHandler {
	return &SettingsHandler{
		cacheService: cacheService,
	}
}

type SettingsData struct {
	ProLabore   float64 `json:"pro_labore"`
	INSSCeiling float64 `json:"inss_ceiling"`
	INSSRate    float64 `json:"inss_rate"`
	INSSAmount  float64 `json:"inss_amount"` // Calculado
}

func (h *SettingsHandler) Get(c echo.Context) error {
	cachedData := h.cacheService.GetSettingsData()
	// Convert services.SettingsData to handlers.SettingsData
	data := SettingsData{
		ProLabore:   cachedData.ProLabore,
		INSSCeiling: cachedData.INSSCeiling,
		INSSRate:    cachedData.INSSRate,
		INSSAmount:  cachedData.INSSAmount,
	}
	return c.Render(http.StatusOK, "settings.html", map[string]interface{}{
		"settings": data,
	})
}

func (h *SettingsHandler) Update(c echo.Context) error {
	proLabore, _ := strconv.ParseFloat(c.FormValue("pro_labore"), 64)
	inssCeiling, _ := strconv.ParseFloat(c.FormValue("inss_ceiling"), 64)
	inssRate, _ := strconv.ParseFloat(c.FormValue("inss_rate"), 64)

	// Atualiza configurações
	updateSetting(models.SettingProLabore, strconv.FormatFloat(proLabore, 'f', 2, 64))
	updateSetting(models.SettingINSSCeiling, strconv.FormatFloat(inssCeiling, 'f', 2, 64))
	updateSetting(models.SettingINSSRate, strconv.FormatFloat(inssRate, 'f', 2, 64))

	// Invalidate cache to force refresh on next request
	h.cacheService.InvalidateCache()

	cachedData := h.cacheService.GetSettingsData()
	// Convert services.SettingsData to handlers.SettingsData
	data := SettingsData{
		ProLabore:   cachedData.ProLabore,
		INSSCeiling: cachedData.INSSCeiling,
		INSSRate:    cachedData.INSSRate,
		INSSAmount:  cachedData.INSSAmount,
	}
	return c.Render(http.StatusOK, "partials/settings-form.html", map[string]interface{}{
		"settings": data,
		"saved":    true,
	})
}

func GetSettingsData() SettingsData {
	data := SettingsData{
		ProLabore:   getSettingFloat(models.SettingProLabore),
		INSSCeiling: getSettingFloat(models.SettingINSSCeiling),
		INSSRate:    getSettingFloat(models.SettingINSSRate),
	}

	// Calcula INSS
	inssConfig := services.INSSConfig{
		ProLabore: data.ProLabore,
		Ceiling:   data.INSSCeiling,
		Rate:      data.INSSRate / 100, // Converte % para decimal
	}
	data.INSSAmount = services.CalculateINSS(inssConfig)

	return data
}

func getSettingFloat(key string) float64 {
	var setting models.Settings
	if err := database.DB.Where("key = ?", key).First(&setting).Error; err != nil {
		return 0
	}
	value, _ := strconv.ParseFloat(setting.Value, 64)
	return value
}

func updateSetting(key, value string) {
	var setting models.Settings
	result := database.DB.Where("key = ?", key).First(&setting)
	if result.Error != nil {
		database.DB.Create(&models.Settings{Key: key, Value: value})
	} else {
		setting.Value = value
		database.DB.Save(&setting)
	}
}
