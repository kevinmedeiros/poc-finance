package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/database"
	"poc-finance/internal/middleware"
	"poc-finance/internal/models"
)

type GroupHandler struct{}

func NewGroupHandler() *GroupHandler {
	return &GroupHandler{}
}

type CreateGroupRequest struct {
	Name        string `form:"name"`
	Description string `form:"description"`
}

func (h *GroupHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)

	var groups []models.FamilyGroup
	database.DB.
		Joins("JOIN group_members ON group_members.group_id = family_groups.id").
		Where("group_members.user_id = ? AND group_members.deleted_at IS NULL", userID).
		Preload("Members").
		Preload("Members.User").
		Find(&groups)

	return c.Render(http.StatusOK, "groups.html", map[string]interface{}{
		"groups": groups,
		"userID": userID,
	})
}

func (h *GroupHandler) Create(c echo.Context) error {
	userID := middleware.GetUserID(c)

	var req CreateGroupRequest
	if err := c.Bind(&req); err != nil {
		return c.String(http.StatusBadRequest, "Dados inválidos")
	}

	if req.Name == "" {
		return c.String(http.StatusBadRequest, "Nome do grupo é obrigatório")
	}

	// Create the group
	group := models.FamilyGroup{
		Name:        req.Name,
		Description: req.Description,
		CreatedByID: userID,
	}

	if err := database.DB.Create(&group).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao criar grupo")
	}

	// Add creator as admin member
	member := models.GroupMember{
		GroupID: group.ID,
		UserID:  userID,
		Role:    "admin",
	}

	if err := database.DB.Create(&member).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao adicionar membro")
	}

	// Return updated list
	var groups []models.FamilyGroup
	database.DB.
		Joins("JOIN group_members ON group_members.group_id = family_groups.id").
		Where("group_members.user_id = ? AND group_members.deleted_at IS NULL", userID).
		Preload("Members").
		Preload("Members.User").
		Find(&groups)

	return c.Render(http.StatusOK, "partials/group-list.html", map[string]interface{}{
		"groups": groups,
		"userID": userID,
	})
}
