package handlers

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/middleware"
	"poc-finance/internal/services"
)

// GroupInviteHandler handles group invite operations
type GroupInviteHandler struct {
	groupService        *services.GroupService
	notificationService *services.NotificationService
	authService         *services.AuthService
}

// NewGroupInviteHandler creates a new group invite handler
func NewGroupInviteHandler() *GroupInviteHandler {
	return &GroupInviteHandler{
		groupService:        services.NewGroupService(),
		notificationService: services.NewNotificationService(),
		authService:         services.NewAuthService(),
	}
}

type RegisterAndJoinRequest struct {
	Email    string `form:"email"`
	Password string `form:"password"`
	Name     string `form:"name"`
}

// GenerateInvite creates a new invite code for a group
func (h *GroupInviteHandler) GenerateInvite(c echo.Context) error {
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
func (h *GroupInviteHandler) ListInvites(c echo.Context) error {
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
func (h *GroupInviteHandler) JoinPage(c echo.Context) error {
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
func (h *GroupInviteHandler) AcceptInvite(c echo.Context) error {
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
func (h *GroupInviteHandler) RevokeInvite(c echo.Context) error {
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

// JoinPagePublic shows the invite page without requiring authentication
// Allows users to see the invite and choose to login or register
func (h *GroupInviteHandler) JoinPagePublic(c echo.Context) error {
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
		claims, err := h.authService.ValidateAccessToken(cookie.Value)
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
func (h *GroupInviteHandler) RegisterAndJoin(c echo.Context) error {
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
			"invite":        invite,
			"code":          code,
			"registerError": "Dados inválidos",
			"email":         req.Email,
			"name":          req.Name,
		})
	}

	// Validate input
	if req.Email == "" || req.Password == "" || req.Name == "" {
		return c.Render(http.StatusOK, "join-group.html", map[string]interface{}{
			"invite":        invite,
			"code":          code,
			"registerError": "Todos os campos são obrigatórios",
			"email":         req.Email,
			"name":          req.Name,
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
	user, err := h.authService.Register(req.Email, req.Password, req.Name)
	if err != nil {
		errorMsg := "Erro ao criar conta. Tente novamente."
		if err == services.ErrUserExists {
			errorMsg = "Este email já está cadastrado. Faça login para aceitar o convite."
		}
		return c.Render(http.StatusOK, "join-group.html", map[string]interface{}{
			"invite":        invite,
			"code":          code,
			"registerError": errorMsg,
			"email":         req.Email,
			"name":          req.Name,
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
	accessToken, _ := h.authService.GenerateAccessToken(user)
	refreshToken, _ := h.authService.GenerateRefreshToken(user)

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
