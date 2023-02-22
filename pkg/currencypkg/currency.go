// Package currencypkg provides common currency related functionality for apps.
package currencypkg

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
