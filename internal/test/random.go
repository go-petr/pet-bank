package test

import (
	"time"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/pkg/randompkg"
)

// RandomAccount returns random account owned by the given owner.
func RandomAccount(owner string) domain.Account {
	return domain.Account{
		ID:        randompkg.IntBetween(1, 100),
		Owner:     owner,
		Balance:   randompkg.MoneyAmountBetween(1000, 10_000),
		Currency:  randompkg.Currency(),
		CreatedAt: time.Now().Truncate(time.Second).UTC(),
	}
}
