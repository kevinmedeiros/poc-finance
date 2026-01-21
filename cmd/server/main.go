package main

import (
	"html/template"
	"io"
	"log"
	"net/http"
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
		// Add CSRF token to all templates
		if csrf := c.Get("csrf"); csrf != nil {
			m["csrf"] = csrf
		}
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
	case strings.Contains(baseName, "goal"):
		templateFile = "internal/templates/goals.html"
	case strings.Contains(baseName, "group"):
		templateFile = "internal/templates/groups.html"
	case strings.Contains(baseName, "recurring"):
		templateFile = "internal/templates/recurring.html"
	case strings.Contains(baseName, "invite"), strings.Contains(baseName, "joint-accounts"), strings.Contains(baseName, "split-members"), strings.Contains(baseName, "notification"):
		return t.renderPartialFile(w, "internal/templates/partials/"+baseName+".html", data)
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

func (t *TemplateRegistry) renderPartialFile(w io.Writer, filePath string, data interface{}) error {
	tmpl, err := template.New("").Funcs(t.funcMap).ParseFiles(filePath)
	if err != nil {
		log.Printf("Error parsing partial template %s: %v", filePath, err)
		return err
	}
	// Extract template name from filepath (e.g., notification-badge.html -> notification-badge)
	baseName := filepath.Base(filePath)
	templateName := strings.TrimSuffix(baseName, ".html")
	return tmpl.ExecuteTemplate(w, templateName, data)
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
		"internal/templates/accounts.html",
		"internal/templates/group-dashboard.html",
		"internal/templates/goals.html",
		"internal/templates/health_score.html",
		"internal/templates/notifications.html",
		"internal/templates/recurring.html",
	}

	// Auth pages have their own base template embedded
	authPages := []string{
		"internal/templates/register.html",
		"internal/templates/login.html",
		"internal/templates/forgot-password.html",
		"internal/templates/reset-password.html",
		"internal/templates/join-group.html",
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

// startRecurringScheduler runs the recurring transaction scheduler in the background
// It checks for due transactions daily at midnight
func startRecurringScheduler(schedulerService *services.RecurringSchedulerService) {
	log.Println("Starting recurring transaction scheduler...")

	// Run immediately on startup
	if err := schedulerService.ProcessDueTransactions(); err != nil {
		log.Printf("Error processing due transactions on startup: %v", err)
	}

	// Calculate time until next midnight
	now := time.Now()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	durationUntilMidnight := nextMidnight.Sub(now)

	// Wait until midnight
	time.Sleep(durationUntilMidnight)

	// Then run every 24 hours
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		log.Println("Running scheduled check for due recurring transactions...")
		if err := schedulerService.ProcessDueTransactions(); err != nil {
			log.Printf("Error processing due transactions: %v", err)
		}
		<-ticker.C
	}
}

// startDueDateScheduler runs the due date notification scheduler in the background
// It checks for upcoming expense due dates daily at midnight
func startDueDateScheduler(schedulerService *services.DueDateSchedulerService) {
	log.Println("Starting due date notification scheduler...")

	// Run immediately on startup
	if err := schedulerService.CheckUpcomingDueDates(); err != nil {
		log.Printf("Error checking upcoming due dates on startup: %v", err)
	}

	// Calculate time until next midnight
	now := time.Now()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	durationUntilMidnight := nextMidnight.Sub(now)

	// Wait until midnight
	time.Sleep(durationUntilMidnight)

	// Then run every 24 hours
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		log.Println("Running scheduled check for upcoming expense due dates...")
		if err := schedulerService.CheckUpcomingDueDates(); err != nil {
			log.Printf("Error checking upcoming due dates: %v", err)
		}
		<-ticker.C
	}
}

func main() {
	// Inicializa banco de dados
	if err := database.Init(); err != nil {
		log.Fatalf("Erro ao inicializar banco de dados: %v", err)
	}

	// Initialize settings cache service
	settingsCacheService := services.NewSettingsCacheService()

	// Start recurring transaction scheduler
	schedulerService := services.NewRecurringSchedulerService()
	go startRecurringScheduler(schedulerService)

	// Start due date notification scheduler
	dueDateSchedulerService := services.NewDueDateSchedulerService()
	go startDueDateScheduler(dueDateSchedulerService)

	// Inicializa Echo
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Security: Add body size limit (2MB)
	e.Use(middleware.BodyLimit("2M"))

	// Security: Add security headers
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:      "1; mode=block",
		ContentTypeNosniff: "nosniff",
		XFrameOptions:      "SAMEORIGIN",
		HSTSMaxAge:         31536000,
		ContentSecurityPolicy: "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' https://unpkg.com https://cdn.tailwindcss.com; " +
			"style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net https://fonts.googleapis.com; " +
			"font-src 'self' https://cdn.jsdelivr.net https://fonts.gstatic.com; " +
			"connect-src 'self'; " +
			"img-src 'self' data:",
	}))

	// Security: Add CSRF protection (header-based for HTMX compatibility)
	e.Use(middleware.CSRFWithConfig(middleware.CSRFConfig{
		TokenLookup:    "header:X-CSRF-Token,form:_csrf",
		CookiePath:     "/",
		CookieHTTPOnly: true,
		CookieSameSite: http.SameSiteLaxMode,
		Skipper: func(c echo.Context) bool {
			// Skip CSRF for logout (it's safe and needs to work even with expired tokens)
			return c.Path() == "/logout"
		},
	}))

	// Rate limiter for auth endpoints (5 requests per second per IP)
	authRateLimiter := middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(5))

	// Carrega templates
	e.Renderer = loadTemplates()

	// Handlers
	authHandler := handlers.NewAuthHandler()
	dashboardHandler := handlers.NewDashboardHandler(settingsCacheService)
	incomeHandler := handlers.NewIncomeHandler()
	expenseHandler := handlers.NewExpenseHandler(settingsCacheService)
	cardHandler := handlers.NewCreditCardHandler()
	exportHandler := handlers.NewExportHandler()
	settingsHandler := handlers.NewSettingsHandler(settingsCacheService)
	groupCrudHandler := handlers.NewGroupCrudHandler()
	groupInviteHandler := handlers.NewGroupInviteHandler()
	groupJointAccountHandler := handlers.NewGroupJointAccountHandler()
	groupDashboardHandler := handlers.NewGroupDashboardHandler()
	groupSummaryHandler := handlers.NewGroupSummaryHandler()
	accountHandler := handlers.NewAccountHandler()
	goalHandler := handlers.NewGoalHandler()
	notificationHandler := handlers.NewNotificationHandler()
	recurringHandler := handlers.NewRecurringTransactionHandler()
	healthScoreHandler := handlers.NewHealthScoreHandler()

	// Auth routes (public - no authentication required)
	e.GET("/register", authHandler.RegisterPage)
	e.POST("/register", authHandler.Register, authRateLimiter)
	e.GET("/login", authHandler.LoginPage)
	e.POST("/login", authHandler.Login, authRateLimiter)
	e.POST("/logout", authHandler.Logout)
	e.GET("/forgot-password", authHandler.ForgotPasswordPage)
	e.POST("/forgot-password", authHandler.ForgotPassword, authRateLimiter)
	e.GET("/reset-password", authHandler.ResetPasswordPage)
	e.POST("/reset-password", authHandler.ResetPassword, authRateLimiter)

	// Public invite page (allows users to see invite before login/register)
	e.GET("/groups/join/:code", groupInviteHandler.JoinPagePublic)
	e.POST("/groups/join/:code/register", groupInviteHandler.RegisterAndJoin)

	// Protected routes (authentication required)
	authService := services.NewAuthService()
	protected := e.Group("")
	protected.Use(authmw.AuthMiddleware(authService))

	// Dashboard
	protected.GET("/", dashboardHandler.Index)

	// Contas (saldo por conta)
	protected.GET("/accounts", accountHandler.List)

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
	protected.GET("/accounts/:accountId/members", expenseHandler.GetAccountMembers)

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
	protected.GET("/groups", groupCrudHandler.List)
	protected.POST("/groups", groupCrudHandler.Create)
	protected.DELETE("/groups/:id", groupCrudHandler.DeleteGroup)
	protected.POST("/groups/:id/invite", groupInviteHandler.GenerateInvite)
	protected.GET("/groups/:id/invites", groupInviteHandler.ListInvites)
	protected.POST("/groups/join/:code", groupInviteHandler.AcceptInvite)
	protected.DELETE("/groups/invites/:id", groupInviteHandler.RevokeInvite)
	protected.POST("/groups/:id/leave", groupCrudHandler.LeaveGroup)
	protected.DELETE("/groups/:id/members/:userId", groupCrudHandler.RemoveMember)

	// Contas conjuntas (joint accounts)
	protected.POST("/groups/:id/accounts", groupJointAccountHandler.CreateJointAccount)
	protected.DELETE("/groups/:id/accounts/:accountId", groupJointAccountHandler.DeleteJointAccount)

	// Dashboard do grupo
	protected.GET("/groups/:id/dashboard", groupDashboardHandler.Dashboard)

	// Resumo periódico do grupo
	protected.POST("/groups/:id/summary/weekly", groupSummaryHandler.GenerateWeeklySummary)
	protected.POST("/groups/:id/summary/monthly", groupSummaryHandler.GenerateMonthlySummary)

	// Metas do grupo
	protected.GET("/groups/:id/goals", goalHandler.GoalsPage)
	protected.POST("/groups/:id/goals", goalHandler.Create)
	protected.DELETE("/goals/:goalId", goalHandler.Delete)
	protected.POST("/goals/:goalId/contribution", goalHandler.AddContribution)

	// Health Score do grupo
	protected.GET("/groups/:id/health-score", healthScoreHandler.GroupScorePage)
	protected.GET("/groups/:id/health-score/history", healthScoreHandler.GetGroupScoreHistory)

	// Notificacoes
	protected.GET("/notifications", notificationHandler.List)
	protected.GET("/notifications/badge", notificationHandler.GetBadge)
	protected.GET("/notifications/dropdown", notificationHandler.GetDropdown)
	protected.POST("/notifications/:id/read", notificationHandler.MarkAsRead)
	protected.POST("/notifications/mark-all-read", notificationHandler.MarkAllAsRead)
	protected.DELETE("/notifications/:id", notificationHandler.Delete)

	// Recurring Transactions
	protected.GET("/recurring", recurringHandler.List)
	protected.POST("/recurring", recurringHandler.Create)
	protected.POST("/recurring/:id", recurringHandler.Update)
	protected.DELETE("/recurring/:id", recurringHandler.Delete)
	protected.POST("/recurring/:id/toggle", recurringHandler.Toggle)

	// Health Score
	protected.GET("/health-score", healthScoreHandler.Index)
	protected.GET("/health-score/current", healthScoreHandler.GetUserScore)
	protected.GET("/health-score/history", healthScoreHandler.GetScoreHistory)

	// Inicia servidor
	log.Println("Servidor iniciado em http://localhost:8080")
	e.Logger.Fatal(e.Start(":8080"))
}
