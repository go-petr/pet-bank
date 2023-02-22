//go:build integration

package accountrepo_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	"testing"

	"github.com/go-petr/pet-bank/internal/accountrepo"
	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/internal/test"
	"github.com/go-petr/pet-bank/pkg/configpkg"
	"github.com/go-petr/pet-bank/pkg/dbpkg"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/go-petr/pet-bank/pkg/randompkg"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var (
	dbDriver string
	dbSource string
)

func TestMain(m *testing.M) {
	config, err := configpkg.Load("../../configs")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	dbDriver = config.DBDriver
	dbSource = config.DBSource

	os.Exit(m.Run())
}

func TestCreate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		wantAccount func(tx *sql.Tx) domain.Account
		wantErr     error
	}{
		{
			name: "OK",
			wantAccount: func(tx *sql.Tx) domain.Account {
				user := test.SeedUser(t, tx)
				return domain.Account{
					Owner:     user.Username,
					Balance:   randompkg.MoneyAmountBetween(100, 1000),
					Currency:  randompkg.Currency(),
					CreatedAt: time.Now().UTC().Truncate(time.Second),
				}
			},
		},
		{
			name: "ErrOwnerNotFound",
			wantAccount: func(tx *sql.Tx) domain.Account {
				return domain.Account{
					Owner:     "ErrOwnerNotFound",
					Balance:   randompkg.MoneyAmountBetween(100, 1000),
					Currency:  randompkg.Currency(),
					CreatedAt: time.Now().UTC().Truncate(time.Second),
				}
			},
			wantErr: domain.ErrOwnerNotFound,
		},
		{
			name: "ErrCurrencyAlreadyExists",
			wantAccount: func(tx *sql.Tx) domain.Account {
				user := test.SeedUser(t, tx)
				account := test.SeedAccountWith1000USDBalance(t, tx, user.Username)
				return domain.Account{
					Owner:     user.Username,
					Balance:   randompkg.MoneyAmountBetween(100, 1000),
					Currency:  account.Currency,
					CreatedAt: time.Now().UTC().Truncate(time.Second),
				}
			},
			wantErr: domain.ErrCurrencyAlreadyExists,
		},
		{
			name: "InvalidBalance",
			wantAccount: func(tx *sql.Tx) domain.Account {
				user := test.SeedUser(t, tx)
				account := test.SeedAccountWith1000USDBalance(t, tx, user.Username)
				return domain.Account{
					Owner:     user.Username,
					Balance:   "",
					Currency:  account.Currency,
					CreatedAt: time.Now().UTC().Truncate(time.Second),
				}
			},
			wantErr: errorspkg.ErrInternal,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Prepare test transaction and seed database
			tx := dbpkg.SetupTX(t, dbDriver, dbSource)
			want := tc.wantAccount(tx)
			accountRepo := accountrepo.NewRepoPGS(tx)

			// Run test
			got, err := accountRepo.Create(context.Background(), want.Owner, want.Balance, want.Currency)
			if err != nil {
				if err == tc.wantErr {
					return
				}
				t.Fatalf(`accountRepo.Create(context.Background(), %v, %v, %v) returned error: %v`,
					want.Owner, want.Balance, want.Currency, err.Error())
			}

			ignoreFields := cmpopts.IgnoreFields(domain.Account{}, "ID")
			compareCreatedAt := cmpopts.EquateApproxTime(time.Second)
			if diff := cmp.Diff(want, got, ignoreFields, compareCreatedAt); diff != "" {
				t.Errorf(`accountRepo.Create(context.Background(), %v, %v, %v) returned unexpected difference (-want +got):\n%s"`,
					want.Owner, want.Balance, want.Currency, diff)
			}

			if got.ID == 0 {
				t.Error("got.ID = 0, want non-zero")
			}
		})
	}
}

func TestGet(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		wantAccount func(tx *sql.Tx) domain.Account
		wantErr     error
	}{
		{
			name: "OK",
			wantAccount: func(tx *sql.Tx) domain.Account {
				user := test.SeedUser(t, tx)
				account := test.SeedAccountWith1000USDBalance(t, tx, user.Username)
				return account
			},
		},
		{
			name: "ErrAccountNotFound",
			wantAccount: func(tx *sql.Tx) domain.Account {
				return domain.Account{ID: 0}
			},
			wantErr: domain.ErrAccountNotFound,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Prepare test transaction and seed database
			tx := dbpkg.SetupTX(t, dbDriver, dbSource)
			want := tc.wantAccount(tx)
			accountRepo := accountrepo.NewRepoPGS(tx)

			// Run test
			got, err := accountRepo.Get(context.Background(), want.ID)
			if err != nil {
				if err == tc.wantErr {
					return
				}
				t.Fatalf(`accountRepo.Get(context.Background(), %v) returned error: %v`,
					want.ID, err.Error())
			}

			compareCreatedAt := cmpopts.EquateApproxTime(time.Second)
			if diff := cmp.Diff(want, got, compareCreatedAt); diff != "" {
				t.Errorf(`accountRepo.Get(context.Background(), %v) returned unexpected difference (-want +got):\n%s"`,
					want.ID, diff)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		wantAccount func(tx *sql.Tx) domain.Account
		wantErr     error
	}{
		{
			name: "OK",
			wantAccount: func(tx *sql.Tx) domain.Account {
				user := test.SeedUser(t, tx)
				account := test.SeedAccountWith1000USDBalance(t, tx, user.Username)
				return account
			},
		},
		{
			name: "ErrAccountNotFound",
			wantAccount: func(tx *sql.Tx) domain.Account {
				return domain.Account{ID: 0}
			},
			wantErr: domain.ErrAccountNotFound,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Prepare test transaction and seed database
			tx := dbpkg.SetupTX(t, dbDriver, dbSource)
			want := tc.wantAccount(tx)
			accountRepo := accountrepo.NewRepoPGS(tx)

			// Run test
			err := accountRepo.Delete(context.Background(), want.ID)
			if err != nil {
				if err == tc.wantErr {
					return
				}
				t.Fatalf(`accountRepo.Delete(context.Background(), %v) returned error: %v`,
					want.ID, err.Error())
			}
		})
	}
}

func TestList(t *testing.T) {
	t.Parallel()
	const entriesCount = 30

	testCases := []struct {
		name         string
		limit        int32
		offset       int32
		wantAccounts func(tx *sql.Tx) []domain.Account
		wantErr      error
	}{
		{
			name:   "ListAll",
			limit:  100,
			offset: 0,
			wantAccounts: func(tx *sql.Tx) []domain.Account {
				user := test.SeedUser(t, tx)
				accounts := test.SeedAllCurrenciesAccountsWith1000Balance(t, tx, user.Username)
				return accounts
			},
		},
		{
			name:   "Limit10",
			limit:  2,
			offset: 0,
			wantAccounts: func(tx *sql.Tx) []domain.Account {
				user := test.SeedUser(t, tx)
				accounts := test.SeedAllCurrenciesAccountsWith1000Balance(t, tx, user.Username)
				return accounts[:2]
			},
		},
		{
			name:   "Limit10Offset10",
			limit:  2,
			offset: 1,
			wantAccounts: func(tx *sql.Tx) []domain.Account {
				user := test.SeedUser(t, tx)
				accounts := test.SeedAllCurrenciesAccountsWith1000Balance(t, tx, user.Username)
				return accounts[1:3]
			},
		},
		{
			name:   "NegativeLimit",
			limit:  -100,
			offset: 0,
			wantAccounts: func(tx *sql.Tx) []domain.Account {
				user := test.SeedUser(t, tx)
				accounts := test.SeedAllCurrenciesAccountsWith1000Balance(t, tx, user.Username)
				return accounts[1:3]
			},
			wantErr: errorspkg.ErrInternal,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Prepare test transaction and seed database
			tx := dbpkg.SetupTX(t, dbDriver, dbSource)
			want := tc.wantAccounts(tx)
			wantOwner := want[0].Owner
			accountRepo := accountrepo.NewRepoPGS(tx)

			// Run test
			got, err := accountRepo.List(context.Background(), wantOwner, tc.limit, tc.offset)
			if err != nil {
				if err == tc.wantErr {
					return
				}
				t.Fatalf(`accountRepo.List(context.Background(), %v, %v, %v) returned unexpected error: %v`,
					wantOwner, tc.limit, tc.offset, err)
			}

			compareCreatedAt := cmpopts.EquateApproxTime(time.Second)
			if diff := cmp.Diff(want, got, compareCreatedAt); diff != "" {
				t.Errorf(`accountRepo.List(context.Background(), %v, %v, %v) returned unexpected difference (-want +got):\n%s"`,
					wantOwner, tc.limit, tc.offset, diff)
			}
		})
	}
}

func TestAddBalance(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		amount      string
		wantAccount func(tx *sql.Tx) domain.Account
		wantErr     error
	}{
		{
			name:   "OK",
			amount: "100",
			wantAccount: func(tx *sql.Tx) domain.Account {
				user := test.SeedUser(t, tx)
				account := test.SeedAccountWith1000Balance(t, tx, user.Username, randompkg.Currency())
				account.Balance = "1100"
				return account
			},
		},
		{
			name:   "ErrAccountNotFound",
			amount: "100",
			wantAccount: func(tx *sql.Tx) domain.Account {
				return domain.Account{ID: 0}
			},
			wantErr: domain.ErrAccountNotFound,
		},
		{
			name:   "IvalidAmount",
			amount: "",
			wantAccount: func(tx *sql.Tx) domain.Account {
				return domain.Account{ID: 0}
			},
			wantErr: errorspkg.ErrInternal,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Prepare test transaction and seed database
			tx := dbpkg.SetupTX(t, dbDriver, dbSource)
			want := tc.wantAccount(tx)
			accountRepo := accountrepo.NewRepoPGS(tx)

			// Run test
			got, err := accountRepo.AddBalance(context.Background(), tc.amount, want.ID)
			if err != nil {
				if err == tc.wantErr {
					return
				}
				t.Fatalf(`accountRepo.AddBalance(context.Background(), %v, %v) returned error: %v`,
					tc.amount, want.ID, err.Error())
			}

			compareCreatedAt := cmpopts.EquateApproxTime(time.Second)
			if diff := cmp.Diff(want, got, compareCreatedAt); diff != "" {
				t.Errorf(`accountRepo.AddBalance(context.Background(), %v, %v) returned unexpected difference (-want +got):\n%s"`,
					tc.amount, want.ID, diff)
			}

			if got.ID == 0 {
				t.Error("got.ID = 0, want non-zero")
			}
		})
	}
}
