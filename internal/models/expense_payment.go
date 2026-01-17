package models

import (
	"time"

	"gorm.io/gorm"
)

// ExpensePayment registra o pagamento de uma despesa em um mês específico
type ExpensePayment struct {
	gorm.Model
	ExpenseID uint      `json:"expense_id" gorm:"not null;index"`
	Expense   Expense   `json:"expense" gorm:"foreignKey:ExpenseID"`
	Month     int       `json:"month" gorm:"not null"`     // 1-12
	Year      int       `json:"year" gorm:"not null"`      // 2024, 2025, etc
	PaidAt    time.Time `json:"paid_at"`                   // Data do pagamento
	Amount    float64   `json:"amount"`                    // Valor pago (pode ser diferente do valor cadastrado)
}

func (e *ExpensePayment) TableName() string {
	return "expense_payments"
}
