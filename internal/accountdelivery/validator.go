package accountdelivery

import (
	"github.com/go-petr/pet-bank/pkg/currencypkg"
	"github.com/go-playground/validator/v10"
)

// ValidCurrency validates whether the currency is supported.
var ValidCurrency validator.Func = func(fl validator.FieldLevel) bool {
	if c, ok := fl.Field().Interface().(string); ok {
		return currencypkg.IsSupportedCurrency(c)
	}
	return false
}
