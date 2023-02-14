package repo

import (
	"context"
	"database/sql"
	"log"
	"os"

	"testing"
	"time"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/internal/user/repo"
	"github.com/go-petr/pet-bank/pkg/configpkg"
	"github.com/go-petr/pet-bank/pkg/passpkg"
	"github.com/go-petr/pet-bank/pkg/randompkg"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

var (
	testAccountRepo *AccountRepo
	testUserRepo    *repo.UserRepo
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

	testAccountRepo = NewAccountRepo(testDB)
	testUserRepo = repo.NewUserRepo(testDB)

	os.Exit(m.Run())
}

func createRandomUser(t *testing.T) domain.User {

	hashedPassword, err := passpkg.Hash(randompkg.String(10))
	require.NoError(t, err)

	arg := domain.CreateUserParams{
		Username:       randompkg.Owner(),
		HashedPassword: hashedPassword,
		FullName:       randompkg.Owner(),
		Email:          randompkg.Owner(),
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

func createRandomAccount(t *testing.T, testUser domain.User) domain.Account {

	testBalance := randompkg.MoneyAmountBetween(1_000, 10_000)
	testCurrency := randompkg.Currency()

	account, err := testAccountRepo.CreateAccount(context.Background(), testUser.Username, testBalance, testCurrency)
	require.NoError(t, err)
	require.NotEmpty(t, account)

	require.Equal(t, testUser.Username, account.Owner)
	require.Equal(t, testBalance, account.Balance)
	require.Equal(t, testCurrency, account.Currency)

	require.NotZero(t, account.ID)
	require.NotZero(t, account.CreatedAt)

	return account
}

func TestCreateAccount(t *testing.T) {
	testUser := createRandomUser(t)
	createRandomAccount(t, testUser)
}

func TestCreateAccountConstraintViolations(t *testing.T) {

	testUser := createRandomUser(t)
	testAccount := createRandomAccount(t, testUser)

	testCases := []struct {
		name  string
		input struct {
			owner    string
			balance  string
			currency string
		}
		checkResponse func(response domain.Account, err error)
	}{
		{
			name: "ErrOwnerNotFound",
			input: struct {
				owner    string
				balance  string
				currency string
			}{
				"NotFound",
				randompkg.MoneyAmountBetween(1_000, 10_000),
				testAccount.Currency,
			},
			checkResponse: func(response domain.Account, err error) {
				require.EqualError(t, err, domain.ErrOwnerNotFound.Error())
				require.Empty(t, response)
			},
		},
		{
			name: "ErrCurrencyAlreadyExists",
			input: struct {
				owner    string
				balance  string
				currency string
			}{
				testUser.Username,
				randompkg.MoneyAmountBetween(1_000, 10_000),
				testAccount.Currency,
			},
			checkResponse: func(response domain.Account, err error) {
				require.EqualError(t, err, domain.ErrCurrencyAlreadyExists.Error())
				require.Empty(t, response)
			},
		},
	}

	for i := range testCases {

		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {

			response, err := testAccountRepo.CreateAccount(context.Background(), tc.input.owner, tc.input.balance, tc.input.currency)

			tc.checkResponse(response, err)
		})
	}

}

func TestGetAccount(t *testing.T) {

	testUser := createRandomUser(t)
	testAccount := createRandomAccount(t, testUser)

	account2, err := testAccountRepo.GetAccount(
		context.Background(),
		testAccount.ID,
	)
	require.NoError(t, err)
	require.NotEmpty(t, account2)

	require.Equal(t, testAccount.ID, account2.ID)
	require.Equal(t, testAccount.Owner, account2.Owner)
	require.Equal(t, testAccount.Balance, account2.Balance)
	require.Equal(t, testAccount.Currency, account2.Currency)
	require.WithinDuration(t, testAccount.CreatedAt, account2.CreatedAt, time.Second)
}

func TestDeleteAccount(t *testing.T) {
	testUser := createRandomUser(t)
	testAccount := createRandomAccount(t, testUser)

	err := testAccountRepo.DeleteAccount(context.Background(), testAccount.ID)
	require.NoError(t, err)

	accountDeleted, err := testAccountRepo.GetAccount(context.Background(), testAccount.ID)
	require.Error(t, err)
	require.EqualError(t, err, domain.ErrAccountNotFound.Error())
	require.Empty(t, accountDeleted)
}

func TestListAccounts(t *testing.T) {
	var lastAccount domain.Account
	for i := 0; i < 10; i++ {
		testUser := createRandomUser(t)
		lastAccount = createRandomAccount(t, testUser)
	}

	accounts, err := testAccountRepo.ListAccounts(context.Background(), lastAccount.Owner, 5, 0)
	require.NoError(t, err)
	require.NotEmpty(t, accounts)

	for _, account := range accounts {
		require.NotEmpty(t, account)
		require.Equal(t, lastAccount.Owner, account.Owner)
	}
}

func TestAddAccountBalance(t *testing.T) {

	testUser := createRandomUser(t)
	testAccount := createRandomAccount(t, testUser)
	testAmount := randompkg.MoneyAmountBetween(100, 1_000)

	account1Balance, err := decimal.NewFromString(testAccount.Balance)
	require.NoError(t, err)
	deltaBalance, err := decimal.NewFromString(testAmount)
	require.NoError(t, err)

	account2, err := testAccountRepo.AddAccountBalance(context.Background(), testAmount, testAccount.ID)
	require.NoError(t, err)
	require.NotEmpty(t, account2)

	account2Balance, err := decimal.NewFromString(account2.Balance)
	require.NoError(t, err)

	require.Equal(t, testAccount.ID, account2.ID)
	require.Equal(t, testAccount.Owner, account2.Owner)
	require.Equal(t, account1Balance.Add(deltaBalance), account2Balance)
	require.Equal(t, testAccount.Currency, account2.Currency)
	require.Equal(t, testAccount.CreatedAt, account2.CreatedAt, time.Second)
}
