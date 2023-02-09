// Package currency provides common currency related functionality for apps.
package currency

// Constants for all supported currencies.
const (
	USD = "USD"
	EUR = "EUR"
	RMB = "RMB"
)

// IsSupportedCurrency returns true if the currncy is supported.
func IsSupportedCurrency(currency string) bool {
	switch currency {
	case USD, EUR, RMB:
		return true
	default:
		return false
	}
}
