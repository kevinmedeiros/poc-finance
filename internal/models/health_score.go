package models

import (
	"time"

	"gorm.io/gorm"
)

// HealthScore represents a calculated financial health score for a user or group.
// Scores range from 0-100 and are based on multiple financial factors including
// savings rate, debt levels, goal progress, and budget adherence. Historical scores
// enable tracking of financial health trends over time.
type HealthScore struct {
	gorm.Model
	UserID       *uint         `json:"user_id" gorm:"index"`
	User         *User         `json:"-" gorm:"foreignKey:UserID"`
	GroupID      *uint         `json:"group_id" gorm:"index"`
	Group        *FamilyGroup  `json:"-" gorm:"foreignKey:GroupID"`
	Score        float64       `json:"score" gorm:"not null"`
	SavingsScore float64       `json:"savings_score" gorm:"not null"`
	DebtScore    float64       `json:"debt_score" gorm:"not null"`
	GoalScore    float64       `json:"goal_score" gorm:"not null"`
	BudgetScore  float64       `json:"budget_score" gorm:"not null"`
	CalculatedAt time.Time     `json:"calculated_at" gorm:"not null;index"`
	Metadata     string        `json:"metadata" gorm:"type:text"`
}

func (h *HealthScore) TableName() string {
	return "health_scores"
}

// IsHealthy returns true if the overall health score is 75 or above.
// Scores of 75+ indicate good financial health practices.
func (h *HealthScore) IsHealthy() bool {
	return h.Score >= 75
}

// IsAtRisk returns true if the overall health score is below 50.
// Scores below 50 indicate significant financial health concerns.
func (h *HealthScore) IsAtRisk() bool {
	return h.Score < 50
}

// GetScoreLevel returns a string representation of the score level.
// Returns "excellent" (90+), "good" (75-89), "fair" (50-74), or "poor" (<50).
func (h *HealthScore) GetScoreLevel() string {
	switch {
	case h.Score >= 90:
		return "excellent"
	case h.Score >= 75:
		return "good"
	case h.Score >= 50:
		return "fair"
	default:
		return "poor"
	}
}

// IsUserScore returns true if this score belongs to an individual user
// rather than a family group.
func (h *HealthScore) IsUserScore() bool {
	return h.UserID != nil && h.GroupID == nil
}

// IsGroupScore returns true if this score belongs to a family group
// rather than an individual user.
func (h *HealthScore) IsGroupScore() bool {
	return h.GroupID != nil && h.UserID == nil
}
