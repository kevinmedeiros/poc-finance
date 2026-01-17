package models

import "gorm.io/gorm"

type AccountType string

const (
	AccountTypeIndividual AccountType = "individual"
	AccountTypeJoint      AccountType = "joint"
)

type Account struct {
	gorm.Model
	Name    string      `json:"name" gorm:"not null"`
	Type    AccountType `json:"type" gorm:"not null;default:'individual'"`
	UserID  uint        `json:"user_id" gorm:"not null;index"` // Owner (for individual) or creator (for joint)
	User    User        `json:"-" gorm:"foreignKey:UserID"`
	GroupID *uint       `json:"group_id" gorm:"index"` // Optional: for joint accounts linked to a group
}

func (a *Account) TableName() string {
	return "accounts"
}

func (a *Account) IsIndividual() bool {
	return a.Type == AccountTypeIndividual
}

func (a *Account) IsJoint() bool {
	return a.Type == AccountTypeJoint
}
