// Package models defines the core data structures for the personal finance application.
// It includes database entities for users, accounts, transactions, expenses, income,
// budgets, goals, and group management. All structs use GORM for ORM database mapping
// and follow standard Go naming conventions and GoDoc documentation practices.
package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents a registered user in the system.
// It stores core user information including authentication credentials
// and is the primary entity for user management and authentication.
type User struct {
	gorm.Model
	Email               string     `json:"email" gorm:"uniqueIndex;not null"`
	PasswordHash        string     `json:"-" gorm:"not null"`
	Name                string     `json:"name" gorm:"not null"`
	FailedLoginAttempts int        `json:"failed_login_attempts" gorm:"default:0"`
	LockedUntil         *time.Time `json:"locked_until,omitempty"`
	LastFailedLoginAt   *time.Time `json:"last_failed_login_at,omitempty"`
	OnboardingCompleted bool       `json:"onboarding_completed" gorm:"default:false"`
}

func (u *User) TableName() string {
	return "users"
}

// IsLocked checks whether the user account is currently locked due to failed login attempts.
func (u *User) IsLocked() bool {
	if u.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*u.LockedUntil)
}

// RefreshToken represents a long-lived token used to obtain new access tokens.
// It enables users to maintain authenticated sessions without repeatedly
// providing credentials, and can be revoked for security purposes.
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

// IsExpired checks whether the refresh token has passed its expiration time.
func (r *RefreshToken) IsExpired() bool {
	return time.Now().After(r.ExpiresAt)
}

// PasswordResetToken represents a one-time use token for resetting user passwords.
// It includes expiration time and usage tracking to ensure secure password recovery.
// Once used or expired, the token becomes invalid.
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

// IsExpired checks whether the password reset token has passed its expiration time.
func (p *PasswordResetToken) IsExpired() bool {
	return time.Now().After(p.ExpiresAt)
}
