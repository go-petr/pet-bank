package accountdelivery

import (
	"github.com/go-petr/pet-bank/pkg/currency"
	"github.com/go-playground/validator/v10"
)

// ValidCurrency validates whether the currency is supported.
var ValidCurrency validator.Func = func(fl validator.FieldLevel) bool {
	if c, ok := fl.Field().Interface().(string); ok {
		return currency.IsSupportedCurrency(c)
	}
	return false
}
