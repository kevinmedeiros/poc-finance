package models

import (
	"time"

	"gorm.io/gorm"
)

// GoalStatus represents the current status of a financial goal
type GoalStatus string

const (
	// GoalStatusActive represents goals that are currently being worked towards
	GoalStatusActive    GoalStatus = "active"
	// GoalStatusCompleted represents goals that have reached their target amount
	GoalStatusCompleted GoalStatus = "completed"
	// GoalStatusCancelled represents goals that have been abandoned or cancelled
	GoalStatusCancelled GoalStatus = "cancelled"
)

// GroupGoal represents a shared financial goal for a family group.
// Goals allow group members to collaboratively save towards a target amount by a specific date.
// Each goal tracks contributions from individual members, calculates progress towards the target,
// and can optionally be linked to a specific account for automated tracking.
type GroupGoal struct {
	gorm.Model
	GroupID       uint        `json:"group_id" gorm:"not null;index"`
	Group         FamilyGroup `json:"group" gorm:"foreignKey:GroupID"`
	AccountID     *uint       `json:"account_id" gorm:"index"`
	Account       *Account    `json:"-" gorm:"foreignKey:AccountID"`
	Name          string      `json:"name" gorm:"not null"`
	Description   string      `json:"description"`
	TargetAmount  float64     `json:"target_amount" gorm:"not null"`
	CurrentAmount float64     `json:"current_amount" gorm:"default:0"`
	StartDate     time.Time   `json:"start_date" gorm:"not null"`
	TargetDate    time.Time   `json:"target_date" gorm:"not null"`
	Status        GoalStatus  `json:"status" gorm:"default:active"`
	CreatedByID   uint        `json:"created_by_id" gorm:"not null"`
	CreatedBy     User        `json:"created_by" gorm:"foreignKey:CreatedByID"`
	Contributions []GoalContribution `json:"contributions" gorm:"foreignKey:GoalID"`
}

func (g *GroupGoal) TableName() string {
	return "group_goals"
}

func (g *GroupGoal) IsCompleted() bool {
	return g.CurrentAmount >= g.TargetAmount
}

func (g *GroupGoal) ProgressPercentage() float64 {
	if g.TargetAmount == 0 {
		return 0
	}
	pct := (g.CurrentAmount / g.TargetAmount) * 100
	if pct > 100 {
		return 100
	}
	return pct
}

func (g *GroupGoal) RemainingAmount() float64 {
	remaining := g.TargetAmount - g.CurrentAmount
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GoalContribution represents an individual contribution made by a user towards a group goal.
// Contributions track which users are actively contributing to goals and by how much,
// enabling fair distribution tracking and contribution history for shared financial objectives.
type GoalContribution struct {
	gorm.Model
	GoalID uint      `json:"goal_id" gorm:"not null;index"`
	Goal   GroupGoal `json:"-" gorm:"foreignKey:GoalID"`
	UserID uint      `json:"user_id" gorm:"not null;index"`
	User   User      `json:"user" gorm:"foreignKey:UserID"`
	Amount float64   `json:"amount" gorm:"not null"`
}

func (c *GoalContribution) TableName() string {
	return "goal_contributions"
}
