package models

import (
	"time"

	"gorm.io/gorm"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeGroupInvite NotificationType = "group_invite"
	NotificationTypeExpense     NotificationType = "expense"
	NotificationTypeGoalReached NotificationType = "goal_reached"
	NotificationTypeBudgetAlert NotificationType = "budget_alert"
	NotificationTypeSummary     NotificationType = "summary"
)

type Notification struct {
	gorm.Model
	UserID    uint             `json:"user_id" gorm:"not null;index"`
	User      User             `json:"user" gorm:"foreignKey:UserID"`
	Type      NotificationType `json:"type" gorm:"not null;index"`
	Title     string           `json:"title" gorm:"not null"`
	Message   string           `json:"message" gorm:"not null"`
	Read      bool             `json:"read" gorm:"default:false;index"`
	ReadAt    *time.Time       `json:"read_at"`
	Link      string           `json:"link"` // Optional link to navigate to
	GroupID   *uint            `json:"group_id"`
	Group     *FamilyGroup     `json:"group" gorm:"foreignKey:GroupID"`
	InviteID  *uint            `json:"invite_id"`
	Invite    *GroupInvite     `json:"invite" gorm:"foreignKey:InviteID"`
}

func (n *Notification) TableName() string {
	return "notifications"
}
