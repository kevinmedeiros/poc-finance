package models

import "gorm.io/gorm"

// ExpenseType represents the type of an expense
type ExpenseType string

const (
	// ExpenseTypeFixed represents fixed expenses with regular amounts
	ExpenseTypeFixed    ExpenseType = "fixed"
	// ExpenseTypeVariable represents variable expenses with fluctuating amounts
	ExpenseTypeVariable ExpenseType = "variable"
)

// Expense represents an expense associated with an account.
// Expenses can be split between group members using ExpenseSplit.
// Each expense has a type (fixed or variable) and a due day for recurring payments.
type Expense struct {
	gorm.Model
	AccountID uint           `json:"account_id" gorm:"not null;index"` // Account ID to which the expense belongs
	Account   Account        `json:"-" gorm:"foreignKey:AccountID"`
	Name      string         `json:"name" gorm:"not null"`                // Descriptive name of the expense
	Amount    float64        `json:"amount" gorm:"not null"`              // Expense amount
	Type      ExpenseType    `json:"type" gorm:"not null"`                // Type of expense (fixed or variable)
	DueDay    int            `json:"due_day" gorm:"default:1"`            // Payment due day (1-31)
	Category  string         `json:"category"`                            // Expense category (food, transport, etc)
	Active    bool           `json:"active" gorm:"default:true"`          // Whether the expense is active
	IsSplit   bool           `json:"is_split" gorm:"default:false"`       // Whether the expense is split between members
	Splits    []ExpenseSplit `json:"splits" gorm:"foreignKey:ExpenseID"` // Expense splits between members
}

func (e *Expense) TableName() string {
	return "expenses"
}
