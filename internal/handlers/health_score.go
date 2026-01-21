package handlers

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/middleware"
	"poc-finance/internal/services"
)

type HealthScoreHandler struct {
	healthScoreService *services.HealthScoreService
	accountService     *services.AccountService
	groupService       *services.GroupService
}

func NewHealthScoreHandler() *HealthScoreHandler {
	return &HealthScoreHandler{
		healthScoreService: services.NewHealthScoreService(),
		accountService:     services.NewAccountService(),
		groupService:       services.NewGroupService(),
	}
}

// Index renders the main health score page for a user
func (h *HealthScoreHandler) Index(c echo.Context) error {
	userID := middleware.GetUserID(c)

	// Get user's account IDs
	accountIDs, err := h.accountService.GetUserAccountIDs(userID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao buscar contas")
	}

	// Calculate current health score
	score, err := h.healthScoreService.CalculateUserScore(userID, accountIDs)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao calcular score de saúde")
	}

	// Get score history (last 6 months)
	history, err := h.healthScoreService.GetScoreHistory(&userID, nil, 6)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao buscar histórico")
	}

	// Get personalized recommendations
	recommendations, err := h.healthScoreService.GetRecommendations(score)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao gerar recomendações")
	}

	return c.Render(http.StatusOK, "health_score.html", map[string]interface{}{
		"score":           score,
		"history":         history,
		"recommendations": recommendations,
		"userID":          userID,
	})
}

// GetUserScore returns the current user score (for HTMX partial updates)
func (h *HealthScoreHandler) GetUserScore(c echo.Context) error {
	userID := middleware.GetUserID(c)

	// Get user's account IDs
	accountIDs, err := h.accountService.GetUserAccountIDs(userID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao buscar contas")
	}

	// Calculate current health score
	score, err := h.healthScoreService.CalculateUserScore(userID, accountIDs)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao calcular score de saúde")
	}

	// Get personalized recommendations
	recommendations, err := h.healthScoreService.GetRecommendations(score)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao gerar recomendações")
	}

	return c.Render(http.StatusOK, "partials/health-score-card.html", map[string]interface{}{
		"score":           score,
		"recommendations": recommendations,
	})
}

// GetScoreHistory returns historical score data as JSON for trend charts
func (h *HealthScoreHandler) GetScoreHistory(c echo.Context) error {
	userID := middleware.GetUserID(c)

	// Get months parameter (default 6)
	months := 6
	if monthsParam := c.QueryParam("months"); monthsParam != "" {
		if m, err := strconv.Atoi(monthsParam); err == nil && m > 0 && m <= 12 {
			months = m
		}
	}

	// Get score history
	history, err := h.healthScoreService.GetScoreHistory(&userID, nil, months)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Erro ao buscar histórico",
		})
	}

	return c.JSON(http.StatusOK, history)
}

// GroupScorePage renders the health score page for a family group
func (h *HealthScoreHandler) GroupScorePage(c echo.Context) error {
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

	// Calculate group health score
	score, err := h.healthScoreService.CalculateGroupScore(uint(groupID))
	if err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao calcular score de saúde do grupo")
	}

	// Get score history (last 6 months)
	history, err := h.healthScoreService.GetScoreHistory(nil, &group.ID, 6)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao buscar histórico")
	}

	// Get personalized recommendations
	recommendations, err := h.healthScoreService.GetRecommendations(score)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao gerar recomendações")
	}

	return c.Render(http.StatusOK, "health_score.html", map[string]interface{}{
		"score":           score,
		"history":         history,
		"recommendations": recommendations,
		"group":           group,
		"groupID":         groupID,
		"userID":          userID,
	})
}

// GetGroupScoreHistory returns historical score data for a group as JSON
func (h *HealthScoreHandler) GetGroupScoreHistory(c echo.Context) error {
	userID := middleware.GetUserID(c)
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "ID do grupo inválido",
		})
	}

	// Verify user is a member
	if !h.groupService.IsGroupMember(uint(groupID), userID) {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Você não é membro deste grupo",
		})
	}

	// Get months parameter (default 6)
	months := 6
	if monthsParam := c.QueryParam("months"); monthsParam != "" {
		if m, err := strconv.Atoi(monthsParam); err == nil && m > 0 && m <= 12 {
			months = m
		}
	}

	// Get score history
	gid := uint(groupID)
	history, err := h.healthScoreService.GetScoreHistory(nil, &gid, months)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Erro ao buscar histórico",
		})
	}

	return c.JSON(http.StatusOK, history)
}
