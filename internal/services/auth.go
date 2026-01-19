package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
)

var (
	ErrInvalidCredentials = errors.New("credenciais inválidas")
	ErrUserExists         = errors.New("email já cadastrado")
	ErrUserNotFound       = errors.New("usuário não encontrado")
	ErrTokenExpired       = errors.New("token expirado")
	ErrTokenInvalid       = errors.New("token inválido")
	ErrAccountLocked      = errors.New("conta bloqueada temporariamente")
)

// JWTSecret is loaded from environment variable
var JWTSecret []byte

func init() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatalf("JWT_SECRET environment variable must be set")
	}
	JWTSecret = []byte(secret)
}

const (
	AccessTokenDuration        = 15 * time.Minute
	RefreshTokenDuration       = 7 * 24 * time.Hour
	PasswordResetTokenDuration = 1 * time.Hour
	BcryptCost                 = 12
	MaxFailedAttempts          = 5
	LockoutDuration            = 15 * time.Minute
)

type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

type AuthService struct{}

func NewAuthService() *AuthService {
	return &AuthService{}
}

// HashPassword creates a bcrypt hash of the password
func (s *AuthService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword compares a password with its hash
func (s *AuthService) CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateAccessToken creates a new JWT access token
func (s *AuthService) GenerateAccessToken(user *models.User) (string, error) {
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.Email,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

// GenerateRefreshToken creates a new refresh token and stores it in database
func (s *AuthService) GenerateRefreshToken(user *models.User) (string, error) {
	// Generate random token
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	tokenString := hex.EncodeToString(bytes)

	// Store in database
	refreshToken := &models.RefreshToken{
		UserID:    user.ID,
		Token:     tokenString,
		ExpiresAt: time.Now().Add(RefreshTokenDuration),
	}

	if err := database.DB.Create(refreshToken).Error; err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateAccessToken validates and parses a JWT access token
func (s *AuthService) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return JWTSecret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token from database
func (s *AuthService) ValidateRefreshToken(tokenString string) (*models.User, error) {
	var refreshToken models.RefreshToken
	if err := database.DB.Where("token = ?", tokenString).Preload("User").First(&refreshToken).Error; err != nil {
		return nil, ErrTokenInvalid
	}

	if refreshToken.IsExpired() {
		// Delete expired token
		database.DB.Delete(&refreshToken)
		return nil, ErrTokenExpired
	}

	return &refreshToken.User, nil
}

// RevokeRefreshToken deletes a refresh token from database
func (s *AuthService) RevokeRefreshToken(tokenString string) error {
	return database.DB.Where("token = ?", tokenString).Delete(&models.RefreshToken{}).Error
}

// RevokeAllUserTokens deletes all refresh tokens for a user
func (s *AuthService) RevokeAllUserTokens(userID uint) error {
	return database.DB.Where("user_id = ?", userID).Delete(&models.RefreshToken{}).Error
}

// Register creates a new user account
func (s *AuthService) Register(email, password, name string) (*models.User, error) {
	// Check if user already exists
	var existingUser models.User
	if err := database.DB.Where("email = ?", email).First(&existingUser).Error; err == nil {
		return nil, ErrUserExists
	}

	// Hash password
	hash, err := s.HashPassword(password)
	if err != nil {
		return nil, err
	}

	// Create user
	user := &models.User{
		Email:        email,
		PasswordHash: hash,
		Name:         name,
	}

	if err := database.DB.Create(user).Error; err != nil {
		return nil, err
	}

	// Auto-create individual account for the user (private data by default)
	account := &models.Account{
		Name:   "Conta Pessoal",
		Type:   models.AccountTypeIndividual,
		UserID: user.ID,
	}

	if err := database.DB.Create(account).Error; err != nil {
		// Rollback user creation if account creation fails
		database.DB.Delete(user)
		return nil, err
	}

	return user, nil
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(email, password string) (*models.User, string, string, error) {
	var user models.User
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, "", "", ErrInvalidCredentials
	}

	// Check if account is locked
	if user.IsLocked() {
		return nil, "", "", ErrAccountLocked
	}

	if !s.CheckPassword(password, user.PasswordHash) {
		// Increment failed login attempts
		user.FailedLoginAttempts++
		now := time.Now()
		user.LastFailedLoginAt = &now

		// Lock account if max attempts reached
		if user.FailedLoginAttempts >= MaxFailedAttempts {
			lockUntil := now.Add(LockoutDuration)
			user.LockedUntil = &lockUntil
		}

		// Save failed attempt tracking
		database.DB.Model(&user).Updates(map[string]interface{}{
			"failed_login_attempts": user.FailedLoginAttempts,
			"last_failed_login_at":  user.LastFailedLoginAt,
			"locked_until":          user.LockedUntil,
		})

		// Return locked error if just locked, otherwise invalid credentials
		if user.FailedLoginAttempts >= MaxFailedAttempts {
			return nil, "", "", ErrAccountLocked
		}
		return nil, "", "", ErrInvalidCredentials
	}

	// Reset failed login attempts on successful login
	if user.FailedLoginAttempts > 0 || user.LockedUntil != nil || user.LastFailedLoginAt != nil {
		database.DB.Model(&user).Updates(map[string]interface{}{
			"failed_login_attempts": 0,
			"last_failed_login_at":  nil,
			"locked_until":          nil,
		})
		user.FailedLoginAttempts = 0
		user.LockedUntil = nil
		user.LastFailedLoginAt = nil
	}

	accessToken, err := s.GenerateAccessToken(&user)
	if err != nil {
		return nil, "", "", err
	}

	refreshToken, err := s.GenerateRefreshToken(&user)
	if err != nil {
		return nil, "", "", err
	}

	return &user, accessToken, refreshToken, nil
}

// RefreshAccessToken generates a new access token using a valid refresh token
func (s *AuthService) RefreshAccessToken(refreshTokenString string) (string, error) {
	user, err := s.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return "", err
	}

	return s.GenerateAccessToken(user)
}

// GetUserByID retrieves a user by ID
func (s *AuthService) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return nil, ErrUserNotFound
	}
	return &user, nil
}

// CleanupExpiredTokens removes all expired refresh tokens
func (s *AuthService) CleanupExpiredTokens() error {
	return database.DB.Where("expires_at < ?", time.Now()).Delete(&models.RefreshToken{}).Error
}

// GeneratePasswordResetToken creates a password reset token for a user
func (s *AuthService) GeneratePasswordResetToken(email string) (string, error) {
	var user models.User
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return "", ErrUserNotFound
	}

	// Invalidate any existing reset tokens for this user
	database.DB.Where("user_id = ? AND used = ?", user.ID, false).Delete(&models.PasswordResetToken{})

	// Generate random token
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	tokenString := hex.EncodeToString(bytes)

	// Store in database
	resetToken := &models.PasswordResetToken{
		UserID:    user.ID,
		Token:     tokenString,
		ExpiresAt: time.Now().Add(PasswordResetTokenDuration),
		Used:      false,
	}

	if err := database.DB.Create(resetToken).Error; err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidatePasswordResetToken validates a password reset token
func (s *AuthService) ValidatePasswordResetToken(tokenString string) (*models.User, error) {
	var resetToken models.PasswordResetToken
	if err := database.DB.Where("token = ? AND used = ?", tokenString, false).Preload("User").First(&resetToken).Error; err != nil {
		return nil, ErrTokenInvalid
	}

	if resetToken.IsExpired() {
		database.DB.Delete(&resetToken)
		return nil, ErrTokenExpired
	}

	return &resetToken.User, nil
}

// ResetPassword changes a user's password using a valid reset token
func (s *AuthService) ResetPassword(tokenString, newPassword string) error {
	var resetToken models.PasswordResetToken
	if err := database.DB.Where("token = ? AND used = ?", tokenString, false).Preload("User").First(&resetToken).Error; err != nil {
		return ErrTokenInvalid
	}

	if resetToken.IsExpired() {
		database.DB.Delete(&resetToken)
		return ErrTokenExpired
	}

	// Hash new password
	hash, err := s.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// Update user's password
	if err := database.DB.Model(&resetToken.User).Update("password_hash", hash).Error; err != nil {
		return err
	}

	// Mark token as used
	database.DB.Model(&resetToken).Update("used", true)

	// Revoke all user's refresh tokens for security
	s.RevokeAllUserTokens(resetToken.UserID)

	return nil
}

// GetUserByEmail retrieves a user by email
func (s *AuthService) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, ErrUserNotFound
	}
	return &user, nil
}
