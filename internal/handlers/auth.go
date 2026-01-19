package handlers

import (
	"errors"
	"html"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/services"
)

// isProduction returns true if ENV is set to "production"
func isProduction() bool {
	return os.Getenv("ENV") == "production"
}

// isValidPassword checks password complexity requirements
func isValidPassword(password string) (bool, string) {
	if len(password) < 8 {
		return false, "A senha deve ter pelo menos 8 caracteres"
	}
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)

	if !hasUpper || !hasLower || !hasNumber {
		return false, "A senha deve conter letras maiúsculas, minúsculas e números"
	}
	return true, ""
}

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		authService: services.NewAuthService(),
	}
}

type RegisterRequest struct {
	Email    string `form:"email"`
	Password string `form:"password"`
	Name     string `form:"name"`
}

func (h *AuthHandler) RegisterPage(c echo.Context) error {
	return c.Render(http.StatusOK, "register.html", nil)
}

func (h *AuthHandler) Register(c echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.Render(http.StatusOK, "register.html", map[string]interface{}{
			"error": "Dados inválidos",
			"email": req.Email,
			"name":  req.Name,
		})
	}

	// Validate and sanitize input
	req.Email = strings.TrimSpace(req.Email)
	req.Name = html.EscapeString(strings.TrimSpace(req.Name))

	if req.Email == "" || req.Password == "" || req.Name == "" {
		return c.Render(http.StatusOK, "register.html", map[string]interface{}{
			"error": "Todos os campos são obrigatórios",
			"email": req.Email,
			"name":  req.Name,
		})
	}

	// Validate password strength
	if valid, errMsg := isValidPassword(req.Password); !valid {
		return c.Render(http.StatusOK, "register.html", map[string]interface{}{
			"error": errMsg,
			"email": req.Email,
			"name":  req.Name,
		})
	}

	// Register user
	_, err := h.authService.Register(req.Email, req.Password, req.Name)
	if err != nil {
		if errors.Is(err, services.ErrUserExists) {
			return c.Render(http.StatusOK, "register.html", map[string]interface{}{
				"error": "Este email já está cadastrado",
				"email": req.Email,
				"name":  req.Name,
			})
		}
		return c.Render(http.StatusOK, "register.html", map[string]interface{}{
			"error": "Erro ao criar conta. Tente novamente.",
			"email": req.Email,
			"name":  req.Name,
		})
	}

	// Redirect to login page with success message
	return c.Redirect(http.StatusSeeOther, "/login?registered=1")
}

func (h *AuthHandler) LoginPage(c echo.Context) error {
	registered := c.QueryParam("registered") == "1"
	reset := c.QueryParam("reset") == "1"
	redirect := c.QueryParam("redirect")
	return c.Render(http.StatusOK, "login.html", map[string]interface{}{
		"registered": registered,
		"reset":      reset,
		"redirect":   redirect,
	})
}

type LoginRequest struct {
	Email    string `form:"email"`
	Password string `form:"password"`
	Redirect string `form:"redirect"`
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.Render(http.StatusOK, "login.html", map[string]interface{}{
			"error":    "Dados inválidos",
			"email":    req.Email,
			"redirect": req.Redirect,
		})
	}

	req.Email = strings.TrimSpace(req.Email)

	if req.Email == "" || req.Password == "" {
		return c.Render(http.StatusOK, "login.html", map[string]interface{}{
			"error":    "Email e senha são obrigatórios",
			"email":    req.Email,
			"redirect": req.Redirect,
		})
	}

	// Authenticate user
	_, accessToken, refreshToken, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		if errors.Is(err, services.ErrAccountLocked) {
			return c.Render(http.StatusOK, "login.html", map[string]interface{}{
				"error":    "Conta bloqueada temporariamente devido a múltiplas tentativas de login. Tente novamente em 15 minutos.",
				"email":    req.Email,
				"redirect": req.Redirect,
			})
		}
		return c.Render(http.StatusOK, "login.html", map[string]interface{}{
			"error":    "Email ou senha incorretos",
			"email":    req.Email,
			"redirect": req.Redirect,
		})
	}

	// Set cookies with proper security flags
	c.SetCookie(&http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   isProduction(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(services.AccessTokenDuration.Seconds()),
	})

	c.SetCookie(&http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   isProduction(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(services.RefreshTokenDuration.Seconds()),
	})

	// Redirect to specified URL or home (with open redirect protection)
	redirectURL := "/"
	if req.Redirect != "" &&
		strings.HasPrefix(req.Redirect, "/") &&
		!strings.HasPrefix(req.Redirect, "//") &&
		!strings.Contains(req.Redirect, "://") {
		redirectURL = req.Redirect
	}
	return c.Redirect(http.StatusSeeOther, redirectURL)
}

func (h *AuthHandler) Logout(c echo.Context) error {
	// Get refresh token to revoke
	cookie, err := c.Cookie("refresh_token")
	if err == nil && cookie.Value != "" {
		h.authService.RevokeRefreshToken(cookie.Value)
	}

	// Clear cookies
	c.SetCookie(&http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	c.SetCookie(&http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	return c.Redirect(http.StatusSeeOther, "/login")
}

func (h *AuthHandler) ForgotPasswordPage(c echo.Context) error {
	return c.Render(http.StatusOK, "forgot-password.html", nil)
}

type ForgotPasswordRequest struct {
	Email string `form:"email"`
}

func (h *AuthHandler) ForgotPassword(c echo.Context) error {
	var req ForgotPasswordRequest
	if err := c.Bind(&req); err != nil {
		return c.Render(http.StatusOK, "forgot-password.html", map[string]interface{}{
			"error": "Dados inválidos",
		})
	}

	req.Email = strings.TrimSpace(req.Email)

	if req.Email == "" {
		return c.Render(http.StatusOK, "forgot-password.html", map[string]interface{}{
			"error": "Email é obrigatório",
		})
	}

	// In production, this would:
	// 1. Generate reset token: h.authService.GeneratePasswordResetToken(req.Email)
	// 2. Send email with reset link
	// For now, password reset is disabled until email is implemented

	// Always show success message to prevent email enumeration
	return c.Render(http.StatusOK, "forgot-password.html", map[string]interface{}{
		"success": true,
	})
}

func (h *AuthHandler) ResetPasswordPage(c echo.Context) error {
	token := c.QueryParam("token")
	if token == "" {
		return c.Redirect(http.StatusSeeOther, "/forgot-password")
	}

	// Validate token before showing form
	_, err := h.authService.ValidatePasswordResetToken(token)
	if err != nil {
		return c.Render(http.StatusOK, "reset-password.html", map[string]interface{}{
			"error":        "Link de recuperação inválido ou expirado",
			"invalidToken": true,
		})
	}

	return c.Render(http.StatusOK, "reset-password.html", map[string]interface{}{
		"token": token,
	})
}

type ResetPasswordRequest struct {
	Token           string `form:"token"`
	Password        string `form:"password"`
	PasswordConfirm string `form:"password_confirm"`
}

func (h *AuthHandler) ResetPassword(c echo.Context) error {
	var req ResetPasswordRequest
	if err := c.Bind(&req); err != nil {
		return c.Render(http.StatusOK, "reset-password.html", map[string]interface{}{
			"error": "Dados inválidos",
			"token": req.Token,
		})
	}

	if req.Token == "" {
		return c.Redirect(http.StatusSeeOther, "/forgot-password")
	}

	if req.Password == "" || req.PasswordConfirm == "" {
		return c.Render(http.StatusOK, "reset-password.html", map[string]interface{}{
			"error": "Todos os campos são obrigatórios",
			"token": req.Token,
		})
	}

	// Validate password strength
	if valid, errMsg := isValidPassword(req.Password); !valid {
		return c.Render(http.StatusOK, "reset-password.html", map[string]interface{}{
			"error": errMsg,
			"token": req.Token,
		})
	}

	if req.Password != req.PasswordConfirm {
		return c.Render(http.StatusOK, "reset-password.html", map[string]interface{}{
			"error": "As senhas não coincidem",
			"token": req.Token,
		})
	}

	// Reset password
	if err := h.authService.ResetPassword(req.Token, req.Password); err != nil {
		if errors.Is(err, services.ErrTokenExpired) || errors.Is(err, services.ErrTokenInvalid) {
			return c.Render(http.StatusOK, "reset-password.html", map[string]interface{}{
				"error":        "Link de recuperação inválido ou expirado",
				"invalidToken": true,
			})
		}
		return c.Render(http.StatusOK, "reset-password.html", map[string]interface{}{
			"error": "Erro ao redefinir senha. Tente novamente.",
			"token": req.Token,
		})
	}

	// Redirect to login with success message
	return c.Redirect(http.StatusSeeOther, "/login?reset=1")
}
