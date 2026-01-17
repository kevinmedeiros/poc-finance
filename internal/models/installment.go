package models

import (
	"time"

	"gorm.io/gorm"
)

type Installment struct {
	gorm.Model
	CreditCardID       uint      `json:"credit_card_id" gorm:"not null"`
	CreditCard         CreditCard `json:"credit_card" gorm:"foreignKey:CreditCardID"`
	Description        string    `json:"description" gorm:"not null"`
	TotalAmount        float64   `json:"total_amount" gorm:"not null"`
	InstallmentAmount  float64   `json:"installment_amount" gorm:"not null"`
	TotalInstallments  int       `json:"total_installments" gorm:"not null"`
	CurrentInstallment int       `json:"current_installment" gorm:"not null;default:1"`
	StartDate          time.Time `json:"start_date" gorm:"not null"`
	Category           string    `json:"category"`
}

func (i *Installment) TableName() string {
	return "installments"
}
