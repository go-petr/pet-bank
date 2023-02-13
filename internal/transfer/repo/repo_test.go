package repo

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/go-petr/pet-bank/internal/account"
	ar "github.com/go-petr/pet-bank/internal/account/repo"
	er "github.com/go-petr/pet-bank/internal/entry/repo"
	"github.com/go-petr/pet-bank/internal/transfer"
	"github.com/go-petr/pet-bank/internal/user"
	ur "github.com/go-petr/pet-bank/internal/user/repo"
	"github.com/go-petr/pet-bank/pkg/configpkg"
	"github.com/go-petr/pet-bank/pkg/passpkg"
	"github.com/go-petr/pet-bank/pkg/apprandom"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

var (
	testTransferRepo *transferRepo
	testAccountRepo  *ar.AccountRepo
	testUserRepo     *ur.UserRepo
	testEntryRepo    *er.EntryRepo
)

func TestMain(m *testing.M) {
	config, err := configpkg.Load("../../../configs")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	testDB, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	testUserRepo = ur.NewUserRepo(testDB)
	testAccountRepo = ar.NewAccountRepo(testDB)
	testEntryRepo = er.NewEntryRepo(testDB)

	testTransferRepo = NewTransferRepo(testDB)

	os.Exit(m.Run())
}

func createRandomUser(t *testing.T) user.User {

	hashedPassword, err := passpkg.Hash(apprandom.String(10))
	require.NoError(t, err)

	arg := user.CreateUserParams{
		Username:       apprandom.Owner(),
		HashedPassword: hashedPassword,
		FullName:       apprandom.Owner(),
		Email:          apprandom.Email(),
	}

	testUser, err := testUserRepo.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, testUser)

	require.Equal(t, arg.Username, testUser.Username)
	require.Equal(t, arg.HashedPassword, testUser.HashedPassword)
	require.Equal(t, arg.FullName, testUser.FullName)
	require.Equal(t, arg.Email, testUser.Email)

	require.NotZero(t, testUser.CreatedAt)

	return testUser
}

func createRandomAccount(t *testing.T, testUser user.User) account.Account {

	// create random account
	argAccount := account.CreateAccountParams{
		Owner:    testUser.Username,
		Balance:  apprandom.MoneyAmountBetween(1_000, 10_000),
		Currency: apprandom.Currency(),
	}

	account, err := testAccountRepo.CreateAccount(context.Background(), argAccount)
	require.NoError(t, err)
	require.NotEmpty(t, account)

	require.Equal(t, argAccount.Owner, account.Owner)
	require.Equal(t, argAccount.Balance, account.Balance)
	require.Equal(t, argAccount.Currency, account.Currency)

	require.NotZero(t, account.ID)
	require.NotZero(t, account.CreatedAt)

	return account
}

func createRandomTransfer(t *testing.T, testAccount1, testAccount2 account.Account) transfer.Transfer {

	arg := transfer.CreateTransferParams{
		FromAccountID: testAccount1.ID,
		ToAccountID:   testAccount2.ID,
		Amount:        apprandom.MoneyAmountBetween(10, 100),
	}

	transfer, err := testTransferRepo.CreateTransfer(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, transfer)

	require.Equal(t, arg.FromAccountID, transfer.FromAccountID)
	require.Equal(t, arg.ToAccountID, transfer.ToAccountID)
	require.Equal(t, arg.Amount, transfer.Amount)

	require.NotZero(t, transfer.ID)
	require.NotZero(t, transfer.CreatedAt)

	return transfer
}

func TestCreateTransfer(t *testing.T) {
	testUser1 := createRandomUser(t)
	testUser2 := createRandomUser(t)
	testAccount1 := createRandomAccount(t, testUser1)
	testAccount2 := createRandomAccount(t, testUser2)
	createRandomTransfer(t, testAccount1, testAccount2)
}

func TestGetTransfer(t *testing.T) {
	testUser1 := createRandomUser(t)
	testUser2 := createRandomUser(t)
	testAccount1 := createRandomAccount(t, testUser1)
	testAccount2 := createRandomAccount(t, testUser2)
	transfer1 := createRandomTransfer(t, testAccount1, testAccount2)

	transfer2, err := testTransferRepo.GetTransfer(context.Background(), transfer1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, transfer2)

	require.Equal(t, transfer1.ID, transfer2.ID)
	require.Equal(t, transfer1.FromAccountID, transfer2.FromAccountID)
	require.Equal(t, transfer1.ToAccountID, transfer2.ToAccountID)
	require.Equal(t, transfer1.Amount, transfer2.Amount)
	require.Equal(t, transfer1.CreatedAt, transfer2.CreatedAt)
}

func TestListTransfers(t *testing.T) {
	testUser1 := createRandomUser(t)
	testUser2 := createRandomUser(t)
	testAccount1 := createRandomAccount(t, testUser1)
	testAccount2 := createRandomAccount(t, testUser2)

	for i := 0; i < 10; i++ {
		createRandomTransfer(t, testAccount1, testAccount2)
	}

	arg := transfer.ListTransfersParams{
		FromAccountID: testAccount1.ID,
		ToAccountID:   testAccount2.ID,
		Limit:         5,
		Offset:        5,
	}

	transfers, err := testTransferRepo.ListTransfers(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, transfers)
	require.Len(t, transfers, 5)

	for _, transfer := range transfers {
		require.NotEmpty(t, transfer)
		require.True(t, transfer.FromAccountID == testAccount1.ID || transfer.ToAccountID == testAccount1.ID)
	}
}

func TestTransferTx(t *testing.T) {

	testUser1 := createRandomUser(t)
	testUser2 := createRandomUser(t)
	testAccount1 := createRandomAccount(t, testUser1)
	testAccount2 := createRandomAccount(t, testUser2)

	testAccount1BalanceBefore, err := decimal.NewFromString(testAccount1.Balance)
	require.NoError(t, err)
	testAccount2BalanceBefore, err := decimal.NewFromString(testAccount2.Balance)
	require.NoError(t, err)

	// run n concurrent transfer transactions
	n := 20
	amount := "10"
	amountDecimal, err := decimal.NewFromString(amount)
	require.NoError(t, err)

	errs := make(chan error)
	results := make(chan transfer.TransferTxResult)

	for i := 0; i < n; i++ {
		go func() {

			result, err := testTransferRepo.TransferTx(context.Background(), transfer.CreateTransferParams{
				FromAccountID: testAccount1.ID,
				ToAccountID:   testAccount2.ID,
				Amount:        amount,
			})

			errs <- err
			results <- result

		}()
	}

	existed := make(map[int]bool)
	// check results
	for i := 0; i < n; i++ {
		err := <-errs
		require.NoError(t, err)

		result := <-results
		require.NotEmpty(t, result)

		// check transfer
		transfer := result.Transfer
		require.NotEmpty(t, transfer)
		require.Equal(t, testAccount1.ID, transfer.FromAccountID)
		require.Equal(t, testAccount2.ID, transfer.ToAccountID)
		require.Equal(t, amount, transfer.Amount)
		require.NotZero(t, transfer.ID)
		require.NotZero(t, transfer.CreatedAt)

		_, err = testTransferRepo.GetTransfer(context.Background(), transfer.ID)
		require.NoError(t, err)

		// check entries
		fromEntry := result.FromEntry
		require.NotEmpty(t, fromEntry)
		require.Equal(t, testAccount1.ID, fromEntry.AccountID)
		require.Equal(t, "-"+amount, fromEntry.Amount)
		require.NotZero(t, fromEntry.ID)
		require.NotZero(t, fromEntry.CreatedAt)

		_, err = testEntryRepo.GetEntry(context.Background(), fromEntry.ID)
		require.NoError(t, err)

		toEntry := result.ToEntry
		require.NotEmpty(t, toEntry)
		require.Equal(t, testAccount2.ID, toEntry.AccountID)
		require.Equal(t, amount, toEntry.Amount)
		require.NotZero(t, toEntry.ID)
		require.NotZero(t, toEntry.CreatedAt)

		_, err = testEntryRepo.GetEntry(context.Background(), toEntry.ID)
		require.NoError(t, err)

		// check accounts
		fromAccount := result.FromAccount
		require.NotEmpty(t, fromAccount)
		require.Equal(t, testAccount1.ID, fromAccount.ID)

		toAccount := result.ToAccount
		require.NotEmpty(t, toAccount)
		require.Equal(t, testAccount2.ID, toAccount.ID)

		// check accounts's balances
		testAccount1BalanceAfter, err := decimal.NewFromString(fromAccount.Balance)
		require.NoError(t, err)
		testAccount2BalanceAfter, err := decimal.NewFromString(toAccount.Balance)
		require.NoError(t, err)

		diff1 := testAccount1BalanceBefore.Sub(testAccount1BalanceAfter)
		diff2 := testAccount2BalanceAfter.Sub(testAccount2BalanceBefore)
		require.Equal(t, diff1.String(), diff2.String())
		require.True(t, diff1.GreaterThan(decimal.Zero))
		require.True(t, diff1.Mod(amountDecimal).IsZero())

		k := int(diff1.Div(amountDecimal).IntPart())
		require.True(t, k >= 1 && k <= n)
		require.NotContains(t, existed, k)
		existed[k] = true
	}

	// check the final updated balance
	updatedAccount1, err := testAccountRepo.GetAccount(context.Background(), testAccount1.ID)
	require.NoError(t, err)

	updatedAccount2, err := testAccountRepo.GetAccount(context.Background(), testAccount2.ID)
	require.NoError(t, err)

	require.Equal(t,
		testAccount1BalanceBefore.Sub(amountDecimal.Mul(decimal.NewFromInt(int64(n)))).String(),
		updatedAccount1.Balance)
	require.Equal(t,
		testAccount2BalanceBefore.Add(amountDecimal.Mul(decimal.NewFromInt(int64(n)))).String(),
		updatedAccount2.Balance)
}

func TestTransferTxDeadlock(t *testing.T) {

	testUser1 := createRandomUser(t)
	testUser2 := createRandomUser(t)
	testAccount1 := createRandomAccount(t, testUser1)
	testAccount2 := createRandomAccount(t, testUser2)

	// run n concurrent transfer transactions
	n := 20
	amount := "10"

	errs := make(chan error)

	for i := 0; i < n; i++ {

		fromAccountID, toAccountID := testAccount1.ID, testAccount2.ID
		if i%2 == 0 {
			fromAccountID, toAccountID = testAccount2.ID, testAccount1.ID
		}

		go func() {

			_, err := testTransferRepo.TransferTx(context.Background(), transfer.CreateTransferParams{
				FromAccountID: fromAccountID,
				ToAccountID:   toAccountID,
				Amount:        amount,
			})

			errs <- err

		}()
	}

	// check results
	for i := 0; i < n; i++ {
		err := <-errs
		require.NoError(t, err)

	}

	// check the final updated balance
	updatedAccount1, err := testAccountRepo.GetAccount(context.Background(), testAccount1.ID)
	require.NoError(t, err)

	updatedAccount2, err := testAccountRepo.GetAccount(context.Background(), testAccount2.ID)
	require.NoError(t, err)

	require.Equal(t, testAccount1.Balance, updatedAccount1.Balance)
	require.Equal(t, testAccount2.Balance, updatedAccount2.Balance)
}
