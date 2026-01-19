package models

import (
	"time"

	"gorm.io/gorm"
)

// ExpensePayment records the payment of an expense in a specific month
type ExpensePayment struct {
	gorm.Model
	ExpenseID uint      `json:"expense_id" gorm:"not null;index"`
	Expense   Expense   `json:"expense" gorm:"foreignKey:ExpenseID"`
	Month     int       `json:"month" gorm:"not null"`     // 1-12
	Year      int       `json:"year" gorm:"not null"`      // 2024, 2025, etc
	PaidAt    time.Time `json:"paid_at"`                   // Payment date
	Amount    float64   `json:"amount"`                    // Amount paid (may differ from the registered amount)
}

func (e *ExpensePayment) TableName() string {
	return "expense_payments"
}
