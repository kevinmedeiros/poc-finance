package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/database"
	"poc-finance/internal/middleware"
	"poc-finance/internal/models"
	"poc-finance/internal/services"
	"poc-finance/internal/testutil"
)

func setupHealthScoreTestHandler() (*HealthScoreHandler, *echo.Echo, uint, uint, uint) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create test user
	authService := services.NewAuthService()
	user, _ := authService.Register("health@example.com", "Password123", "Health User")

	// Create test account
	account := models.Account{
		Name:   "Test Account",
		Type:   models.AccountTypeIndividual,
		UserID: user.ID,
	}
	database.DB.Create(&account)

	// Create test group
	group := testutil.CreateTestGroup(db, "Test Group", user.ID)

	// Create group membership
	testutil.CreateTestGroupMember(db, group.ID, user.ID, "admin")

	// Create test joint account for the group
	jointAccount := models.Account{
		Name:    "Test Joint Account",
		Type:    models.AccountTypeJoint,
		GroupID: &group.ID,
	}
	database.DB.Create(&jointAccount)

	// Create some financial data for scoring
	now := time.Now()
	income := models.Income{
		AccountID:   account.ID,
		Date:        time.Date(now.Year(), now.Month(), 15, 0, 0, 0, 0, time.Local),
		GrossAmount: 10000.00,
		NetAmount:   8500.00,
		TaxAmount:   1500.00,
		Description: "Test Income",
	}
	database.DB.Create(&income)

	expense := models.Expense{
		AccountID: account.ID,
		Name:      "Test Expense",
		Amount:    2000.00,
		Type:      models.ExpenseTypeFixed,
		DueDay:    10,
		Active:    true,
	}
	database.DB.Create(&expense)

	e := echo.New()
	e.Renderer = &testutil.MockRenderer{}
	handler := NewHealthScoreHandler()
	return handler, e, user.ID, account.ID, group.ID
}

func TestHealthScoreHandler_Index_Success(t *testing.T) {
	handler, e, userID, _, _ := setupHealthScoreTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/health-score", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.Index(c)
	if err != nil {
		t.Fatalf("Index() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHealthScoreHandler_GetUserScore_Success(t *testing.T) {
	handler, e, userID, _, _ := setupHealthScoreTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/health-score/score", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.GetUserScore(c)
	if err != nil {
		t.Fatalf("GetUserScore() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHealthScoreHandler_GetScoreHistory_Success(t *testing.T) {
	handler, e, userID, _, _ := setupHealthScoreTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/health-score/history", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.GetScoreHistory(c)
	if err != nil {
		t.Fatalf("GetScoreHistory() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHealthScoreHandler_GetScoreHistory_WithMonthsParameter(t *testing.T) {
	handler, e, userID, _, _ := setupHealthScoreTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/health-score/history?months=3", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.GetScoreHistory(c)
	if err != nil {
		t.Fatalf("GetScoreHistory() with months parameter returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHealthScoreHandler_GetScoreHistory_WithInvalidMonthsParameter(t *testing.T) {
	handler, e, userID, _, _ := setupHealthScoreTestHandler()

	// Invalid months parameter should default to 6
	req := httptest.NewRequest(http.MethodGet, "/health-score/history?months=invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.GetScoreHistory(c)
	if err != nil {
		t.Fatalf("GetScoreHistory() with invalid months parameter returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHealthScoreHandler_GetScoreHistory_WithMonthsTooHigh(t *testing.T) {
	handler, e, userID, _, _ := setupHealthScoreTestHandler()

	// Months > 12 should default to 6
	req := httptest.NewRequest(http.MethodGet, "/health-score/history?months=24", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.GetScoreHistory(c)
	if err != nil {
		t.Fatalf("GetScoreHistory() with months too high returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHealthScoreHandler_GetScoreHistory_WithMonthsZero(t *testing.T) {
	handler, e, userID, _, _ := setupHealthScoreTestHandler()

	// Months <= 0 should default to 6
	req := httptest.NewRequest(http.MethodGet, "/health-score/history?months=0", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, userID)

	err := handler.GetScoreHistory(c)
	if err != nil {
		t.Fatalf("GetScoreHistory() with months zero returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHealthScoreHandler_GroupScorePage_Success(t *testing.T) {
	handler, e, userID, _, groupID := setupHealthScoreTestHandler()

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/groups/%d/health-score", groupID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", groupID))
	c.Set(middleware.UserIDKey, userID)

	err := handler.GroupScorePage(c)
	if err != nil {
		t.Fatalf("GroupScorePage() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHealthScoreHandler_GroupScorePage_InvalidGroupID(t *testing.T) {
	handler, e, userID, _, _ := setupHealthScoreTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/groups/invalid/health-score", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("invalid")
	c.Set(middleware.UserIDKey, userID)

	err := handler.GroupScorePage(c)
	if err != nil {
		t.Fatalf("GroupScorePage() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHealthScoreHandler_GroupScorePage_NotAMember(t *testing.T) {
	handler, e, _, _, groupID := setupHealthScoreTestHandler()

	// Create another user who is not a member
	authService := services.NewAuthService()
	otherUser, _ := authService.Register("other@example.com", "Password123", "Other User")

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/groups/%d/health-score", groupID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", groupID))
	c.Set(middleware.UserIDKey, otherUser.ID)

	err := handler.GroupScorePage(c)
	if err != nil {
		t.Fatalf("GroupScorePage() returned error: %v", err)
	}

	if rec.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestHealthScoreHandler_GroupScorePage_GroupNotFound(t *testing.T) {
	handler, e, userID, _, _ := setupHealthScoreTestHandler()

	// User is not a member of non-existent group, so it returns forbidden
	nonExistentGroupID := uint(99999)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/groups/%d/health-score", nonExistentGroupID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", nonExistentGroupID))
	c.Set(middleware.UserIDKey, userID)

	err := handler.GroupScorePage(c)
	if err != nil {
		t.Fatalf("GroupScorePage() returned error: %v", err)
	}

	if rec.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestHealthScoreHandler_GetGroupScoreHistory_Success(t *testing.T) {
	handler, e, userID, _, groupID := setupHealthScoreTestHandler()

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/groups/%d/health-score/history", groupID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", groupID))
	c.Set(middleware.UserIDKey, userID)

	err := handler.GetGroupScoreHistory(c)
	if err != nil {
		t.Fatalf("GetGroupScoreHistory() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHealthScoreHandler_GetGroupScoreHistory_WithMonthsParameter(t *testing.T) {
	handler, e, userID, _, groupID := setupHealthScoreTestHandler()

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/groups/%d/health-score/history?months=3", groupID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", groupID))
	c.Set(middleware.UserIDKey, userID)

	err := handler.GetGroupScoreHistory(c)
	if err != nil {
		t.Fatalf("GetGroupScoreHistory() with months parameter returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHealthScoreHandler_GetGroupScoreHistory_WithInvalidMonthsParameter(t *testing.T) {
	handler, e, userID, _, groupID := setupHealthScoreTestHandler()

	// Invalid months parameter should default to 6
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/groups/%d/health-score/history?months=invalid", groupID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", groupID))
	c.Set(middleware.UserIDKey, userID)

	err := handler.GetGroupScoreHistory(c)
	if err != nil {
		t.Fatalf("GetGroupScoreHistory() with invalid months parameter returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHealthScoreHandler_GetGroupScoreHistory_InvalidGroupID(t *testing.T) {
	handler, e, userID, _, _ := setupHealthScoreTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/groups/invalid/health-score/history", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("invalid")
	c.Set(middleware.UserIDKey, userID)

	err := handler.GetGroupScoreHistory(c)
	if err != nil {
		t.Fatalf("GetGroupScoreHistory() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHealthScoreHandler_GetGroupScoreHistory_NotAMember(t *testing.T) {
	handler, e, _, _, groupID := setupHealthScoreTestHandler()

	// Create another user who is not a member
	authService := services.NewAuthService()
	otherUser, _ := authService.Register("other2@example.com", "Password123", "Other User 2")

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/groups/%d/health-score/history", groupID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", groupID))
	c.Set(middleware.UserIDKey, otherUser.ID)

	err := handler.GetGroupScoreHistory(c)
	if err != nil {
		t.Fatalf("GetGroupScoreHistory() returned error: %v", err)
	}

	if rec.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestHealthScoreHandler_GetGroupScoreHistory_GroupNotFound(t *testing.T) {
	handler, e, userID, _, _ := setupHealthScoreTestHandler()

	// User is not a member of non-existent group, so it returns forbidden
	nonExistentGroupID := uint(99999)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/groups/%d/health-score/history", nonExistentGroupID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", nonExistentGroupID))
	c.Set(middleware.UserIDKey, userID)

	err := handler.GetGroupScoreHistory(c)
	if err != nil {
		t.Fatalf("GetGroupScoreHistory() returned error: %v", err)
	}

	if rec.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}
