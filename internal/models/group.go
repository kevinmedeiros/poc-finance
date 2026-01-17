package models

import (
	"gorm.io/gorm"
)

type FamilyGroup struct {
	gorm.Model
	Name        string        `json:"name" gorm:"not null"`
	Description string        `json:"description"`
	CreatedByID uint          `json:"created_by_id" gorm:"not null"`
	CreatedBy   User          `json:"created_by" gorm:"foreignKey:CreatedByID"`
	Members     []GroupMember `json:"members" gorm:"foreignKey:GroupID"`
}

func (g *FamilyGroup) TableName() string {
	return "family_groups"
}

type GroupMember struct {
	gorm.Model
	GroupID uint        `json:"group_id" gorm:"not null;index"`
	Group   FamilyGroup `json:"group" gorm:"foreignKey:GroupID"`
	UserID  uint        `json:"user_id" gorm:"not null;index"`
	User    User        `json:"user" gorm:"foreignKey:UserID"`
	Role    string      `json:"role" gorm:"default:admin"` // "admin" or "member"
}

func (m *GroupMember) TableName() string {
	return "group_members"
}
