package models

import "gorm.io/gorm"

// Settings represents system-wide configuration key-value pairs.
// Settings store application configuration that can be modified at runtime without code changes,
// such as financial calculation parameters (pro-labore, INSS rates and ceilings).
// Each setting is identified by a unique key and stores its value as a string.
type Settings struct {
	gorm.Model
	Key   string `json:"key" gorm:"uniqueIndex;not null"`
	Value string `json:"value"`
}

func (s *Settings) TableName() string {
	return "settings"
}

const (
	// SettingProLabore represents the monthly pro-labore (owner's salary) value setting
	SettingProLabore = "pro_labore"
	// SettingINSSCeiling represents the INSS (social security) contribution ceiling for limit calculations
	SettingINSSCeiling = "inss_ceiling"
	// SettingINSSRate represents the INSS contribution rate (default 11%)
	SettingINSSRate = "inss_rate"
	// SettingBudgetWarningThreshold represents the budget warning threshold percentage for alerts
	SettingBudgetWarningThreshold = "budget_warning_threshold"
	// SettingRecordStartDate represents the start date from which records should be displayed (format: YYYY-MM-DD)
	SettingRecordStartDate = "record_start_date"
)
