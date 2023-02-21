//go:build integration

package entryrepo

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"
	"time"

	"github.com/go-petr/pet-bank/internal/accountrepo"
	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/internal/userrepo"
	"github.com/go-petr/pet-bank/pkg/configpkg"
	"github.com/go-petr/pet-bank/pkg/currencypkg"
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

func SeedUser(t *testing.T, tx *sql.Tx) domain.User {
	t.Helper()

	arg := domain.CreateUserParams{
		Username:       "Username",
		HashedPassword: "HashedPassword",
		FullName:       "FullName",
		Email:          "Email@Email.com",
	}

	userRepo := userrepo.NewRepoPGS(tx)
	user, err := userRepo.Create(context.Background(), arg)
	if err != nil {
		t.Fatalf("userRepo.Create(context.Background(), %+v) returned error: %v", arg, err)
	}

	return user
}

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

func TestCreate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		wantEntry func(tx *sql.Tx) domain.Entry
		wantErr   error
	}{
		{
			name: "OK",
			wantEntry: func(tx *sql.Tx) domain.Entry {
				user := SeedUser(t, tx)
				account := SeedAccountWith1000USDBalance(t, tx, user.Username)
				return domain.Entry{AccountID: account.ID, Amount: randompkg.MoneyAmountBetween(-100, 100)}
			},
		},
		{
			name: "NullAmount",
			wantEntry: func(tx *sql.Tx) domain.Entry {
				user := SeedUser(t, tx)
				account := SeedAccountWith1000USDBalance(t, tx, user.Username)
				return domain.Entry{AccountID: account.ID, Amount: ""}
			},
			wantErr: errorspkg.ErrInternal,
		},
		{
			name: "ConstraintViolation:entries_account_id_fkey",
			wantEntry: func(tx *sql.Tx) domain.Entry {
				return domain.Entry{AccountID: -100500, Amount: randompkg.MoneyAmountBetween(-100, 100)}
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
			want := tc.wantEntry(tx)
			entryRepo := NewRepoPGS(tx)

			// Run test
			got, err := entryRepo.Create(context.Background(), want.Amount, want.AccountID)
			if err != nil {
				if err == tc.wantErr {
					return
				}
				t.Fatalf(`entryRepo.Create(context.Background(), %v, %v) returned error: %v`,
					want.Amount, want.AccountID, err.Error())
			}

			ignoreFields := cmpopts.IgnoreFields(domain.Entry{}, "ID", "CreatedAt")
			if diff := cmp.Diff(want, got, ignoreFields); diff != "" {
				t.Errorf(`entryRepo.Create(context.Background(), %v, %v) returned unexpected difference (-want +got):\n%s"`,
					want.Amount, want.AccountID, diff)
			}

			if got.ID == 0 {
				t.Error("got.ID = 0, want non-zero")
			}

			if !cmp.Equal(got.CreatedAt, time.Now(), cmpopts.EquateApproxTime(time.Second)) {
				t.Errorf("got.CreatedAt = %v, want %v +- minute",
					got.CreatedAt.Truncate(time.Second), time.Now().UTC().Truncate(time.Second))
			}
		})
	}
}

func SeedEntry(t *testing.T, tx *sql.Tx, amount string, accountID int32) domain.Entry {
	t.Helper()

	entryRepo := NewRepoPGS(tx)

	entry, err := entryRepo.Create(context.Background(), amount, accountID)
	if err != nil {
		t.Fatalf("entryRepo.Create(context.Background(), %v, %v) returned error: %v",
			amount, accountID, err)
	}

	return entry
}

func SeedEntries(t *testing.T, tx *sql.Tx, count, accountID int32) []domain.Entry {
	t.Helper()

	entries := make([]domain.Entry, count)

	for i := range entries {
		entries[i] = SeedEntry(t, tx, randompkg.MoneyAmountBetween(-1000, 1000), accountID)
	}

	return entries
}

func TestGet(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		wantEntry func(tx *sql.Tx) domain.Entry
		wantErr   error
	}{
		{
			name: "OK",
			wantEntry: func(tx *sql.Tx) domain.Entry {
				user := SeedUser(t, tx)
				account := SeedAccountWith1000USDBalance(t, tx, user.Username)
				wantEntry := SeedEntry(t, tx, randompkg.MoneyAmountBetween(-10, 10), account.ID)

				return wantEntry
			},
		},
		{
			name: "ErrEntryNotFound",
			wantEntry: func(tx *sql.Tx) domain.Entry {
				return domain.Entry{ID: 0}
			},
			wantErr: domain.ErrEntryNotFound,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			// t.Parallel()

			// Prepare test transaction and seed database
			tx := dbpkg.SetupTX(t, dbDriver, dbSource)
			want := tc.wantEntry(tx)
			entryRepo := NewRepoPGS(tx)

			// Run test
			got, err := entryRepo.Get(context.Background(), want.ID)
			if err != nil {
				if err == tc.wantErr {
					return
				}

				t.Errorf(`entryRepo.Get(context.Background(), %v) returned unexpected error: %v`,
					want.ID, err)
				return
			}

			ignoreFields := cmpopts.IgnoreFields(domain.Entry{}, "CreatedAt")
			if diff := cmp.Diff(want, got, ignoreFields); diff != "" {
				t.Errorf(`entryRepo.Get(context.Background(), %v) returned unexpected difference (-want +got):\n%s"`,
					want.ID, diff)
			}

			if got.ID == 0 {
				t.Error("got.ID = 0, want non-zero")
			}

			if !cmp.Equal(got.CreatedAt, want.CreatedAt, cmpopts.EquateApproxTime(time.Second)) {
				t.Errorf("got.CreatedAt = %v, want %v +- minute",
					got.CreatedAt.Truncate(time.Second), want.CreatedAt.Truncate(time.Second))
			}
		})
	}
}

func TestList(t *testing.T) {
	t.Parallel()
	const entriesCount = 30

	testCases := []struct {
		name                    string
		limit                   int32
		offset                  int32
		wantAccountIDAndEntries func(tx *sql.Tx) (int32, []domain.Entry)
		wantErr                 error
	}{
		{
			name:   "ListAll",
			limit:  100,
			offset: 0,
			wantAccountIDAndEntries: func(tx *sql.Tx) (int32, []domain.Entry) {
				user := SeedUser(t, tx)
				account := SeedAccountWith1000USDBalance(t, tx, user.Username)
				entries := SeedEntries(t, tx, entriesCount, account.ID)

				return account.ID, entries
			},
		},
		{
			name:   "Limit10",
			limit:  10,
			offset: 0,
			wantAccountIDAndEntries: func(tx *sql.Tx) (int32, []domain.Entry) {
				user := SeedUser(t, tx)
				account := SeedAccountWith1000USDBalance(t, tx, user.Username)
				entries := SeedEntries(t, tx, entriesCount, account.ID)

				return account.ID, entries[:10]
			},
		},
		{
			name:   "Limit10Offset10",
			limit:  10,
			offset: 10,
			wantAccountIDAndEntries: func(tx *sql.Tx) (int32, []domain.Entry) {
				user := SeedUser(t, tx)
				account := SeedAccountWith1000USDBalance(t, tx, user.Username)
				entries := SeedEntries(t, tx, entriesCount, account.ID)

				return account.ID, entries[10:20]
			},
		},
		{
			name:   "NoEntries",
			limit:  100,
			offset: 0,
			wantAccountIDAndEntries: func(tx *sql.Tx) (int32, []domain.Entry) {
				return 0, []domain.Entry{}
			},
		},
		{
			name:    "NegativeLimit",
			limit:   -100,
			offset:  0,
			wantErr: errorspkg.ErrInternal,
			wantAccountIDAndEntries: func(tx *sql.Tx) (int32, []domain.Entry) {
				return 0, []domain.Entry{}
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Prepare test transaction and seed database
			tx := dbpkg.SetupTX(t, dbDriver, dbSource)
			wantAccountID, wantEntries := tc.wantAccountIDAndEntries(tx)
			entryRepo := NewRepoPGS(tx)

			// Run test
			entries, err := entryRepo.List(context.Background(), wantAccountID, tc.limit, tc.offset)
			if err != nil {
				if err == tc.wantErr {
					return
				}

				t.Fatalf(`entryRepo.List(context.Background(), %v, %v, %v) returned unexpected error: %v`,
					wantAccountID, tc.limit, tc.offset, err)
			}

			ignoreFields := cmpopts.IgnoreFields(domain.Entry{}, "CreatedAt")
			if diff := cmp.Diff(wantEntries, entries, ignoreFields); diff != "" {
				t.Errorf(`entryRepo.List(context.Background(), %v, %v, %v) returned unexpected difference (-want +got):\n%s"`,
					wantAccountID, tc.limit, tc.offset, diff)
			}
		})
	}
}
