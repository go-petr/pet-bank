//go:build integration

package transferrepo

import (
	"testing"
)

func TestCreate(t *testing.T) {

}

func TestGet(t *testing.T) {

}

func TestListTransfers(t *testing.T) {

}

// func TestTransferTx(t *testing.T) {
// 	testUser1 := createRandomUser(t)
// 	testUser2 := createRandomUser(t)
// 	testAccount1 := createRandomAccount(t, testUser1)
// 	testAccount2 := createRandomAccount(t, testUser2)

// 	testAccount1BalanceBefore, err := decimal.NewFromString(testAccount1.Balance)
// 	require.NoError(t, err)
// 	testAccount2BalanceBefore, err := decimal.NewFromString(testAccount2.Balance)
// 	require.NoError(t, err)

// 	// run n concurrent transfer transactions
// 	n := 20
// 	amount := "10"
// 	amountDecimal, err := decimal.NewFromString(amount)
// 	require.NoError(t, err)

// 	errs := make(chan error)
// 	results := make(chan domain.TransferTxResult)

// 	for i := 0; i < n; i++ {
// 		go func() {
// 			result, err := testTransferRepo.Transfer(context.Background(), domain.CreateTransferParams{
// 				FromAccountID: testAccount1.ID,
// 				ToAccountID:   testAccount2.ID,
// 				Amount:        amount,
// 			})

// 			errs <- err
// 			results <- result
// 		}()
// 	}

// 	existed := make(map[int]bool)

// 	// check results
// 	for i := 0; i < n; i++ {
// 		err := <-errs
// 		require.NoError(t, err)

// 		result := <-results
// 		require.NotEmpty(t, result)

// 		// check transfer
// 		transfer := result.Transfer
// 		require.NotEmpty(t, transfer)
// 		require.Equal(t, testAccount1.ID, transfer.FromAccountID)
// 		require.Equal(t, testAccount2.ID, transfer.ToAccountID)
// 		require.Equal(t, amount, transfer.Amount)
// 		require.NotZero(t, transfer.ID)
// 		require.NotZero(t, transfer.CreatedAt)

// 		_, err = testTransferRepo.Get(context.Background(), transfer.ID)
// 		require.NoError(t, err)

// 		// check entries
// 		fromEntry := result.FromEntry
// 		require.NotEmpty(t, fromEntry)
// 		require.Equal(t, testAccount1.ID, fromEntry.AccountID)
// 		require.Equal(t, "-"+amount, fromEntry.Amount)
// 		require.NotZero(t, fromEntry.ID)
// 		require.NotZero(t, fromEntry.CreatedAt)

// 		_, err = testEntryRepo.Get(context.Background(), fromEntry.ID)
// 		require.NoError(t, err)

// 		toEntry := result.ToEntry
// 		require.NotEmpty(t, toEntry)
// 		require.Equal(t, testAccount2.ID, toEntry.AccountID)
// 		require.Equal(t, amount, toEntry.Amount)
// 		require.NotZero(t, toEntry.ID)
// 		require.NotZero(t, toEntry.CreatedAt)

// 		_, err = testEntryRepo.Get(context.Background(), toEntry.ID)
// 		require.NoError(t, err)

// 		// check accounts
// 		fromAccount := result.FromAccount
// 		require.NotEmpty(t, fromAccount)
// 		require.Equal(t, testAccount1.ID, fromAccount.ID)

// 		toAccount := result.ToAccount
// 		require.NotEmpty(t, toAccount)
// 		require.Equal(t, testAccount2.ID, toAccount.ID)

// 		// check accounts's balances
// 		testAccount1BalanceAfter, err := decimal.NewFromString(fromAccount.Balance)
// 		require.NoError(t, err)
// 		testAccount2BalanceAfter, err := decimal.NewFromString(toAccount.Balance)
// 		require.NoError(t, err)

// 		diff1 := testAccount1BalanceBefore.Sub(testAccount1BalanceAfter)
// 		diff2 := testAccount2BalanceAfter.Sub(testAccount2BalanceBefore)
// 		require.Equal(t, diff1.String(), diff2.String())
// 		require.True(t, diff1.GreaterThan(decimal.Zero))
// 		require.True(t, diff1.Mod(amountDecimal).IsZero())

// 		k := int(diff1.Div(amountDecimal).IntPart())
// 		require.True(t, k >= 1 && k <= n)
// 		require.NotContains(t, existed, k)
// 		existed[k] = true
// 	}

// 	// check the final updated balance
// 	updatedAccount1, err := testAccountRepo.Get(context.Background(), testAccount1.ID)
// 	require.NoError(t, err)

// 	updatedAccount2, err := testAccountRepo.Get(context.Background(), testAccount2.ID)
// 	require.NoError(t, err)

// 	require.Equal(t,
// 		testAccount1BalanceBefore.Sub(amountDecimal.Mul(decimal.NewFromInt(int64(n)))).String(),
// 		updatedAccount1.Balance)
// 	require.Equal(t,
// 		testAccount2BalanceBefore.Add(amountDecimal.Mul(decimal.NewFromInt(int64(n)))).String(),
// 		updatedAccount2.Balance)
// }

// func TestTransferTxDeadlock(t *testing.T) {
// 	testUser1 := createRandomUser(t)
// 	testUser2 := createRandomUser(t)
// 	testAccount1 := createRandomAccount(t, testUser1)
// 	testAccount2 := createRandomAccount(t, testUser2)

// 	// run n concurrent transfer transactions
// 	n := 20
// 	amount := "10"

// 	errs := make(chan error)

// 	for i := 0; i < n; i++ {
// 		fromAccountID, toAccountID := testAccount1.ID, testAccount2.ID
// 		if i%2 == 0 {
// 			fromAccountID, toAccountID = testAccount2.ID, testAccount1.ID
// 		}

// 		go func() {
// 			_, err := testTransferRepo.Transfer(context.Background(), domain.CreateTransferParams{
// 				FromAccountID: fromAccountID,
// 				ToAccountID:   toAccountID,
// 				Amount:        amount,
// 			})

// 			errs <- err
// 		}()
// 	}

// 	// check results
// 	for i := 0; i < n; i++ {
// 		err := <-errs
// 		require.NoError(t, err)
// 	}

// 	// check the final updated balance
// 	updatedAccount1, err := testAccountRepo.Get(context.Background(), testAccount1.ID)
// 	require.NoError(t, err)

// 	updatedAccount2, err := testAccountRepo.Get(context.Background(), testAccount2.ID)
// 	require.NoError(t, err)

// 	require.Equal(t, testAccount1.Balance, updatedAccount1.Balance)
// 	require.Equal(t, testAccount2.Balance, updatedAccount2.Balance)
// }
