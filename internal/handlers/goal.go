package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/middleware"
	"poc-finance/internal/services"
)

type GoalHandler struct {
	goalService    *services.GoalService
	groupService   *services.GroupService
	accountService *services.AccountService
}

func NewGoalHandler() *GoalHandler {
	return &GoalHandler{
		goalService:    services.NewGoalService(),
		groupService:   services.NewGroupService(),
		accountService: services.NewAccountService(),
	}
}

type CreateGoalRequest struct {
	Name         string  `form:"name"`
	Description  string  `form:"description"`
	TargetAmount float64 `form:"target_amount"`
	TargetDate   string  `form:"target_date"`
	AccountID    *uint   `form:"account_id"`
}

// GoalsPage returns the goals page for a group
func (h *GoalHandler) GoalsPage(c echo.Context) error {
	userID := middleware.GetUserID(c)
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID do grupo inválido")
	}

	// Verify user is a member
	if !h.groupService.IsGroupMember(uint(groupID), userID) {
		return c.String(http.StatusForbidden, "Você não é membro deste grupo")
	}

	// Get group info
	group, err := h.groupService.GetGroupByID(uint(groupID))
	if err != nil {
		return c.String(http.StatusNotFound, "Grupo não encontrado")
	}

	// Get goals
	goals, err := h.goalService.GetGroupGoals(uint(groupID), userID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao buscar metas")
	}

	// Get joint accounts for the dropdown
	accounts, _ := h.accountService.GetGroupJointAccounts(uint(groupID))

	return c.Render(http.StatusOK, "goals.html", map[string]interface{}{
		"group":    group,
		"goals":    goals,
		"accounts": accounts,
		"groupID":  groupID,
		"userID":   userID,
	})
}

// List returns all goals for a group (partial)
func (h *GoalHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID do grupo inválido")
	}

	goals, err := h.goalService.GetGroupGoals(uint(groupID), userID)
	if err != nil {
		if err == services.ErrUnauthorized {
			return c.String(http.StatusForbidden, "Você não é membro deste grupo")
		}
		return c.String(http.StatusInternalServerError, "Erro ao buscar metas")
	}

	return c.Render(http.StatusOK, "partials/goal-list.html", map[string]interface{}{
		"goals":   goals,
		"groupID": groupID,
		"userID":  userID,
	})
}

// Create creates a new goal
func (h *GoalHandler) Create(c echo.Context) error {
	userID := middleware.GetUserID(c)
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID do grupo inválido")
	}

	var req CreateGoalRequest
	if err := c.Bind(&req); err != nil {
		return c.String(http.StatusBadRequest, "Dados inválidos")
	}

	if req.Name == "" || req.TargetAmount <= 0 {
		return c.String(http.StatusBadRequest, "Nome e valor alvo são obrigatórios")
	}

	targetDate, err := time.Parse("2006-01-02", req.TargetDate)
	if err != nil {
		return c.String(http.StatusBadRequest, "Data alvo inválida")
	}

	// Handle optional account ID
	var accountID *uint
	if req.AccountID != nil && *req.AccountID > 0 {
		accountID = req.AccountID
	}

	_, err = h.goalService.CreateGoal(uint(groupID), userID, req.Name, req.Description,
		req.TargetAmount, targetDate, accountID)
	if err != nil {
		if err == services.ErrUnauthorized {
			return c.String(http.StatusForbidden, "Você não é membro deste grupo")
		}
		return c.String(http.StatusInternalServerError, "Erro ao criar meta")
	}

	// Return the full list updated
	goals, _ := h.goalService.GetGroupGoals(uint(groupID), userID)

	return c.Render(http.StatusOK, "partials/goal-list.html", map[string]interface{}{
		"goals":   goals,
		"groupID": groupID,
		"userID":  userID,
	})
}

// Delete deletes a goal
func (h *GoalHandler) Delete(c echo.Context) error {
	userID := middleware.GetUserID(c)
	goalID, err := strconv.ParseUint(c.Param("goalId"), 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID da meta inválido")
	}

	// Get goal to find group ID
	goal, err := h.goalService.GetGoalByID(uint(goalID))
	if err != nil {
		return c.String(http.StatusNotFound, "Meta não encontrada")
	}

	groupID := goal.GroupID

	if err := h.goalService.DeleteGoal(uint(goalID), userID); err != nil {
		if err == services.ErrUnauthorized {
			return c.String(http.StatusForbidden, "Você não tem permissão para deletar esta meta")
		}
		if err == services.ErrGoalNotFound {
			return c.String(http.StatusNotFound, "Meta não encontrada")
		}
		return c.String(http.StatusInternalServerError, "Erro ao deletar meta")
	}

	// Return updated list
	goals, _ := h.goalService.GetGroupGoals(groupID, userID)

	return c.Render(http.StatusOK, "partials/goal-list.html", map[string]interface{}{
		"goals":   goals,
		"groupID": groupID,
		"userID":  userID,
	})
}

// AddContribution adds a contribution to a goal
func (h *GoalHandler) AddContribution(c echo.Context) error {
	userID := middleware.GetUserID(c)
	goalID, err := strconv.ParseUint(c.Param("goalId"), 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID da meta inválido")
	}

	amount, err := strconv.ParseFloat(c.FormValue("amount"), 64)
	if err != nil || amount <= 0 {
		return c.String(http.StatusBadRequest, "Valor inválido")
	}

	_, err = h.goalService.AddContribution(uint(goalID), userID, amount)
	if err != nil {
		if err == services.ErrGoalNotFound {
			return c.String(http.StatusNotFound, "Meta não encontrada")
		}
		if err == services.ErrGoalCompleted {
			return c.String(http.StatusBadRequest, "Meta já foi concluída")
		}
		if err == services.ErrUnauthorized {
			return c.String(http.StatusForbidden, "Você não é membro deste grupo")
		}
		return c.String(http.StatusInternalServerError, "Erro ao registrar contribuição")
	}

	// Get updated goal
	goal, _ := h.goalService.GetGoalByID(uint(goalID))

	// Return updated list
	goals, _ := h.goalService.GetGroupGoals(goal.GroupID, userID)

	return c.Render(http.StatusOK, "partials/goal-list.html", map[string]interface{}{
		"goals":   goals,
		"groupID": goal.GroupID,
		"userID":  userID,
	})
}
