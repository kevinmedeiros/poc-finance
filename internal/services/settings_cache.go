package services

import (
	"log"
	"strconv"
	"sync"
	"time"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
)

// SettingsData represents cached application settings
type SettingsData struct {
	ProLabore   float64 `json:"pro_labore"`
	INSSCeiling float64 `json:"inss_ceiling"`
	INSSRate    float64 `json:"inss_rate"`
	INSSAmount  float64 `json:"inss_amount"` // Calculated value
}

// SettingsCacheService provides thread-safe caching for application settings with TTL-based expiration
type SettingsCacheService struct {
	cachedData SettingsData
	lastFetch  time.Time
	ttl        time.Duration
	mu         sync.RWMutex
}

// NewSettingsCacheService creates a new settings cache service with 5-minute TTL
func NewSettingsCacheService() *SettingsCacheService {
	return &SettingsCacheService{
		ttl: 5 * time.Minute,
	}
}

// GetSettingsData returns cached settings data or fetches from database if cache is expired
func (s *SettingsCacheService) GetSettingsData() SettingsData {
	// Try to read from cache first (read lock)
	s.mu.RLock()
	if time.Since(s.lastFetch) < s.ttl && !s.lastFetch.IsZero() {
		// Cache hit - return cached data
		age := time.Since(s.lastFetch)
		log.Printf("[SettingsCache] CACHE HIT - Serving cached data (age: %v, TTL: %v)", age.Round(time.Second), s.ttl)
		data := s.cachedData
		s.mu.RUnlock()
		return data
	}
	s.mu.RUnlock()

	// Cache miss or expired - acquire write lock to refresh
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring write lock (another goroutine might have refreshed)
	if time.Since(s.lastFetch) < s.ttl && !s.lastFetch.IsZero() {
		log.Printf("[SettingsCache] CACHE HIT (double-check) - Another goroutine refreshed cache")
		return s.cachedData
	}

	// Fetch fresh data from database
	if s.lastFetch.IsZero() {
		log.Printf("[SettingsCache] CACHE MISS - Initial fetch from database")
	} else {
		log.Printf("[SettingsCache] CACHE EXPIRED - Refreshing from database (last fetch: %v ago)", time.Since(s.lastFetch).Round(time.Second))
	}
	s.cachedData = s.fetchSettingsFromDB()
	s.lastFetch = time.Now()
	log.Printf("[SettingsCache] Cache refreshed - ProLabore: %.2f, INSS: %.2f", s.cachedData.ProLabore, s.cachedData.INSSAmount)

	return s.cachedData
}

// InvalidateCache forces a cache refresh on the next GetSettingsData call
func (s *SettingsCacheService) InvalidateCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	log.Printf("[SettingsCache] CACHE INVALIDATED - Next request will fetch from database")
	s.lastFetch = time.Time{} // Reset to zero value to force refresh
}

// fetchSettingsFromDB queries the database for all settings and calculates derived values
func (s *SettingsCacheService) fetchSettingsFromDB() SettingsData {
	data := SettingsData{
		ProLabore:   getSettingFloat(models.SettingProLabore),
		INSSCeiling: getSettingFloat(models.SettingINSSCeiling),
		INSSRate:    getSettingFloat(models.SettingINSSRate),
	}

	// Calculate INSS amount
	inssConfig := INSSConfig{
		ProLabore: data.ProLabore,
		Ceiling:   data.INSSCeiling,
		Rate:      data.INSSRate / 100, // Convert % to decimal
	}
	data.INSSAmount = CalculateINSS(inssConfig)

	return data
}

// getSettingFloat retrieves a setting value from database and converts to float64
func getSettingFloat(key string) float64 {
	var setting models.Settings
	if err := database.DB.Where("key = ?", key).First(&setting).Error; err != nil {
		return 0
	}
	value, _ := strconv.ParseFloat(setting.Value, 64)
	return value
}
