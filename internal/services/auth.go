package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
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
)

// JWTSecret should be set via environment variable in production
var JWTSecret = []byte("your-super-secret-key-change-in-production")

const (
	AccessTokenDuration  = 15 * time.Minute
	RefreshTokenDuration = 7 * 24 * time.Hour
	BcryptCost           = 12
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

	return user, nil
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(email, password string) (*models.User, string, string, error) {
	var user models.User
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, "", "", ErrInvalidCredentials
	}

	if !s.CheckPassword(password, user.PasswordHash) {
		return nil, "", "", ErrInvalidCredentials
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
