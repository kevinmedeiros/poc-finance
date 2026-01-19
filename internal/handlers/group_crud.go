package handlers

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/database"
	"poc-finance/internal/middleware"
	"poc-finance/internal/models"
	"poc-finance/internal/services"
)

type GroupCrudHandler struct {
	groupService   *services.GroupService
	accountService *services.AccountService
}

func NewGroupCrudHandler() *GroupCrudHandler {
	return &GroupCrudHandler{
		groupService:   services.NewGroupService(),
		accountService: services.NewAccountService(),
	}
}

func (h *GroupCrudHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)

	var groups []models.FamilyGroup
	database.DB.
		Joins("JOIN group_members ON group_members.group_id = family_groups.id").
		Where("group_members.user_id = ? AND group_members.deleted_at IS NULL", userID).
		Preload("Members").
		Preload("Members.User").
		Find(&groups)

	// Get joint accounts for each group
	groupAccounts := make(map[uint][]models.Account)
	for _, group := range groups {
		accounts, _ := h.accountService.GetGroupJointAccounts(group.ID)
		groupAccounts[group.ID] = accounts
	}

	return c.Render(http.StatusOK, "groups.html", map[string]interface{}{
		"groups":        groups,
		"userID":        userID,
		"groupAccounts": groupAccounts,
	})
}

func (h *GroupCrudHandler) Create(c echo.Context) error {
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

	// Get joint accounts for each group
	groupAccounts := make(map[uint][]models.Account)
	for _, g := range groups {
		accounts, _ := h.accountService.GetGroupJointAccounts(g.ID)
		groupAccounts[g.ID] = accounts
	}

	return c.Render(http.StatusOK, "partials/group-list.html", map[string]interface{}{
		"groups":        groups,
		"userID":        userID,
		"groupAccounts": groupAccounts,
	})
}

func (h *GroupCrudHandler) LeaveGroup(c echo.Context) error {
	userID := middleware.GetUserID(c)
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID do grupo inválido")
	}

	if err := h.groupService.LeaveGroup(uint(groupID), userID); err != nil {
		switch err {
		case services.ErrNotGroupMember:
			return c.String(http.StatusBadRequest, "Você não é membro deste grupo")
		case services.ErrLastAdminCannotLeave:
			return c.String(http.StatusBadRequest, "Você é o único administrador e não pode sair do grupo")
		default:
			return c.String(http.StatusInternalServerError, "Erro ao sair do grupo")
		}
	}

	// Return updated list
	var groups []models.FamilyGroup
	database.DB.
		Joins("JOIN group_members ON group_members.group_id = family_groups.id").
		Where("group_members.user_id = ? AND group_members.deleted_at IS NULL", userID).
		Preload("Members").
		Preload("Members.User").
		Find(&groups)

	// Get joint accounts for each group
	groupAccounts := make(map[uint][]models.Account)
	for _, g := range groups {
		accounts, _ := h.accountService.GetGroupJointAccounts(g.ID)
		groupAccounts[g.ID] = accounts
	}

	return c.Render(http.StatusOK, "partials/group-list.html", map[string]interface{}{
		"groups":        groups,
		"userID":        userID,
		"groupAccounts": groupAccounts,
	})
}

// DeleteGroup deletes a group (only admin can delete)
func (h *GroupCrudHandler) DeleteGroup(c echo.Context) error {
	userID := middleware.GetUserID(c)
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID do grupo inválido")
	}

	if err := h.groupService.DeleteGroup(uint(groupID), userID); err != nil {
		if err == services.ErrNotGroupAdmin {
			return c.String(http.StatusForbidden, "Apenas administradores podem excluir o grupo")
		}
		return c.String(http.StatusInternalServerError, "Erro ao excluir grupo")
	}

	// Return updated list
	var groups []models.FamilyGroup
	database.DB.
		Joins("JOIN group_members ON group_members.group_id = family_groups.id").
		Where("group_members.user_id = ? AND group_members.deleted_at IS NULL", userID).
		Preload("Members").
		Preload("Members.User").
		Find(&groups)

	groupAccounts := make(map[uint][]models.Account)
	for _, g := range groups {
		accounts, _ := h.accountService.GetGroupJointAccounts(g.ID)
		groupAccounts[g.ID] = accounts
	}

	return c.Render(http.StatusOK, "partials/group-list.html", map[string]interface{}{
		"groups":        groups,
		"userID":        userID,
		"groupAccounts": groupAccounts,
	})
}

// RemoveMember removes a member from the group (only admin can remove)
func (h *GroupCrudHandler) RemoveMember(c echo.Context) error {
	userID := middleware.GetUserID(c)
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID do grupo inválido")
	}

	memberUserID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID do membro inválido")
	}

	if err := h.groupService.RemoveMember(uint(groupID), uint(memberUserID), userID); err != nil {
		switch err {
		case services.ErrNotGroupAdmin:
			return c.String(http.StatusForbidden, "Apenas administradores podem remover membros")
		case services.ErrNotGroupMember:
			return c.String(http.StatusBadRequest, "Membro não encontrado")
		default:
			return c.String(http.StatusInternalServerError, "Erro ao remover membro")
		}
	}

	// Return updated list
	var groups []models.FamilyGroup
	database.DB.
		Joins("JOIN group_members ON group_members.group_id = family_groups.id").
		Where("group_members.user_id = ? AND group_members.deleted_at IS NULL", userID).
		Preload("Members").
		Preload("Members.User").
		Find(&groups)

	groupAccounts := make(map[uint][]models.Account)
	for _, g := range groups {
		accounts, _ := h.accountService.GetGroupJointAccounts(g.ID)
		groupAccounts[g.ID] = accounts
	}

	return c.Render(http.StatusOK, "partials/group-list.html", map[string]interface{}{
		"groups":        groups,
		"userID":        userID,
		"groupAccounts": groupAccounts,
	})
}
