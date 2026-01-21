package handlers

import (
	"time"

	"poc-finance/internal/security"
)

// Request types for group operations
type CreateGroupRequest struct {
	Name        string `form:"name"`
	Description string `form:"description"`
}

type CreateJointAccountRequest struct {
	Name string `form:"name"`
}

type RegisterAndJoinRequest struct {
	Email    string `form:"email"`
	Password string `form:"password"`
	Name     string `form:"name"`
}

// isValidGroupPassword validates a password and returns whether it's valid and an error message
func isValidGroupPassword(password string) (bool, string) {
	return security.ValidatePassword(password)
}

var monthNames = map[time.Month]string{
	time.January:   "Janeiro",
	time.February:  "Fevereiro",
	time.March:     "Março",
	time.April:     "Abril",
	time.May:       "Maio",
	time.June:      "Junho",
	time.July:      "Julho",
	time.August:    "Agosto",
	time.September: "Setembro",
	time.October:   "Outubro",
	time.November:  "Novembro",
	time.December:  "Dezembro",
}
