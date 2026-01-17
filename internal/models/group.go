package models

import (
	"time"

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

type GroupInvite struct {
	gorm.Model
	Code        string      `json:"code" gorm:"uniqueIndex;not null"`
	GroupID     uint        `json:"group_id" gorm:"not null;index"`
	Group       FamilyGroup `json:"group" gorm:"foreignKey:GroupID"`
	CreatedByID uint        `json:"created_by_id" gorm:"not null"`
	CreatedBy   User        `json:"created_by" gorm:"foreignKey:CreatedByID"`
	ExpiresAt   time.Time   `json:"expires_at" gorm:"not null"`
	MaxUses     int         `json:"max_uses" gorm:"default:0"`    // 0 = unlimited
	UsedCount   int         `json:"used_count" gorm:"default:0"`
	Revoked     bool        `json:"revoked" gorm:"default:false"`
}

func (i *GroupInvite) TableName() string {
	return "group_invites"
}

func (i *GroupInvite) IsValid() bool {
	if i.Revoked {
		return false
	}
	if time.Now().After(i.ExpiresAt) {
		return false
	}
	if i.MaxUses > 0 && i.UsedCount >= i.MaxUses {
		return false
	}
	return true
}
