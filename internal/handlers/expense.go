package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/database"
	"poc-finance/internal/middleware"
	"poc-finance/internal/models"
	"poc-finance/internal/services"
)

type ExpenseHandler struct {
	accountService      *services.AccountService
	notificationService *services.NotificationService
}

func NewExpenseHandler() *ExpenseHandler {
	return &ExpenseHandler{
		accountService:      services.NewAccountService(),
		notificationService: services.NewNotificationService(),
	}
}

type CreateExpenseRequest struct {
	AccountID   uint      `form:"account_id"`
	Name        string    `form:"name"`
	Amount      float64   `form:"amount"`
	Type        string    `form:"type"`
	DueDay      int       `form:"due_day"`
	Category    string    `form:"category"`
	IsSplit     bool      `form:"is_split"`
	SplitUsers  []uint    `form:"split_user_ids"`
	SplitPcts   []float64 `form:"split_percentages"`
}

type ExpenseWithStatus struct {
	models.Expense
	IsPaid bool
}

func (h *ExpenseHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	accountIDs, _ := h.accountService.GetUserAccountIDs(userID)
	accounts, _ := h.accountService.GetUserAccounts(userID)

	// Handle category filter from query parameter
	var selectedCategory string
	categoryParam := c.QueryParam("category")

	if categoryParam != "" && categoryParam != "all" {
		selectedCategory = categoryParam
	}

	now := time.Now()
	month := int(now.Month())
	year := now.Year()

	var fixedExpenses []models.Expense
	var variableExpenses []models.Expense

	// Build queries with category filter if applicable
	fixedQuery := database.DB.Preload("Splits").Preload("Splits.User").Where("type = ? AND account_id IN ?", models.ExpenseTypeFixed, accountIDs)
	variableQuery := database.DB.Preload("Splits").Preload("Splits.User").Where("type = ? AND account_id IN ?", models.ExpenseTypeVariable, accountIDs)

	if selectedCategory != "" {
		fixedQuery = fixedQuery.Where("category = ?", selectedCategory)
		variableQuery = variableQuery.Where("category = ?", selectedCategory)
	}

	fixedQuery.Order("due_day, name").Find(&fixedExpenses)
	variableQuery.Order("created_at DESC").Find(&variableExpenses)

	// Verifica status de pagamento para cada despesa fixa
	fixedWithStatus := make([]ExpenseWithStatus, len(fixedExpenses))
	for i, e := range fixedExpenses {
		fixedWithStatus[i] = ExpenseWithStatus{
			Expense: e,
			IsPaid:  isExpensePaid(e.ID, month, year),
		}
	}

	// Calcula totais
	var totalFixed, totalVariable, totalPaid, totalPending float64
	for _, e := range fixedWithStatus {
		if e.Active {
			totalFixed += e.Amount
			if e.IsPaid {
				totalPaid += e.Amount
			} else {
				totalPending += e.Amount
			}
		}
	}
	for _, e := range variableExpenses {
		if e.Active {
			totalVariable += e.Amount
		}
	}

	data := map[string]interface{}{
		"fixedExpenses":    fixedWithStatus,
		"variableExpenses": variableExpenses,
		"accounts":         accounts,
		"totalFixed":       totalFixed,
		"totalVariable":    totalVariable,
		"totalPaid":        totalPaid,
		"totalPending":     totalPending,
		"categories":       getExpenseCategories(),
		"currentMonth":     month,
		"currentYear":      year,
		"selectedCategory": selectedCategory,
	}

	return c.Render(http.StatusOK, "expenses.html", data)
}

func (h *ExpenseHandler) Create(c echo.Context) error {
	userID := middleware.GetUserID(c)

	var req CreateExpenseRequest
	if err := c.Bind(&req); err != nil {
		return c.String(http.StatusBadRequest, "Dados inválidos")
	}

	// Parse split data from form
	splitUserIDs := c.Request().Form["split_user_ids"]
	splitPercentages := c.Request().Form["split_percentages"]

	// Validate user has access to selected account
	accountID := req.AccountID
	if accountID == 0 {
		// Fallback to individual account if not specified
		account, err := h.accountService.GetUserIndividualAccount(userID)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Conta não encontrada")
		}
		accountID = account.ID
	} else if !h.accountService.CanUserAccessAccount(userID, accountID) {
		return c.String(http.StatusForbidden, "Acesso negado à conta selecionada")
	}

	expenseType := models.ExpenseTypeFixed
	if req.Type == "variable" {
		expenseType = models.ExpenseTypeVariable
	}

	// Check if this is a split expense
	isSplit := c.FormValue("is_split") == "on" || c.FormValue("is_split") == "true"

	expense := models.Expense{
		AccountID: accountID,
		Name:      req.Name,
		Amount:    req.Amount,
		Type:      expenseType,
		DueDay:    req.DueDay,
		Category:  req.Category,
		Active:    true,
		IsSplit:   isSplit,
	}

	// Start transaction for expense + splits
	tx := database.DB.Begin()

	if err := tx.Create(&expense).Error; err != nil {
		tx.Rollback()
		return c.String(http.StatusInternalServerError, "Erro ao criar despesa")
	}

	// Create splits if this is a split expense
	if isSplit && len(splitUserIDs) > 0 && len(splitUserIDs) == len(splitPercentages) {
		var totalPercentage float64
		for i := range splitUserIDs {
			uid, err := strconv.ParseUint(splitUserIDs[i], 10, 32)
			if err != nil {
				continue
			}
			pct, err := strconv.ParseFloat(splitPercentages[i], 64)
			if err != nil || pct <= 0 {
				continue
			}

			totalPercentage += pct
			splitAmount := req.Amount * pct / 100

			split := models.ExpenseSplit{
				ExpenseID:  expense.ID,
				UserID:     uint(uid),
				Percentage: pct,
				Amount:     splitAmount,
			}

			if err := tx.Create(&split).Error; err != nil {
				tx.Rollback()
				return c.String(http.StatusInternalServerError, "Erro ao criar divisão")
			}
		}

		// Validate total percentage equals 100
		if totalPercentage < 99.99 || totalPercentage > 100.01 {
			tx.Rollback()
			return c.String(http.StatusBadRequest, "A soma dos percentuais deve ser 100%")
		}
	}

	tx.Commit()

	// Notify group members if this is a joint account expense
	h.notifyPartnerExpense(userID, accountID, &expense)

	// Check budget limit and notify if exceeded
	h.checkBudgetLimit(accountID)

	return h.renderExpenseList(c, string(expenseType))
}

// notifyPartnerExpense sends notifications to group members when an expense is added to a joint account
func (h *ExpenseHandler) notifyPartnerExpense(creatorID uint, accountID uint, expense *models.Expense) {
	// Get the account to check if it's a joint account
	account, err := h.accountService.GetAccountByID(accountID)
	if err != nil || account.Type != models.AccountTypeJoint || account.GroupID == nil {
		return
	}

	// Get the creator's name
	var creator models.User
	if err := database.DB.First(&creator, creatorID).Error; err != nil {
		return
	}

	// Get all group members
	members, err := h.accountService.GetAccountMembers(accountID)
	if err != nil {
		return
	}

	// Send notifications (errors are logged but don't fail the request)
	h.notificationService.NotifyPartnerExpense(expense, account, creatorID, creator.Name, members)
}

// checkBudgetLimit checks if account has reached its budget limit and sends notification
func (h *ExpenseHandler) checkBudgetLimit(accountID uint) {
	// Get account balance to check total expenses
	balance, err := h.accountService.GetAccountBalance(accountID)
	if err != nil {
		return
	}

	// Check if budget limit is set
	if balance.Account.BudgetLimit == nil || *balance.Account.BudgetLimit <= 0 {
		return
	}

	budgetLimit := *balance.Account.BudgetLimit
	percentage := (balance.TotalExpenses / budgetLimit) * 100

	// Only notify if expenses reach or exceed 100% of budget
	if percentage < 100 {
		return
	}

	// Get account members to notify
	members, err := h.accountService.GetAccountMembers(accountID)
	if err != nil {
		return
	}

	// Send budget alert notification
	alertData := services.BudgetAlertData{
		Account:       &balance.Account,
		TotalExpenses: balance.TotalExpenses,
		BudgetLimit:   budgetLimit,
		Percentage:    percentage,
	}
	h.notificationService.NotifyBudgetLimitReached(alertData, members)
}

func (h *ExpenseHandler) Toggle(c echo.Context) error {
	userID := middleware.GetUserID(c)
	accountIDs, _ := h.accountService.GetUserAccountIDs(userID)

	id, _ := strconv.Atoi(c.Param("id"))

	var expense models.Expense
	if err := database.DB.Where("id = ? AND account_id IN ?", id, accountIDs).First(&expense).Error; err != nil {
		return c.String(http.StatusNotFound, "Despesa não encontrada")
	}

	expense.Active = !expense.Active
	database.DB.Save(&expense)

	return h.renderExpenseList(c, string(expense.Type))
}

func (h *ExpenseHandler) Delete(c echo.Context) error {
	userID := middleware.GetUserID(c)
	accountIDs, _ := h.accountService.GetUserAccountIDs(userID)

	id, _ := strconv.Atoi(c.Param("id"))

	var expense models.Expense
	if err := database.DB.Where("id = ? AND account_id IN ?", id, accountIDs).First(&expense).Error; err != nil {
		return c.String(http.StatusNotFound, "Despesa não encontrada")
	}

	expenseType := string(expense.Type)
	database.DB.Delete(&expense)

	return h.renderExpenseList(c, expenseType)
}

// MarkPaid marca uma despesa como paga no mês/ano atual
func (h *ExpenseHandler) MarkPaid(c echo.Context) error {
	userID := middleware.GetUserID(c)
	accountIDs, _ := h.accountService.GetUserAccountIDs(userID)

	id, _ := strconv.Atoi(c.Param("id"))
	now := time.Now()
	month := int(now.Month())
	year := now.Year()

	var expense models.Expense
	if err := database.DB.Where("id = ? AND account_id IN ?", id, accountIDs).First(&expense).Error; err != nil {
		return c.String(http.StatusNotFound, "Despesa não encontrada")
	}

	// Verifica se já existe um pagamento
	var existing models.ExpensePayment
	result := database.DB.Where("expense_id = ? AND month = ? AND year = ?", id, month, year).First(&existing)

	if result.Error != nil {
		// Cria novo pagamento
		payment := models.ExpensePayment{
			ExpenseID: uint(id),
			Month:     month,
			Year:      year,
			PaidAt:    now,
			Amount:    expense.Amount,
		}
		database.DB.Create(&payment)
	}

	return h.renderExpenseList(c, string(expense.Type))
}

// MarkUnpaid remove a marcação de pago
func (h *ExpenseHandler) MarkUnpaid(c echo.Context) error {
	userID := middleware.GetUserID(c)
	accountIDs, _ := h.accountService.GetUserAccountIDs(userID)

	id, _ := strconv.Atoi(c.Param("id"))
	now := time.Now()
	month := int(now.Month())
	year := now.Year()

	var expense models.Expense
	if err := database.DB.Where("id = ? AND account_id IN ?", id, accountIDs).First(&expense).Error; err != nil {
		return c.String(http.StatusNotFound, "Despesa não encontrada")
	}

	// Remove pagamento
	database.DB.Where("expense_id = ? AND month = ? AND year = ?", id, month, year).Delete(&models.ExpensePayment{})

	return h.renderExpenseList(c, string(expense.Type))
}

func (h *ExpenseHandler) renderExpenseList(c echo.Context, expenseType string) error {
	userID := middleware.GetUserID(c)
	accountIDs, _ := h.accountService.GetUserAccountIDs(userID)

	now := time.Now()
	month := int(now.Month())
	year := now.Year()

	var expenses []models.Expense
	database.DB.Preload("Splits").Preload("Splits.User").Where("type = ? AND account_id IN ?", expenseType, accountIDs).Order("due_day, name").Find(&expenses)

	template := "partials/fixed-expense-list.html"
	if expenseType == "variable" {
		template = "partials/variable-expense-list.html"
		return c.Render(http.StatusOK, template, map[string]interface{}{
			"expenses": expenses,
		})
	}

	// Para despesas fixas, adiciona status de pagamento
	expensesWithStatus := make([]ExpenseWithStatus, len(expenses))
	for i, e := range expenses {
		expensesWithStatus[i] = ExpenseWithStatus{
			Expense: e,
			IsPaid:  isExpensePaid(e.ID, month, year),
		}
	}

	return c.Render(http.StatusOK, template, map[string]interface{}{
		"expenses": expensesWithStatus,
	})
}

func isExpensePaid(expenseID uint, month, year int) bool {
	var count int64
	database.DB.Model(&models.ExpensePayment{}).
		Where("expense_id = ? AND month = ? AND year = ?", expenseID, month, year).
		Count(&count)
	return count > 0
}

func getExpenseCategories() []string {
	return []string{
		"Moradia",
		"Alimentação",
		"Transporte",
		"Saúde",
		"Educação",
		"Lazer",
		"Serviços",
		"Impostos",
		"Outros",
	}
}

// GetAccountMembers returns members of an account for split configuration
func (h *ExpenseHandler) GetAccountMembers(c echo.Context) error {
	userID := middleware.GetUserID(c)
	accountID, _ := strconv.Atoi(c.Param("accountId"))

	// Verify user has access
	if !h.accountService.CanUserAccessAccount(userID, uint(accountID)) {
		return c.String(http.StatusForbidden, "Acesso negado")
	}

	account, err := h.accountService.GetAccountByID(uint(accountID))
	if err != nil {
		return c.String(http.StatusNotFound, "Conta não encontrada")
	}

	members, err := h.accountService.GetAccountMembers(uint(accountID))
	if err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao buscar membros")
	}

	// Return HTML for member split inputs
	return c.Render(http.StatusOK, "partials/split-members.html", map[string]interface{}{
		"members":   members,
		"account":   account,
		"isJoint":   account.Type == models.AccountTypeJoint,
	})
}
