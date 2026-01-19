package models

import (
	"time"

	"gorm.io/gorm"
)

// TransactionType represents the direction of a recurring transaction
type TransactionType string

const (
	// TransactionTypeExpense represents money flowing out of the account (bills, subscriptions, etc.)
	TransactionTypeExpense TransactionType = "expense"
	// TransactionTypeIncome represents money flowing into the account (salary, recurring payments, etc.)
	TransactionTypeIncome  TransactionType = "income"
)

// Frequency represents how often a recurring transaction repeats
type Frequency string

const (
	// FrequencyDaily represents transactions that occur every day
	FrequencyDaily   Frequency = "daily"
	// FrequencyWeekly represents transactions that occur every week
	FrequencyWeekly  Frequency = "weekly"
	// FrequencyMonthly represents transactions that occur every month
	FrequencyMonthly Frequency = "monthly"
	// FrequencyYearly represents transactions that occur every year
	FrequencyYearly  Frequency = "yearly"
)

// RecurringTransaction represents an automated transaction that repeats on a regular schedule.
// Recurring transactions are used to track predictable income (salary, rental income) and
// expenses (subscriptions, bills, loan payments) that occur at fixed intervals. The system
// uses NextRunDate to automatically generate transactions when they become due.
// Transactions can be configured with optional end dates and can be activated or deactivated.
type RecurringTransaction struct {
	gorm.Model
	AccountID       uint            `json:"account_id" gorm:"not null;index"`
	Account         Account         `json:"-" gorm:"foreignKey:AccountID"`
	TransactionType TransactionType `json:"transaction_type" gorm:"not null"`
	Frequency       Frequency       `json:"frequency" gorm:"not null"`
	Amount          float64         `json:"amount" gorm:"not null"`
	Description     string          `json:"description"`
	StartDate       time.Time       `json:"start_date" gorm:"not null"`
	EndDate         *time.Time      `json:"end_date"`
	NextRunDate     time.Time       `json:"next_run_date" gorm:"not null"`
	Active          bool            `json:"active" gorm:"default:true"`
	Category        string          `json:"category"`
}

func (rt *RecurringTransaction) TableName() string {
	return "recurring_transactions"
}
