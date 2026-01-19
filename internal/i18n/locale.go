// Package i18n provides internationalization utilities for the poc-finance application.
package i18n

import "time"

// MonthNames maps time.Month values to their Portuguese names.
// This is used for direct lookups when you have a time.Month value.
var MonthNames = map[time.Month]string{
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

// MonthNamesSlice provides Portuguese month names as an ordered slice (Janeiro to Dezembro).
// This is used for index-based access where the index represents the month number (0 = Janeiro).
var MonthNamesSlice = []string{
	"Janeiro",
	"Fevereiro",
	"Março",
	"Abril",
	"Maio",
	"Junho",
	"Julho",
	"Agosto",
	"Setembro",
	"Outubro",
	"Novembro",
	"Dezembro",
}
