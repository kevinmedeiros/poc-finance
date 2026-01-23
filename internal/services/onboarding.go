package services

import (
	"errors"
	"time"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
)

var (
	ErrInvalidTemplate = errors.New("template de categoria inválido")
)

// CategoryTemplate represents a predefined set of budget categories for onboarding
type CategoryTemplate struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Categories  []CategoryTemplate `json:"categories"`
}

// CategoryTemplateItem represents a single category in a template
type CategoryTemplateItem struct {
	Name  string  `json:"name"`
	Limit float64 `json:"limit"`
}

// OnboardingService handles the onboarding wizard flow for new users
type OnboardingService struct{}

func NewOnboardingService() *OnboardingService {
	return &OnboardingService{}
}

// GetCategoryTemplates returns all available category templates for onboarding
func (s *OnboardingService) GetCategoryTemplates() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "personal",
			"display_name": "Pessoal",
			"description": "Categorias básicas para controle pessoal",
			"categories": []CategoryTemplateItem{
				{Name: "Alimentação", Limit: 800.00},
				{Name: "Transporte", Limit: 400.00},
				{Name: "Moradia", Limit: 1500.00},
				{Name: "Saúde", Limit: 300.00},
				{Name: "Lazer", Limit: 400.00},
				{Name: "Educação", Limit: 500.00},
				{Name: "Outros", Limit: 300.00},
			},
		},
		{
			"name":        "family",
			"display_name": "Família",
			"description": "Categorias para orçamento familiar",
			"categories": []CategoryTemplateItem{
				{Name: "Alimentação", Limit: 1500.00},
				{Name: "Transporte", Limit: 800.00},
				{Name: "Moradia", Limit: 2500.00},
				{Name: "Saúde", Limit: 600.00},
				{Name: "Educação", Limit: 1000.00},
				{Name: "Lazer", Limit: 600.00},
				{Name: "Filhos", Limit: 800.00},
				{Name: "Despesas Domésticas", Limit: 500.00},
				{Name: "Outros", Limit: 400.00},
			},
		},
		{
			"name":        "brazilian",
			"display_name": "Brasileiro",
			"description": "Categorias adaptadas para o Brasil",
			"categories": []CategoryTemplateItem{
				{Name: "Alimentação", Limit: 1000.00},
				{Name: "Transporte", Limit: 500.00},
				{Name: "Moradia (Aluguel/Financiamento)", Limit: 2000.00},
				{Name: "Contas (Água, Luz, Internet)", Limit: 400.00},
				{Name: "Saúde e Farmácia", Limit: 400.00},
				{Name: "Educação", Limit: 600.00},
				{Name: "Lazer e Entretenimento", Limit: 500.00},
				{Name: "Vestuário", Limit: 300.00},
				{Name: "Impostos e Taxas", Limit: 300.00},
				{Name: "Outros", Limit: 300.00},
			},
		},
	}
}

// CreateDefaultBudget creates a budget with categories from a template for a user
func (s *OnboardingService) CreateDefaultBudget(userID uint, templateName string) (*models.Budget, error) {
	// Find the template
	templates := s.GetCategoryTemplates()
	var selectedTemplate map[string]interface{}

	for _, tmpl := range templates {
		if tmpl["name"].(string) == templateName {
			selectedTemplate = tmpl
			break
		}
	}

	if selectedTemplate == nil {
		return nil, ErrInvalidTemplate
	}

	// Get current month and year
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	// Create budget
	budget := &models.Budget{
		UserID: userID,
		Year:   year,
		Month:  month,
		Name:   "Orçamento " + selectedTemplate["display_name"].(string),
		Status: models.BudgetStatusActive,
	}

	if err := database.DB.Create(budget).Error; err != nil {
		return nil, err
	}

	// Create categories
	categoryItems := selectedTemplate["categories"].([]CategoryTemplateItem)
	for _, catItem := range categoryItems {
		category := &models.BudgetCategory{
			BudgetID: budget.ID,
			Category: catItem.Name,
			Limit:    catItem.Limit,
			Spent:    0,
		}

		if err := database.DB.Create(category).Error; err != nil {
			// Rollback budget creation if category creation fails
			database.DB.Delete(budget)
			return nil, err
		}
	}

	// Reload budget with categories
	if err := database.DB.Preload("Categories").First(budget, budget.ID).Error; err != nil {
		return nil, err
	}

	return budget, nil
}

// CompleteOnboarding marks the user's onboarding as completed
func (s *OnboardingService) CompleteOnboarding(userID uint) error {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return ErrUserNotFound
	}

	user.OnboardingCompleted = true
	return database.DB.Save(&user).Error
}

// IsOnboardingCompleted checks if a user has completed onboarding
func (s *OnboardingService) IsOnboardingCompleted(userID uint) (bool, error) {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return false, ErrUserNotFound
	}

	return user.OnboardingCompleted, nil
}

// SkipOnboarding marks onboarding as completed without creating any data
func (s *OnboardingService) SkipOnboarding(userID uint) error {
	return s.CompleteOnboarding(userID)
}
