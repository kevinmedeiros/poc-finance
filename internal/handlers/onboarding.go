package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/database"
	"poc-finance/internal/middleware"
	"poc-finance/internal/models"
	"poc-finance/internal/services"
)

type OnboardingHandler struct {
	onboardingService *services.OnboardingService
	accountService    *services.AccountService
}

func NewOnboardingHandler() *OnboardingHandler {
	return &OnboardingHandler{
		onboardingService: services.NewOnboardingService(),
		accountService:    services.NewAccountService(),
	}
}

// OnboardingData holds the current state of the onboarding wizard
type OnboardingData struct {
	Step            int
	AccountCreated  bool
	AccountID       uint
	AccountName     string
	CategoriesSet   bool
	BudgetID        uint
	TemplateName    string
	TransactionDone bool
	Templates       []map[string]interface{}
	Categories      []models.BudgetCategory
}

// ShowWizard displays the onboarding wizard at the current step
func (h *OnboardingHandler) ShowWizard(c echo.Context) error {
	userID := middleware.GetUserID(c)

	// Check if user has already completed onboarding
	completed, err := h.onboardingService.IsOnboardingCompleted(userID)
	if err == nil && completed {
		return c.Redirect(http.StatusSeeOther, "/")
	}

	// Get current step from query param (default to step 1)
	step := 1
	if stepParam := c.QueryParam("step"); stepParam != "" {
		if s, err := strconv.Atoi(stepParam); err == nil && s >= 1 && s <= 5 {
			step = s
		}
	}

	// Get category templates for step 3
	templates := h.onboardingService.GetCategoryTemplates()

	data := map[string]interface{}{
		"step":      step,
		"templates": templates,
	}

	return c.Render(http.StatusOK, "onboarding.html", data)
}

// CreateAccount handles the account creation step (step 2)
func (h *OnboardingHandler) CreateAccount(c echo.Context) error {
	userID := middleware.GetUserID(c)

	accountName := strings.TrimSpace(c.FormValue("account_name"))
	if accountName == "" {
		accountName = "Conta Pessoal"
	}

	// Check if user already has an individual account
	existingAccount, err := h.accountService.GetUserIndividualAccount(userID)
	if err == nil && existingAccount != nil {
		// User already has an account, redirect to next step
		return c.Redirect(http.StatusSeeOther, "/onboarding?step=3")
	}

	// Create new individual account
	account := &models.Account{
		Name:   accountName,
		Type:   models.AccountTypeIndividual,
		UserID: userID,
	}

	if err := database.DB.Create(account).Error; err != nil {
		return c.Render(http.StatusOK, "onboarding.html", map[string]interface{}{
			"step":  2,
			"error": "Erro ao criar conta. Tente novamente.",
		})
	}

	// Redirect to next step
	return c.Redirect(http.StatusSeeOther, "/onboarding?step=3")
}

// SelectCategories handles the category template selection step (step 3)
func (h *OnboardingHandler) SelectCategories(c echo.Context) error {
	userID := middleware.GetUserID(c)

	templateName := c.FormValue("template")
	if templateName == "" {
		templates := h.onboardingService.GetCategoryTemplates()
		return c.Render(http.StatusOK, "onboarding.html", map[string]interface{}{
			"step":      3,
			"templates": templates,
			"error":     "Por favor, selecione um modelo de categorias",
		})
	}

	// Create budget with categories from template
	_, err := h.onboardingService.CreateDefaultBudget(userID, templateName)
	if err != nil {
		templates := h.onboardingService.GetCategoryTemplates()
		return c.Render(http.StatusOK, "onboarding.html", map[string]interface{}{
			"step":      3,
			"templates": templates,
			"error":     "Erro ao criar categorias. Tente novamente.",
		})
	}

	// Redirect to next step
	return c.Redirect(http.StatusSeeOther, "/onboarding?step=4")
}

// CreateTransaction handles the first transaction creation step (step 4)
func (h *OnboardingHandler) CreateTransaction(c echo.Context) error {
	userID := middleware.GetUserID(c)

	// Get form values
	name := strings.TrimSpace(c.FormValue("name"))
	amountStr := c.FormValue("amount")
	category := c.FormValue("category")

	// Validate inputs
	if name == "" || amountStr == "" || category == "" {
		// Get user's account and categories for re-rendering
		account, _ := h.accountService.GetUserIndividualAccount(userID)

		// Get current budget categories
		var categories []models.BudgetCategory
		now := time.Now()
		var budget models.Budget
		database.DB.Where("user_id = ? AND year = ? AND month = ?",
			userID, now.Year(), int(now.Month())).
			Preload("Categories").
			First(&budget)
		if budget.ID != 0 {
			categories = budget.Categories
		}

		return c.Render(http.StatusOK, "onboarding.html", map[string]interface{}{
			"step":       4,
			"error":      "Todos os campos são obrigatórios",
			"name":       name,
			"amount":     amountStr,
			"category":   category,
			"account":    account,
			"categories": categories,
		})
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount <= 0 {
		// Get user's account and categories for re-rendering
		account, _ := h.accountService.GetUserIndividualAccount(userID)

		// Get current budget categories
		var categories []models.BudgetCategory
		now := time.Now()
		var budget models.Budget
		database.DB.Where("user_id = ? AND year = ? AND month = ?",
			userID, now.Year(), int(now.Month())).
			Preload("Categories").
			First(&budget)
		if budget.ID != 0 {
			categories = budget.Categories
		}

		return c.Render(http.StatusOK, "onboarding.html", map[string]interface{}{
			"step":       4,
			"error":      "Por favor, insira um valor válido",
			"name":       name,
			"amount":     amountStr,
			"category":   category,
			"account":    account,
			"categories": categories,
		})
	}

	// Get user's account
	account, err := h.accountService.GetUserIndividualAccount(userID)
	if err != nil {
		return c.Render(http.StatusOK, "onboarding.html", map[string]interface{}{
			"step":  4,
			"error": "Conta não encontrada. Por favor, volte e crie uma conta.",
		})
	}

	// Create expense as a variable expense
	expense := &models.Expense{
		AccountID: account.ID,
		Name:      name,
		Amount:    amount,
		Type:      models.ExpenseTypeVariable,
		Category:  category,
		Active:    true,
		DueDay:    time.Now().Day(),
	}

	if err := database.DB.Create(expense).Error; err != nil {
		// Get categories for re-rendering
		var categories []models.BudgetCategory
		now := time.Now()
		var budget models.Budget
		database.DB.Where("user_id = ? AND year = ? AND month = ?",
			userID, now.Year(), int(now.Month())).
			Preload("Categories").
			First(&budget)
		if budget.ID != 0 {
			categories = budget.Categories
		}

		return c.Render(http.StatusOK, "onboarding.html", map[string]interface{}{
			"step":       4,
			"error":      "Erro ao criar transação. Tente novamente.",
			"name":       name,
			"amount":     amountStr,
			"category":   category,
			"account":    account,
			"categories": categories,
		})
	}

	// Redirect to completion step
	return c.Redirect(http.StatusSeeOther, "/onboarding?step=5")
}

// Complete marks the onboarding as completed and redirects to dashboard
func (h *OnboardingHandler) Complete(c echo.Context) error {
	userID := middleware.GetUserID(c)

	if err := h.onboardingService.CompleteOnboarding(userID); err != nil {
		return c.Render(http.StatusOK, "onboarding.html", map[string]interface{}{
			"step":  5,
			"error": "Erro ao finalizar. Tente novamente.",
		})
	}

	return c.Redirect(http.StatusSeeOther, "/")
}

// Skip allows users to skip the onboarding wizard
func (h *OnboardingHandler) Skip(c echo.Context) error {
	userID := middleware.GetUserID(c)

	if err := h.onboardingService.SkipOnboarding(userID); err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao pular onboarding")
	}

	return c.Redirect(http.StatusSeeOther, "/")
}
