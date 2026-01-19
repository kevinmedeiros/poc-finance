package models

import "gorm.io/gorm"

// AccountType represents the type of account (individual or joint)
type AccountType string

const (
	// AccountTypeIndividual represents an account owned by a single user
	AccountTypeIndividual AccountType = "individual"
	// AccountTypeJoint represents an account shared between multiple users in a group
	AccountTypeJoint      AccountType = "joint"
)

// Account represents a financial account for tracking income, expenses, and budgets.
// Accounts can be individual (single user) or joint (shared between group members).
// Joint accounts are typically linked to a FamilyGroup and allow multiple users to collaborate.
type Account struct {
	gorm.Model
	Name        string      `json:"name" gorm:"not null"`
	Type        AccountType `json:"type" gorm:"not null;default:'individual'"`
	UserID      uint        `json:"user_id" gorm:"not null;index"` // Owner (for individual) or creator (for joint)
	User        User        `json:"-" gorm:"foreignKey:UserID"`
	GroupID     *uint       `json:"group_id" gorm:"index"` // Optional: for joint accounts linked to a group
	BudgetLimit *float64    `json:"budget_limit"`          // Optional monthly expense limit
}

func (a *Account) TableName() string {
	return "accounts"
}

// IsIndividual returns true if this is an individual account owned by a single user
func (a *Account) IsIndividual() bool {
	return a.Type == AccountTypeIndividual
}

// IsJoint returns true if this is a joint account shared between group members
func (a *Account) IsJoint() bool {
	return a.Type == AccountTypeJoint
}
