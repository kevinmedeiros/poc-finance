package handlers

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/middleware"
	"poc-finance/internal/services"
)

type GroupJointAccountHandler struct {
	accountService *services.AccountService
	groupService   *services.GroupService
}

func NewGroupJointAccountHandler() *GroupJointAccountHandler {
	return &GroupJointAccountHandler{
		accountService: services.NewAccountService(),
		groupService:   services.NewGroupService(),
	}
}

type CreateJointAccountRequest struct {
	Name string `form:"name"`
}

// CreateJointAccount creates a new joint account for a group
func (h *GroupJointAccountHandler) CreateJointAccount(c echo.Context) error {
	userID := middleware.GetUserID(c)
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID do grupo inválido")
	}

	var req CreateJointAccountRequest
	if err := c.Bind(&req); err != nil {
		return c.String(http.StatusBadRequest, "Dados inválidos")
	}

	if req.Name == "" {
		return c.String(http.StatusBadRequest, "Nome da conta é obrigatório")
	}

	_, err = h.accountService.CreateJointAccount(req.Name, uint(groupID), userID)
	if err != nil {
		if err == services.ErrUnauthorized {
			return c.String(http.StatusForbidden, "Você não é membro deste grupo")
		}
		return c.String(http.StatusInternalServerError, "Erro ao criar conta conjunta")
	}

	// Return updated accounts list
	accounts, _ := h.accountService.GetGroupJointAccounts(uint(groupID))
	return c.Render(http.StatusOK, "partials/joint-accounts-list.html", map[string]interface{}{
		"accounts": accounts,
		"groupID":  groupID,
		"userID":   userID,
	})
}

// DeleteJointAccount deletes a joint account
func (h *GroupJointAccountHandler) DeleteJointAccount(c echo.Context) error {
	userID := middleware.GetUserID(c)
	accountID, err := strconv.ParseUint(c.Param("accountId"), 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID da conta inválido")
	}

	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID do grupo inválido")
	}

	if err := h.accountService.DeleteJointAccount(uint(accountID), userID); err != nil {
		if err == services.ErrUnauthorized {
			return c.String(http.StatusForbidden, "Você não é membro deste grupo")
		}
		if err == services.ErrAccountNotFound {
			return c.String(http.StatusNotFound, "Conta não encontrada")
		}
		return c.String(http.StatusInternalServerError, "Erro ao excluir conta")
	}

	// Return updated accounts list
	accounts, _ := h.accountService.GetGroupJointAccounts(uint(groupID))
	return c.Render(http.StatusOK, "partials/joint-accounts-list.html", map[string]interface{}{
		"accounts": accounts,
		"groupID":  groupID,
		"userID":   userID,
	})
}
