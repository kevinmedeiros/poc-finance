package models

import "gorm.io/gorm"

type Settings struct {
	gorm.Model
	Key   string `json:"key" gorm:"uniqueIndex;not null"`
	Value string `json:"value"`
}

func (s *Settings) TableName() string {
	return "settings"
}

// Chaves de configuração
const (
	SettingProLabore    = "pro_labore"      // Valor do pró-labore mensal
	SettingINSSCeiling  = "inss_ceiling"    // Teto do INSS (para cálculo do limite)
	SettingINSSRate     = "inss_rate"       // Alíquota do INSS (padrão 11%)
)
