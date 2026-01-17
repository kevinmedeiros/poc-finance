package models

import "gorm.io/gorm"

type CreditCard struct {
	gorm.Model
	Name         string        `json:"name" gorm:"not null"`
	ClosingDay   int           `json:"closing_day" gorm:"not null"`
	DueDay       int           `json:"due_day" gorm:"not null"`
	LimitAmount  float64       `json:"limit_amount"`
	Installments []Installment `json:"installments" gorm:"foreignKey:CreditCardID"`
}

func (c *CreditCard) TableName() string {
	return "credit_cards"
}
