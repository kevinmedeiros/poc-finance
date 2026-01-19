package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/database"
	"poc-finance/internal/middleware"
	"poc-finance/internal/models"
	"poc-finance/internal/services"
)

type GroupDashboardHandler struct {
	groupService   *services.GroupService
	accountService *services.AccountService
}

func NewGroupDashboardHandler() *GroupDashboardHandler {
	return &GroupDashboardHandler{
		groupService:   services.NewGroupService(),
		accountService: services.NewAccountService(),
	}
}

// Dashboard shows the consolidated dashboard for a group's joint accounts
func (h *GroupDashboardHandler) Dashboard(c echo.Context) error {
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

	// Get monthly summary for joint accounts
	currentSummary := services.GetMonthlySummaryForAccounts(database.DB, year, month, accountIDs)

	// 6-month projection
	var monthSummaries []services.MonthlySummary
	for i := 0; i < 6; i++ {
		m := month + i
		y := year
		if m > 12 {
			m -= 12
			y++
		}
		monthSummaries = append(monthSummaries, services.GetMonthlySummaryForAccounts(database.DB, y, m, accountIDs))
	}

	// Get account balances
	accountBalances, _ := h.accountService.GetGroupJointAccountsWithBalances(uint(groupID))

	// Calculate totals
	var totalIncome, totalExpenses, totalBalance float64
	for _, ab := range accountBalances {
		totalIncome += ab.TotalIncome
		totalExpenses += ab.TotalExpenses
		totalBalance += ab.Balance
	}

	// Upcoming bills for joint accounts
	upcomingBills := getUpcomingBillsForAccounts(now, accountIDs)

	// Get group members
	var members []models.GroupMember
	database.DB.Preload("User").Where("group_id = ?", groupID).Find(&members)

	// Get member contributions
	memberContributions := services.GetMemberContributions(database.DB, uint(groupID), accountIDs)

	// HOLISTIC SUMMARY - All accounts (individual + joint) for the family group
	allAccountIDs, _ := h.accountService.GetAllGroupAccountIDs(uint(groupID))
	holisticSummary := services.GetMonthlySummaryForAccounts(database.DB, year, month, allAccountIDs)
	allAccountBalances, _ := h.accountService.GetAllGroupAccountsWithBalances(uint(groupID))

	// Calculate holistic totals
	var holisticIncome, holisticExpenses, holisticBalance float64
	for _, ab := range allAccountBalances {
		holisticIncome += ab.TotalIncome
		holisticExpenses += ab.TotalExpenses
		holisticBalance += ab.Balance
	}

	// 6-month holistic projection
	var holisticMonthSummaries []services.MonthlySummary
	for i := 0; i < 6; i++ {
		m := month + i
		y := year
		if m > 12 {
			m -= 12
			y++
		}
		holisticMonthSummaries = append(holisticMonthSummaries, services.GetMonthlySummaryForAccounts(database.DB, y, m, allAccountIDs))
	}

	data := map[string]interface{}{
		"group":               group,
		"members":             members,
		"currentMonth":        currentSummary,
		"monthSummaries":      monthSummaries,
		"accountBalances":     accountBalances,
		"totalIncome":         totalIncome,
		"totalExpenses":       totalExpenses,
		"totalBalance":        totalBalance,
		"upcomingBills":       upcomingBills,
		"memberContributions": memberContributions,
		"now":                 now,
		// Holistic summary data
		"holisticSummary":        holisticSummary,
		"holisticIncome":         holisticIncome,
		"holisticExpenses":       holisticExpenses,
		"holisticBalance":        holisticBalance,
		"holisticMonthSummaries": holisticMonthSummaries,
		"allAccountBalances":     allAccountBalances,
	}

	return c.Render(http.StatusOK, "group-dashboard.html", data)
}
