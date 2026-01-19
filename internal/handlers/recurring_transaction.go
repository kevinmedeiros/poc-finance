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

type RecurringTransactionHandler struct {
	accountService *services.AccountService
}

func NewRecurringTransactionHandler() *RecurringTransactionHandler {
	return &RecurringTransactionHandler{
		accountService: services.NewAccountService(),
	}
}

type CreateRecurringTransactionRequest struct {
	AccountID       uint    `form:"account_id"`
	TransactionType string  `form:"transaction_type"`
	Frequency       string  `form:"frequency"`
	Amount          float64 `form:"amount"`
	Description     string  `form:"description"`
	StartDate       string  `form:"start_date"`
	EndDate         string  `form:"end_date"`
	Category        string  `form:"category"`
}

func (h *RecurringTransactionHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	accountIDs, _ := h.accountService.GetUserAccountIDs(userID)
	accounts, _ := h.accountService.GetUserAccounts(userID)

	var activeRecurringTransactions []models.RecurringTransaction
	var pausedRecurringTransactions []models.RecurringTransaction

	database.DB.Preload("Account").Where("account_id IN ? AND active = ?", accountIDs, true).Order("next_run_date ASC").Find(&activeRecurringTransactions)
	database.DB.Preload("Account").Where("account_id IN ? AND active = ?", accountIDs, false).Order("next_run_date ASC").Find(&pausedRecurringTransactions)

	data := map[string]interface{}{
		"activeRecurringTransactions": activeRecurringTransactions,
		"pausedRecurringTransactions": pausedRecurringTransactions,
		"accounts":                    accounts,
		"transactionTypes":            []string{"expense", "income"},
		"frequencies":                 []string{"daily", "weekly", "monthly", "yearly"},
	}

	return c.Render(http.StatusOK, "recurring.html", data)
}

func (h *RecurringTransactionHandler) Create(c echo.Context) error {
	userID := middleware.GetUserID(c)

	var req CreateRecurringTransactionRequest
	if err := c.Bind(&req); err != nil {
		return c.String(http.StatusBadRequest, "Dados inválidos")
	}

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

	// Validate transaction type
	transactionType := models.TransactionType(req.TransactionType)
	if transactionType != models.TransactionTypeExpense && transactionType != models.TransactionTypeIncome {
		return c.String(http.StatusBadRequest, "Tipo de transação inválido")
	}

	// Validate frequency
	frequency := models.Frequency(req.Frequency)
	if frequency != models.FrequencyDaily && frequency != models.FrequencyWeekly &&
		frequency != models.FrequencyMonthly && frequency != models.FrequencyYearly {
		return c.String(http.StatusBadRequest, "Frequência inválida")
	}

	// Parse start date
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return c.String(http.StatusBadRequest, "Data de início inválida")
	}

	// Parse end date (optional)
	var endDate *time.Time
	if req.EndDate != "" {
		parsedEndDate, err := time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			return c.String(http.StatusBadRequest, "Data de término inválida")
		}
		endDate = &parsedEndDate
	}

	// Validate amount
	if req.Amount <= 0 {
		return c.String(http.StatusBadRequest, "Valor deve ser maior que zero")
	}

	// Set next run date to start date
	nextRunDate := startDate

	recurringTransaction := models.RecurringTransaction{
		AccountID:       accountID,
		TransactionType: transactionType,
		Frequency:       frequency,
		Amount:          req.Amount,
		Description:     req.Description,
		StartDate:       startDate,
		EndDate:         endDate,
		NextRunDate:     nextRunDate,
		Active:          true,
		Category:        req.Category,
	}

	if err := database.DB.Create(&recurringTransaction).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao criar transação recorrente")
	}

	// Return updated list
	accountIDs, _ := h.accountService.GetUserAccountIDs(userID)
	var activeRecurringTransactions []models.RecurringTransaction
	database.DB.Preload("Account").Where("account_id IN ? AND active = ?", accountIDs, true).Order("next_run_date ASC").Find(&activeRecurringTransactions)

	return c.Render(http.StatusOK, "partials/recurring-list.html", map[string]interface{}{
		"activeRecurringTransactions": activeRecurringTransactions,
	})
}

func (h *RecurringTransactionHandler) Update(c echo.Context) error {
	userID := middleware.GetUserID(c)
	accountIDs, _ := h.accountService.GetUserAccountIDs(userID)

	id, _ := strconv.Atoi(c.Param("id"))

	var recurringTransaction models.RecurringTransaction
	if err := database.DB.Where("id = ? AND account_id IN ?", id, accountIDs).First(&recurringTransaction).Error; err != nil {
		return c.String(http.StatusNotFound, "Transação recorrente não encontrada")
	}

	var req CreateRecurringTransactionRequest
	if err := c.Bind(&req); err != nil {
		return c.String(http.StatusBadRequest, "Dados inválidos")
	}

	// Validate account access if changing account
	if req.AccountID != 0 && req.AccountID != recurringTransaction.AccountID {
		if !h.accountService.CanUserAccessAccount(userID, req.AccountID) {
			return c.String(http.StatusForbidden, "Acesso negado à conta selecionada")
		}
		recurringTransaction.AccountID = req.AccountID
	}

	// Validate and update transaction type
	if req.TransactionType != "" {
		transactionType := models.TransactionType(req.TransactionType)
		if transactionType != models.TransactionTypeExpense && transactionType != models.TransactionTypeIncome {
			return c.String(http.StatusBadRequest, "Tipo de transação inválido")
		}
		recurringTransaction.TransactionType = transactionType
	}

	// Validate and update frequency
	if req.Frequency != "" {
		frequency := models.Frequency(req.Frequency)
		if frequency != models.FrequencyDaily && frequency != models.FrequencyWeekly &&
			frequency != models.FrequencyMonthly && frequency != models.FrequencyYearly {
			return c.String(http.StatusBadRequest, "Frequência inválida")
		}
		recurringTransaction.Frequency = frequency
	}

	// Update amount
	if req.Amount > 0 {
		recurringTransaction.Amount = req.Amount
	}

	// Update description
	if req.Description != "" {
		recurringTransaction.Description = req.Description
	}

	// Update category
	if req.Category != "" {
		recurringTransaction.Category = req.Category
	}

	// Update start date
	if req.StartDate != "" {
		startDate, err := time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			return c.String(http.StatusBadRequest, "Data de início inválida")
		}
		recurringTransaction.StartDate = startDate
	}

	// Update end date (optional)
	if req.EndDate != "" {
		parsedEndDate, err := time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			return c.String(http.StatusBadRequest, "Data de término inválida")
		}
		recurringTransaction.EndDate = &parsedEndDate
	}

	if err := database.DB.Save(&recurringTransaction).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao atualizar transação recorrente")
	}

	// Return updated list
	var activeRecurringTransactions []models.RecurringTransaction
	database.DB.Preload("Account").Where("account_id IN ? AND active = ?", accountIDs, true).Order("next_run_date ASC").Find(&activeRecurringTransactions)

	return c.Render(http.StatusOK, "partials/recurring-list.html", map[string]interface{}{
		"activeRecurringTransactions": activeRecurringTransactions,
	})
}

func (h *RecurringTransactionHandler) Delete(c echo.Context) error {
	userID := middleware.GetUserID(c)
	accountIDs, _ := h.accountService.GetUserAccountIDs(userID)

	id, _ := strconv.Atoi(c.Param("id"))

	// Verify the recurring transaction belongs to user's accounts before deleting
	var recurringTransaction models.RecurringTransaction
	if err := database.DB.Where("id = ? AND account_id IN ?", id, accountIDs).First(&recurringTransaction).Error; err != nil {
		return c.String(http.StatusNotFound, "Transação recorrente não encontrada")
	}

	if err := database.DB.Delete(&recurringTransaction).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao deletar")
	}

	// Return updated list
	var activeRecurringTransactions []models.RecurringTransaction
	database.DB.Preload("Account").Where("account_id IN ? AND active = ?", accountIDs, true).Order("next_run_date ASC").Find(&activeRecurringTransactions)

	return c.Render(http.StatusOK, "partials/recurring-list.html", map[string]interface{}{
		"activeRecurringTransactions": activeRecurringTransactions,
	})
}

func (h *RecurringTransactionHandler) Toggle(c echo.Context) error {
	userID := middleware.GetUserID(c)
	accountIDs, _ := h.accountService.GetUserAccountIDs(userID)

	id, _ := strconv.Atoi(c.Param("id"))

	var recurringTransaction models.RecurringTransaction
	if err := database.DB.Where("id = ? AND account_id IN ?", id, accountIDs).First(&recurringTransaction).Error; err != nil {
		return c.String(http.StatusNotFound, "Transação recorrente não encontrada")
	}

	recurringTransaction.Active = !recurringTransaction.Active
	database.DB.Save(&recurringTransaction)

	// Return updated lists (both active and paused)
	var activeRecurringTransactions []models.RecurringTransaction
	var pausedRecurringTransactions []models.RecurringTransaction

	database.DB.Preload("Account").Where("account_id IN ? AND active = ?", accountIDs, true).Order("next_run_date ASC").Find(&activeRecurringTransactions)
	database.DB.Preload("Account").Where("account_id IN ? AND active = ?", accountIDs, false).Order("next_run_date ASC").Find(&pausedRecurringTransactions)

	return c.Render(http.StatusOK, "partials/recurring-list.html", map[string]interface{}{
		"activeRecurringTransactions": activeRecurringTransactions,
		"pausedRecurringTransactions": pausedRecurringTransactions,
	})
}
