package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email        string `json:"email" gorm:"uniqueIndex;not null"`
	PasswordHash string `json:"-" gorm:"not null"`
	Name         string `json:"name" gorm:"not null"`
}

func (u *User) TableName() string {
	return "users"
}

type RefreshToken struct {
	gorm.Model
	UserID    uint      `json:"user_id" gorm:"not null;index"`
	User      User      `json:"-" gorm:"foreignKey:UserID"`
	Token     string    `json:"token" gorm:"uniqueIndex;not null"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
}

func (r *RefreshToken) TableName() string {
	return "refresh_tokens"
}

func (r *RefreshToken) IsExpired() bool {
	return time.Now().After(r.ExpiresAt)
}

type PasswordResetToken struct {
	gorm.Model
	UserID    uint      `json:"user_id" gorm:"not null;index"`
	User      User      `json:"-" gorm:"foreignKey:UserID"`
	Token     string    `json:"token" gorm:"uniqueIndex;not null"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	Used      bool      `json:"used" gorm:"default:false"`
}

func (p *PasswordResetToken) TableName() string {
	return "password_reset_tokens"
}

func (p *PasswordResetToken) IsExpired() bool {
	return time.Now().After(p.ExpiresAt)
}
