package models

import "gorm.io/gorm"

// ExpenseSplit represents the division of an expense between group members
type ExpenseSplit struct {
	gorm.Model
	ExpenseID  uint    `json:"expense_id" gorm:"not null;index"`
	Expense    Expense `json:"-" gorm:"foreignKey:ExpenseID"`
	UserID     uint    `json:"user_id" gorm:"not null;index"`
	User       User    `json:"user" gorm:"foreignKey:UserID"`
	Percentage float64 `json:"percentage" gorm:"not null"` // 0-100
	Amount     float64 `json:"amount" gorm:"not null"`     // Calculated: expense.amount * percentage / 100
}

func (e *ExpenseSplit) TableName() string {
	return "expense_splits"
}
