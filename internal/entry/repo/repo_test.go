package repo

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/go-petr/pet-bank/internal/account"
	ar "github.com/go-petr/pet-bank/internal/account/repo"
	"github.com/go-petr/pet-bank/internal/entry"
	"github.com/go-petr/pet-bank/internal/user"
	ur "github.com/go-petr/pet-bank/internal/user/repo"
	"github.com/go-petr/pet-bank/pkg/util"
	"github.com/stretchr/testify/require"
)

var (
	testEntryRepo   *EntryRepo
	testAccountRepo *ar.AccountRepo
	testUserRepo    *ur.UserRepo
)

func TestMain(m *testing.M) {
	config, err := util.LoadConfig("../../..")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	testDB, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	testEntryRepo = NewEntryRepo(testDB)
	testUserRepo = ur.NewUserRepo(testDB)
	testAccountRepo = ar.NewAccountRepo(testDB)

	os.Exit(m.Run())
}

func createRandomEntry(t *testing.T, account account.Account) entry.Entry {
	arg := entry.CreateEntryParams{
		AccountID: account.ID,
		Amount:    util.RandomMoneyAmountBetween(100, 1_000),
	}

	entry, err := testEntryRepo.CreateEntry(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, entry)

	require.Equal(t, arg.AccountID, entry.AccountID)
	require.Equal(t, arg.Amount, entry.Amount)

	require.NotZero(t, entry.ID)
	require.NotZero(t, entry.CreatedAt)

	return entry
}

func createRandomUser(t *testing.T) user.User {

	hashedPassword, err := util.HashPassword(util.RandomString(10))
	require.NoError(t, err)

	arg := user.CreateUserParams{
		Username:       util.RandomOwner(),
		HashedPassword: hashedPassword,
		FullName:       util.RandomOwner(),
		Email:          util.RandomEmail(),
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
		Balance:  util.RandomMoneyAmountBetween(1_000, 10_000),
		Currency: util.RandomCurrency(),
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

func TestCreateEntry(t *testing.T) {

	testUser1 := createRandomUser(t)
	testAccount1 := createRandomAccount(t, testUser1)
	createRandomEntry(t, testAccount1)
}

func TestGetEntry(t *testing.T) {

	testUser1 := createRandomUser(t)
	testAccount1 := createRandomAccount(t, testUser1)
	entry1 := createRandomEntry(t, testAccount1)

	entry2, err := testEntryRepo.GetEntry(context.Background(), entry1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, entry2)

	require.Equal(t, entry1.ID, entry2.ID)
	require.Equal(t, entry1.AccountID, entry2.AccountID)
	require.Equal(t, entry1.Amount, entry2.Amount)
	require.Equal(t, entry1.CreatedAt, entry2.CreatedAt)

}

func TestListEntries(t *testing.T) {

	testUser1 := createRandomUser(t)
	testAccount1 := createRandomAccount(t, testUser1)

	for i := 0; i < 10; i++ {
		createRandomEntry(t, testAccount1)
	}

	arg := entry.ListEntriesParams{
		AccountID: testAccount1.ID,
		Limit:     5,
		Offset:    5,
	}

	entries, err := testEntryRepo.ListEntries(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, entries, 5)

	for _, e := range entries {
		require.NotEmpty(t, e)
		require.Equal(t, arg.AccountID, e.AccountID)
	}
}
