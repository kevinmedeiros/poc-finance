package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/database"
	"poc-finance/internal/middleware"
	"poc-finance/internal/models"
	"poc-finance/internal/services"
	"poc-finance/internal/testutil"
)

func setupGoalTestHandler() (*GoalHandler, *echo.Echo, uint, uint, uint) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create test user
	authService := services.NewAuthService()
	user, _ := authService.Register("test@example.com", "Password123", "Test User")

	// Create test group
	group := testutil.CreateTestGroup(db, "Test Group", user.ID)

	// Create group membership
	testutil.CreateTestGroupMember(db, group.ID, user.ID, "admin")

	// Create test account for the group
	account := models.Account{
		Name:    "Test Joint Account",
		Type:    models.AccountTypeJoint,
		GroupID: &group.ID,
	}
	database.DB.Create(&account)

	e := echo.New()
	handler := NewGoalHandler()
	return handler, e, user.ID, group.ID, account.ID
}

func TestGoalHandler_GoalsPage_Success(t *testing.T) {
	handler, e, userID, groupID, _ := setupGoalTestHandler()

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/groups/%d/goals", groupID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", groupID))
	c.Set(middleware.UserIDKey, userID)

	// Mock renderer
	e.Renderer = &testutil.MockRenderer{}

	err := handler.GoalsPage(c)
	if err != nil {
		t.Fatalf("GoalsPage() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestGoalHandler_GoalsPage_InvalidGroupID(t *testing.T) {
	handler, e, userID, _, _ := setupGoalTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/groups/invalid/goals", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("invalid")
	c.Set(middleware.UserIDKey, userID)

	err := handler.GoalsPage(c)
	if err != nil {
		t.Fatalf("GoalsPage() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	if !strings.Contains(rec.Body.String(), "ID do grupo inválido") {
		t.Errorf("Response body = %s, want to contain 'ID do grupo inválido'", rec.Body.String())
	}
}

func TestGoalHandler_GoalsPage_NotAMember(t *testing.T) {
	handler, e, _, groupID, _ := setupGoalTestHandler()

	// Create another user who is not a member
	authService := services.NewAuthService()
	otherUser, _ := authService.Register("other@example.com", "Password123", "Other User")

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/groups/%d/goals", groupID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", groupID))
	c.Set(middleware.UserIDKey, otherUser.ID)

	err := handler.GoalsPage(c)
	if err != nil {
		t.Fatalf("GoalsPage() returned error: %v", err)
	}

	if rec.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	if !strings.Contains(rec.Body.String(), "Você não é membro deste grupo") {
		t.Errorf("Response body = %s, want to contain 'Você não é membro deste grupo'", rec.Body.String())
	}
}

func TestGoalHandler_GoalsPage_GroupNotFound(t *testing.T) {
	handler, e, userID, _, _ := setupGoalTestHandler()

	// User is not a member of non-existent group, so it returns forbidden instead of not found
	// This is correct behavior - membership is checked before group existence
	nonExistentGroupID := uint(99999)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/groups/%d/goals", nonExistentGroupID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", nonExistentGroupID))
	c.Set(middleware.UserIDKey, userID)

	err := handler.GoalsPage(c)
	if err != nil {
		t.Fatalf("GoalsPage() returned error: %v", err)
	}

	if rec.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	if !strings.Contains(rec.Body.String(), "Você não é membro deste grupo") {
		t.Errorf("Response body = %s, want to contain 'Você não é membro deste grupo'", rec.Body.String())
	}
}

func TestGoalHandler_List_Success(t *testing.T) {
	handler, e, userID, groupID, _ := setupGoalTestHandler()

	// Create a test goal
	goalService := services.NewGoalService()
	targetDate := time.Now().AddDate(0, 6, 0)
	goalService.CreateGoal(groupID, userID, "Test Goal", "Test Description", 1000.0, targetDate, nil)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/groups/%d/goals/list", groupID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", groupID))
	c.Set(middleware.UserIDKey, userID)

	// Mock renderer
	e.Renderer = &testutil.MockRenderer{}

	err := handler.List(c)
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestGoalHandler_List_InvalidGroupID(t *testing.T) {
	handler, e, userID, _, _ := setupGoalTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/groups/invalid/goals/list", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("invalid")
	c.Set(middleware.UserIDKey, userID)

	err := handler.List(c)
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	if !strings.Contains(rec.Body.String(), "ID do grupo inválido") {
		t.Errorf("Response body = %s, want to contain 'ID do grupo inválido'", rec.Body.String())
	}
}

func TestGoalHandler_List_Unauthorized(t *testing.T) {
	handler, e, _, groupID, _ := setupGoalTestHandler()

	// Create another user who is not a member
	authService := services.NewAuthService()
	otherUser, _ := authService.Register("other2@example.com", "Password123", "Other User")

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/groups/%d/goals/list", groupID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", groupID))
	c.Set(middleware.UserIDKey, otherUser.ID)

	err := handler.List(c)
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}

	if rec.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	if !strings.Contains(rec.Body.String(), "Você não é membro deste grupo") {
		t.Errorf("Response body = %s, want to contain 'Você não é membro deste grupo'", rec.Body.String())
	}
}

func TestGoalHandler_Create_Success(t *testing.T) {
	handler, e, userID, groupID, accountID := setupGoalTestHandler()

	form := url.Values{}
	form.Set("name", "Test Goal")
	form.Set("description", "Test Description")
	form.Set("target_amount", "1000.00")
	form.Set("target_date", "2025-12-31")
	form.Set("account_id", fmt.Sprintf("%d", accountID))

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/groups/%d/goals", groupID), strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", groupID))
	c.Set(middleware.UserIDKey, userID)

	// Mock renderer
	e.Renderer = &testutil.MockRenderer{}

	err := handler.Create(c)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	// Verify goal was created
	var goal models.GroupGoal
	if err := database.DB.Where("group_id = ? AND name = ?", groupID, "Test Goal").First(&goal).Error; err != nil {
		t.Fatalf("Failed to find created goal: %v", err)
	}

	if goal.TargetAmount != 1000.00 {
		t.Errorf("TargetAmount = %f, want %f", goal.TargetAmount, 1000.00)
	}

	if goal.Description != "Test Description" {
		t.Errorf("Description = %s, want %s", goal.Description, "Test Description")
	}

	if goal.AccountID == nil || *goal.AccountID != accountID {
		t.Errorf("AccountID = %v, want %d", goal.AccountID, accountID)
	}
}

func TestGoalHandler_Create_WithoutAccountID(t *testing.T) {
	handler, e, userID, groupID, _ := setupGoalTestHandler()

	form := url.Values{}
	form.Set("name", "Test Goal No Account")
	form.Set("description", "Test Description")
	form.Set("target_amount", "500.00")
	form.Set("target_date", "2025-06-30")

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/groups/%d/goals", groupID), strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", groupID))
	c.Set(middleware.UserIDKey, userID)

	// Mock renderer
	e.Renderer = &testutil.MockRenderer{}

	err := handler.Create(c)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	// Verify goal was created
	var goal models.GroupGoal
	if err := database.DB.Where("group_id = ? AND name = ?", groupID, "Test Goal No Account").First(&goal).Error; err != nil {
		t.Fatalf("Failed to find created goal: %v", err)
	}

	if goal.AccountID != nil {
		t.Errorf("AccountID = %v, want nil", goal.AccountID)
	}
}

func TestGoalHandler_Create_InvalidGroupID(t *testing.T) {
	handler, e, userID, _, _ := setupGoalTestHandler()

	form := url.Values{}
	form.Set("name", "Test Goal")
	form.Set("target_amount", "1000.00")
	form.Set("target_date", "2025-12-31")

	req := httptest.NewRequest(http.MethodPost, "/groups/invalid/goals", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("invalid")
	c.Set(middleware.UserIDKey, userID)

	err := handler.Create(c)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	if !strings.Contains(rec.Body.String(), "ID do grupo inválido") {
		t.Errorf("Response body = %s, want to contain 'ID do grupo inválido'", rec.Body.String())
	}
}

func TestGoalHandler_Create_EmptyName(t *testing.T) {
	handler, e, userID, groupID, _ := setupGoalTestHandler()

	form := url.Values{}
	form.Set("name", "")
	form.Set("target_amount", "1000.00")
	form.Set("target_date", "2025-12-31")

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/groups/%d/goals", groupID), strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", groupID))
	c.Set(middleware.UserIDKey, userID)

	err := handler.Create(c)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	if !strings.Contains(rec.Body.String(), "Nome e valor alvo são obrigatórios") {
		t.Errorf("Response body = %s, want to contain 'Nome e valor alvo são obrigatórios'", rec.Body.String())
	}
}

func TestGoalHandler_Create_InvalidTargetAmount(t *testing.T) {
	handler, e, userID, groupID, _ := setupGoalTestHandler()

	tests := []struct {
		name   string
		amount string
	}{
		{
			name:   "zero amount",
			amount: "0",
		},
		{
			name:   "negative amount",
			amount: "-100.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{}
			form.Set("name", "Test Goal")
			form.Set("target_amount", tt.amount)
			form.Set("target_date", "2025-12-31")

			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/groups/%d/goals", groupID), strings.NewReader(form.Encode()))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(fmt.Sprintf("%d", groupID))
			c.Set(middleware.UserIDKey, userID)

			err := handler.Create(c)
			if err != nil {
				t.Fatalf("Create() returned error: %v", err)
			}

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
			}

			if !strings.Contains(rec.Body.String(), "Nome e valor alvo são obrigatórios") {
				t.Errorf("Response body = %s, want to contain 'Nome e valor alvo são obrigatórios'", rec.Body.String())
			}
		})
	}
}

func TestGoalHandler_Create_InvalidTargetDate(t *testing.T) {
	handler, e, userID, groupID, _ := setupGoalTestHandler()

	form := url.Values{}
	form.Set("name", "Test Goal")
	form.Set("target_amount", "1000.00")
	form.Set("target_date", "invalid-date")

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/groups/%d/goals", groupID), strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", groupID))
	c.Set(middleware.UserIDKey, userID)

	err := handler.Create(c)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	if !strings.Contains(rec.Body.String(), "Data alvo inválida") {
		t.Errorf("Response body = %s, want to contain 'Data alvo inválida'", rec.Body.String())
	}
}

func TestGoalHandler_Create_Unauthorized(t *testing.T) {
	handler, e, _, groupID, _ := setupGoalTestHandler()

	// Create another user who is not a member
	authService := services.NewAuthService()
	otherUser, _ := authService.Register("other3@example.com", "Password123", "Other User")

	form := url.Values{}
	form.Set("name", "Test Goal")
	form.Set("target_amount", "1000.00")
	form.Set("target_date", "2025-12-31")

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/groups/%d/goals", groupID), strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", groupID))
	c.Set(middleware.UserIDKey, otherUser.ID)

	err := handler.Create(c)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if rec.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	if !strings.Contains(rec.Body.String(), "Você não é membro deste grupo") {
		t.Errorf("Response body = %s, want to contain 'Você não é membro deste grupo'", rec.Body.String())
	}
}

func TestGoalHandler_Delete_Success(t *testing.T) {
	handler, e, userID, groupID, _ := setupGoalTestHandler()

	// Create a test goal
	goalService := services.NewGoalService()
	targetDate := time.Now().AddDate(0, 6, 0)
	goal, _ := goalService.CreateGoal(groupID, userID, "Test Goal to Delete", "Test Description", 1000.0, targetDate, nil)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/goals/%d", goal.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("goalId")
	c.SetParamValues(fmt.Sprintf("%d", goal.ID))
	c.Set(middleware.UserIDKey, userID)

	// Mock renderer
	e.Renderer = &testutil.MockRenderer{}

	err := handler.Delete(c)
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	// Verify goal was deleted (soft delete)
	var deletedGoal models.GroupGoal
	if err := database.DB.Unscoped().Where("id = ?", goal.ID).First(&deletedGoal).Error; err != nil {
		t.Fatalf("Failed to find deleted goal: %v", err)
	}

	if deletedGoal.DeletedAt.Time.IsZero() {
		t.Errorf("Goal was not soft deleted")
	}
}

func TestGoalHandler_Delete_InvalidGoalID(t *testing.T) {
	handler, e, userID, _, _ := setupGoalTestHandler()

	req := httptest.NewRequest(http.MethodDelete, "/goals/invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("goalId")
	c.SetParamValues("invalid")
	c.Set(middleware.UserIDKey, userID)

	err := handler.Delete(c)
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	if !strings.Contains(rec.Body.String(), "ID da meta inválido") {
		t.Errorf("Response body = %s, want to contain 'ID da meta inválido'", rec.Body.String())
	}
}

func TestGoalHandler_Delete_GoalNotFound(t *testing.T) {
	handler, e, userID, _, _ := setupGoalTestHandler()

	nonExistentGoalID := uint(99999)
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/goals/%d", nonExistentGoalID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("goalId")
	c.SetParamValues(fmt.Sprintf("%d", nonExistentGoalID))
	c.Set(middleware.UserIDKey, userID)

	err := handler.Delete(c)
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	if !strings.Contains(rec.Body.String(), "Meta não encontrada") {
		t.Errorf("Response body = %s, want to contain 'Meta não encontrada'", rec.Body.String())
	}
}

func TestGoalHandler_Delete_Unauthorized(t *testing.T) {
	handler, e, userID, groupID, _ := setupGoalTestHandler()

	// Create a test goal
	goalService := services.NewGoalService()
	targetDate := time.Now().AddDate(0, 6, 0)
	goal, _ := goalService.CreateGoal(groupID, userID, "Test Goal", "Test Description", 1000.0, targetDate, nil)

	// Create another user who is not a member
	authService := services.NewAuthService()
	otherUser, _ := authService.Register("other4@example.com", "Password123", "Other User")

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/goals/%d", goal.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("goalId")
	c.SetParamValues(fmt.Sprintf("%d", goal.ID))
	c.Set(middleware.UserIDKey, otherUser.ID)

	err := handler.Delete(c)
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	if rec.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	if !strings.Contains(rec.Body.String(), "Você não tem permissão para deletar esta meta") {
		t.Errorf("Response body = %s, want to contain 'Você não tem permissão para deletar esta meta'", rec.Body.String())
	}
}

func TestGoalHandler_AddContribution_Success(t *testing.T) {
	handler, e, userID, groupID, _ := setupGoalTestHandler()

	// Create a test goal
	goalService := services.NewGoalService()
	targetDate := time.Now().AddDate(0, 6, 0)
	goal, _ := goalService.CreateGoal(groupID, userID, "Test Goal", "Test Description", 1000.0, targetDate, nil)

	form := url.Values{}
	form.Set("amount", "100.00")

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/goals/%d/contributions", goal.ID), strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("goalId")
	c.SetParamValues(fmt.Sprintf("%d", goal.ID))
	c.Set(middleware.UserIDKey, userID)

	// Mock renderer
	e.Renderer = &testutil.MockRenderer{}

	err := handler.AddContribution(c)
	if err != nil {
		t.Fatalf("AddContribution() returned error: %v", err)
	}

	// Verify contribution was added
	var contribution models.GoalContribution
	if err := database.DB.Where("goal_id = ? AND user_id = ?", goal.ID, userID).First(&contribution).Error; err != nil {
		t.Fatalf("Failed to find created contribution: %v", err)
	}

	if contribution.Amount != 100.00 {
		t.Errorf("Amount = %f, want %f", contribution.Amount, 100.00)
	}

	// Verify goal's current amount was updated
	var updatedGoal models.GroupGoal
	database.DB.First(&updatedGoal, goal.ID)
	if updatedGoal.CurrentAmount != 100.00 {
		t.Errorf("CurrentAmount = %f, want %f", updatedGoal.CurrentAmount, 100.00)
	}
}

func TestGoalHandler_AddContribution_InvalidGoalID(t *testing.T) {
	handler, e, userID, _, _ := setupGoalTestHandler()

	form := url.Values{}
	form.Set("amount", "100.00")

	req := httptest.NewRequest(http.MethodPost, "/goals/invalid/contributions", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("goalId")
	c.SetParamValues("invalid")
	c.Set(middleware.UserIDKey, userID)

	err := handler.AddContribution(c)
	if err != nil {
		t.Fatalf("AddContribution() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	if !strings.Contains(rec.Body.String(), "ID da meta inválido") {
		t.Errorf("Response body = %s, want to contain 'ID da meta inválido'", rec.Body.String())
	}
}

func TestGoalHandler_AddContribution_InvalidAmount(t *testing.T) {
	handler, e, userID, groupID, _ := setupGoalTestHandler()

	// Create a test goal
	goalService := services.NewGoalService()
	targetDate := time.Now().AddDate(0, 6, 0)
	goal, _ := goalService.CreateGoal(groupID, userID, "Test Goal", "Test Description", 1000.0, targetDate, nil)

	tests := []struct {
		name   string
		amount string
	}{
		{
			name:   "not a number",
			amount: "invalid",
		},
		{
			name:   "zero amount",
			amount: "0",
		},
		{
			name:   "negative amount",
			amount: "-50.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{}
			form.Set("amount", tt.amount)

			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/goals/%d/contributions", goal.ID), strings.NewReader(form.Encode()))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("goalId")
			c.SetParamValues(fmt.Sprintf("%d", goal.ID))
			c.Set(middleware.UserIDKey, userID)

			err := handler.AddContribution(c)
			if err != nil {
				t.Fatalf("AddContribution() returned error: %v", err)
			}

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
			}

			if !strings.Contains(rec.Body.String(), "Valor inválido") {
				t.Errorf("Response body = %s, want to contain 'Valor inválido'", rec.Body.String())
			}
		})
	}
}

func TestGoalHandler_AddContribution_GoalNotFound(t *testing.T) {
	handler, e, userID, _, _ := setupGoalTestHandler()

	nonExistentGoalID := uint(99999)
	form := url.Values{}
	form.Set("amount", "100.00")

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/goals/%d/contributions", nonExistentGoalID), strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("goalId")
	c.SetParamValues(fmt.Sprintf("%d", nonExistentGoalID))
	c.Set(middleware.UserIDKey, userID)

	err := handler.AddContribution(c)
	if err != nil {
		t.Fatalf("AddContribution() returned error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	if !strings.Contains(rec.Body.String(), "Meta não encontrada") {
		t.Errorf("Response body = %s, want to contain 'Meta não encontrada'", rec.Body.String())
	}
}

func TestGoalHandler_AddContribution_GoalCompleted(t *testing.T) {
	handler, e, userID, groupID, _ := setupGoalTestHandler()

	// Create a test goal with target amount 100
	goalService := services.NewGoalService()
	targetDate := time.Now().AddDate(0, 6, 0)
	goal, _ := goalService.CreateGoal(groupID, userID, "Test Goal", "Test Description", 100.0, targetDate, nil)

	// Add contribution that completes the goal
	goalService.AddContribution(goal.ID, userID, 100.0)

	// Try to add another contribution
	form := url.Values{}
	form.Set("amount", "50.00")

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/goals/%d/contributions", goal.ID), strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("goalId")
	c.SetParamValues(fmt.Sprintf("%d", goal.ID))
	c.Set(middleware.UserIDKey, userID)

	err := handler.AddContribution(c)
	if err != nil {
		t.Fatalf("AddContribution() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	if !strings.Contains(rec.Body.String(), "Meta já foi concluída") {
		t.Errorf("Response body = %s, want to contain 'Meta já foi concluída'", rec.Body.String())
	}
}

func TestGoalHandler_AddContribution_Unauthorized(t *testing.T) {
	handler, e, userID, groupID, _ := setupGoalTestHandler()

	// Create a test goal
	goalService := services.NewGoalService()
	targetDate := time.Now().AddDate(0, 6, 0)
	goal, _ := goalService.CreateGoal(groupID, userID, "Test Goal", "Test Description", 1000.0, targetDate, nil)

	// Create another user who is not a member
	authService := services.NewAuthService()
	otherUser, _ := authService.Register("other5@example.com", "Password123", "Other User")

	form := url.Values{}
	form.Set("amount", "100.00")

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/goals/%d/contributions", goal.ID), strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("goalId")
	c.SetParamValues(fmt.Sprintf("%d", goal.ID))
	c.Set(middleware.UserIDKey, otherUser.ID)

	err := handler.AddContribution(c)
	if err != nil {
		t.Fatalf("AddContribution() returned error: %v", err)
	}

	if rec.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	if !strings.Contains(rec.Body.String(), "Você não é membro deste grupo") {
		t.Errorf("Response body = %s, want to contain 'Você não é membro deste grupo'", rec.Body.String())
	}
}
