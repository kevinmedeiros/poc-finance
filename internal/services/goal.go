package services

import (
	"errors"
	"time"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
)

var (
	ErrGoalNotFound  = errors.New("meta não encontrada")
	ErrGoalCompleted = errors.New("meta já foi concluída")
)

type GoalService struct {
	groupService *GroupService
}

func NewGoalService() *GoalService {
	return &GoalService{
		groupService: NewGroupService(),
	}
}

// CreateGoal creates a new group goal
func (s *GoalService) CreateGoal(groupID, userID uint, name string, description string,
	targetAmount float64, targetDate time.Time, accountID *uint) (*models.GroupGoal, error) {

	// Verify user is group member
	if !s.groupService.IsGroupMember(groupID, userID) {
		return nil, ErrUnauthorized
	}

	goal := &models.GroupGoal{
		GroupID:       groupID,
		AccountID:     accountID,
		Name:          name,
		Description:   description,
		TargetAmount:  targetAmount,
		CurrentAmount: 0,
		StartDate:     time.Now(),
		TargetDate:    targetDate,
		Status:        models.GoalStatusActive,
		CreatedByID:   userID,
	}

	if err := database.DB.Create(goal).Error; err != nil {
		return nil, err
	}

	database.DB.Preload("Group").Preload("CreatedBy").Preload("Account").First(goal, goal.ID)
	return goal, nil
}

// GetGroupGoals returns all goals for a group
func (s *GoalService) GetGroupGoals(groupID, userID uint) ([]models.GroupGoal, error) {
	if !s.groupService.IsGroupMember(groupID, userID) {
		return nil, ErrUnauthorized
	}

	var goals []models.GroupGoal
	database.DB.Where("group_id = ?", groupID).
		Preload("CreatedBy").
		Preload("Account").
		Preload("Contributions.User").
		Order("target_date ASC").
		Find(&goals)

	return goals, nil
}

// GetGoalByID retrieves a goal by ID
func (s *GoalService) GetGoalByID(goalID uint) (*models.GroupGoal, error) {
	var goal models.GroupGoal
	if err := database.DB.Preload("Group").Preload("CreatedBy").Preload("Account").Preload("Contributions.User").
		First(&goal, goalID).Error; err != nil {
		return nil, ErrGoalNotFound
	}
	return &goal, nil
}

// UpdateGoal updates goal details
func (s *GoalService) UpdateGoal(goalID, userID uint, name, description string,
	targetAmount float64, targetDate time.Time) error {

	goal, err := s.GetGoalByID(goalID)
	if err != nil {
		return err
	}

	// Only creator or group admin can update
	if !s.isGoalOwnerOrAdmin(goal.GroupID, userID, goal.CreatedByID) {
		return ErrUnauthorized
	}

	return database.DB.Model(goal).Updates(map[string]interface{}{
		"name":          name,
		"description":   description,
		"target_amount": targetAmount,
		"target_date":   targetDate,
	}).Error
}

// DeleteGoal deletes a goal
func (s *GoalService) DeleteGoal(goalID, userID uint) error {
	goal, err := s.GetGoalByID(goalID)
	if err != nil {
		return err
	}

	if !s.isGoalOwnerOrAdmin(goal.GroupID, userID, goal.CreatedByID) {
		return ErrUnauthorized
	}

	// Delete contributions first
	database.DB.Where("goal_id = ?", goalID).Delete(&models.GoalContribution{})

	return database.DB.Delete(goal).Error
}

// AddContribution adds/updates contribution from a member
func (s *GoalService) AddContribution(goalID, userID uint, amount float64) (*models.GoalContribution, error) {
	goal, err := s.GetGoalByID(goalID)
	if err != nil {
		return nil, err
	}

	// Verify user is member of the group
	if !s.groupService.IsGroupMember(goal.GroupID, userID) {
		return nil, ErrUnauthorized
	}

	if goal.Status != models.GoalStatusActive {
		return nil, ErrGoalCompleted
	}

	// Find existing contribution or create new one
	var contrib models.GoalContribution
	result := database.DB.Where("goal_id = ? AND user_id = ?", goalID, userID).First(&contrib)

	if result.Error == nil {
		// Update existing
		contrib.Amount += amount
		database.DB.Save(&contrib)
	} else {
		// Create new
		contrib = models.GoalContribution{
			GoalID: goalID,
			UserID: userID,
			Amount: amount,
		}
		database.DB.Create(&contrib)
	}

	// Update goal current amount
	s.updateGoalCurrentAmount(goalID)

	database.DB.Preload("User").First(&contrib, contrib.ID)
	return &contrib, nil
}

// GetContributions returns all contributions for a goal
func (s *GoalService) GetContributions(goalID uint) ([]models.GoalContribution, error) {
	var contribs []models.GoalContribution
	database.DB.Where("goal_id = ?", goalID).
		Preload("User").
		Order("amount DESC").
		Find(&contribs)

	return contribs, nil
}

// updateGoalCurrentAmount recalculates the goal's current amount from contributions
func (s *GoalService) updateGoalCurrentAmount(goalID uint) {
	var totalAmount float64
	database.DB.Model(&models.GoalContribution{}).
		Where("goal_id = ?", goalID).
		Select("COALESCE(SUM(amount), 0)").
		Row().
		Scan(&totalAmount)

	var goal models.GroupGoal
	database.DB.First(&goal, goalID)

	updates := map[string]interface{}{
		"current_amount": totalAmount,
	}

	// Check if goal is completed
	if totalAmount >= goal.TargetAmount && goal.Status == models.GoalStatusActive {
		updates["status"] = models.GoalStatusCompleted
	}

	database.DB.Model(&goal).Updates(updates)
}

// isGoalOwnerOrAdmin checks if user is the goal creator or a group admin
func (s *GoalService) isGoalOwnerOrAdmin(groupID, userID, goalCreatorID uint) bool {
	// Creator of goal
	if userID == goalCreatorID {
		return true
	}

	// Group admin
	return s.groupService.IsGroupAdmin(groupID, userID)
}
