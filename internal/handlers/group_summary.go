package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/database"
	"poc-finance/internal/middleware"
	"poc-finance/internal/models"
	"poc-finance/internal/services"
)

type GroupSummaryHandler struct {
	groupService        *services.GroupService
	accountService      *services.AccountService
	notificationService *services.NotificationService
}

func NewGroupSummaryHandler() *GroupSummaryHandler {
	return &GroupSummaryHandler{
		groupService:        services.NewGroupService(),
		accountService:      services.NewAccountService(),
		notificationService: services.NewNotificationService(),
	}
}

// GenerateWeeklySummary generates a weekly summary notification for a group
func (h *GroupSummaryHandler) GenerateWeeklySummary(c echo.Context) error {
	userID := middleware.GetUserID(c)
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID do grupo inválido")
	}

	// Verify user is a member of the group
	if !h.groupService.IsGroupMember(uint(groupID), userID) {
		return c.String(http.StatusForbidden, "Você não é membro deste grupo")
	}

	// Get group info
	group, err := h.groupService.GetGroupByID(uint(groupID))
	if err != nil {
		return c.String(http.StatusNotFound, "Grupo não encontrado")
	}

	// Get joint account IDs for this group
	accountIDs, _ := h.accountService.GetGroupJointAccountIDs(uint(groupID))

	// Calculate weekly summary (last 7 days)
	now := time.Now()
	weekStart := now.AddDate(0, 0, -7)

	// Get income for the period
	var totalIncome float64
	var incomeCount int64
	database.DB.Model(&models.Income{}).
		Where("account_id IN ? AND date >= ? AND date <= ?", accountIDs, weekStart, now).
		Select("COALESCE(SUM(net_amount), 0)").
		Scan(&totalIncome)
	database.DB.Model(&models.Income{}).
		Where("account_id IN ? AND date >= ? AND date <= ?", accountIDs, weekStart, now).
		Count(&incomeCount)

	// Get expenses for the period
	var totalExpenses float64
	var expenseCount int64
	database.DB.Model(&models.Expense{}).
		Where("account_id IN ? AND created_at >= ? AND created_at <= ? AND active = ?", accountIDs, weekStart, now, true).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalExpenses)
	database.DB.Model(&models.Expense{}).
		Where("account_id IN ? AND created_at >= ? AND created_at <= ? AND active = ?", accountIDs, weekStart, now, true).
		Count(&expenseCount)

	// Build summary data
	summaryData := services.GroupSummaryData{
		GroupName:     group.Name,
		GroupID:       group.ID,
		PeriodType:    "weekly",
		PeriodLabel:   fmt.Sprintf("Semana de %s a %s", weekStart.Format("02/01"), now.Format("02/01")),
		TotalIncome:   totalIncome,
		TotalExpenses: totalExpenses,
		Balance:       totalIncome - totalExpenses,
		ExpenseCount:  int(expenseCount),
		IncomeCount:   int(incomeCount),
	}

	// Get group members
	groupMembers, err := h.groupService.GetGroupMembers(uint(groupID))
	if err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao buscar membros do grupo")
	}

	// Send weekly summary notification
	if err := h.notificationService.NotifyWeeklySummary(summaryData, groupMembers); err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao criar notificação de resumo")
	}

	return c.String(http.StatusOK, "Resumo semanal enviado com sucesso!")
}

// GenerateMonthlySummary generates a monthly summary notification for a group
func (h *GroupSummaryHandler) GenerateMonthlySummary(c echo.Context) error {
	userID := middleware.GetUserID(c)
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID do grupo inválido")
	}

	// Verify user is a member of the group
	if !h.groupService.IsGroupMember(uint(groupID), userID) {
		return c.String(http.StatusForbidden, "Você não é membro deste grupo")
	}

	// Get group info
	group, err := h.groupService.GetGroupByID(uint(groupID))
	if err != nil {
		return c.String(http.StatusNotFound, "Grupo não encontrado")
	}

	// Get joint account IDs for this group
	accountIDs, _ := h.accountService.GetGroupJointAccountIDs(uint(groupID))

	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	// Get monthly summary
	monthlySummary := services.GetMonthlySummaryForAccounts(database.DB, year, month, accountIDs)

	// Build summary data
	summaryData := services.GroupSummaryData{
		GroupName:     group.Name,
		GroupID:       group.ID,
		PeriodType:    "monthly",
		PeriodLabel:   monthNames[now.Month()],
		TotalIncome:   monthlySummary.TotalIncomeNet,
		TotalExpenses: monthlySummary.TotalExpenses,
		Balance:       monthlySummary.Balance,
	}

	// Get group members
	groupMembers, err := h.groupService.GetGroupMembers(uint(groupID))
	if err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao buscar membros do grupo")
	}

	// Send monthly summary notification
	if err := h.notificationService.NotifyMonthlySummary(summaryData, groupMembers); err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao criar notificação de resumo")
	}

	return c.String(http.StatusOK, "Resumo mensal enviado com sucesso!")
}
