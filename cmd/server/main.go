package main

import (
	"html/template"
	"io"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"poc-finance/internal/database"
	"poc-finance/internal/handlers"
	authmw "poc-finance/internal/middleware"
	"poc-finance/internal/services"
)

type TemplateRegistry struct {
	templates map[string]*template.Template
	funcMap   template.FuncMap
}

func (t *TemplateRegistry) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	// Adiciona dados comuns a todos os templates
	if data == nil {
		data = make(map[string]interface{})
	}
	if m, ok := data.(map[string]interface{}); ok {
		m["Year"] = time.Now().Year()
	}

	// Para partials (HTMX), renderiza apenas o fragmento HTML
	if strings.HasPrefix(name, "partials/") {
		return t.renderPartial(w, name, data)
	}

	tmpl, ok := t.templates[name]
	if !ok {
		log.Printf("Template %s not found", name)
		return echo.ErrNotFound
	}

	return tmpl.ExecuteTemplate(w, "base", data)
}

func (t *TemplateRegistry) renderPartial(w io.Writer, name string, data interface{}) error {
	// Extrai o nome do template define a ser executado
	// partials/income-list.html -> income-list
	baseName := strings.TrimPrefix(name, "partials/")
	baseName = strings.TrimSuffix(baseName, ".html")

	// Encontra o arquivo de template original
	var templateFile string
	switch {
	case strings.Contains(baseName, "income"):
		templateFile = "internal/templates/income.html"
	case strings.Contains(baseName, "expense") || strings.Contains(baseName, "fixed") || strings.Contains(baseName, "variable"):
		templateFile = "internal/templates/expenses.html"
	case strings.Contains(baseName, "card") || strings.Contains(baseName, "installment"):
		templateFile = "internal/templates/cards.html"
	case strings.Contains(baseName, "settings"):
		templateFile = "internal/templates/settings.html"
	case strings.Contains(baseName, "group"):
		templateFile = "internal/templates/groups.html"
	default:
		return echo.ErrNotFound
	}

	tmpl, err := template.New("").Funcs(t.funcMap).ParseFiles(templateFile)
	if err != nil {
		log.Printf("Error parsing template %s: %v", templateFile, err)
		return err
	}

	return tmpl.ExecuteTemplate(w, baseName, data)
}

func loadTemplates() *TemplateRegistry {
	templates := make(map[string]*template.Template)

	funcMap := template.FuncMap{
		"dict": func(values ...interface{}) map[string]interface{} {
			if len(values)%2 != 0 {
				return nil
			}
			dict := make(map[string]interface{})
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil
				}
				dict[key] = values[i+1]
			}
			return dict
		},
	}

	baseTemplate := "internal/templates/base.html"
	pages := []string{
		"internal/templates/dashboard.html",
		"internal/templates/income.html",
		"internal/templates/expenses.html",
		"internal/templates/cards.html",
		"internal/templates/settings.html",
		"internal/templates/groups.html",
	}

	// Auth pages have their own base template embedded
	authPages := []string{
		"internal/templates/register.html",
		"internal/templates/login.html",
		"internal/templates/forgot-password.html",
		"internal/templates/reset-password.html",
	}

	for _, page := range pages {
		name := filepath.Base(page)
		tmpl := template.Must(template.New("").Funcs(funcMap).ParseFiles(baseTemplate, page))
		templates[name] = tmpl
	}

	// Auth pages define their own base template
	for _, page := range authPages {
		name := filepath.Base(page)
		tmpl := template.Must(template.New("").Funcs(funcMap).ParseFiles(page))
		templates[name] = tmpl
	}

	return &TemplateRegistry{templates: templates, funcMap: funcMap}
}

func main() {
	// Inicializa banco de dados
	if err := database.Init(); err != nil {
		log.Fatalf("Erro ao inicializar banco de dados: %v", err)
	}

	// Inicializa Echo
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Carrega templates
	e.Renderer = loadTemplates()

	// Handlers
	authHandler := handlers.NewAuthHandler()
	dashboardHandler := handlers.NewDashboardHandler()
	incomeHandler := handlers.NewIncomeHandler()
	expenseHandler := handlers.NewExpenseHandler()
	cardHandler := handlers.NewCreditCardHandler()
	exportHandler := handlers.NewExportHandler()
	settingsHandler := handlers.NewSettingsHandler()
	groupHandler := handlers.NewGroupHandler()

	// Auth routes (public - no authentication required)
	e.GET("/register", authHandler.RegisterPage)
	e.POST("/register", authHandler.Register)
	e.GET("/login", authHandler.LoginPage)
	e.POST("/login", authHandler.Login)
	e.POST("/logout", authHandler.Logout)
	e.GET("/forgot-password", authHandler.ForgotPasswordPage)
	e.POST("/forgot-password", authHandler.ForgotPassword)
	e.GET("/reset-password", authHandler.ResetPasswordPage)
	e.POST("/reset-password", authHandler.ResetPassword)

	// Protected routes (authentication required)
	authService := services.NewAuthService()
	protected := e.Group("")
	protected.Use(authmw.AuthMiddleware(authService))

	// Dashboard
	protected.GET("/", dashboardHandler.Index)

	// Recebimentos
	protected.GET("/incomes", incomeHandler.List)
	protected.POST("/incomes", incomeHandler.Create)
	protected.DELETE("/incomes/:id", incomeHandler.Delete)
	protected.GET("/incomes/preview", incomeHandler.CalculatePreview)

	// Despesas
	protected.GET("/expenses", expenseHandler.List)
	protected.POST("/expenses", expenseHandler.Create)
	protected.POST("/expenses/:id/toggle", expenseHandler.Toggle)
	protected.POST("/expenses/:id/paid", expenseHandler.MarkPaid)
	protected.POST("/expenses/:id/unpaid", expenseHandler.MarkUnpaid)
	protected.DELETE("/expenses/:id", expenseHandler.Delete)

	// Cartões
	protected.GET("/cards", cardHandler.List)
	protected.POST("/cards", cardHandler.CreateCard)
	protected.DELETE("/cards/:id", cardHandler.DeleteCard)
	protected.POST("/installments", cardHandler.CreateInstallment)
	protected.DELETE("/installments/:id", cardHandler.DeleteInstallment)

	// Exportação
	protected.GET("/export", exportHandler.ExportYear)

	// Configurações
	protected.GET("/settings", settingsHandler.Get)
	protected.POST("/settings", settingsHandler.Update)

	// Grupos familiares
	protected.GET("/groups", groupHandler.List)
	protected.POST("/groups", groupHandler.Create)

	// Inicia servidor
	log.Println("Servidor iniciado em http://localhost:8080")
	e.Logger.Fatal(e.Start(":8080"))
}
