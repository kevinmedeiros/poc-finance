package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
)

type ExpenseHandler struct{}

func NewExpenseHandler() *ExpenseHandler {
	return &ExpenseHandler{}
}

type CreateExpenseRequest struct {
	Name     string  `form:"name"`
	Amount   float64 `form:"amount"`
	Type     string  `form:"type"`
	DueDay   int     `form:"due_day"`
	Category string  `form:"category"`
}

type ExpenseWithStatus struct {
	models.Expense
	IsPaid bool
}

func (h *ExpenseHandler) List(c echo.Context) error {
	now := time.Now()
	month := int(now.Month())
	year := now.Year()

	var fixedExpenses []models.Expense
	var variableExpenses []models.Expense

	database.DB.Where("type = ?", models.ExpenseTypeFixed).Order("due_day, name").Find(&fixedExpenses)
	database.DB.Where("type = ?", models.ExpenseTypeVariable).Order("created_at DESC").Find(&variableExpenses)

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
		"totalFixed":       totalFixed,
		"totalVariable":    totalVariable,
		"totalPaid":        totalPaid,
		"totalPending":     totalPending,
		"categories":       getExpenseCategories(),
		"currentMonth":     month,
		"currentYear":      year,
	}

	return c.Render(http.StatusOK, "expenses.html", data)
}

func (h *ExpenseHandler) Create(c echo.Context) error {
	var req CreateExpenseRequest
	if err := c.Bind(&req); err != nil {
		return c.String(http.StatusBadRequest, "Dados inválidos")
	}

	expenseType := models.ExpenseTypeFixed
	if req.Type == "variable" {
		expenseType = models.ExpenseTypeVariable
	}

	expense := models.Expense{
		Name:     req.Name,
		Amount:   req.Amount,
		Type:     expenseType,
		DueDay:   req.DueDay,
		Category: req.Category,
		Active:   true,
	}

	if err := database.DB.Create(&expense).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao criar despesa")
	}

	return h.renderExpenseList(c, string(expenseType))
}

func (h *ExpenseHandler) Toggle(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	var expense models.Expense
	if err := database.DB.First(&expense, id).Error; err != nil {
		return c.String(http.StatusNotFound, "Despesa não encontrada")
	}

	expense.Active = !expense.Active
	database.DB.Save(&expense)

	return h.renderExpenseList(c, string(expense.Type))
}

func (h *ExpenseHandler) Delete(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	var expense models.Expense
	if err := database.DB.First(&expense, id).Error; err != nil {
		return c.String(http.StatusNotFound, "Despesa não encontrada")
	}

	expenseType := string(expense.Type)
	database.DB.Delete(&expense)

	return h.renderExpenseList(c, expenseType)
}

// MarkPaid marca uma despesa como paga no mês/ano atual
func (h *ExpenseHandler) MarkPaid(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	now := time.Now()
	month := int(now.Month())
	year := now.Year()

	var expense models.Expense
	if err := database.DB.First(&expense, id).Error; err != nil {
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
	id, _ := strconv.Atoi(c.Param("id"))
	now := time.Now()
	month := int(now.Month())
	year := now.Year()

	var expense models.Expense
	if err := database.DB.First(&expense, id).Error; err != nil {
		return c.String(http.StatusNotFound, "Despesa não encontrada")
	}

	// Remove pagamento
	database.DB.Where("expense_id = ? AND month = ? AND year = ?", id, month, year).Delete(&models.ExpensePayment{})

	return h.renderExpenseList(c, string(expense.Type))
}

func (h *ExpenseHandler) renderExpenseList(c echo.Context, expenseType string) error {
	now := time.Now()
	month := int(now.Month())
	year := now.Year()

	var expenses []models.Expense
	database.DB.Where("type = ?", expenseType).Order("due_day, name").Find(&expenses)

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
