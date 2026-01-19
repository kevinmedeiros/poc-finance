package models

import "gorm.io/gorm"

// CreditCard represents a credit card with billing cycle information and credit limit.
// It tracks the closing day, due day, and associated installment payments.
type CreditCard struct {
	gorm.Model
	AccountID    uint          `json:"account_id" gorm:"not null;index"`
	Account      Account       `json:"-" gorm:"foreignKey:AccountID"`
	Name         string        `json:"name" gorm:"not null"`
	ClosingDay   int           `json:"closing_day" gorm:"not null"`
	DueDay       int           `json:"due_day" gorm:"not null"`
	LimitAmount  float64       `json:"limit_amount"`
	Installments []Installment `json:"installments" gorm:"foreignKey:CreditCardID"`
}

func (c *CreditCard) TableName() string {
	return "credit_cards"
}
