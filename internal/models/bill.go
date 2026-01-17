package models

import (
	"time"

	"gorm.io/gorm"
)

type Bill struct {
	gorm.Model
	Name      string    `json:"name" gorm:"not null"`
	Amount    float64   `json:"amount" gorm:"not null"`
	DueDate   time.Time `json:"due_date" gorm:"not null"`
	Paid      bool      `json:"paid" gorm:"default:false"`
	Recurring bool      `json:"recurring" gorm:"default:false"`
	Category  string    `json:"category"`
}

func (b *Bill) TableName() string {
	return "bills"
}
