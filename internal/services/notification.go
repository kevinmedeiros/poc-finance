package services

import (
	"fmt"
	"time"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
)

type NotificationService struct{}

func NewNotificationService() *NotificationService {
	return &NotificationService{}
}

// Create creates a new notification for a user
func (s *NotificationService) Create(notification *models.Notification) error {
	return database.DB.Create(notification).Error
}

// GetUserNotifications retrieves all notifications for a user
func (s *NotificationService) GetUserNotifications(userID uint, limit int) ([]models.Notification, error) {
	var notifications []models.Notification
	query := database.DB.Where("user_id = ?", userID).
		Preload("Group").
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&notifications).Error
	return notifications, err
}

// GetUnreadNotifications retrieves only unread notifications for a user
func (s *NotificationService) GetUnreadNotifications(userID uint) ([]models.Notification, error) {
	var notifications []models.Notification
	err := database.DB.Where("user_id = ? AND read = ?", userID, false).
		Preload("Group").
		Order("created_at DESC").
		Find(&notifications).Error
	return notifications, err
}

// GetUnreadCount returns the count of unread notifications for a user
func (s *NotificationService) GetUnreadCount(userID uint) (int64, error) {
	var count int64
	err := database.DB.Model(&models.Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Count(&count).Error
	return count, err
}

// MarkAsRead marks a single notification as read
func (s *NotificationService) MarkAsRead(notificationID, userID uint) error {
	now := time.Now()
	return database.DB.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", notificationID, userID).
		Updates(map[string]interface{}{
			"read":    true,
			"read_at": now,
		}).Error
}

// MarkAllAsRead marks all notifications as read for a user
func (s *NotificationService) MarkAllAsRead(userID uint) error {
	now := time.Now()
	return database.DB.Model(&models.Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Updates(map[string]interface{}{
			"read":    true,
			"read_at": now,
		}).Error
}

// DeleteNotification deletes a notification
func (s *NotificationService) DeleteNotification(notificationID, userID uint) error {
	return database.DB.Where("id = ? AND user_id = ?", notificationID, userID).
		Delete(&models.Notification{}).Error
}

// NotifyGroupInvite creates a notification when a user is added to a group
func (s *NotificationService) NotifyGroupInvite(userID uint, group *models.FamilyGroup, inviterName string) error {
	notification := &models.Notification{
		UserID:  userID,
		Type:    models.NotificationTypeGroupInvite,
		Title:   "Convite para grupo",
		Message: fmt.Sprintf("Você foi adicionado ao grupo \"%s\" por %s", group.Name, inviterName),
		Link:    fmt.Sprintf("/groups/%d/dashboard", group.ID),
		GroupID: &group.ID,
	}
	return s.Create(notification)
}

// NotifyPartnerExpense creates notifications for group members when a new expense is added to a joint account
func (s *NotificationService) NotifyPartnerExpense(expense *models.Expense, account *models.Account, creatorID uint, creatorName string, groupMembers []models.User) error {
	if account.GroupID == nil {
		return nil
	}

	for _, member := range groupMembers {
		// Don't notify the creator
		if member.ID == creatorID {
			continue
		}

		notification := &models.Notification{
			UserID:  member.ID,
			Type:    models.NotificationTypeExpense,
			Title:   "Novo gasto do parceiro",
			Message: fmt.Sprintf("%s adicionou \"%s\" (R$ %.2f) na conta conjunta \"%s\"", creatorName, expense.Name, expense.Amount, account.Name),
			Link:    "/expenses",
			GroupID: account.GroupID,
		}
		if err := s.Create(notification); err != nil {
			return err
		}
	}
	return nil
}
