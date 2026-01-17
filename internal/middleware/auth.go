package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/services"
)

// Context keys for user information
const (
	UserIDKey    = "user_id"
	UserEmailKey = "user_email"
)

// AuthMiddleware creates a middleware that validates JWT access tokens
// and protects routes from unauthorized access
func AuthMiddleware(authService *services.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get access token from cookie
			accessCookie, err := c.Cookie("access_token")
			if err != nil || accessCookie.Value == "" {
				// No access token - try to refresh using refresh token
				if refreshed := tryRefreshToken(c, authService); !refreshed {
					return redirectToLogin(c)
				}
				// Token was refreshed, get the new access token
				accessCookie, _ = c.Cookie("access_token")
			}

			// Validate access token
			claims, err := authService.ValidateAccessToken(accessCookie.Value)
			if err != nil {
				if err == services.ErrTokenExpired {
					// Try to refresh the token
					if refreshed := tryRefreshToken(c, authService); !refreshed {
						return redirectToLogin(c)
					}
					// Token was refreshed, get the new access token and validate
					accessCookie, _ = c.Cookie("access_token")
					claims, err = authService.ValidateAccessToken(accessCookie.Value)
					if err != nil {
						return redirectToLogin(c)
					}
				} else {
					return redirectToLogin(c)
				}
			}

			// Store user info in context for handlers to use
			c.Set(UserIDKey, claims.UserID)
			c.Set(UserEmailKey, claims.Email)

			return next(c)
		}
	}
}

// tryRefreshToken attempts to refresh the access token using the refresh token
func tryRefreshToken(c echo.Context, authService *services.AuthService) bool {
	refreshCookie, err := c.Cookie("refresh_token")
	if err != nil || refreshCookie.Value == "" {
		return false
	}

	// Validate refresh token and get new access token
	newAccessToken, err := authService.RefreshAccessToken(refreshCookie.Value)
	if err != nil {
		// Clear invalid cookies
		clearAuthCookies(c)
		return false
	}

	// Set new access token cookie
	c.SetCookie(&http.Cookie{
		Name:     "access_token",
		Value:    newAccessToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(services.AccessTokenDuration.Seconds()),
	})

	return true
}

// clearAuthCookies removes authentication cookies
func clearAuthCookies(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	c.SetCookie(&http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

// redirectToLogin redirects the user to the login page
func redirectToLogin(c echo.Context) error {
	// For HTMX requests, return a special header to trigger full page redirect
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Redirect", "/login")
		return c.NoContent(http.StatusUnauthorized)
	}
	return c.Redirect(http.StatusFound, "/login")
}

// GetUserID extracts user ID from context (set by AuthMiddleware)
func GetUserID(c echo.Context) uint {
	if userID, ok := c.Get(UserIDKey).(uint); ok {
		return userID
	}
	return 0
}

// GetUserEmail extracts user email from context (set by AuthMiddleware)
func GetUserEmail(c echo.Context) string {
	if email, ok := c.Get(UserEmailKey).(string); ok {
		return email
	}
	return ""
}
