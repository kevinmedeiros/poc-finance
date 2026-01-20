package handlers

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/database"
	"poc-finance/internal/i18n"
	"poc-finance/internal/middleware"
	"poc-finance/internal/models"
	"poc-finance/internal/services"
)

type GroupHandler struct{
	groupService        *services.GroupService
	accountService      *services.AccountService
	notificationService *services.NotificationService
}

func NewGroupHandler() *GroupHandler {
	return &GroupHandler{
		groupService:        services.NewGroupService(),
		accountService:      services.NewAccountService(),
		notificationService: services.NewNotificationService(),
	}
}

func (h *GroupHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)

	groups, groupAccounts := h.getUserGroupsWithAccounts(userID)

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
	groups, groupAccounts := h.getUserGroupsWithAccounts(userID)

	return c.Render(http.StatusOK, "partials/group-list.html", map[string]interface{}{
		"groups":        groups,
		"userID":        userID,
		"groupAccounts": groupAccounts,
	})
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

	// Get invite info before accepting (to get inviter name)
	invite, _ := h.groupService.GetInviteByCode(code)

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

	// Send notification to the new member
	inviterName := "um membro"
	if invite != nil && invite.CreatedBy.Name != "" {
		inviterName = invite.CreatedBy.Name
	}
	h.notificationService.NotifyGroupInvite(userID, group, inviterName)

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
	groups, groupAccounts := h.getUserGroupsWithAccounts(userID)

	return c.Render(http.StatusOK, "partials/group-list.html", map[string]interface{}{
		"groups":        groups,
		"userID":        userID,
		"groupAccounts": groupAccounts,
	})
}

// DeleteGroup deletes a group (only admin can delete)
func (h *GroupHandler) DeleteGroup(c echo.Context) error {
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
	groups, groupAccounts := h.getUserGroupsWithAccounts(userID)

	return c.Render(http.StatusOK, "partials/group-list.html", map[string]interface{}{
		"groups":        groups,
		"userID":        userID,
		"groupAccounts": groupAccounts,
	})
}

// RemoveMember removes a member from the group (only admin can remove)
func (h *GroupHandler) RemoveMember(c echo.Context) error {
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
	groups, groupAccounts := h.getUserGroupsWithAccounts(userID)

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

// GenerateWeeklySummary generates a weekly summary notification for a group
func (h *GroupHandler) GenerateWeeklySummary(c echo.Context) error {
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
func (h *GroupHandler) GenerateMonthlySummary(c echo.Context) error {
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
		PeriodLabel:   i18n.MonthNames[now.Month()],
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

// JoinPagePublic shows the invite page without requiring authentication
// Allows users to see the invite and choose to login or register
func (h *GroupHandler) JoinPagePublic(c echo.Context) error {
	code := c.Param("code")

	invite, err := h.groupService.ValidateInvite(code)
	if err != nil {
		return c.Render(http.StatusOK, "join-group.html", map[string]interface{}{
			"error": "Convite inválido ou expirado",
		})
	}

	// Check if user is already logged in
	isLoggedIn := false
	userID := uint(0)
	cookie, err := c.Cookie("access_token")
	if err == nil && cookie.Value != "" {
		authService := services.NewAuthService()
		claims, err := authService.ValidateAccessToken(cookie.Value)
		if err == nil {
			isLoggedIn = true
			userID = claims.UserID
		}
	}

	// If logged in, check if already a member
	if isLoggedIn && h.groupService.IsGroupMember(invite.GroupID, userID) {
		return c.Render(http.StatusOK, "join-group.html", map[string]interface{}{
			"error": "Você já é membro deste grupo",
		})
	}

	return c.Render(http.StatusOK, "join-group.html", map[string]interface{}{
		"invite":     invite,
		"code":       code,
		"isLoggedIn": isLoggedIn,
	})
}

// RegisterAndJoin creates a new user account and adds them to the group
func (h *GroupHandler) RegisterAndJoin(c echo.Context) error {
	code := c.Param("code")

	// Validate invite first
	invite, err := h.groupService.ValidateInvite(code)
	if err != nil {
		return c.Render(http.StatusOK, "join-group.html", map[string]interface{}{
			"error": "Convite inválido ou expirado",
		})
	}

	var req RegisterAndJoinRequest
	if err := c.Bind(&req); err != nil {
		return c.Render(http.StatusOK, "join-group.html", map[string]interface{}{
			"invite":       invite,
			"code":         code,
			"registerError": "Dados inválidos",
			"email":        req.Email,
			"name":         req.Name,
		})
	}

	// Validate input
	if req.Email == "" || req.Password == "" || req.Name == "" {
		return c.Render(http.StatusOK, "join-group.html", map[string]interface{}{
			"invite":       invite,
			"code":         code,
			"registerError": "Todos os campos são obrigatórios",
			"email":        req.Email,
			"name":         req.Name,
		})
	}

	// Validate password strength
	if valid, errMsg := isValidGroupPassword(req.Password); !valid {
		return c.Render(http.StatusOK, "join-group.html", map[string]interface{}{
			"invite":        invite,
			"code":          code,
			"registerError": errMsg,
			"email":         req.Email,
			"name":          req.Name,
		})
	}

	// Register user
	authService := services.NewAuthService()
	user, err := authService.Register(req.Email, req.Password, req.Name)
	if err != nil {
		errorMsg := "Erro ao criar conta. Tente novamente."
		if err == services.ErrUserExists {
			errorMsg = "Este email já está cadastrado. Faça login para aceitar o convite."
		}
		return c.Render(http.StatusOK, "join-group.html", map[string]interface{}{
			"invite":       invite,
			"code":         code,
			"registerError": errorMsg,
			"email":        req.Email,
			"name":         req.Name,
		})
	}

	// Accept the invite (add user to group)
	group, err := h.groupService.AcceptInvite(code, user.ID)
	if err != nil {
		// User was created but couldn't join group - still a success for registration
		return c.Render(http.StatusOK, "join-group.html", map[string]interface{}{
			"registerSuccess": true,
			"registerError":   "Conta criada, mas houve um erro ao entrar no grupo. Faça login e tente novamente.",
		})
	}

	// Send notification
	inviterName := "um membro"
	if invite.CreatedBy.Name != "" {
		inviterName = invite.CreatedBy.Name
	}
	h.notificationService.NotifyGroupInvite(user.ID, group, inviterName)

	// Generate tokens and set cookies for auto-login
	accessToken, _ := authService.GenerateAccessToken(user)
	refreshToken, _ := authService.GenerateRefreshToken(user)

	isSecure := os.Getenv("ENV") == "production"
	c.SetCookie(&http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(services.AccessTokenDuration.Seconds()),
	})

	c.SetCookie(&http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(services.RefreshTokenDuration.Seconds()),
	})

	return c.Render(http.StatusOK, "join-group.html", map[string]interface{}{
		"success": true,
		"group":   group,
	})
}

// getUserGroupsWithAccounts fetches all groups for a user along with their joint accounts
func (h *GroupHandler) getUserGroupsWithAccounts(userID uint) ([]models.FamilyGroup, map[uint][]models.Account) {
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

	return groups, groupAccounts
}
