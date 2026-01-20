package services

import (
	"testing"
	"time"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
	"poc-finance/internal/testutil"
)

func TestUser_IsLocked(t *testing.T) {
	tests := []struct {
		name        string
		lockedUntil *time.Time
		want        bool
	}{
		{
			name:        "no lockout",
			lockedUntil: nil,
			want:        false,
		},
		{
			name:        "locked in future",
			lockedUntil: func() *time.Time { t := time.Now().Add(10 * time.Minute); return &t }(),
			want:        true,
		},
		{
			name:        "lockout expired",
			lockedUntil: func() *time.Time { t := time.Now().Add(-10 * time.Minute); return &t }(),
			want:        false,
		},
		{
			name:        "lockout just expired",
			lockedUntil: func() *time.Time { t := time.Now().Add(-1 * time.Second); return &t }(),
			want:        false,
		},
		{
			name:        "lockout expires soon",
			lockedUntil: func() *time.Time { t := time.Now().Add(1 * time.Second); return &t }(),
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &models.User{
				Email:        "test@example.com",
				Name:         "Test User",
				PasswordHash: "hash",
				LockedUntil:  tt.lockedUntil,
			}

			got := user.IsLocked()
			if got != tt.want {
				t.Errorf("IsLocked() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthService_Login_FailedAttemptsTracking(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	// Create test user
	user, err := authService.Register("test@example.com", "Password123", "Test User")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name                    string
		attemptNumber           int
		password                string
		wantErr                 error
		wantFailedAttempts      int
		wantLocked              bool
		wantLastFailedLoginAt   bool
	}{
		{
			name:                  "first failed attempt",
			attemptNumber:         1,
			password:              "WrongPassword",
			wantErr:               ErrInvalidCredentials,
			wantFailedAttempts:    1,
			wantLocked:            false,
			wantLastFailedLoginAt: true,
		},
		{
			name:                  "second failed attempt",
			attemptNumber:         2,
			password:              "WrongPassword",
			wantErr:               ErrInvalidCredentials,
			wantFailedAttempts:    2,
			wantLocked:            false,
			wantLastFailedLoginAt: true,
		},
		{
			name:                  "third failed attempt",
			attemptNumber:         3,
			password:              "WrongPassword",
			wantErr:               ErrInvalidCredentials,
			wantFailedAttempts:    3,
			wantLocked:            false,
			wantLastFailedLoginAt: true,
		},
		{
			name:                  "fourth failed attempt",
			attemptNumber:         4,
			password:              "WrongPassword",
			wantErr:               ErrInvalidCredentials,
			wantFailedAttempts:    4,
			wantLocked:            false,
			wantLastFailedLoginAt: true,
		},
		{
			name:                  "fifth failed attempt - account locked",
			attemptNumber:         5,
			password:              "WrongPassword",
			wantErr:               ErrAccountLocked,
			wantFailedAttempts:    5,
			wantLocked:            true,
			wantLastFailedLoginAt: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := authService.Login(user.Email, tt.password)

			if err != tt.wantErr {
				t.Errorf("Login() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Reload user from database
			var updatedUser models.User
			database.DB.First(&updatedUser, user.ID)

			if updatedUser.FailedLoginAttempts != tt.wantFailedAttempts {
				t.Errorf("FailedLoginAttempts = %d, want %d", updatedUser.FailedLoginAttempts, tt.wantFailedAttempts)
			}

			if updatedUser.IsLocked() != tt.wantLocked {
				t.Errorf("IsLocked() = %v, want %v", updatedUser.IsLocked(), tt.wantLocked)
			}

			if tt.wantLastFailedLoginAt && updatedUser.LastFailedLoginAt == nil {
				t.Error("LastFailedLoginAt should be set")
			}

			if tt.wantLocked && updatedUser.LockedUntil == nil {
				t.Error("LockedUntil should be set when account is locked")
			}
		})
	}
}

func TestAuthService_Login_AccountLocked(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	// Create test user
	user, err := authService.Register("locked@example.com", "Password123", "Locked User")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Lock the account
	lockUntil := time.Now().Add(15 * time.Minute)
	database.DB.Model(&user).Updates(map[string]interface{}{
		"failed_login_attempts": 5,
		"locked_until":          lockUntil,
	})

	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{
			name:     "locked with correct password",
			password: "Password123",
			wantErr:  ErrAccountLocked,
		},
		{
			name:     "locked with wrong password",
			password: "WrongPassword",
			wantErr:  ErrAccountLocked,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := authService.Login(user.Email, tt.password)

			if err != tt.wantErr {
				t.Errorf("Login() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthService_Login_LockoutExpired(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	// Create test user
	user, err := authService.Register("expired@example.com", "Password123", "Expired User")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Set lockout that has expired
	lockUntil := time.Now().Add(-1 * time.Minute)
	lastFailed := time.Now().Add(-2 * time.Minute)
	database.DB.Model(&user).Updates(map[string]interface{}{
		"failed_login_attempts": 5,
		"locked_until":          lockUntil,
		"last_failed_login_at":  lastFailed,
	})

	// Try to login with correct password - should succeed
	loggedInUser, accessToken, refreshToken, err := authService.Login(user.Email, "Password123")
	if err != nil {
		t.Fatalf("Login() should succeed after lockout expired, got error: %v", err)
	}

	if loggedInUser == nil {
		t.Error("Login() should return user")
	}

	if accessToken == "" {
		t.Error("Login() should return access token")
	}

	if refreshToken == "" {
		t.Error("Login() should return refresh token")
	}

	// Verify lockout fields are reset
	var updatedUser models.User
	database.DB.First(&updatedUser, user.ID)

	if updatedUser.FailedLoginAttempts != 0 {
		t.Errorf("FailedLoginAttempts = %d, want 0", updatedUser.FailedLoginAttempts)
	}

	if updatedUser.LockedUntil != nil {
		t.Error("LockedUntil should be nil after successful login")
	}

	if updatedUser.LastFailedLoginAt != nil {
		t.Error("LastFailedLoginAt should be nil after successful login")
	}
}

func TestAuthService_Login_ResetFailedAttemptsOnSuccess(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	// Create test user
	user, err := authService.Register("reset@example.com", "Password123", "Reset User")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Make some failed attempts
	for i := 0; i < 3; i++ {
		authService.Login(user.Email, "WrongPassword")
	}

	// Verify failed attempts were tracked
	var userBeforeSuccess models.User
	database.DB.First(&userBeforeSuccess, user.ID)
	if userBeforeSuccess.FailedLoginAttempts != 3 {
		t.Errorf("FailedLoginAttempts before success = %d, want 3", userBeforeSuccess.FailedLoginAttempts)
	}

	// Successful login should reset failed attempts
	loggedInUser, _, _, err := authService.Login(user.Email, "Password123")
	if err != nil {
		t.Fatalf("Login() should succeed with correct password: %v", err)
	}

	if loggedInUser == nil {
		t.Error("Login() should return user")
	}

	// Verify all lockout fields are reset
	var updatedUser models.User
	database.DB.First(&updatedUser, user.ID)

	if updatedUser.FailedLoginAttempts != 0 {
		t.Errorf("FailedLoginAttempts = %d, want 0 after successful login", updatedUser.FailedLoginAttempts)
	}

	if updatedUser.LockedUntil != nil {
		t.Error("LockedUntil should be nil after successful login")
	}

	if updatedUser.LastFailedLoginAt != nil {
		t.Error("LastFailedLoginAt should be nil after successful login")
	}
}

func TestAuthService_Login_LockoutDuration(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	// Create test user
	user, err := authService.Register("duration@example.com", "Password123", "Duration User")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Make 5 failed attempts to trigger lockout
	for i := 0; i < 5; i++ {
		authService.Login(user.Email, "WrongPassword")
	}

	// Get user and check lockout duration
	var lockedUser models.User
	database.DB.First(&lockedUser, user.ID)

	if lockedUser.LockedUntil == nil {
		t.Fatal("LockedUntil should be set")
	}

	// Verify lockout duration is approximately 15 minutes
	lockoutDuration := lockedUser.LockedUntil.Sub(time.Now())
	expectedDuration := LockoutDuration

	// Allow 5 second tolerance for test execution time
	if lockoutDuration < expectedDuration-5*time.Second || lockoutDuration > expectedDuration+5*time.Second {
		t.Errorf("Lockout duration = %v, want approximately %v", lockoutDuration, expectedDuration)
	}
}

func TestAuthService_Login_MaxFailedAttemptsConstant(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	// Create test user
	user, err := authService.Register("max@example.com", "Password123", "Max User")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Make MaxFailedAttempts - 1 failed attempts
	for i := 0; i < MaxFailedAttempts-1; i++ {
		_, _, _, err := authService.Login(user.Email, "WrongPassword")
		if err != ErrInvalidCredentials {
			t.Fatalf("Attempt %d should return ErrInvalidCredentials, got: %v", i+1, err)
		}
	}

	// Verify account is not locked yet
	var userBeforeLock models.User
	database.DB.First(&userBeforeLock, user.ID)
	if userBeforeLock.IsLocked() {
		t.Error("Account should not be locked before MaxFailedAttempts")
	}

	// One more failed attempt should lock the account
	_, _, _, err = authService.Login(user.Email, "WrongPassword")
	if err != ErrAccountLocked {
		t.Errorf("Attempt %d should return ErrAccountLocked, got: %v", MaxFailedAttempts, err)
	}

	// Verify account is locked
	var lockedUser models.User
	database.DB.First(&lockedUser, user.ID)
	if !lockedUser.IsLocked() {
		t.Error("Account should be locked after MaxFailedAttempts")
	}

	if lockedUser.FailedLoginAttempts != MaxFailedAttempts {
		t.Errorf("FailedLoginAttempts = %d, want %d", lockedUser.FailedLoginAttempts, MaxFailedAttempts)
	}
}

func TestAuthService_Login_MultipleUsers_IndependentLockout(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	// Create two test users
	user1, err := authService.Register("user1@example.com", "Password123", "User 1")
	if err != nil {
		t.Fatalf("Failed to create user1: %v", err)
	}

	user2, err := authService.Register("user2@example.com", "Password123", "User 2")
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}

	// Lock user1 with failed attempts
	for i := 0; i < 5; i++ {
		authService.Login(user1.Email, "WrongPassword")
	}

	// Verify user1 is locked
	var lockedUser1 models.User
	database.DB.First(&lockedUser1, user1.ID)
	if !lockedUser1.IsLocked() {
		t.Error("User1 should be locked")
	}

	// Verify user2 is not affected
	var user2Check models.User
	database.DB.First(&user2Check, user2.ID)
	if user2Check.FailedLoginAttempts != 0 {
		t.Error("User2 should have 0 failed attempts")
	}
	if user2Check.IsLocked() {
		t.Error("User2 should not be locked")
	}

	// Verify user2 can still login
	_, _, _, err = authService.Login(user2.Email, "Password123")
	if err != nil {
		t.Errorf("User2 should be able to login, got error: %v", err)
	}
}

func TestAuthService_Login_LockedAccountNoAttemptsIncrement(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	authService := NewAuthService()

	// Create test user
	user, err := authService.Register("noincrement@example.com", "Password123", "No Increment User")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Lock the account
	lockUntil := time.Now().Add(15 * time.Minute)
	database.DB.Model(&user).Updates(map[string]interface{}{
		"failed_login_attempts": 5,
		"locked_until":          lockUntil,
	})

	// Try to login with wrong password while locked
	_, _, _, err = authService.Login(user.Email, "WrongPassword")
	if err != ErrAccountLocked {
		t.Errorf("Login() should return ErrAccountLocked, got: %v", err)
	}

	// Verify failed attempts didn't increment (stays at 5)
	var lockedUser models.User
	database.DB.First(&lockedUser, user.ID)
	if lockedUser.FailedLoginAttempts != 5 {
		t.Errorf("FailedLoginAttempts = %d, should stay at 5 when account is locked", lockedUser.FailedLoginAttempts)
	}
}
