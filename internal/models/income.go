package models

import (
	"time"

	"gorm.io/gorm"
)

type Income struct {
	gorm.Model
	AccountID    uint      `json:"account_id" gorm:"not null;index"`
	Account      Account   `json:"-" gorm:"foreignKey:AccountID"`
	Date         time.Time `json:"date" gorm:"not null"`
	AmountUSD    float64   `json:"amount_usd" gorm:"not null"`
	ExchangeRate float64   `json:"exchange_rate" gorm:"not null"`
	AmountBRL    float64   `json:"amount_brl" gorm:"not null"`
	GrossAmount  float64   `json:"gross_amount" gorm:"not null"`
	TaxAmount    float64   `json:"tax_amount" gorm:"not null"`
	NetAmount    float64   `json:"net_amount" gorm:"not null"`
	Description  string    `json:"description"`
}

func (i *Income) TableName() string {
	return "incomes"
}
