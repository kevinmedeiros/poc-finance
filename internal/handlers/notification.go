package handlers

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/middleware"
	"poc-finance/internal/services"
)

type NotificationHandler struct {
	notificationService *services.NotificationService
}

func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{
		notificationService: services.NewNotificationService(),
	}
}

// List returns all notifications for the current user
func (h *NotificationHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)

	notifications, err := h.notificationService.GetUserNotifications(userID, 0)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao buscar notificações")
	}

	unreadCount, _ := h.notificationService.GetUnreadCount(userID)

	return c.Render(http.StatusOK, "notifications.html", map[string]interface{}{
		"notifications": notifications,
		"unreadCount":   unreadCount,
	})
}

// GetDropdown returns the notification dropdown content (HTMX partial)
func (h *NotificationHandler) GetDropdown(c echo.Context) error {
	userID := middleware.GetUserID(c)

	notifications, err := h.notificationService.GetUserNotifications(userID, 5)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao buscar notificações")
	}

	unreadCount, _ := h.notificationService.GetUnreadCount(userID)

	return c.Render(http.StatusOK, "partials/notification-dropdown.html", map[string]interface{}{
		"notifications": notifications,
		"unreadCount":   unreadCount,
	})
}

// GetBadge returns the notification badge count (HTMX partial)
func (h *NotificationHandler) GetBadge(c echo.Context) error {
	userID := middleware.GetUserID(c)

	unreadCount, _ := h.notificationService.GetUnreadCount(userID)

	return c.Render(http.StatusOK, "partials/notification-badge.html", map[string]interface{}{
		"unreadCount": unreadCount,
	})
}

// MarkAsRead marks a single notification as read
func (h *NotificationHandler) MarkAsRead(c echo.Context) error {
	userID := middleware.GetUserID(c)
	notificationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID da notificação inválido")
	}

	if err := h.notificationService.MarkAsRead(uint(notificationID), userID); err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao marcar como lida")
	}

	// Return updated dropdown
	return h.GetDropdown(c)
}

// MarkAllAsRead marks all notifications as read
func (h *NotificationHandler) MarkAllAsRead(c echo.Context) error {
	userID := middleware.GetUserID(c)

	if err := h.notificationService.MarkAllAsRead(userID); err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao marcar todas como lidas")
	}

	// Return updated dropdown
	return h.GetDropdown(c)
}

// Delete deletes a notification
func (h *NotificationHandler) Delete(c echo.Context) error {
	userID := middleware.GetUserID(c)
	notificationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID da notificação inválido")
	}

	if err := h.notificationService.DeleteNotification(uint(notificationID), userID); err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao excluir notificação")
	}

	return c.String(http.StatusOK, "")
}
