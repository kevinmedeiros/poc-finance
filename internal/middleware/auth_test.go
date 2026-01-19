package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/database"
	"poc-finance/internal/services"
	"poc-finance/internal/testutil"
)

func TestGetUserID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	tests := []struct {
		name     string
		setup    func(c echo.Context)
		expected uint
	}{
		{
			name: "user id present",
			setup: func(c echo.Context) {
				c.Set(UserIDKey, uint(123))
			},
			expected: 123,
		},
		{
			name: "user id not present",
			setup: func(c echo.Context) {
				// Don't set anything
			},
			expected: 0,
		},
		{
			name: "wrong type in context",
			setup: func(c echo.Context) {
				c.Set(UserIDKey, "not-a-uint")
			},
			expected: 0,
		},
		{
			name: "int instead of uint",
			setup: func(c echo.Context) {
				c.Set(UserIDKey, int(123))
			},
			expected: 0, // Wrong type, should return 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset context
			c = e.NewContext(req, rec)
			tt.setup(c)

			result := GetUserID(c)
			if result != tt.expected {
				t.Errorf("GetUserID() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestGetUserEmail(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	tests := []struct {
		name     string
		setup    func(c echo.Context)
		expected string
	}{
		{
			name: "email present",
			setup: func(c echo.Context) {
				c.Set(UserEmailKey, "test@example.com")
			},
			expected: "test@example.com",
		},
		{
			name: "email not present",
			setup: func(c echo.Context) {
				// Don't set anything
			},
			expected: "",
		},
		{
			name: "wrong type in context",
			setup: func(c echo.Context) {
				c.Set(UserEmailKey, 123)
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset context
			c = e.NewContext(req, rec)
			tt.setup(c)

			result := GetUserEmail(c)
			if result != tt.expected {
				t.Errorf("GetUserEmail() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestAuthMiddleware_NoToken(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	e := echo.New()
	authService := services.NewAuthService()

	// Create a test handler that should not be reached
	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	}

	// Apply middleware
	middlewareHandler := AuthMiddleware(authService)(handler)

	// Make request without any cookies
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := middlewareHandler(c)

	// Should redirect to login
	if err != nil {
		t.Fatalf("Middleware returned error: %v", err)
	}

	if rec.Code != http.StatusFound {
		t.Errorf("Status code = %d, want %d (redirect)", rec.Code, http.StatusFound)
	}

	location := rec.Header().Get("Location")
	if location != "/login" {
		t.Errorf("Location = %s, want /login", location)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := services.NewAuthService()

	// Register and login a user
	user, _ := authService.Register("test@example.com", "password123", "Test User")
	_, accessToken, _, _ := authService.Login("test@example.com", "password123")

	e := echo.New()

	// Create a test handler
	handlerCalled := false
	handler := func(c echo.Context) error {
		handlerCalled = true
		// Verify user info was set in context
		userID := GetUserID(c)
		userEmail := GetUserEmail(c)

		if userID != user.ID {
			t.Errorf("UserID in context = %d, want %d", userID, user.ID)
		}
		if userEmail != user.Email {
			t.Errorf("UserEmail in context = %s, want %s", userEmail, user.Email)
		}

		return c.String(http.StatusOK, "success")
	}

	// Apply middleware
	middlewareHandler := AuthMiddleware(authService)(handler)

	// Make request with access token cookie
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  "access_token",
		Value: accessToken,
	})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := middlewareHandler(c)

	if err != nil {
		t.Fatalf("Middleware returned error: %v", err)
	}

	if !handlerCalled {
		t.Error("Handler was not called")
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	e := echo.New()
	authService := services.NewAuthService()

	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	}

	middlewareHandler := AuthMiddleware(authService)(handler)

	// Make request with invalid access token
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  "access_token",
		Value: "invalid-token",
	})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := middlewareHandler(c)

	if err != nil {
		t.Fatalf("Middleware returned error: %v", err)
	}

	if rec.Code != http.StatusFound {
		t.Errorf("Status code = %d, want %d (redirect)", rec.Code, http.StatusFound)
	}
}

func TestTryRefreshToken(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := services.NewAuthService()

	// Register and login a user
	authService.Register("refresh@example.com", "password123", "Refresh User")
	_, _, refreshToken, _ := authService.Login("refresh@example.com", "password123")

	e := echo.New()

	// Test with valid refresh token
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: refreshToken,
	})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	result := tryRefreshToken(c, authService)

	if !result {
		t.Error("tryRefreshToken should return true for valid refresh token")
	}

	// Check that a new access token cookie was set
	cookies := rec.Result().Cookies()
	foundAccessToken := false
	for _, cookie := range cookies {
		if cookie.Name == "access_token" {
			foundAccessToken = true
			if cookie.Value == "" {
				t.Error("Access token cookie should have a value")
			}
			if !cookie.HttpOnly {
				t.Error("Access token cookie should be HttpOnly")
			}
		}
	}

	if !foundAccessToken {
		t.Error("New access token cookie should have been set")
	}
}

func TestTryRefreshToken_InvalidToken(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := services.NewAuthService()

	e := echo.New()

	// Test with invalid refresh token
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: "invalid-token",
	})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	result := tryRefreshToken(c, authService)

	if result {
		t.Error("tryRefreshToken should return false for invalid refresh token")
	}

	// Check that auth cookies were cleared
	cookies := rec.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "access_token" || cookie.Name == "refresh_token" {
			if cookie.MaxAge != -1 {
				t.Errorf("Cookie %s should be cleared (MaxAge=-1)", cookie.Name)
			}
		}
	}
}

func TestTryRefreshToken_NoToken(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := services.NewAuthService()

	e := echo.New()

	// Test with no refresh token
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	result := tryRefreshToken(c, authService)

	if result {
		t.Error("tryRefreshToken should return false when no refresh token is present")
	}
}

func TestAuthMiddleware_HTMXRequest(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	e := echo.New()
	authService := services.NewAuthService()

	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	}

	middlewareHandler := AuthMiddleware(authService)(handler)

	// Make HTMX request without token
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := middlewareHandler(c)

	if err != nil {
		t.Fatalf("Middleware returned error: %v", err)
	}

	// For HTMX, should return 401 with HX-Redirect header
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Status code = %d, want %d for HTMX", rec.Code, http.StatusUnauthorized)
	}

	hxRedirect := rec.Header().Get("HX-Redirect")
	if hxRedirect != "/login" {
		t.Errorf("HX-Redirect = %s, want /login", hxRedirect)
	}
}

func TestClearAuthCookies(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	clearAuthCookies(c)

	cookies := rec.Result().Cookies()

	if len(cookies) != 2 {
		t.Errorf("Expected 2 cookies to be cleared, got %d", len(cookies))
	}

	for _, cookie := range cookies {
		if cookie.Name != "access_token" && cookie.Name != "refresh_token" {
			t.Errorf("Unexpected cookie: %s", cookie.Name)
		}
		if cookie.MaxAge != -1 {
			t.Errorf("Cookie %s MaxAge = %d, want -1 (delete)", cookie.Name, cookie.MaxAge)
		}
		if cookie.Value != "" {
			t.Errorf("Cookie %s should have empty value", cookie.Name)
		}
	}
}

func TestRedirectToLogin_Normal(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := redirectToLogin(c)

	if err != nil {
		t.Fatalf("redirectToLogin returned error: %v", err)
	}

	if rec.Code != http.StatusFound {
		t.Errorf("Status code = %d, want %d", rec.Code, http.StatusFound)
	}

	location := rec.Header().Get("Location")
	if location != "/login" {
		t.Errorf("Location = %s, want /login", location)
	}
}

func TestRedirectToLogin_HTMX(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := redirectToLogin(c)

	if err != nil {
		t.Fatalf("redirectToLogin returned error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Status code = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	hxRedirect := rec.Header().Get("HX-Redirect")
	if hxRedirect != "/login" {
		t.Errorf("HX-Redirect = %s, want /login", hxRedirect)
	}
}
