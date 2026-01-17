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

type CreditCardHandler struct {
	accountService *services.AccountService
}

func NewCreditCardHandler() *CreditCardHandler {
	return &CreditCardHandler{
		accountService: services.NewAccountService(),
	}
}

type CreateCardRequest struct {
	Name        string  `form:"name"`
	ClosingDay  int     `form:"closing_day"`
	DueDay      int     `form:"due_day"`
	LimitAmount float64 `form:"limit_amount"`
}

type CreateInstallmentRequest struct {
	CreditCardID      uint    `form:"credit_card_id"`
	Description       string  `form:"description"`
	TotalAmount       float64 `form:"total_amount"`
	TotalInstallments int     `form:"total_installments"`
	StartDate         string  `form:"start_date"`
	Category          string  `form:"category"`
}

func (h *CreditCardHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	accountIDs, _ := h.accountService.GetUserAccountIDs(userID)

	var cards []models.CreditCard
	database.DB.Where("account_id IN ?", accountIDs).Preload("Installments").Find(&cards)

	// Calcula o total de cada cartão para o mês atual
	now := time.Now()
	cardTotals := make(map[uint]float64)

	for _, card := range cards {
		for _, inst := range card.Installments {
			// Verifica se a parcela ainda está ativa neste mês
			monthsPassed := monthsBetween(inst.StartDate, now)
			if monthsPassed < inst.TotalInstallments {
				cardTotals[card.ID] += inst.InstallmentAmount
			}
		}
	}

	// Lista de parcelas ativas (only from user's cards)
	var activeInstallments []models.Installment
	cardIDs := make([]uint, len(cards))
	for i, card := range cards {
		cardIDs[i] = card.ID
	}
	if len(cardIDs) > 0 {
		database.DB.Where("credit_card_id IN ?", cardIDs).Preload("CreditCard").Find(&activeInstallments)
	}

	// Filtra parcelas ativas
	var filtered []models.Installment
	for _, inst := range activeInstallments {
		monthsPassed := monthsBetween(inst.StartDate, now)
		if monthsPassed < inst.TotalInstallments {
			inst.CurrentInstallment = monthsPassed + 1
			filtered = append(filtered, inst)
		}
	}

	data := map[string]interface{}{
		"cards":        cards,
		"cardTotals":   cardTotals,
		"installments": filtered,
		"categories":   getExpenseCategories(),
	}

	return c.Render(http.StatusOK, "cards.html", data)
}

func (h *CreditCardHandler) CreateCard(c echo.Context) error {
	userID := middleware.GetUserID(c)
	account, err := h.accountService.GetUserIndividualAccount(userID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Conta não encontrada")
	}

	var req CreateCardRequest
	if err := c.Bind(&req); err != nil {
		return c.String(http.StatusBadRequest, "Dados inválidos")
	}

	card := models.CreditCard{
		AccountID:   account.ID,
		Name:        req.Name,
		ClosingDay:  req.ClosingDay,
		DueDay:      req.DueDay,
		LimitAmount: req.LimitAmount,
	}

	if err := database.DB.Create(&card).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao criar cartão")
	}

	accountIDs, _ := h.accountService.GetUserAccountIDs(userID)
	var cards []models.CreditCard
	database.DB.Where("account_id IN ?", accountIDs).Find(&cards)

	return c.Render(http.StatusOK, "partials/card-list.html", map[string]interface{}{
		"cards": cards,
	})
}

func (h *CreditCardHandler) DeleteCard(c echo.Context) error {
	userID := middleware.GetUserID(c)
	accountIDs, _ := h.accountService.GetUserAccountIDs(userID)

	id, _ := strconv.Atoi(c.Param("id"))

	// Verify card belongs to user before deleting
	var card models.CreditCard
	if err := database.DB.Where("id = ? AND account_id IN ?", id, accountIDs).First(&card).Error; err != nil {
		return c.String(http.StatusNotFound, "Cartão não encontrado")
	}

	// Deleta parcelas associadas
	database.DB.Where("credit_card_id = ?", id).Delete(&models.Installment{})
	database.DB.Delete(&card)

	var cards []models.CreditCard
	database.DB.Where("account_id IN ?", accountIDs).Find(&cards)

	return c.Render(http.StatusOK, "partials/card-list.html", map[string]interface{}{
		"cards": cards,
	})
}

func (h *CreditCardHandler) CreateInstallment(c echo.Context) error {
	userID := middleware.GetUserID(c)
	accountIDs, _ := h.accountService.GetUserAccountIDs(userID)

	var req CreateInstallmentRequest
	if err := c.Bind(&req); err != nil {
		return c.String(http.StatusBadRequest, "Dados inválidos")
	}

	// Verify card belongs to user
	var card models.CreditCard
	if err := database.DB.Where("id = ? AND account_id IN ?", req.CreditCardID, accountIDs).First(&card).Error; err != nil {
		return c.String(http.StatusNotFound, "Cartão não encontrado")
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return c.String(http.StatusBadRequest, "Data inválida")
	}

	installmentAmount := req.TotalAmount / float64(req.TotalInstallments)

	installment := models.Installment{
		CreditCardID:       req.CreditCardID,
		Description:        req.Description,
		TotalAmount:        req.TotalAmount,
		InstallmentAmount:  installmentAmount,
		TotalInstallments:  req.TotalInstallments,
		CurrentInstallment: 1,
		StartDate:          startDate,
		Category:           req.Category,
	}

	if err := database.DB.Create(&installment).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao criar parcelamento")
	}

	return h.renderInstallmentList(c)
}

func (h *CreditCardHandler) DeleteInstallment(c echo.Context) error {
	userID := middleware.GetUserID(c)
	accountIDs, _ := h.accountService.GetUserAccountIDs(userID)

	id, _ := strconv.Atoi(c.Param("id"))

	// Verify installment belongs to user's card
	var installment models.Installment
	if err := database.DB.Preload("CreditCard").First(&installment, id).Error; err != nil {
		return c.String(http.StatusNotFound, "Parcela não encontrada")
	}

	// Check if the credit card belongs to user's accounts
	found := false
	for _, accID := range accountIDs {
		if installment.CreditCard.AccountID == accID {
			found = true
			break
		}
	}
	if !found {
		return c.String(http.StatusNotFound, "Parcela não encontrada")
	}

	database.DB.Delete(&installment)
	return h.renderInstallmentList(c)
}

func (h *CreditCardHandler) renderInstallmentList(c echo.Context) error {
	userID := middleware.GetUserID(c)
	accountIDs, _ := h.accountService.GetUserAccountIDs(userID)

	// Get user's cards first
	var cards []models.CreditCard
	database.DB.Where("account_id IN ?", accountIDs).Find(&cards)

	cardIDs := make([]uint, len(cards))
	for i, card := range cards {
		cardIDs[i] = card.ID
	}

	now := time.Now()
	var installments []models.Installment
	if len(cardIDs) > 0 {
		database.DB.Where("credit_card_id IN ?", cardIDs).Preload("CreditCard").Find(&installments)
	}

	var filtered []models.Installment
	for _, inst := range installments {
		monthsPassed := monthsBetween(inst.StartDate, now)
		if monthsPassed < inst.TotalInstallments {
			inst.CurrentInstallment = monthsPassed + 1
			filtered = append(filtered, inst)
		}
	}

	return c.Render(http.StatusOK, "partials/installment-list.html", map[string]interface{}{
		"installments": filtered,
	})
}

func monthsBetween(start, end time.Time) int {
	years := end.Year() - start.Year()
	months := int(end.Month()) - int(start.Month())
	return years*12 + months
}
