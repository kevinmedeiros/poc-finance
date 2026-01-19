package models

import (
	"time"

	"gorm.io/gorm"
)

// Income represents an income transaction with multi-currency support.
// It tracks income amounts in both USD and BRL with exchange rate information,
// and includes gross, tax, and net amounts for financial reporting.
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

// TableName returns the table name for the Income model
func (i *Income) TableName() string {
	return "incomes"
}
