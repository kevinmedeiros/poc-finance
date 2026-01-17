package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"poc-finance/internal/middleware"
	"poc-finance/internal/services"
)

type AccountHandler struct {
	accountService *services.AccountService
}

func NewAccountHandler() *AccountHandler {
	return &AccountHandler{
		accountService: services.NewAccountService(),
	}
}

func (h *AccountHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)

	balances, err := h.accountService.GetUserAccountsWithBalances(userID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao buscar contas")
	}

	// Calculate total balance across all accounts
	var totalBalance float64
	for _, b := range balances {
		totalBalance += b.Balance
	}

	data := map[string]interface{}{
		"accounts":     balances,
		"totalBalance": totalBalance,
	}

	return c.Render(http.StatusOK, "accounts.html", data)
}
