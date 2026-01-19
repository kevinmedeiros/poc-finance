package models

import (
	"time"

	"gorm.io/gorm"
)

// Bill represents a scheduled payment or recurring bill associated with an account
type Bill struct {
	gorm.Model
	AccountID uint      `json:"account_id" gorm:"not null;index"`
	Account   Account   `json:"-" gorm:"foreignKey:AccountID"`
	Name      string    `json:"name" gorm:"not null"`
	Amount    float64   `json:"amount" gorm:"not null"`
	DueDate   time.Time `json:"due_date" gorm:"not null"`
	Paid      bool      `json:"paid" gorm:"default:false"`
	Recurring bool      `json:"recurring" gorm:"default:false"` // Whether this bill repeats on a schedule
	Category  string    `json:"category"`                       // Optional category for organizing bills
}

func (b *Bill) TableName() string {
	return "bills"
}
