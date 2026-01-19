package models

import (
	"time"

	"gorm.io/gorm"
)

type TransactionType string

const (
	TransactionTypeExpense TransactionType = "expense"
	TransactionTypeIncome  TransactionType = "income"
)

type Frequency string

const (
	FrequencyDaily   Frequency = "daily"
	FrequencyWeekly  Frequency = "weekly"
	FrequencyMonthly Frequency = "monthly"
	FrequencyYearly  Frequency = "yearly"
)

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
