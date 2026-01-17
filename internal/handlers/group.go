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

type GroupHandler struct {
	groupService   *services.GroupService
	accountService *services.AccountService
}

func NewGroupHandler() *GroupHandler {
	return &GroupHandler{
		groupService:   services.NewGroupService(),
		accountService: services.NewAccountService(),
	}
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

type CreateJointAccountRequest struct {
	Name string `form:"name"`
}

// CreateJointAccount creates a new joint account for a group
func (h *GroupHandler) CreateJointAccount(c echo.Context) error {
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
func (h *GroupHandler) DeleteJointAccount(c echo.Context) error {
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

// GenerateInvite creates a new invite code for a group
func (h *GroupHandler) GenerateInvite(c echo.Context) error {
	userID := middleware.GetUserID(c)
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID do grupo inválido")
	}

	invite, err := h.groupService.GenerateInviteCode(uint(groupID), userID)
	if err != nil {
		if err == services.ErrNotGroupAdmin {
			return c.String(http.StatusForbidden, "Você não é administrador deste grupo")
		}
		return c.String(http.StatusInternalServerError, "Erro ao gerar convite")
	}

	// Build invite link
	scheme := "http"
	if c.Request().TLS != nil {
		scheme = "https"
	}
	inviteLink := fmt.Sprintf("%s://%s/groups/join/%s", scheme, c.Request().Host, invite.Code)

	return c.Render(http.StatusOK, "partials/invite-modal.html", map[string]interface{}{
		"invite":     invite,
		"inviteLink": inviteLink,
	})
}

// ListInvites shows all active invites for a group
func (h *GroupHandler) ListInvites(c echo.Context) error {
	userID := middleware.GetUserID(c)
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID do grupo inválido")
	}

	invites, err := h.groupService.GetGroupInvites(uint(groupID), userID)
	if err != nil {
		if err == services.ErrNotGroupAdmin {
			return c.String(http.StatusForbidden, "Você não é administrador deste grupo")
		}
		return c.String(http.StatusInternalServerError, "Erro ao buscar convites")
	}

	group, _ := h.groupService.GetGroupByID(uint(groupID))

	// Build invite links
	scheme := "http"
	if c.Request().TLS != nil {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s/groups/join/", scheme, c.Request().Host)

	return c.Render(http.StatusOK, "partials/invite-list.html", map[string]interface{}{
		"invites": invites,
		"group":   group,
		"baseURL": baseURL,
	})
}

// JoinPage shows the page to accept an invite
func (h *GroupHandler) JoinPage(c echo.Context) error {
	code := c.Param("code")

	invite, err := h.groupService.ValidateInvite(code)
	if err != nil {
		return c.Render(http.StatusOK, "join-group.html", map[string]interface{}{
			"error": "Convite inválido ou expirado",
		})
	}

	return c.Render(http.StatusOK, "join-group.html", map[string]interface{}{
		"invite": invite,
		"code":   code,
	})
}

// AcceptInvite adds the user to the group
func (h *GroupHandler) AcceptInvite(c echo.Context) error {
	userID := middleware.GetUserID(c)
	code := c.Param("code")

	group, err := h.groupService.AcceptInvite(code, userID)
	if err != nil {
		errorMsg := "Erro ao aceitar convite"
		switch err {
		case services.ErrInviteNotFound:
			errorMsg = "Convite não encontrado"
		case services.ErrInviteExpired:
			errorMsg = "Convite expirado"
		case services.ErrInviteInvalid:
			errorMsg = "Convite inválido"
		case services.ErrInviteMaxUsed:
			errorMsg = "Convite atingiu o limite de usos"
		case services.ErrAlreadyMember:
			errorMsg = "Você já é membro deste grupo"
		}
		return c.Render(http.StatusOK, "join-group.html", map[string]interface{}{
			"error": errorMsg,
		})
	}

	return c.Render(http.StatusOK, "join-group.html", map[string]interface{}{
		"success": true,
		"group":   group,
	})
}

// RevokeInvite revokes an invite
func (h *GroupHandler) RevokeInvite(c echo.Context) error {
	userID := middleware.GetUserID(c)
	inviteID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID do convite inválido")
	}

	if err := h.groupService.RevokeInvite(uint(inviteID), userID); err != nil {
		if err == services.ErrNotGroupAdmin {
			return c.String(http.StatusForbidden, "Você não é administrador deste grupo")
		}
		return c.String(http.StatusInternalServerError, "Erro ao revogar convite")
	}

	return c.String(http.StatusOK, "")
}

// LeaveGroup removes the current user from a group
func (h *GroupHandler) LeaveGroup(c echo.Context) error {
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

// Dashboard shows the consolidated dashboard for a group's joint accounts
func (h *GroupHandler) Dashboard(c echo.Context) error {
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
	}

	return c.Render(http.StatusOK, "group-dashboard.html", data)
}
