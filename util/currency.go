package util

// Constants for all supported currencies
const (
	USD = "USD"
	EUR = "EUR"
	RMB = "RMB"
)

// IsSupportedCurrency returns true if the currncy is supported
func IsSupportedCurrency(currency string) bool {
	switch currency {
	case USD, EUR, RMB:
		return true
	default:
		return false
	}
}
