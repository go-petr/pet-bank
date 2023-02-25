// Package currencypkg provides common currency related functionality for apps.
package currencypkg

import "github.com/go-playground/validator/v10"

// Constants for all supported currencies.
const (
	USD = "USD"
	EUR = "EUR"
	RMB = "RMB"
)

// SupportedCurrencies holds all the supported currencies.
var SupportedCurrencies = []string{
	USD,
	EUR,
	RMB,
}

// IsSupportedCurrency returns true if the currncy is supported.
func IsSupportedCurrency(currency string) bool {
	for _, c := range SupportedCurrencies {
		if c == currency {
			return true
		}
	}

	return false
}

// ValidCurrency validates whether the currency is supported.
var ValidCurrency validator.Func = func(fl validator.FieldLevel) bool {
	if c, ok := fl.Field().Interface().(string); ok {
		return IsSupportedCurrency(c)
	}
	return false
}
