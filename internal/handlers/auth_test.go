package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/database"
	"poc-finance/internal/services"
	"poc-finance/internal/testutil"
)

func TestIsValidPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		valid    bool
		errMsg   string
	}{
		{
			name:     "valid password",
			password: "Password123",
			valid:    true,
			errMsg:   "",
		},
		{
			name:     "too short",
			password: "Pass1",
			valid:    false,
			errMsg:   "A senha deve ter pelo menos 8 caracteres",
		},
		{
			name:     "no uppercase",
			password: "password123",
			valid:    false,
			errMsg:   "A senha deve conter letras maiúsculas, minúsculas e números",
		},
		{
			name:     "no lowercase",
			password: "PASSWORD123",
			valid:    false,
			errMsg:   "A senha deve conter letras maiúsculas, minúsculas e números",
		},
		{
			name:     "no numbers",
			password: "PasswordABC",
			valid:    false,
			errMsg:   "A senha deve conter letras maiúsculas, minúsculas e números",
		},
		{
			name:     "only numbers",
			password: "12345678",
			valid:    false,
			errMsg:   "A senha deve conter letras maiúsculas, minúsculas e números",
		},
		{
			name:     "only lowercase",
			password: "password",
			valid:    false,
			errMsg:   "A senha deve conter letras maiúsculas, minúsculas e números",
		},
		{
			name:     "only uppercase",
			password: "PASSWORD",
			valid:    false,
			errMsg:   "A senha deve conter letras maiúsculas, minúsculas e números",
		},
		{
			name:     "special characters allowed",
			password: "Password123!@#",
			valid:    true,
			errMsg:   "",
		},
		{
			name:     "exactly 8 characters",
			password: "Pass123A",
			valid:    true,
			errMsg:   "",
		},
		{
			name:     "7 characters too short",
			password: "Pass12A",
			valid:    false,
			errMsg:   "A senha deve ter pelo menos 8 caracteres",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, errMsg := isValidPassword(tt.password)

			if valid != tt.valid {
				t.Errorf("isValidPassword(%q) valid = %v, want %v", tt.password, valid, tt.valid)
			}

			if errMsg != tt.errMsg {
				t.Errorf("isValidPassword(%q) errMsg = %q, want %q", tt.password, errMsg, tt.errMsg)
			}
		})
	}
}

func setupTestHandler() (*AuthHandler, *echo.Echo) {
	db := testutil.SetupTestDB()
	database.DB = db

	e := echo.New()
	handler := NewAuthHandler()
	return handler, e
}

func TestAuthHandler_Register_Success(t *testing.T) {
	handler, e := setupTestHandler()

	form := url.Values{}
	form.Set("email", "test@example.com")
	form.Set("password", "Password123")
	form.Set("name", "Test User")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Register(c)
	if err != nil {
		t.Fatalf("Register() returned error: %v", err)
	}

	// Should redirect to login on success
	if rec.Code != http.StatusSeeOther {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusSeeOther)
	}

	location := rec.Header().Get("Location")
	if location != "/login?registered=1" {
		t.Errorf("Location = %s, want /login?registered=1", location)
	}
}

func TestAuthHandler_Register_DuplicateEmail(t *testing.T) {
	handler, e := setupTestHandler()

	// Register first user
	authService := services.NewAuthService()
	authService.Register("duplicate@example.com", "Password123", "First User")

	// Try to register with same email
	form := url.Values{}
	form.Set("email", "duplicate@example.com")
	form.Set("password", "Password456")
	form.Set("name", "Second User")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Need to set up a mock renderer for this test to work fully
	// For now, just verify no panic occurs
	_ = handler.Register(c)

	// Handler will render template with error, status should be OK (form re-render)
	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d (re-render with error)", rec.Code, http.StatusOK)
	}
}

func TestAuthHandler_Register_WeakPassword(t *testing.T) {
	handler, e := setupTestHandler()

	form := url.Values{}
	form.Set("email", "weak@example.com")
	form.Set("password", "weak")
	form.Set("name", "Weak Password User")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler.Register(c)

	// Should re-render form with error (status 200)
	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuthHandler_Register_EmptyFields(t *testing.T) {
	handler, e := setupTestHandler()

	tests := []struct {
		name     string
		email    string
		password string
		userName string
	}{
		{"empty email", "", "Password123", "Test User"},
		{"empty password", "test@example.com", "", "Test User"},
		{"empty name", "test@example.com", "Password123", ""},
		{"all empty", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{}
			form.Set("email", tt.email)
			form.Set("password", tt.password)
			form.Set("name", tt.userName)

			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			_ = handler.Register(c)

			// Should re-render form with error
			if rec.Code != http.StatusOK {
				t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
			}
		})
	}
}

func TestAuthHandler_Login_Success(t *testing.T) {
	handler, e := setupTestHandler()

	// Register a user first
	authService := services.NewAuthService()
	authService.Register("login@example.com", "Password123", "Login User")

	form := url.Values{}
	form.Set("email", "login@example.com")
	form.Set("password", "Password123")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Login(c)
	if err != nil {
		t.Fatalf("Login() returned error: %v", err)
	}

	// Should redirect to home
	if rec.Code != http.StatusSeeOther {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusSeeOther)
	}

	location := rec.Header().Get("Location")
	if location != "/" {
		t.Errorf("Location = %s, want /", location)
	}

	// Should set cookies
	cookies := rec.Result().Cookies()
	foundAccess := false
	foundRefresh := false
	for _, cookie := range cookies {
		if cookie.Name == "access_token" && cookie.Value != "" {
			foundAccess = true
			if !cookie.HttpOnly {
				t.Error("access_token should be HttpOnly")
			}
		}
		if cookie.Name == "refresh_token" && cookie.Value != "" {
			foundRefresh = true
			if !cookie.HttpOnly {
				t.Error("refresh_token should be HttpOnly")
			}
		}
	}

	if !foundAccess {
		t.Error("access_token cookie not set")
	}
	if !foundRefresh {
		t.Error("refresh_token cookie not set")
	}
}

func TestAuthHandler_Login_WithRedirect(t *testing.T) {
	handler, e := setupTestHandler()

	// Register a user first
	authService := services.NewAuthService()
	authService.Register("redirect@example.com", "Password123", "Redirect User")

	form := url.Values{}
	form.Set("email", "redirect@example.com")
	form.Set("password", "Password123")
	form.Set("redirect", "/dashboard")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler.Login(c)

	location := rec.Header().Get("Location")
	if location != "/dashboard" {
		t.Errorf("Location = %s, want /dashboard", location)
	}
}

func TestAuthHandler_Login_OpenRedirectProtection(t *testing.T) {
	handler, e := setupTestHandler()

	// Register a user first
	authService := services.NewAuthService()
	authService.Register("security@example.com", "Password123", "Security User")

	tests := []struct {
		name        string
		redirect    string
		expectedLoc string
	}{
		{"normal redirect", "/dashboard", "/dashboard"},
		{"external url rejected", "https://evil.com", "/"},
		{"protocol-relative rejected", "//evil.com", "/"},
		{"contains scheme rejected", "javascript://alert(1)", "/"},
		{"empty redirect", "", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{}
			form.Set("email", "security@example.com")
			form.Set("password", "Password123")
			form.Set("redirect", tt.redirect)

			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			_ = handler.Login(c)

			location := rec.Header().Get("Location")
			if location != tt.expectedLoc {
				t.Errorf("Location = %s, want %s", location, tt.expectedLoc)
			}
		})
	}
}

func TestAuthHandler_Login_WrongCredentials(t *testing.T) {
	handler, e := setupTestHandler()

	// Register a user first
	authService := services.NewAuthService()
	authService.Register("wrong@example.com", "Password123", "Wrong User")

	form := url.Values{}
	form.Set("email", "wrong@example.com")
	form.Set("password", "WrongPassword123")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler.Login(c)

	// Should re-render form with error
	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuthHandler_Login_EmptyFields(t *testing.T) {
	handler, e := setupTestHandler()

	tests := []struct {
		name     string
		email    string
		password string
	}{
		{"empty email", "", "Password123"},
		{"empty password", "test@example.com", ""},
		{"both empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{}
			form.Set("email", tt.email)
			form.Set("password", tt.password)

			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			_ = handler.Login(c)

			if rec.Code != http.StatusOK {
				t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
			}
		})
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	handler, e := setupTestHandler()

	// Login first to get tokens
	authService := services.NewAuthService()
	authService.Register("logout@example.com", "Password123", "Logout User")
	_, _, refreshToken, _ := authService.Login("logout@example.com", "Password123")

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: refreshToken,
	})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Logout(c)
	if err != nil {
		t.Fatalf("Logout() returned error: %v", err)
	}

	// Should redirect to login
	if rec.Code != http.StatusSeeOther {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusSeeOther)
	}

	location := rec.Header().Get("Location")
	if location != "/login" {
		t.Errorf("Location = %s, want /login", location)
	}

	// Cookies should be cleared
	cookies := rec.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "access_token" || cookie.Name == "refresh_token" {
			if cookie.MaxAge != -1 {
				t.Errorf("%s MaxAge = %d, want -1 (delete)", cookie.Name, cookie.MaxAge)
			}
		}
	}
}

func TestAuthHandler_ResetPassword_Success(t *testing.T) {
	handler, e := setupTestHandler()

	// Register a user and generate reset token
	authService := services.NewAuthService()
	authService.Register("reset@example.com", "OldPassword123", "Reset User")
	token, _ := authService.GeneratePasswordResetToken("reset@example.com")

	form := url.Values{}
	form.Set("token", token)
	form.Set("password", "NewPassword123")
	form.Set("password_confirm", "NewPassword123")

	req := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.ResetPassword(c)
	if err != nil {
		t.Fatalf("ResetPassword() returned error: %v", err)
	}

	// Should redirect to login with success
	if rec.Code != http.StatusSeeOther {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusSeeOther)
	}

	location := rec.Header().Get("Location")
	if location != "/login?reset=1" {
		t.Errorf("Location = %s, want /login?reset=1", location)
	}
}

func TestAuthHandler_ResetPassword_PasswordMismatch(t *testing.T) {
	handler, e := setupTestHandler()

	authService := services.NewAuthService()
	authService.Register("mismatch@example.com", "OldPassword123", "Mismatch User")
	token, _ := authService.GeneratePasswordResetToken("mismatch@example.com")

	form := url.Values{}
	form.Set("token", token)
	form.Set("password", "NewPassword123")
	form.Set("password_confirm", "DifferentPassword123")

	req := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler.ResetPassword(c)

	// Should re-render form with error
	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuthHandler_ResetPassword_WeakPassword(t *testing.T) {
	handler, e := setupTestHandler()

	authService := services.NewAuthService()
	authService.Register("weakreset@example.com", "OldPassword123", "Weak Reset User")
	token, _ := authService.GeneratePasswordResetToken("weakreset@example.com")

	form := url.Values{}
	form.Set("token", token)
	form.Set("password", "weak")
	form.Set("password_confirm", "weak")

	req := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler.ResetPassword(c)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuthHandler_ResetPassword_InvalidToken(t *testing.T) {
	handler, e := setupTestHandler()

	form := url.Values{}
	form.Set("token", "invalid-token")
	form.Set("password", "NewPassword123")
	form.Set("password_confirm", "NewPassword123")

	req := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler.ResetPassword(c)

	// Should re-render with invalid token error
	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuthHandler_ResetPassword_EmptyToken(t *testing.T) {
	handler, e := setupTestHandler()

	form := url.Values{}
	form.Set("token", "")
	form.Set("password", "NewPassword123")
	form.Set("password_confirm", "NewPassword123")

	req := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler.ResetPassword(c)

	// Should redirect to forgot password
	if rec.Code != http.StatusSeeOther {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusSeeOther)
	}

	location := rec.Header().Get("Location")
	if location != "/forgot-password" {
		t.Errorf("Location = %s, want /forgot-password", location)
	}
}

func TestAuthHandler_ForgotPassword_PreventEnumeration(t *testing.T) {
	handler, e := setupTestHandler()

	// Register one user
	authService := services.NewAuthService()
	authService.Register("exists@example.com", "Password123", "Exists User")

	tests := []struct {
		name  string
		email string
	}{
		{"existing email", "exists@example.com"},
		{"nonexistent email", "notexists@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{}
			form.Set("email", tt.email)

			req := httptest.NewRequest(http.MethodPost, "/forgot-password", strings.NewReader(form.Encode()))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			_ = handler.ForgotPassword(c)

			// Both should return the same status to prevent enumeration
			if rec.Code != http.StatusOK {
				t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
			}
		})
	}
}

func TestAuthHandler_Register_XSSPrevention(t *testing.T) {
	handler, e := setupTestHandler()

	form := url.Values{}
	form.Set("email", "xss@example.com")
	form.Set("password", "Password123")
	form.Set("name", "<script>alert('xss')</script>")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler.Register(c)

	// Should redirect on success (XSS characters should be escaped)
	if rec.Code != http.StatusSeeOther {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusSeeOther)
	}

	// Verify the user was created with escaped name
	authService := services.NewAuthService()
	user, err := authService.GetUserByEmail("xss@example.com")
	if err != nil {
		t.Fatalf("User should be created: %v", err)
	}

	// Name should be HTML escaped
	if strings.Contains(user.Name, "<script>") {
		t.Error("Name should have script tags escaped")
	}
}
