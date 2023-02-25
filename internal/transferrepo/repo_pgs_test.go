//go:build integration

package transferrepo_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"
	"time"

	"github.com/go-petr/pet-bank/internal/accountrepo"
	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/internal/integrationtest"
	"github.com/go-petr/pet-bank/internal/integrationtest/helpers"
	"github.com/go-petr/pet-bank/internal/middleware"
	"github.com/go-petr/pet-bank/internal/transferrepo"
	"github.com/go-petr/pet-bank/pkg/configpkg"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/go-petr/pet-bank/pkg/randompkg"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/shopspring/decimal"
)

var (
	dbDriver string
	dbSource string
	ctx      context.Context
)

func TestMain(m *testing.M) {
	config, err := configpkg.Load("../../configs")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	dbDriver = config.DBDriver
	dbSource = config.DBSource

	logger := middleware.CreateLogger(config)
	ctx = logger.WithContext(context.Background())
	// ctx = context.Background()

	os.Exit(m.Run())
}

func TestCreate(t *testing.T) {
	testCases := []struct {
		name         string
		wantTransfer func(tx *sql.Tx) domain.Transfer
		wantErr      error
	}{
		{
			name: "OK",
			wantTransfer: func(tx *sql.Tx) domain.Transfer {
				user1 := helpers.SeedUser(t, tx)
				account1 := helpers.SeedAccountWith1000USDBalance(t, tx, user1.Username)
				user2 := helpers.SeedUser(t, tx)
				account2 := helpers.SeedAccountWith1000USDBalance(t, tx, user2.Username)
				transfer := domain.Transfer{
					FromAccountID: account1.ID,
					ToAccountID:   account2.ID,
					Amount:        randompkg.MoneyAmountBetween(100, 1000),
					CreatedAt:     time.Now().UTC().Truncate(time.Second),
				}

				return transfer
			},
		},
		{
			name: "ErrFromAccountNotFound",
			wantTransfer: func(tx *sql.Tx) domain.Transfer {
				user1 := helpers.SeedUser(t, tx)
				account1 := helpers.SeedAccountWith1000USDBalance(t, tx, user1.Username)
				transfer := domain.Transfer{
					FromAccountID: account1.ID,
					ToAccountID:   0,
					Amount:        randompkg.MoneyAmountBetween(100, 1000),
					CreatedAt:     time.Now().UTC().Truncate(time.Second),
				}

				return transfer
			},
			wantErr: domain.ErrAccountNotFound,
		},
		{
			name: "ErrFromAccountNotFound",
			wantTransfer: func(tx *sql.Tx) domain.Transfer {
				user2 := helpers.SeedUser(t, tx)
				account2 := helpers.SeedAccountWith1000USDBalance(t, tx, user2.Username)
				transfer := domain.Transfer{
					FromAccountID: 0,
					ToAccountID:   account2.ID,
					Amount:        randompkg.MoneyAmountBetween(100, 1000),
					CreatedAt:     time.Now().UTC().Truncate(time.Second),
				}

				return transfer
			},
			wantErr: domain.ErrAccountNotFound,
		},
		{
			name: "InvalidAmount",
			wantTransfer: func(tx *sql.Tx) domain.Transfer {
				user1 := helpers.SeedUser(t, tx)
				account1 := helpers.SeedAccountWith1000USDBalance(t, tx, user1.Username)
				user2 := helpers.SeedUser(t, tx)
				account2 := helpers.SeedAccountWith1000USDBalance(t, tx, user2.Username)
				transfer := domain.Transfer{
					FromAccountID: account1.ID,
					ToAccountID:   account2.ID,
					Amount:        "0",
					CreatedAt:     time.Now().UTC().Truncate(time.Second),
				}

				return transfer
			},
			wantErr: domain.ErrInvalidAmount,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Prepare test transaction and seed database
			tx := integrationtest.SetupTX(t, dbDriver, dbSource)
			want := tc.wantTransfer(tx)
			transferRepo := transferrepo.NewTxRepoPGS(tx)

			arg := domain.CreateTransferParams{
				FromAccountID: want.FromAccountID,
				ToAccountID:   want.ToAccountID,
				Amount:        want.Amount,
			}

			// Run test
			got, err := transferRepo.Create(context.Background(), arg)
			if err != nil {
				if err == tc.wantErr {
					return
				}
				t.Fatalf(`transferRepo.Create(context.Background(), %v) returned error: %v`,
					arg, err.Error())
			}

			ignoreFields := cmpopts.IgnoreFields(domain.Transfer{}, "ID")
			compareCreatedAt := cmpopts.EquateApproxTime(time.Second)
			if diff := cmp.Diff(want, got, ignoreFields, compareCreatedAt); diff != "" {
				t.Errorf(`transferRepo.Create(context.Background(), %v) returned unexpected difference (-want +got):\n%s"`,
					arg, diff)
			}

			if got.ID == 0 {
				t.Error("got.ID = 0, want non-zero")
			}
		})
	}
}

func SeedTransfer(t *testing.T, tx *sql.Tx, fromAccountID, toAccountID int32, amount string) domain.Transfer {
	transferRepo := transferrepo.NewTxRepoPGS(tx)

	arg := domain.CreateTransferParams{
		FromAccountID: fromAccountID,
		ToAccountID:   toAccountID,
		Amount:        amount,
	}

	transfer, err := transferRepo.Create(context.Background(), arg)
	if err != nil {
		t.Fatalf(`transferRepo.Create(context.Background(), %v) returned error: %v`,
			arg, err.Error())
	}

	return transfer
}

func TestGet(t *testing.T) {
	testCases := []struct {
		name         string
		wantTransfer func(tx *sql.Tx) domain.Transfer
		wantErr      error
	}{
		{
			name: "OK",
			wantTransfer: func(tx *sql.Tx) domain.Transfer {
				user1 := helpers.SeedUser(t, tx)
				account1 := helpers.SeedAccountWith1000USDBalance(t, tx, user1.Username)
				user2 := helpers.SeedUser(t, tx)
				account2 := helpers.SeedAccountWith1000USDBalance(t, tx, user2.Username)
				transfer := SeedTransfer(t, tx, account1.ID, account2.ID, randompkg.MoneyAmountBetween(10, 100))

				return transfer
			},
		},
		{
			name: "ErrTransferNotFound",
			wantTransfer: func(tx *sql.Tx) domain.Transfer {
				return domain.Transfer{ID: 0}
			},
			wantErr: domain.ErrTransferNotFound,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Prepare test transaction and seed database
			tx := integrationtest.SetupTX(t, dbDriver, dbSource)
			want := tc.wantTransfer(tx)
			transferRepo := transferrepo.NewTxRepoPGS(tx)

			// Run test
			got, err := transferRepo.Get(context.Background(), want.ID)
			if err != nil {
				if err == tc.wantErr {
					return
				}
				t.Fatalf(`transferRepo.Get(context.Background(), %v) returned error: %v`,
					want.ID, err.Error())
			}

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf(`transferRepo.Get(context.Background(), %v) returned unexpected difference (-want +got):\n%s"`,
					want.ID, diff)
			}

			if got.ID == 0 {
				t.Error("got.ID = 0, want non-zero")
			}
		})
	}
}

func SeedTransfers(t *testing.T, tx *sql.Tx, fromAccountID, toAccountID int32, count int) []domain.Transfer {
	transfers := make([]domain.Transfer, count)

	for i := range transfers {
		transfers[i] = SeedTransfer(t, tx, fromAccountID, toAccountID, randompkg.MoneyAmountBetween(1, 10))
	}

	return transfers
}

func TestListTransfers(t *testing.T) {
	const transfersCount = 15

	testCases := []struct {
		name          string
		limit         int32
		offset        int32
		wantTransfers func(tx *sql.Tx, account1ID, account2ID int32) []domain.Transfer
		wantErr       error
	}{
		{
			name:   "ListAll",
			limit:  100,
			offset: 0,
			wantTransfers: func(tx *sql.Tx, account1ID, account2ID int32) []domain.Transfer {
				transfers := SeedTransfers(t, tx, account1ID, account2ID, transfersCount)
				return transfers
			},
		},
		{
			name:   "Limit5",
			limit:  5,
			offset: 0,
			wantTransfers: func(tx *sql.Tx, account1ID, account2ID int32) []domain.Transfer {
				transfers := SeedTransfers(t, tx, account1ID, account2ID, transfersCount)
				return transfers[:5]
			},
		},
		{
			name:   "Limit5Offset5",
			limit:  5,
			offset: 5,
			wantTransfers: func(tx *sql.Tx, account1ID, account2ID int32) []domain.Transfer {
				transfers := SeedTransfers(t, tx, account1ID, account2ID, transfersCount)
				return transfers[5:10]
			},
		},
		{
			name:   "NegativeLimit",
			limit:  -100,
			offset: 0,
			wantTransfers: func(tx *sql.Tx, account1ID, account2ID int32) []domain.Transfer {
				transfers := SeedTransfers(t, tx, account1ID, account2ID, transfersCount)
				return transfers[5:10]
			},
			wantErr: errorspkg.ErrInternal,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Prepare test transaction and seed database
			tx := integrationtest.SetupTX(t, dbDriver, dbSource)

			user1 := helpers.SeedUser(t, tx)
			account1 := helpers.SeedAccountWith1000USDBalance(t, tx, user1.Username)
			user2 := helpers.SeedUser(t, tx)
			account2 := helpers.SeedAccountWith1000USDBalance(t, tx, user2.Username)

			want := tc.wantTransfers(tx, account1.ID, account2.ID)
			firstTransfer := want[0]
			transferRepo := transferrepo.NewTxRepoPGS(tx)

			arg := domain.ListTransfersParams{
				FromAccountID: firstTransfer.FromAccountID,
				ToAccountID:   firstTransfer.ToAccountID,
				Limit:         tc.limit,
				Offset:        tc.offset,
			}

			// Run test
			got, err := transferRepo.List(context.Background(), arg)
			if err != nil {
				if err == tc.wantErr {
					return
				}
				t.Fatalf(`transferRepo.List(context.Background(), %v) returned unexpected error: %v`,
					arg, err)
			}

			compareCreatedAt := cmpopts.EquateApproxTime(time.Second)
			if diff := cmp.Diff(want, got, compareCreatedAt); diff != "" {
				t.Errorf(`transferRepo.List(context.Background(), %v) returned unexpected difference (-want +got):\n%s"`,
					arg, diff)
			}
		})
	}
}

func TestTransferTx(t *testing.T) {
	db := integrationtest.SetupDB(t, dbDriver, dbSource)

	user1 := helpers.SeedUser(t, db)
	account1 := helpers.SeedAccountWith1000USDBalance(t, db, user1.Username)
	user2 := helpers.SeedUser(t, db)
	account2 := helpers.SeedAccountWith1000USDBalance(t, db, user2.Username)

	transferRepo := transferrepo.NewRepoPGS(db)

	// run n concurrent transfer transactions
	n := 20
	amount := "10"

	errs := make(chan error)
	results := make(chan domain.TransferTxResult)

	arg := domain.CreateTransferParams{
		FromAccountID: account1.ID,
		ToAccountID:   account2.ID,
		Amount:        amount,
	}

	for i := 0; i < n; i++ {
		go func() {
			result, err := transferRepo.Transfer(ctx, arg)

			errs <- err
			results <- result
		}()
	}

	// check results

	existed := make(map[int]bool)

	wantTransfer := domain.Transfer{
		FromAccountID: account1.ID,
		ToAccountID:   account2.ID,
		Amount:        amount,
	}
	wantFromEntry := domain.Entry{AccountID: account1.ID, Amount: "-" + amount}
	wantToEntry := domain.Entry{AccountID: account2.ID, Amount: amount}

	account1BalanceBefore, err := decimal.NewFromString(account1.Balance)
	if err != nil {
		t.Fatalf("decimal.NewFromString(%v) returned error: %v", account1.Balance, err)
	}

	account2BalanceBefore, err := decimal.NewFromString(account2.Balance)
	if err != nil {
		t.Fatalf("decimal.NewFromString(%v) returned error: %v", account2.Balance, err)
	}

	amountDecimal, err := decimal.NewFromString(amount)
	if err != nil {
		t.Fatalf("decimal.NewFromString(%v) returned error: %v", amount, err)
	}

	for i := 0; i < n; i++ {
		err := <-errs
		if err != nil {
			t.Fatalf("transferRepo.Transfer(ctx, %+v) returned error: %v", arg, err)
		}

		got := <-results

		// check transfer
		ignoreFields := cmpopts.IgnoreFields(domain.Transfer{}, "ID", "CreatedAt")
		if diff := cmp.Diff(wantTransfer, got.Transfer, ignoreFields); diff != "" {
			t.Errorf(`transferRepo.Transfer(ctx, %v) returned unexpected difference (-want +got):\n%s"`,
				arg, diff)
		}

		// check entries
		ignoreFields = cmpopts.IgnoreFields(domain.Entry{}, "ID", "CreatedAt")
		if diff := cmp.Diff(wantFromEntry, got.FromEntry, ignoreFields); diff != "" {
			t.Errorf(`transferRepo.Transfer(ctx, %v) returned unexpected difference (-want +got):\n%s"`,
				arg, diff)
		}

		if diff := cmp.Diff(wantToEntry, got.ToEntry, ignoreFields); diff != "" {
			t.Errorf(`transferRepo.Transfer(ctx, %v) returned unexpected difference (-want +got):\n%s"`,
				arg, diff)
		}

		// check accounts's balances
		account1BalanceAfter, err := decimal.NewFromString(got.FromAccount.Balance)
		if err != nil {
			t.Fatalf("decimal.NewFromString(%v) returned error: %v", got.FromAccount.Balance, err)
		}

		account2BalanceAfter, err := decimal.NewFromString(got.ToAccount.Balance)
		if err != nil {
			t.Fatalf("decimal.NewFromString(%v) returned error: %v", got.ToAccount.Balance, err)
		}

		diff1 := account1BalanceBefore.Sub(account1BalanceAfter)
		diff2 := account2BalanceAfter.Sub(account2BalanceBefore)

		if !diff1.Equal(diff2) {
			t.Fatalf("diff1 = %v, diff2 = %v, want equal", diff1, diff2)
		}

		k := int(diff1.Div(amountDecimal).IntPart())
		if k < 1 || k > n {
			t.Fatalf("k = %v, want k >= 1 && k <= n", k)
		}

		if existed[k] {
			t.Fatalf("k = %v already exists, want k to be unique", k)
		}

		existed[k] = true
	}

	// check the final updated balance
	accountRepo := accountrepo.NewRepoPGS(db)

	updatedAccount1, err := accountRepo.Get(ctx, account1.ID)
	if err != nil {
		t.Errorf("accountRepo.Get(ctx, %v) returned error: %v", account1.ID, err)
	}

	updatedAccount2, err := accountRepo.Get(ctx, account2.ID)
	if err != nil {
		t.Errorf("accountRepo.Get(ctx, %v) returned error: %v", account2.ID, err)
	}

	amountTransfered := amountDecimal.Mul(decimal.NewFromInt(int64(n)))

	account1BalanceAfter := account1BalanceBefore.Sub(amountTransfered).String()
	if account1BalanceAfter != updatedAccount1.Balance {
		t.Errorf("account1BalanceAfter = %v, updatedAccount1.Balance = %v, want equal",
			account1BalanceAfter, updatedAccount1.Balance)
	}

	account2BalanceAfter := account1BalanceBefore.Add(amountTransfered).String()
	if account2BalanceAfter != updatedAccount2.Balance {
		t.Errorf("account2BalanceAfter = %v, updatedAccount2.Balance = %v, want equal",
			account2BalanceAfter, updatedAccount2.Balance)
	}
}

func TestTransferTxDeadlock(t *testing.T) {
	db := integrationtest.SetupDB(t, dbDriver, dbSource)

	user1 := helpers.SeedUser(t, db)
	account1 := helpers.SeedAccountWith1000USDBalance(t, db, user1.Username)
	user2 := helpers.SeedUser(t, db)
	account2 := helpers.SeedAccountWith1000USDBalance(t, db, user2.Username)

	transferRepo := transferrepo.NewRepoPGS(db)

	// run n concurrent transfer transactions
	n := 30
	amount := "10"

	errs := make(chan error)

	for i := 0; i < n; i++ {
		fromAccountID, toAccountID := account1.ID, account2.ID
		// Change transfer direction
		if i%2 == 0 {
			fromAccountID, toAccountID = account2.ID, account1.ID
		}

		arg := domain.CreateTransferParams{
			FromAccountID: fromAccountID,
			ToAccountID:   toAccountID,
			Amount:        amount,
		}

		go func() {
			_, err := transferRepo.Transfer(context.Background(), arg)
			errs <- err
		}()
	}

	// check results
	for i := 0; i < n; i++ {
		err := <-errs
		if err != nil {
			t.Errorf("transferRepo.Transfer(ctx, arg) returned error: %v", err)
		}
	}

	// check the final updated balance
	accountRepo := accountrepo.NewRepoPGS(db)

	updatedAccount1, err := accountRepo.Get(context.Background(), account1.ID)
	if err != nil {
		t.Errorf("accountRepo.Get(ctx, %v) returned error: %v", account1.ID, err)
	}

	updatedAccount2, err := accountRepo.Get(context.Background(), account2.ID)
	if err != nil {
		t.Errorf("accountRepo.Get(ctx, %v) returned error: %v", account2.ID, err)
	}

	if account1.Balance != updatedAccount1.Balance {
		t.Errorf("account1.Balance = %v, updatedAccount1.Balance = %v, want equal",
			account1.Balance, updatedAccount1.Balance)
	}

	if account2.Balance != updatedAccount2.Balance {
		t.Errorf("account2BalanceAfter = %v, updatedAccount2.Balance = %v, want equal",
			account2.Balance, updatedAccount2.Balance)
	}
}
