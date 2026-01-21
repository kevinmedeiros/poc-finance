package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/database"
	"poc-finance/internal/middleware"
	"poc-finance/internal/models"
	"poc-finance/internal/services"
)

type IncomeHandler struct {
	accountService *services.AccountService
	cacheService   *services.SettingsCacheService
}

func NewIncomeHandler(cacheService *services.SettingsCacheService) *IncomeHandler {
	return &IncomeHandler{
		accountService: services.NewAccountService(),
		cacheService:   cacheService,
	}
}

type CreateIncomeRequest struct {
	AccountID    uint    `form:"account_id"`
	Date         string  `form:"date"`
	AmountUSD    float64 `form:"amount_usd"`
	ExchangeRate float64 `form:"exchange_rate"`
	Description  string  `form:"description"`
}

func (h *IncomeHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	accountIDs, _ := h.accountService.GetUserAccountIDs(userID)
	accounts, _ := h.accountService.GetUserAccounts(userID)

	var incomes []models.Income
	database.DB.Where("account_id IN ?", accountIDs).Order("date DESC").Find(&incomes)

	// Get settings for manual bracket override
	settingsData := h.cacheService.GetSettingsData()

	// Calcula faturamento 12 meses para mostrar na tela
	revenue12M := services.GetRevenue12MonthsForAccounts(database.DB, accountIDs)
	bracket, rate, nextAt := services.GetBracketInfoWithManualOverride(revenue12M, settingsData.ManualBracket)

	data := map[string]interface{}{
		"incomes":        incomes,
		"accounts":       accounts,
		"revenue12m":     revenue12M,
		"currentBracket": bracket,
		"effectiveRate":  rate,
		"nextBracketAt":  nextAt,
		"manualBracket":  settingsData.ManualBracket,
	}

	return c.Render(http.StatusOK, "income.html", data)
}

func (h *IncomeHandler) Create(c echo.Context) error {
	userID := middleware.GetUserID(c)

	var req CreateIncomeRequest
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

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return c.String(http.StatusBadRequest, "Data inválida")
	}

	// Calcula valores
	amountBRL := req.AmountUSD * req.ExchangeRate

	// Get settings for manual bracket override
	settingsData := h.cacheService.GetSettingsData()

	// Busca faturamento dos últimos 12 meses para calcular imposto
	accountIDs, _ := h.accountService.GetUserAccountIDs(userID)
	revenue12M := services.GetRevenue12MonthsForAccounts(database.DB, accountIDs)
	log.Printf("[Income] Creating income - ManualBracket: %d, Revenue12M: %.2f, AmountBRL: %.2f", settingsData.ManualBracket, revenue12M, amountBRL)
	taxCalc := services.CalculateTaxWithManualBracket(revenue12M, amountBRL, settingsData.ManualBracket)

	income := models.Income{
		AccountID:    accountID,
		Date:         date,
		AmountUSD:    req.AmountUSD,
		ExchangeRate: req.ExchangeRate,
		AmountBRL:    amountBRL,
		GrossAmount:  amountBRL,
		TaxAmount:    taxCalc.TaxAmount,
		NetAmount:    taxCalc.NetAmount,
		Description:  req.Description,
	}

	if err := database.DB.Create(&income).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao criar recebimento")
	}

	// Retorna a lista atualizada (para HTMX)
	var incomes []models.Income
	database.DB.Where("account_id IN ?", accountIDs).Order("date DESC").Find(&incomes)

	return c.Render(http.StatusOK, "partials/income-list.html", map[string]interface{}{
		"incomes": incomes,
	})
}

func (h *IncomeHandler) Delete(c echo.Context) error {
	userID := middleware.GetUserID(c)
	accountIDs, _ := h.accountService.GetUserAccountIDs(userID)

	id, _ := strconv.Atoi(c.Param("id"))

	// Verify the income belongs to user's accounts before deleting
	var income models.Income
	if err := database.DB.Where("id = ? AND account_id IN ?", id, accountIDs).First(&income).Error; err != nil {
		return c.String(http.StatusNotFound, "Recebimento não encontrado")
	}

	if err := database.DB.Delete(&income).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao deletar")
	}

	var incomes []models.Income
	database.DB.Where("account_id IN ?", accountIDs).Order("date DESC").Find(&incomes)

	return c.Render(http.StatusOK, "partials/income-list.html", map[string]interface{}{
		"incomes": incomes,
	})
}

// CalculatePreview retorna uma prévia do cálculo sem salvar
func (h *IncomeHandler) CalculatePreview(c echo.Context) error {
	userID := middleware.GetUserID(c)
	accountIDs, _ := h.accountService.GetUserAccountIDs(userID)

	amountUSD, _ := strconv.ParseFloat(c.QueryParam("amount_usd"), 64)
	exchangeRate, _ := strconv.ParseFloat(c.QueryParam("exchange_rate"), 64)

	if amountUSD <= 0 || exchangeRate <= 0 {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"amount_brl": 0,
			"tax":        0,
			"net":        0,
		})
	}

	amountBRL := amountUSD * exchangeRate

	// Get settings for manual bracket override
	settingsData := h.cacheService.GetSettingsData()

	revenue12M := services.GetRevenue12MonthsForAccounts(database.DB, accountIDs)
	taxCalc := services.CalculateTaxWithManualBracket(revenue12M, amountBRL, settingsData.ManualBracket)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"amount_brl":     amountBRL,
		"tax":            taxCalc.TaxAmount,
		"net":            taxCalc.NetAmount,
		"effective_rate": taxCalc.EffectiveRate * 100,
	})
}
