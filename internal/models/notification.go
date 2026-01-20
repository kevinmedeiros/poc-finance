package models

import (
	"time"

	"gorm.io/gorm"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	// NotificationTypeGroupInvite represents notifications for group membership invitations
	NotificationTypeGroupInvite NotificationType = "group_invite"
	// NotificationTypeExpense represents notifications about shared expenses and payments
	NotificationTypeExpense NotificationType = "expense"
	// NotificationTypeGoalReached represents notifications when a financial goal is achieved
	NotificationTypeGoalReached NotificationType = "goal_reached"
	// NotificationTypeBudgetAlert represents notifications when budget limits are approaching or exceeded
	NotificationTypeBudgetAlert NotificationType = "budget_alert"
	// NotificationTypeSummary represents periodic financial summary notifications
	NotificationTypeSummary NotificationType = "summary"
	// NotificationTypeDueDate represents notifications for upcoming expense due dates
	NotificationTypeDueDate NotificationType = "due_date"
)

// Notification represents an in-app notification sent to a user.
// Notifications inform users about important events such as group invites, expense updates,
// budget alerts, and goal achievements. They can optionally link to related entities
// (groups, invites) and track read status for user experience.
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
