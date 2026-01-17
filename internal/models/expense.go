package models

import "gorm.io/gorm"

type ExpenseType string

const (
	ExpenseTypeFixed    ExpenseType = "fixed"
	ExpenseTypeVariable ExpenseType = "variable"
)

type Expense struct {
	gorm.Model
	Name     string      `json:"name" gorm:"not null"`
	Amount   float64     `json:"amount" gorm:"not null"`
	Type     ExpenseType `json:"type" gorm:"not null"`
	DueDay   int         `json:"due_day" gorm:"default:1"`
	Category string      `json:"category"`
	Active   bool        `json:"active" gorm:"default:true"`
}

func (e *Expense) TableName() string {
	return "expenses"
}
