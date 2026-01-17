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
	}

	// Auth pages have their own base template embedded
	authPages := []string{
		"internal/templates/register.html",
		"internal/templates/login.html",
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

	// Auth routes
	e.GET("/register", authHandler.RegisterPage)
	e.POST("/register", authHandler.Register)
	e.GET("/login", authHandler.LoginPage)
	e.POST("/login", authHandler.Login)
	e.POST("/logout", authHandler.Logout)

	// Rotas
	e.GET("/", dashboardHandler.Index)

	// Recebimentos
	e.GET("/incomes", incomeHandler.List)
	e.POST("/incomes", incomeHandler.Create)
	e.DELETE("/incomes/:id", incomeHandler.Delete)
	e.GET("/incomes/preview", incomeHandler.CalculatePreview)

	// Despesas
	e.GET("/expenses", expenseHandler.List)
	e.POST("/expenses", expenseHandler.Create)
	e.POST("/expenses/:id/toggle", expenseHandler.Toggle)
	e.POST("/expenses/:id/paid", expenseHandler.MarkPaid)
	e.POST("/expenses/:id/unpaid", expenseHandler.MarkUnpaid)
	e.DELETE("/expenses/:id", expenseHandler.Delete)

	// Cartões
	e.GET("/cards", cardHandler.List)
	e.POST("/cards", cardHandler.CreateCard)
	e.DELETE("/cards/:id", cardHandler.DeleteCard)
	e.POST("/installments", cardHandler.CreateInstallment)
	e.DELETE("/installments/:id", cardHandler.DeleteInstallment)

	// Exportação
	e.GET("/export", exportHandler.ExportYear)

	// Configurações
	e.GET("/settings", settingsHandler.Get)
	e.POST("/settings", settingsHandler.Update)

	// Inicia servidor
	log.Println("Servidor iniciado em http://localhost:8080")
	e.Logger.Fatal(e.Start(":8080"))
}
