package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/services"
)

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

	// Validate input
	req.Email = strings.TrimSpace(req.Email)
	req.Name = strings.TrimSpace(req.Name)

	if req.Email == "" || req.Password == "" || req.Name == "" {
		return c.Render(http.StatusOK, "register.html", map[string]interface{}{
			"error": "Todos os campos são obrigatórios",
			"email": req.Email,
			"name":  req.Name,
		})
	}

	if len(req.Password) < 6 {
		return c.Render(http.StatusOK, "register.html", map[string]interface{}{
			"error": "A senha deve ter pelo menos 6 caracteres",
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
	return c.Render(http.StatusOK, "login.html", map[string]interface{}{
		"registered": registered,
	})
}

type LoginRequest struct {
	Email    string `form:"email"`
	Password string `form:"password"`
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.Render(http.StatusOK, "login.html", map[string]interface{}{
			"error": "Dados inválidos",
			"email": req.Email,
		})
	}

	req.Email = strings.TrimSpace(req.Email)

	if req.Email == "" || req.Password == "" {
		return c.Render(http.StatusOK, "login.html", map[string]interface{}{
			"error": "Email e senha são obrigatórios",
			"email": req.Email,
		})
	}

	// Authenticate user
	_, accessToken, refreshToken, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		return c.Render(http.StatusOK, "login.html", map[string]interface{}{
			"error": "Email ou senha incorretos",
			"email": req.Email,
		})
	}

	// Set cookies
	c.SetCookie(&http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	})

	c.SetCookie(&http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	})

	return c.Redirect(http.StatusSeeOther, "/")
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
