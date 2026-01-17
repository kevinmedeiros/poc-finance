package models

import "gorm.io/gorm"

type ExpenseType string

const (
	ExpenseTypeFixed    ExpenseType = "fixed"
	ExpenseTypeVariable ExpenseType = "variable"
)

type Expense struct {
	gorm.Model
	AccountID uint        `json:"account_id" gorm:"not null;index"`
	Account   Account     `json:"-" gorm:"foreignKey:AccountID"`
	Name      string      `json:"name" gorm:"not null"`
	Amount    float64     `json:"amount" gorm:"not null"`
	Type      ExpenseType `json:"type" gorm:"not null"`
	DueDay    int         `json:"due_day" gorm:"default:1"`
	Category  string      `json:"category"`
	Active    bool        `json:"active" gorm:"default:true"`
	IsSplit   bool        `json:"is_split" gorm:"default:false"`
	Splits    []ExpenseSplit `json:"splits" gorm:"foreignKey:ExpenseID"`
}

func (e *Expense) TableName() string {
	return "expenses"
}
