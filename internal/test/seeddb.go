// Package test provides shared test helpers.
package test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/go-petr/pet-bank/internal/accountrepo"
	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/internal/entryrepo"
	"github.com/go-petr/pet-bank/internal/userrepo"
	"github.com/go-petr/pet-bank/pkg/currencypkg"
	"github.com/go-petr/pet-bank/pkg/passpkg"
	"github.com/go-petr/pet-bank/pkg/randompkg"
)

// SeedUser creates random User inside a test transaction.
func SeedUser(t *testing.T, tx *sql.Tx) domain.User {
	t.Helper()

	hashedPassword, err := passpkg.Hash(randompkg.String(32))
	if err != nil {
		t.Fatalf("passpkg.Hash(randompkg.String(10)) returned error: %v", err)
	}

	arg := domain.CreateUserParams{
		Username:       randompkg.Owner(),
		HashedPassword: hashedPassword,
		FullName:       randompkg.String(10),
		Email:          randompkg.Email(),
	}

	userRepo := userrepo.NewRepoPGS(tx)
	user, err := userRepo.Create(context.Background(), arg)

	if err != nil {
		t.Fatalf("userRepo.Create(context.Background(), %+v) returned error: %v", arg, err)
	}

	return user
}

// SeedEntry creates Entry inside a test transaction.
func SeedEntry(t *testing.T, tx *sql.Tx, amount string, accountID int32) domain.Entry {
	t.Helper()

	entryRepo := entryrepo.NewRepoPGS(tx)

	entry, err := entryRepo.Create(context.Background(), amount, accountID)
	if err != nil {
		t.Fatalf("entryRepo.Create(context.Background(), %v, %v) returned error: %v",
			amount, accountID, err)
	}

	return entry
}

// SeedEntries creates Entries with random amounts inside a test transaction.
func SeedEntries(t *testing.T, tx *sql.Tx, count, accountID int32) []domain.Entry {
	t.Helper()

	entries := make([]domain.Entry, count)

	for i := range entries {
		entries[i] = SeedEntry(t, tx, randompkg.MoneyAmountBetween(-1000, 1000), accountID)
	}

	return entries
}

// SeedAccountWith1000USDBalance creates USD Account with 1000 USD on balance inside a test transaction.
func SeedAccountWith1000USDBalance(t *testing.T, tx *sql.Tx, username string) domain.Account {
	t.Helper()

	accountRepo := accountrepo.NewRepoPGS(tx)

	const balance = "1000"

	account, err := accountRepo.Create(context.Background(), username, balance, currencypkg.USD)
	if err != nil {
		stmt := `accountRepo.Create(context.Background(), %v, %v, %v) returned error: %v`
		t.Fatalf(stmt, username, balance, currencypkg.USD, err)
	}

	return account
}
