package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
	"poc-finance/internal/services"
)

type IncomeHandler struct{}

func NewIncomeHandler() *IncomeHandler {
	return &IncomeHandler{}
}

type CreateIncomeRequest struct {
	Date         string  `form:"date"`
	AmountUSD    float64 `form:"amount_usd"`
	ExchangeRate float64 `form:"exchange_rate"`
	Description  string  `form:"description"`
}

func (h *IncomeHandler) List(c echo.Context) error {
	var incomes []models.Income
	database.DB.Order("date DESC").Find(&incomes)

	// Calcula faturamento 12 meses para mostrar na tela
	revenue12M := services.GetRevenue12Months(database.DB)
	bracket, rate, nextAt := services.GetBracketInfo(revenue12M)

	data := map[string]interface{}{
		"incomes":       incomes,
		"revenue12m":    revenue12M,
		"currentBracket": bracket,
		"effectiveRate": rate,
		"nextBracketAt": nextAt,
	}

	return c.Render(http.StatusOK, "income.html", data)
}

func (h *IncomeHandler) Create(c echo.Context) error {
	var req CreateIncomeRequest
	if err := c.Bind(&req); err != nil {
		return c.String(http.StatusBadRequest, "Dados inválidos")
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return c.String(http.StatusBadRequest, "Data inválida")
	}

	// Calcula valores
	amountBRL := req.AmountUSD * req.ExchangeRate

	// Busca faturamento dos últimos 12 meses para calcular imposto
	revenue12M := services.GetRevenue12Months(database.DB)
	taxCalc := services.CalculateTax(revenue12M, amountBRL)

	income := models.Income{
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
	database.DB.Order("date DESC").Find(&incomes)

	return c.Render(http.StatusOK, "partials/income-list.html", map[string]interface{}{
		"incomes": incomes,
	})
}

func (h *IncomeHandler) Delete(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	if err := database.DB.Delete(&models.Income{}, id).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao deletar")
	}

	var incomes []models.Income
	database.DB.Order("date DESC").Find(&incomes)

	return c.Render(http.StatusOK, "partials/income-list.html", map[string]interface{}{
		"incomes": incomes,
	})
}

// CalculatePreview retorna uma prévia do cálculo sem salvar
func (h *IncomeHandler) CalculatePreview(c echo.Context) error {
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
	revenue12M := services.GetRevenue12Months(database.DB)
	taxCalc := services.CalculateTax(revenue12M, amountBRL)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"amount_brl":     amountBRL,
		"tax":            taxCalc.TaxAmount,
		"net":            taxCalc.NetAmount,
		"effective_rate": taxCalc.EffectiveRate * 100,
	})
}
