package handlers

import (
	"regexp"
	"time"
)

// isValidGroupPassword checks password complexity requirements
func isValidGroupPassword(password string) (bool, string) {
	if len(password) < 8 {
		return false, "A senha deve ter pelo menos 8 caracteres"
	}
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)

	if !hasUpper || !hasLower || !hasNumber {
		return false, "A senha deve conter letras maiúsculas, minúsculas e números"
	}
	return true, ""
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
