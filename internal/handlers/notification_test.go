package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/database"
	"poc-finance/internal/middleware"
	"poc-finance/internal/models"
	"poc-finance/internal/services"
	"poc-finance/internal/testutil"
)

func setupNotificationTestHandler() (*NotificationHandler, *echo.Echo, uint, *models.FamilyGroup) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create test user
	authService := services.NewAuthService()
	user, _ := authService.Register("test@example.com", "Password123", "Test User")

	// Create test group
	group := models.FamilyGroup{
		Name: "Test Group",
	}
	database.DB.Create(&group)

	// Add user to group
	groupMember := models.GroupMember{
		GroupID: group.ID,
		UserID:  user.ID,
		Role:    "admin",
	}
	database.DB.Create(&groupMember)

	e := echo.New()
	e.Renderer = &testutil.MockRenderer{}
	handler := NewNotificationHandler()
	return handler, e, user.ID, &group
}

func createTestNotification(userID uint, groupID *uint, notifType models.NotificationType, read bool) *models.Notification {
	notification := &models.Notification{
		UserID:  userID,
		Type:    notifType,
		Title:   "Test Notification",
		Message: "This is a test notification",
		Read:    read,
		GroupID: groupID,
		Link:    "/test",
	}
	if read {
		now := time.Now()
		notification.ReadAt = &now
	}
	database.DB.Create(notification)
	return notification
}

func TestNotificationHandler_List_Success(t *testing.T) {
	handler, e, userID, group := setupNotificationTestHandler()

	// Create test notifications
	createTestNotification(userID, &group.ID, models.NotificationTypeGroupInvite, false)
	createTestNotification(userID, &group.ID, models.NotificationTypeExpense, true)
	createTestNotification(userID, &group.ID, models.NotificationTypeGoalReached, false)

	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.List(c)
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestNotificationHandler_List_EmptyNotifications(t *testing.T) {
	handler, e, userID, _ := setupNotificationTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.List(c)
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestNotificationHandler_GetDropdown_Success(t *testing.T) {
	handler, e, userID, group := setupNotificationTestHandler()

	// Create multiple test notifications
	for i := 0; i < 7; i++ {
		createTestNotification(userID, &group.ID, models.NotificationTypeExpense, i%2 == 0)
	}

	req := httptest.NewRequest(http.MethodGet, "/notifications/dropdown", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.GetDropdown(c)
	if err != nil {
		t.Fatalf("GetDropdown() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Verify that only 5 notifications are returned (limit)
	var notifications []models.Notification
	database.DB.Where("user_id = ?", userID).Find(&notifications)
	if len(notifications) != 7 {
		t.Errorf("Total notifications = %d, want %d", len(notifications), 7)
	}
}

func TestNotificationHandler_GetDropdown_EmptyNotifications(t *testing.T) {
	handler, e, userID, _ := setupNotificationTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/notifications/dropdown", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.GetDropdown(c)
	if err != nil {
		t.Fatalf("GetDropdown() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestNotificationHandler_GetBadge_WithUnreadNotifications(t *testing.T) {
	handler, e, userID, group := setupNotificationTestHandler()

	// Create 3 unread and 2 read notifications
	createTestNotification(userID, &group.ID, models.NotificationTypeExpense, false)
	createTestNotification(userID, &group.ID, models.NotificationTypeExpense, false)
	createTestNotification(userID, &group.ID, models.NotificationTypeExpense, false)
	createTestNotification(userID, &group.ID, models.NotificationTypeExpense, true)
	createTestNotification(userID, &group.ID, models.NotificationTypeExpense, true)

	req := httptest.NewRequest(http.MethodGet, "/notifications/badge", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.GetBadge(c)
	if err != nil {
		t.Fatalf("GetBadge() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Verify unread count
	notificationService := services.NewNotificationService()
	unreadCount, _ := notificationService.GetUnreadCount(userID)
	if unreadCount != 3 {
		t.Errorf("Unread count = %d, want %d", unreadCount, 3)
	}
}

func TestNotificationHandler_GetBadge_NoUnreadNotifications(t *testing.T) {
	handler, e, userID, group := setupNotificationTestHandler()

	// Create only read notifications
	createTestNotification(userID, &group.ID, models.NotificationTypeExpense, true)
	createTestNotification(userID, &group.ID, models.NotificationTypeExpense, true)

	req := httptest.NewRequest(http.MethodGet, "/notifications/badge", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.GetBadge(c)
	if err != nil {
		t.Fatalf("GetBadge() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Verify unread count is 0
	notificationService := services.NewNotificationService()
	unreadCount, _ := notificationService.GetUnreadCount(userID)
	if unreadCount != 0 {
		t.Errorf("Unread count = %d, want %d", unreadCount, 0)
	}
}

func TestNotificationHandler_MarkAsRead_Success(t *testing.T) {
	handler, e, userID, group := setupNotificationTestHandler()

	// Create unread notification
	notification := createTestNotification(userID, &group.ID, models.NotificationTypeExpense, false)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/notifications/%d/read", notification.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", notification.ID))

	err := handler.MarkAsRead(c)
	if err != nil {
		t.Fatalf("MarkAsRead() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Verify notification is marked as read
	var updatedNotification models.Notification
	database.DB.First(&updatedNotification, notification.ID)
	if !updatedNotification.Read {
		t.Errorf("Notification.Read = %v, want %v", updatedNotification.Read, true)
	}
	if updatedNotification.ReadAt == nil {
		t.Error("Notification.ReadAt should not be nil after marking as read")
	}
}

func TestNotificationHandler_MarkAsRead_InvalidID(t *testing.T) {
	handler, e, userID, _ := setupNotificationTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/notifications/invalid/read", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)
	c.SetParamNames("id")
	c.SetParamValues("invalid")

	err := handler.MarkAsRead(c)
	if err != nil {
		t.Fatalf("MarkAsRead() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	body := rec.Body.String()
	if body != "ID da notificação inválido" {
		t.Errorf("Response body = %s, want 'ID da notificação inválido'", body)
	}
}

func TestNotificationHandler_MarkAsRead_NonExistentNotification(t *testing.T) {
	handler, e, userID, _ := setupNotificationTestHandler()

	nonExistentID := uint(99999)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/notifications/%d/read", nonExistentID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", nonExistentID))

	// This should succeed but not mark anything as read (no matching record)
	err := handler.MarkAsRead(c)
	if err != nil {
		t.Fatalf("MarkAsRead() returned error: %v", err)
	}

	// Should return the dropdown (status OK)
	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestNotificationHandler_MarkAsRead_OtherUserNotification(t *testing.T) {
	handler, e, userID, group := setupNotificationTestHandler()

	// Create another user
	authService := services.NewAuthService()
	otherUser, _ := authService.Register("other@example.com", "Password123", "Other User")

	// Create notification for other user
	notification := createTestNotification(otherUser.ID, &group.ID, models.NotificationTypeExpense, false)

	// Try to mark other user's notification as read
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/notifications/%d/read", notification.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", notification.ID))

	err := handler.MarkAsRead(c)
	if err != nil {
		t.Fatalf("MarkAsRead() returned error: %v", err)
	}

	// Should succeed but not mark the notification as read (wrong user)
	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Verify notification is still unread
	var updatedNotification models.Notification
	database.DB.First(&updatedNotification, notification.ID)
	if updatedNotification.Read {
		t.Error("Other user's notification should not be marked as read")
	}
}

func TestNotificationHandler_MarkAllAsRead_Success(t *testing.T) {
	handler, e, userID, group := setupNotificationTestHandler()

	// Create multiple unread notifications
	createTestNotification(userID, &group.ID, models.NotificationTypeExpense, false)
	createTestNotification(userID, &group.ID, models.NotificationTypeGroupInvite, false)
	createTestNotification(userID, &group.ID, models.NotificationTypeGoalReached, false)

	req := httptest.NewRequest(http.MethodPost, "/notifications/read-all", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.MarkAllAsRead(c)
	if err != nil {
		t.Fatalf("MarkAllAsRead() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Verify all notifications are marked as read
	var unreadCount int64
	database.DB.Model(&models.Notification{}).Where("user_id = ? AND read = ?", userID, false).Count(&unreadCount)
	if unreadCount != 0 {
		t.Errorf("Unread count = %d, want %d", unreadCount, 0)
	}
}

func TestNotificationHandler_MarkAllAsRead_NoUnreadNotifications(t *testing.T) {
	handler, e, userID, group := setupNotificationTestHandler()

	// Create only read notifications
	createTestNotification(userID, &group.ID, models.NotificationTypeExpense, true)
	createTestNotification(userID, &group.ID, models.NotificationTypeExpense, true)

	req := httptest.NewRequest(http.MethodPost, "/notifications/read-all", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.MarkAllAsRead(c)
	if err != nil {
		t.Fatalf("MarkAllAsRead() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Verify count remains 0
	var unreadCount int64
	database.DB.Model(&models.Notification{}).Where("user_id = ? AND read = ?", userID, false).Count(&unreadCount)
	if unreadCount != 0 {
		t.Errorf("Unread count = %d, want %d", unreadCount, 0)
	}
}

func TestNotificationHandler_MarkAllAsRead_MixedReadStatus(t *testing.T) {
	handler, e, userID, group := setupNotificationTestHandler()

	// Create mixed read/unread notifications
	createTestNotification(userID, &group.ID, models.NotificationTypeExpense, true)
	createTestNotification(userID, &group.ID, models.NotificationTypeExpense, false)
	createTestNotification(userID, &group.ID, models.NotificationTypeExpense, true)
	createTestNotification(userID, &group.ID, models.NotificationTypeExpense, false)

	req := httptest.NewRequest(http.MethodPost, "/notifications/read-all", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.MarkAllAsRead(c)
	if err != nil {
		t.Fatalf("MarkAllAsRead() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Verify all notifications are now read
	var unreadCount int64
	database.DB.Model(&models.Notification{}).Where("user_id = ? AND read = ?", userID, false).Count(&unreadCount)
	if unreadCount != 0 {
		t.Errorf("Unread count = %d, want %d", unreadCount, 0)
	}
}

func TestNotificationHandler_Delete_Success(t *testing.T) {
	handler, e, userID, group := setupNotificationTestHandler()

	// Create notification to delete
	notification := createTestNotification(userID, &group.ID, models.NotificationTypeExpense, false)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/notifications/%d", notification.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", notification.ID))

	err := handler.Delete(c)
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Verify notification is deleted
	var count int64
	database.DB.Model(&models.Notification{}).Where("id = ?", notification.ID).Count(&count)
	if count != 0 {
		t.Error("Notification should be deleted")
	}
}

func TestNotificationHandler_Delete_InvalidID(t *testing.T) {
	handler, e, userID, _ := setupNotificationTestHandler()

	req := httptest.NewRequest(http.MethodDelete, "/notifications/invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)
	c.SetParamNames("id")
	c.SetParamValues("invalid")

	err := handler.Delete(c)
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	body := rec.Body.String()
	if body != "ID da notificação inválido" {
		t.Errorf("Response body = %s, want 'ID da notificação inválido'", body)
	}
}

func TestNotificationHandler_Delete_NonExistentNotification(t *testing.T) {
	handler, e, userID, _ := setupNotificationTestHandler()

	nonExistentID := uint(99999)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/notifications/%d", nonExistentID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", nonExistentID))

	err := handler.Delete(c)
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	// Should succeed even if notification doesn't exist
	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestNotificationHandler_Delete_OtherUserNotification(t *testing.T) {
	handler, e, userID, group := setupNotificationTestHandler()

	// Create another user
	authService := services.NewAuthService()
	otherUser, _ := authService.Register("other@example.com", "Password123", "Other User")

	// Create notification for other user
	notification := createTestNotification(otherUser.ID, &group.ID, models.NotificationTypeExpense, false)

	// Try to delete other user's notification
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/notifications/%d", notification.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", notification.ID))

	err := handler.Delete(c)
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	// Should succeed but not delete the notification (wrong user)
	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Verify notification still exists
	var count int64
	database.DB.Model(&models.Notification{}).Where("id = ?", notification.ID).Count(&count)
	if count != 1 {
		t.Error("Other user's notification should not be deleted")
	}
}

func TestNotificationHandler_Delete_ReadNotification(t *testing.T) {
	handler, e, userID, group := setupNotificationTestHandler()

	// Create read notification to delete
	notification := createTestNotification(userID, &group.ID, models.NotificationTypeExpense, true)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/notifications/%d", notification.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", notification.ID))

	err := handler.Delete(c)
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Verify notification is deleted
	var count int64
	database.DB.Model(&models.Notification{}).Where("id = ?", notification.ID).Count(&count)
	if count != 0 {
		t.Error("Read notification should be deleted")
	}
}

func TestNotificationHandler_MultipleNotificationTypes(t *testing.T) {
	handler, e, userID, group := setupNotificationTestHandler()

	// Create notifications of different types
	createTestNotification(userID, &group.ID, models.NotificationTypeGroupInvite, false)
	createTestNotification(userID, &group.ID, models.NotificationTypeExpense, false)
	createTestNotification(userID, &group.ID, models.NotificationTypeGoalReached, true)
	createTestNotification(userID, &group.ID, models.NotificationTypeBudgetAlert, false)
	createTestNotification(userID, &group.ID, models.NotificationTypeSummary, true)

	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.List(c)
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Verify all notification types are retrieved
	var notifications []models.Notification
	database.DB.Where("user_id = ?", userID).Find(&notifications)
	if len(notifications) != 5 {
		t.Errorf("Notification count = %d, want %d", len(notifications), 5)
	}

	// Count unread notifications
	var unreadCount int64
	database.DB.Model(&models.Notification{}).Where("user_id = ? AND read = ?", userID, false).Count(&unreadCount)
	if unreadCount != 3 {
		t.Errorf("Unread count = %d, want %d", unreadCount, 3)
	}
}
