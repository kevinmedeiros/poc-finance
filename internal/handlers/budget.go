package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/middleware"
	"poc-finance/internal/services"
)

type BudgetHandler struct {
	budgetService *services.BudgetService
	groupService  *services.GroupService
}

func NewBudgetHandler() *BudgetHandler {
	return &BudgetHandler{
		budgetService: services.NewBudgetService(),
		groupService:  services.NewGroupService(),
	}
}

type CreateBudgetRequest struct {
	Name    string `form:"name"`
	Year    int    `form:"year"`
	Month   int    `form:"month"`
	GroupID *uint  `form:"group_id"`
}

type AddCategoryRequest struct {
	Category string  `form:"category"`
	Limit    float64 `form:"limit"`
}

type UpdateCategoryRequest struct {
	Category string  `form:"category"`
	Limit    float64 `form:"limit"`
}

// BudgetsPage returns the budgets page for individual user
func (h *BudgetHandler) BudgetsPage(c echo.Context) error {
	userID := middleware.GetUserID(c)

	// Get current month/year
	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())

	// Get user budgets
	budgets, err := h.budgetService.GetUserBudgets(userID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao buscar orçamentos")
	}

	// Get user's groups for the dropdown
	groups, _ := h.groupService.GetUserGroups(userID)

	return c.Render(http.StatusOK, "budgets.html", map[string]interface{}{
		"budgets":      budgets,
		"groups":       groups,
		"currentYear":  currentYear,
		"currentMonth": currentMonth,
		"categories":   getExpenseCategories(),
		"userID":       userID,
	})
}

// GroupBudgetsPage returns the budgets page for a group
func (h *BudgetHandler) GroupBudgetsPage(c echo.Context) error {
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

	// Get group budgets
	budgets, err := h.budgetService.GetGroupBudgets(uint(groupID), userID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao buscar orçamentos")
	}

	// Get current month/year
	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())

	return c.Render(http.StatusOK, "budgets.html", map[string]interface{}{
		"budgets":      budgets,
		"group":        group,
		"groupID":      groupID,
		"currentYear":  currentYear,
		"currentMonth": currentMonth,
		"categories":   getExpenseCategories(),
		"userID":       userID,
	})
}

// List returns all budgets (partial for HTMX)
func (h *BudgetHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	groupIDStr := c.QueryParam("group_id")

	var budgets []interface{}
	var err error

	if groupIDStr != "" {
		groupID, parseErr := strconv.ParseUint(groupIDStr, 10, 32)
		if parseErr != nil {
			return c.String(http.StatusBadRequest, "ID do grupo inválido")
		}
		budgets, err = h.budgetService.GetGroupBudgets(uint(groupID), userID)
	} else {
		budgets, err = h.budgetService.GetUserBudgets(userID)
	}

	if err != nil {
		if err == services.ErrUnauthorized {
			return c.String(http.StatusForbidden, "Você não tem permissão para acessar estes orçamentos")
		}
		return c.String(http.StatusInternalServerError, "Erro ao buscar orçamentos")
	}

	return c.Render(http.StatusOK, "partials/budget-list.html", map[string]interface{}{
		"budgets": budgets,
		"userID":  userID,
	})
}

// Create creates a new budget
func (h *BudgetHandler) Create(c echo.Context) error {
	userID := middleware.GetUserID(c)

	var req CreateBudgetRequest
	if err := c.Bind(&req); err != nil {
		return c.String(http.StatusBadRequest, "Dados inválidos")
	}

	if req.Name == "" {
		return c.String(http.StatusBadRequest, "Nome é obrigatório")
	}

	if req.Year == 0 {
		req.Year = time.Now().Year()
	}
	if req.Month == 0 {
		req.Month = int(time.Now().Month())
	}

	// Create empty budget (categories will be added separately)
	var categories []struct {
		Category string
		Limit    float64
	}

	_, err := h.budgetService.CreateBudget(userID, req.GroupID, req.Year, req.Month, req.Name, categories)
	if err != nil {
		if err == services.ErrUnauthorized {
			return c.String(http.StatusForbidden, "Você não tem permissão para criar orçamento para este grupo")
		}
		if err == services.ErrInvalidBudgetMonth || err == services.ErrInvalidBudgetYear {
			return c.String(http.StatusBadRequest, err.Error())
		}
		return c.String(http.StatusInternalServerError, "Erro ao criar orçamento")
	}

	// Return updated list
	var budgets interface{}
	if req.GroupID != nil {
		budgets, _ = h.budgetService.GetGroupBudgets(*req.GroupID, userID)
	} else {
		budgets, _ = h.budgetService.GetUserBudgets(userID)
	}

	return c.Render(http.StatusOK, "partials/budget-list.html", map[string]interface{}{
		"budgets": budgets,
		"userID":  userID,
	})
}

// Delete deletes a budget
func (h *BudgetHandler) Delete(c echo.Context) error {
	userID := middleware.GetUserID(c)
	budgetID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID do orçamento inválido")
	}

	// Get budget to find group ID (for returning updated list)
	budget, err := h.budgetService.GetBudgetByID(uint(budgetID), userID)
	if err != nil {
		if err == services.ErrBudgetNotFound {
			return c.String(http.StatusNotFound, "Orçamento não encontrado")
		}
		return c.String(http.StatusForbidden, "Você não tem permissão para acessar este orçamento")
	}

	if err := h.budgetService.DeleteBudget(uint(budgetID), userID); err != nil {
		if err == services.ErrUnauthorized {
			return c.String(http.StatusForbidden, "Você não tem permissão para deletar este orçamento")
		}
		if err == services.ErrBudgetNotFound {
			return c.String(http.StatusNotFound, "Orçamento não encontrado")
		}
		return c.String(http.StatusInternalServerError, "Erro ao deletar orçamento")
	}

	// Return updated list
	var budgets interface{}
	if budget.GroupID != nil {
		budgets, _ = h.budgetService.GetGroupBudgets(*budget.GroupID, userID)
	} else {
		budgets, _ = h.budgetService.GetUserBudgets(userID)
	}

	return c.Render(http.StatusOK, "partials/budget-list.html", map[string]interface{}{
		"budgets": budgets,
		"userID":  userID,
	})
}

// AddCategory adds a category to a budget
func (h *BudgetHandler) AddCategory(c echo.Context) error {
	userID := middleware.GetUserID(c)
	budgetID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID do orçamento inválido")
	}

	var req AddCategoryRequest
	if err := c.Bind(&req); err != nil {
		return c.String(http.StatusBadRequest, "Dados inválidos")
	}

	if req.Category == "" || req.Limit <= 0 {
		return c.String(http.StatusBadRequest, "Categoria e limite são obrigatórios")
	}

	_, err = h.budgetService.AddCategory(uint(budgetID), userID, req.Category, req.Limit)
	if err != nil {
		if err == services.ErrUnauthorized {
			return c.String(http.StatusForbidden, "Você não tem permissão para modificar este orçamento")
		}
		if err == services.ErrBudgetNotFound {
			return c.String(http.StatusNotFound, "Orçamento não encontrado")
		}
		return c.String(http.StatusInternalServerError, "Erro ao adicionar categoria")
	}

	// Get updated budget
	budget, _ := h.budgetService.GetBudgetByID(uint(budgetID), userID)

	// Return updated budget detail
	return c.Render(http.StatusOK, "partials/budget-detail.html", map[string]interface{}{
		"budget": budget,
		"userID": userID,
	})
}

// UpdateCategory updates a budget category
func (h *BudgetHandler) UpdateCategory(c echo.Context) error {
	userID := middleware.GetUserID(c)
	budgetID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID do orçamento inválido")
	}
	categoryID, err := strconv.ParseUint(c.Param("catId"), 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID da categoria inválido")
	}

	var req UpdateCategoryRequest
	if err := c.Bind(&req); err != nil {
		return c.String(http.StatusBadRequest, "Dados inválidos")
	}

	if req.Category == "" || req.Limit <= 0 {
		return c.String(http.StatusBadRequest, "Categoria e limite são obrigatórios")
	}

	err = h.budgetService.UpdateCategory(uint(categoryID), userID, req.Category, req.Limit)
	if err != nil {
		if err == services.ErrUnauthorized {
			return c.String(http.StatusForbidden, "Você não tem permissão para modificar este orçamento")
		}
		if err == services.ErrCategoryNotFound {
			return c.String(http.StatusNotFound, "Categoria não encontrada")
		}
		return c.String(http.StatusInternalServerError, "Erro ao atualizar categoria")
	}

	// Get updated budget
	budget, _ := h.budgetService.GetBudgetByID(uint(budgetID), userID)

	// Return updated budget detail
	return c.Render(http.StatusOK, "partials/budget-detail.html", map[string]interface{}{
		"budget": budget,
		"userID": userID,
	})
}

// DeleteCategory deletes a budget category
func (h *BudgetHandler) DeleteCategory(c echo.Context) error {
	userID := middleware.GetUserID(c)
	budgetID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID do orçamento inválido")
	}
	categoryID, err := strconv.ParseUint(c.Param("catId"), 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID da categoria inválido")
	}

	err = h.budgetService.DeleteCategory(uint(categoryID), userID)
	if err != nil {
		if err == services.ErrUnauthorized {
			return c.String(http.StatusForbidden, "Você não tem permissão para modificar este orçamento")
		}
		if err == services.ErrCategoryNotFound {
			return c.String(http.StatusNotFound, "Categoria não encontrada")
		}
		return c.String(http.StatusInternalServerError, "Erro ao deletar categoria")
	}

	// Get updated budget
	budget, _ := h.budgetService.GetBudgetByID(uint(budgetID), userID)

	// Return updated budget detail
	return c.Render(http.StatusOK, "partials/budget-detail.html", map[string]interface{}{
		"budget": budget,
		"userID": userID,
	})
}

// CopyFromPreviousMonth copies a budget from the previous month
func (h *BudgetHandler) CopyFromPreviousMonth(c echo.Context) error {
	userID := middleware.GetUserID(c)

	// Get target year and month from query params
	yearStr := c.QueryParam("year")
	monthStr := c.QueryParam("month")
	groupIDStr := c.QueryParam("group_id")

	year, err := strconv.Atoi(yearStr)
	if err != nil || year == 0 {
		year = time.Now().Year()
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month == 0 {
		month = int(time.Now().Month())
	}

	var groupID *uint
	if groupIDStr != "" {
		gid, err := strconv.ParseUint(groupIDStr, 10, 32)
		if err == nil {
			gidUint := uint(gid)
			groupID = &gidUint
		}
	}

	_, err = h.budgetService.CopyFromPreviousMonth(userID, groupID, year, month)
	if err != nil {
		if err == services.ErrUnauthorized {
			return c.String(http.StatusForbidden, "Você não tem permissão para criar orçamento para este grupo")
		}
		if err == services.ErrBudgetNotFound {
			return c.String(http.StatusNotFound, "Nenhum orçamento encontrado no mês anterior")
		}
		return c.String(http.StatusInternalServerError, "Erro ao copiar orçamento")
	}

	// Return updated list
	var budgets interface{}
	if groupID != nil {
		budgets, _ = h.budgetService.GetGroupBudgets(*groupID, userID)
	} else {
		budgets, _ = h.budgetService.GetUserBudgets(userID)
	}

	return c.Render(http.StatusOK, "partials/budget-list.html", map[string]interface{}{
		"budgets": budgets,
		"userID":  userID,
	})
}
