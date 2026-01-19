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
