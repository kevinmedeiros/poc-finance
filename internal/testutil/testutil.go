// Package testutil provides test utilities and helpers for the poc-finance application.
package testutil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/labstack/echo/v4"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"poc-finance/internal/models"
)

// SetupTestDB creates an in-memory SQLite database for testing.
// It automatically migrates all models and returns the database instance.
func SetupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic("failed to connect to test database: " + err.Error())
	}

	// Migrate all models
	err = db.AutoMigrate(
		&models.User{},
		&models.Account{},
		&models.RefreshToken{},
		&models.PasswordResetToken{},
		&models.FamilyGroup{},
		&models.GroupMember{},
		&models.GroupInvite{},
		&models.Income{},
		&models.Expense{},
		&models.ExpensePayment{},
		&models.ExpenseSplit{},
		&models.Bill{},
		&models.CreditCard{},
		&models.Installment{},
		&models.GroupGoal{},
		&models.GoalContribution{},
		&models.Notification{},
		&models.Settings{},
		&models.RecurringTransaction{},
		&models.HealthScore{},
		&models.Budget{},
		&models.BudgetCategory{},
	)
	if err != nil {
		panic("failed to migrate test database: " + err.Error())
	}

	return db
}

// NewTestContext creates a new Echo context for testing HTTP handlers.
func NewTestContext(method, path string, body string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

// NewTestContextWithForm creates a new Echo context with form data.
func NewTestContextWithForm(method, path string, body string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

// NewTestContextWithCookies creates a new Echo context with cookies.
func NewTestContextWithCookies(method, path string, body string, cookies []*http.Cookie) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

// CreateTestUser creates a user in the test database with a hashed password.
func CreateTestUser(db *gorm.DB, email, name, passwordHash string) *models.User {
	user := &models.User{
		Email:        email,
		Name:         name,
		PasswordHash: passwordHash,
	}
	db.Create(user)
	return user
}

// CreateTestAccount creates an account in the test database.
func CreateTestAccount(db *gorm.DB, name string, accountType models.AccountType, userID uint, groupID *uint) *models.Account {
	account := &models.Account{
		Name:    name,
		Type:    accountType,
		UserID:  userID,
		GroupID: groupID,
	}
	db.Create(account)
	return account
}

// CreateTestGroup creates a family group in the test database.
func CreateTestGroup(db *gorm.DB, name string, createdByID uint) *models.FamilyGroup {
	group := &models.FamilyGroup{
		Name:        name,
		CreatedByID: createdByID,
	}
	db.Create(group)
	return group
}

// CreateTestGroupMember creates a group member in the test database.
func CreateTestGroupMember(db *gorm.DB, groupID, userID uint, role string) *models.GroupMember {
	member := &models.GroupMember{
		GroupID: groupID,
		UserID:  userID,
		Role:    role,
	}
	db.Create(member)
	return member
}

// Float64Ptr returns a pointer to a float64 value.
func Float64Ptr(v float64) *float64 {
	return &v
}

// UintPtr returns a pointer to a uint value.
func UintPtr(v uint) *uint {
	return &v
}

// MockRenderer is a simple mock renderer for testing.
type MockRenderer struct{}

// Render renders a template with data.
func (m *MockRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	// Write a simple response to avoid nil pointer issues
	return nil
}
