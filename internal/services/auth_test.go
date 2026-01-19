package services

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
	"poc-finance/internal/testutil"
)

func TestAuthService_HashPassword(t *testing.T) {
	authService := NewAuthService()

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "simple password",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  false, // bcrypt handles empty strings
		},
		{
			name:     "long password",
			password: "this-is-a-very-long-password-that-should-still-work-fine",
			wantErr:  false,
		},
		{
			name:     "special characters",
			password: "p@$$w0rd!#$%^&*()",
			wantErr:  false,
		},
		{
			name:     "unicode password",
			password: "senha123àéîõü",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := authService.HashPassword(tt.password)

			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && hash == "" {
				t.Error("HashPassword() returned empty hash")
			}

			// Hash should be different from password
			if hash == tt.password {
				t.Error("HashPassword() hash equals password")
			}
		})
	}
}

func TestAuthService_HashPassword_UniqueSalts(t *testing.T) {
	authService := NewAuthService()
	password := "samepassword"

	hash1, _ := authService.HashPassword(password)
	hash2, _ := authService.HashPassword(password)

	if hash1 == hash2 {
		t.Error("HashPassword() should produce different hashes for same password (different salts)")
	}
}

func TestAuthService_CheckPassword(t *testing.T) {
	authService := NewAuthService()

	tests := []struct {
		name     string
		password string
		valid    bool
	}{
		{
			name:     "correct password",
			password: "correctpassword",
			valid:    true,
		},
		{
			name:     "empty password",
			password: "",
			valid:    true,
		},
		{
			name:     "special characters",
			password: "p@$$w0rd!#$%",
			valid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := authService.HashPassword(tt.password)
			if err != nil {
				t.Fatalf("Failed to hash password: %v", err)
			}

			result := authService.CheckPassword(tt.password, hash)
			if result != tt.valid {
				t.Errorf("CheckPassword() = %v, want %v", result, tt.valid)
			}
		})
	}
}

func TestAuthService_CheckPassword_WrongPassword(t *testing.T) {
	authService := NewAuthService()

	hash, _ := authService.HashPassword("correctpassword")

	tests := []struct {
		name     string
		password string
	}{
		{name: "wrong password", password: "wrongpassword"},
		{name: "similar password", password: "correctpassworD"},
		{name: "empty password", password: ""},
		{name: "password with space", password: "correctpassword "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if authService.CheckPassword(tt.password, hash) {
				t.Errorf("CheckPassword() should return false for wrong password: %s", tt.password)
			}
		})
	}
}

func TestAuthService_GenerateAccessToken(t *testing.T) {
	authService := NewAuthService()

	user := &models.User{
		Email: "test@example.com",
		Name:  "Test User",
	}
	user.ID = 123

	token, err := authService.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	if token == "" {
		t.Error("GenerateAccessToken() returned empty token")
	}

	// Validate the token
	claims, err := authService.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("UserID = %v, want %v", claims.UserID, user.ID)
	}

	if claims.Email != user.Email {
		t.Errorf("Email = %v, want %v", claims.Email, user.Email)
	}

	if claims.Subject != user.Email {
		t.Errorf("Subject = %v, want %v", claims.Subject, user.Email)
	}
}

func TestAuthService_ValidateAccessToken_InvalidToken(t *testing.T) {
	authService := NewAuthService()

	tests := []struct {
		name  string
		token string
	}{
		{name: "empty token", token: ""},
		{name: "random string", token: "notavalidtoken"},
		{name: "malformed jwt", token: "header.payload.signature"},
		{name: "missing parts", token: "just.two"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := authService.ValidateAccessToken(tt.token)
			if err == nil {
				t.Error("ValidateAccessToken() should return error for invalid token")
			}
			if err != ErrTokenInvalid {
				t.Errorf("Expected ErrTokenInvalid, got %v", err)
			}
		})
	}
}

func TestAuthService_ValidateAccessToken_ExpiredToken(t *testing.T) {
	// Create a token that's already expired
	claims := &Claims{
		UserID: 123,
		Email:  "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // expired 1 hour ago
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Subject:   "test@example.com",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(JWTSecret)

	authService := NewAuthService()
	_, err := authService.ValidateAccessToken(tokenString)

	if err != ErrTokenExpired {
		t.Errorf("Expected ErrTokenExpired, got %v", err)
	}
}

func TestAuthService_ValidateAccessToken_WrongSigningMethod(t *testing.T) {
	// Create a token with a different signing method (none)
	claims := &Claims{
		UserID: 123,
		Email:  "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Use a different secret to simulate wrong signature
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte("different-secret"))

	authService := NewAuthService()
	_, err := authService.ValidateAccessToken(tokenString)

	if err != ErrTokenInvalid {
		t.Errorf("Expected ErrTokenInvalid, got %v", err)
	}
}

func TestAuthService_Register(t *testing.T) {
	// Setup test database
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	user, err := authService.Register("test@example.com", "password123", "Test User")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if user == nil {
		t.Fatal("Register() returned nil user")
	}

	if user.Email != "test@example.com" {
		t.Errorf("Email = %v, want test@example.com", user.Email)
	}

	if user.Name != "Test User" {
		t.Errorf("Name = %v, want Test User", user.Name)
	}

	if user.PasswordHash == "password123" {
		t.Error("PasswordHash should be hashed, not plain text")
	}

	// Verify user can be found in database
	var dbUser models.User
	if err := db.First(&dbUser, user.ID).Error; err != nil {
		t.Errorf("User not found in database: %v", err)
	}

	// Verify individual account was created
	var account models.Account
	if err := db.Where("user_id = ? AND type = ?", user.ID, models.AccountTypeIndividual).First(&account).Error; err != nil {
		t.Errorf("Individual account not created: %v", err)
	}

	if account.Name != "Conta Pessoal" {
		t.Errorf("Account name = %v, want Conta Pessoal", account.Name)
	}
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	// Register first user
	_, err := authService.Register("duplicate@example.com", "password123", "First User")
	if err != nil {
		t.Fatalf("First Register() error = %v", err)
	}

	// Try to register with same email
	_, err = authService.Register("duplicate@example.com", "differentpass", "Second User")
	if err != ErrUserExists {
		t.Errorf("Expected ErrUserExists, got %v", err)
	}
}

func TestAuthService_Login(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	// Register a user first
	_, err := authService.Register("login@example.com", "password123", "Login User")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Test login
	user, accessToken, refreshToken, err := authService.Login("login@example.com", "password123")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if user == nil {
		t.Fatal("Login() returned nil user")
	}

	if user.Email != "login@example.com" {
		t.Errorf("Email = %v, want login@example.com", user.Email)
	}

	if accessToken == "" {
		t.Error("Login() returned empty access token")
	}

	if refreshToken == "" {
		t.Error("Login() returned empty refresh token")
	}

	// Verify refresh token was stored in database
	var dbToken models.RefreshToken
	if err := db.Where("token = ?", refreshToken).First(&dbToken).Error; err != nil {
		t.Errorf("Refresh token not found in database: %v", err)
	}
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	// Register a user first
	_, err := authService.Register("wrong@example.com", "correctpassword", "Wrong Pass User")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Try login with wrong password
	_, _, _, err = authService.Login("wrong@example.com", "wrongpassword")
	if err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_Login_NonexistentUser(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	_, _, _, err := authService.Login("nonexistent@example.com", "password123")
	if err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_RefreshAccessToken(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	// Register and login
	_, err := authService.Register("refresh@example.com", "password123", "Refresh User")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	_, _, refreshToken, err := authService.Login("refresh@example.com", "password123")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	// Refresh the access token
	newAccessToken, err := authService.RefreshAccessToken(refreshToken)
	if err != nil {
		t.Fatalf("RefreshAccessToken() error = %v", err)
	}

	if newAccessToken == "" {
		t.Error("RefreshAccessToken() returned empty token")
	}

	// Validate the new token
	claims, err := authService.ValidateAccessToken(newAccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}

	if claims.Email != "refresh@example.com" {
		t.Errorf("Email = %v, want refresh@example.com", claims.Email)
	}
}

func TestAuthService_RefreshAccessToken_InvalidToken(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	_, err := authService.RefreshAccessToken("invalid-refresh-token")
	if err != ErrTokenInvalid {
		t.Errorf("Expected ErrTokenInvalid, got %v", err)
	}
}

func TestAuthService_RevokeRefreshToken(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	// Register and login
	_, err := authService.Register("revoke@example.com", "password123", "Revoke User")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	_, _, refreshToken, err := authService.Login("revoke@example.com", "password123")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	// Revoke the token
	err = authService.RevokeRefreshToken(refreshToken)
	if err != nil {
		t.Fatalf("RevokeRefreshToken() error = %v", err)
	}

	// Try to use the revoked token
	_, err = authService.RefreshAccessToken(refreshToken)
	if err != ErrTokenInvalid {
		t.Errorf("Expected ErrTokenInvalid after revocation, got %v", err)
	}
}

func TestAuthService_RevokeAllUserTokens(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	// Register and login multiple times
	user, err := authService.Register("revokeall@example.com", "password123", "RevokeAll User")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	_, _, refreshToken1, _ := authService.Login("revokeall@example.com", "password123")
	_, _, refreshToken2, _ := authService.Login("revokeall@example.com", "password123")
	_, _, refreshToken3, _ := authService.Login("revokeall@example.com", "password123")

	// Revoke all tokens
	err = authService.RevokeAllUserTokens(user.ID)
	if err != nil {
		t.Fatalf("RevokeAllUserTokens() error = %v", err)
	}

	// Verify all tokens are revoked
	tokens := []string{refreshToken1, refreshToken2, refreshToken3}
	for _, token := range tokens {
		_, err := authService.RefreshAccessToken(token)
		if err != ErrTokenInvalid {
			t.Errorf("Token should be invalid after RevokeAllUserTokens")
		}
	}
}

func TestAuthService_GetUserByID(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	// Register a user
	registered, err := authService.Register("getbyid@example.com", "password123", "GetByID User")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Get user by ID
	user, err := authService.GetUserByID(registered.ID)
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}

	if user.Email != "getbyid@example.com" {
		t.Errorf("Email = %v, want getbyid@example.com", user.Email)
	}
}

func TestAuthService_GetUserByID_NotFound(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	_, err := authService.GetUserByID(99999)
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestAuthService_GetUserByEmail(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	// Register a user
	_, err := authService.Register("getbyemail@example.com", "password123", "GetByEmail User")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Get user by email
	user, err := authService.GetUserByEmail("getbyemail@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail() error = %v", err)
	}

	if user.Name != "GetByEmail User" {
		t.Errorf("Name = %v, want GetByEmail User", user.Name)
	}
}

func TestAuthService_GetUserByEmail_NotFound(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	_, err := authService.GetUserByEmail("nonexistent@example.com")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestAuthService_GeneratePasswordResetToken(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	// Register a user
	_, err := authService.Register("reset@example.com", "password123", "Reset User")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Generate reset token
	token, err := authService.GeneratePasswordResetToken("reset@example.com")
	if err != nil {
		t.Fatalf("GeneratePasswordResetToken() error = %v", err)
	}

	if token == "" {
		t.Error("GeneratePasswordResetToken() returned empty token")
	}

	// Verify token was stored
	var dbToken models.PasswordResetToken
	if err := db.Where("token = ?", token).First(&dbToken).Error; err != nil {
		t.Errorf("Reset token not found in database: %v", err)
	}
}

func TestAuthService_GeneratePasswordResetToken_NonexistentUser(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	_, err := authService.GeneratePasswordResetToken("nonexistent@example.com")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestAuthService_ResetPassword(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	// Register a user
	_, err := authService.Register("resetpw@example.com", "oldpassword", "ResetPW User")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Generate reset token
	token, err := authService.GeneratePasswordResetToken("resetpw@example.com")
	if err != nil {
		t.Fatalf("GeneratePasswordResetToken() error = %v", err)
	}

	// Reset password
	err = authService.ResetPassword(token, "newpassword")
	if err != nil {
		t.Fatalf("ResetPassword() error = %v", err)
	}

	// Old password should not work
	_, _, _, err = authService.Login("resetpw@example.com", "oldpassword")
	if err != ErrInvalidCredentials {
		t.Error("Old password should not work after reset")
	}

	// New password should work
	_, _, _, err = authService.Login("resetpw@example.com", "newpassword")
	if err != nil {
		t.Errorf("Login with new password failed: %v", err)
	}
}

func TestAuthService_ResetPassword_InvalidToken(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	err := authService.ResetPassword("invalid-token", "newpassword")
	if err != ErrTokenInvalid {
		t.Errorf("Expected ErrTokenInvalid, got %v", err)
	}
}

func TestAuthService_ResetPassword_TokenReuse(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	// Register a user
	_, err := authService.Register("tokenreuse@example.com", "password123", "TokenReuse User")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Generate reset token
	token, err := authService.GeneratePasswordResetToken("tokenreuse@example.com")
	if err != nil {
		t.Fatalf("GeneratePasswordResetToken() error = %v", err)
	}

	// Use the token
	err = authService.ResetPassword(token, "newpassword1")
	if err != nil {
		t.Fatalf("ResetPassword() error = %v", err)
	}

	// Try to use the same token again
	err = authService.ResetPassword(token, "newpassword2")
	if err != ErrTokenInvalid {
		t.Errorf("Expected ErrTokenInvalid for reused token, got %v", err)
	}
}

func TestAuthService_ValidatePasswordResetToken(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	// Register a user
	registered, err := authService.Register("validatetoken@example.com", "password123", "ValidateToken User")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Generate reset token
	token, err := authService.GeneratePasswordResetToken("validatetoken@example.com")
	if err != nil {
		t.Fatalf("GeneratePasswordResetToken() error = %v", err)
	}

	// Validate the token
	user, err := authService.ValidatePasswordResetToken(token)
	if err != nil {
		t.Fatalf("ValidatePasswordResetToken() error = %v", err)
	}

	if user.ID != registered.ID {
		t.Errorf("UserID = %v, want %v", user.ID, registered.ID)
	}
}

func TestAuthService_ValidatePasswordResetToken_Invalid(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	_, err := authService.ValidatePasswordResetToken("invalid-token")
	if err != ErrTokenInvalid {
		t.Errorf("Expected ErrTokenInvalid, got %v", err)
	}
}
