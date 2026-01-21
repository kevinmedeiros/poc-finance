package models

import (
	"time"

	"gorm.io/gorm"
)

// BudgetStatus represents the current status of a budget
type BudgetStatus string

const (
	// BudgetStatusActive represents budgets that are currently being tracked
	BudgetStatusActive   BudgetStatus = "active"
	// BudgetStatusArchived represents budgets that have been archived for historical reference
	BudgetStatusArchived BudgetStatus = "archived"
)

// CategoryStatus represents the visual indicator status for a budget category
type CategoryStatus string

const (
	// CategoryStatusGood indicates spending is under 80% of limit (green)
	CategoryStatusGood    CategoryStatus = "good"
	// CategoryStatusWarning indicates spending is between 80-100% of limit (yellow)
	CategoryStatusWarning CategoryStatus = "warning"
	// CategoryStatusExceeded indicates spending has exceeded 100% of limit (red)
	CategoryStatusExceeded CategoryStatus = "exceeded"
)

// Budget represents a monthly budget for tracking spending across categories.
// Budgets can be individual (UserID set, GroupID null) or shared for a family group (GroupID set).
// Each budget tracks spending across multiple categories with individual limits.
type Budget struct {
	gorm.Model
	GroupID    *uint             `json:"group_id" gorm:"index"`
	Group      *FamilyGroup      `json:"group,omitempty" gorm:"foreignKey:GroupID"`
	UserID     uint              `json:"user_id" gorm:"not null;index"`
	User       User              `json:"user" gorm:"foreignKey:UserID"`
	Year       int               `json:"year" gorm:"not null;index"`
	Month      int               `json:"month" gorm:"not null;index"` // 1-12
	Name       string            `json:"name" gorm:"not null"`
	Status     BudgetStatus      `json:"status" gorm:"default:active"`
	Categories []BudgetCategory  `json:"categories" gorm:"foreignKey:BudgetID"`
}

func (b *Budget) TableName() string {
	return "budgets"
}

// IsGroupBudget returns true if this budget is shared with a family group
func (b *Budget) IsGroupBudget() bool {
	return b.GroupID != nil
}

// TotalLimit calculates the sum of all category limits in this budget
func (b *Budget) TotalLimit() float64 {
	var total float64
	for _, cat := range b.Categories {
		total += cat.Limit
	}
	return total
}

// TotalSpent calculates the sum of all category spending in this budget
func (b *Budget) TotalSpent() float64 {
	var total float64
	for _, cat := range b.Categories {
		total += cat.Spent
	}
	return total
}

// OverallProgressPercentage calculates the overall budget progress across all categories (0-100+).
// Returns 0 if the total limit is zero.
func (b *Budget) OverallProgressPercentage() float64 {
	totalLimit := b.TotalLimit()
	if totalLimit == 0 {
		return 0
	}
	return (b.TotalSpent() / totalLimit) * 100
}

// RemainingAmount returns the total amount still available across all categories.
// Returns 0 if the budget has been exceeded.
func (b *Budget) RemainingAmount() float64 {
	remaining := b.TotalLimit() - b.TotalSpent()
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetOverallStatus returns the visual indicator status for the entire budget
func (b *Budget) GetOverallStatus() CategoryStatus {
	percentage := b.OverallProgressPercentage()
	if percentage > 100 {
		return CategoryStatusExceeded
	}
	if percentage >= 80 {
		return CategoryStatusWarning
	}
	return CategoryStatusGood
}

// BudgetCategory represents a spending category within a budget with its own limit.
// Categories track actual spending and provide progress indicators, remaining amounts,
// and visual status (green/yellow/red) based on spending thresholds.
type BudgetCategory struct {
	gorm.Model
	BudgetID      uint      `json:"budget_id" gorm:"not null;index"`
	Budget        Budget    `json:"-" gorm:"foreignKey:BudgetID"`
	Category      string    `json:"category" gorm:"not null"`
	Limit         float64   `json:"limit" gorm:"not null"`
	Spent         float64   `json:"spent" gorm:"default:0"`
	NotifiedAt80  *time.Time `json:"notified_at_80"`  // Timestamp when 80% notification was sent
	NotifiedAt100 *time.Time `json:"notified_at_100"` // Timestamp when 100% notification was sent
}

func (c *BudgetCategory) TableName() string {
	return "budget_categories"
}

// ProgressPercentage calculates the percentage of the category limit spent (0-100+).
// Returns 0 if the limit is zero.
func (c *BudgetCategory) ProgressPercentage() float64 {
	if c.Limit == 0 {
		return 0
	}
	return (c.Spent / c.Limit) * 100
}

// RemainingAmount returns the amount still available in this category.
// Returns 0 if the category limit has been exceeded.
func (c *BudgetCategory) RemainingAmount() float64 {
	remaining := c.Limit - c.Spent
	if remaining < 0 {
		return 0
	}
	return remaining
}

// IsExceeded returns true if spending has exceeded the category limit
func (c *BudgetCategory) IsExceeded() bool {
	return c.Spent > c.Limit
}

// GetStatus returns the visual indicator status for this category based on spending percentage.
// Returns "good" (green) if under 80%, "warning" (yellow) if 80-100%, or "exceeded" (red) if over 100%.
func (c *BudgetCategory) GetStatus() CategoryStatus {
	percentage := c.ProgressPercentage()
	if percentage > 100 {
		return CategoryStatusExceeded
	}
	if percentage >= 80 {
		return CategoryStatusWarning
	}
	return CategoryStatusGood
}

// ShouldNotifyAt80 returns true if the category has reached 80% threshold and hasn't been notified yet
func (c *BudgetCategory) ShouldNotifyAt80() bool {
	return c.ProgressPercentage() >= 80 && c.NotifiedAt80 == nil
}

// ShouldNotifyAt100 returns true if the category has reached 100% threshold and hasn't been notified yet
func (c *BudgetCategory) ShouldNotifyAt100() bool {
	return c.ProgressPercentage() >= 100 && c.NotifiedAt100 == nil
}
