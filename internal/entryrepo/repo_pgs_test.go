//go:build integration

package entryrepo

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/go-petr/pet-bank/internal/accountrepo"
	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/internal/userrepo"
	"github.com/go-petr/pet-bank/pkg/configpkg"
	"github.com/go-petr/pet-bank/pkg/passpkg"
	"github.com/go-petr/pet-bank/pkg/randompkg"
	"github.com/stretchr/testify/require"
)

var (
	testEntryRepo   *RepoPGS
	testAccountRepo *accountrepo.RepoPGS
	testUserRepo    *userrepo.RepoPGS
)

func TestMain(m *testing.M) {
	config, err := configpkg.Load("../../configs")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	testDB, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	testEntryRepo = NewRepoPGS(testDB)
	testUserRepo = userrepo.NewRepoPGS(testDB)
	testAccountRepo = accountrepo.NewRepoPGS(testDB)

	os.Exit(m.Run())
}

func createRandomEntry(t *testing.T, account domain.Account) domain.Entry {
	testAmount := randompkg.MoneyAmountBetween(100, 1_000)

	entry, err := testEntryRepo.Create(context.Background(), testAmount, account.ID)
	require.NoError(t, err)
	require.NotEmpty(t, entry)

	require.Equal(t, account.ID, entry.AccountID)
	require.Equal(t, testAmount, entry.Amount)

	require.NotZero(t, entry.ID)
	require.NotZero(t, entry.CreatedAt)

	return entry
}

func createRandomUser(t *testing.T) domain.User {
	hashedPassword, err := passpkg.Hash(randompkg.String(10))
	require.NoError(t, err)

	arg := domain.CreateUserParams{
		Username:       randompkg.Owner(),
		HashedPassword: hashedPassword,
		FullName:       randompkg.Owner(),
		Email:          randompkg.Email(),
	}

	testUser, err := testUserRepo.Create(context.Background(), arg)
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

	account, err := testAccountRepo.Create(context.Background(), testUser.Username, testBalance, testCurrency)
	require.NoError(t, err)
	require.NotEmpty(t, account)

	require.Equal(t, testUser.Username, account.Owner)
	require.Equal(t, testBalance, account.Balance)
	require.Equal(t, testCurrency, account.Currency)

	require.NotZero(t, account.ID)
	require.NotZero(t, account.CreatedAt)

	return account
}

func TestCreate(t *testing.T) {
	testUser1 := createRandomUser(t)
	testAccount1 := createRandomAccount(t, testUser1)
	createRandomEntry(t, testAccount1)
}

func TestGet(t *testing.T) {
	testUser1 := createRandomUser(t)
	testAccount1 := createRandomAccount(t, testUser1)
	entry1 := createRandomEntry(t, testAccount1)

	entry2, err := testEntryRepo.Get(context.Background(), entry1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, entry2)

	require.Equal(t, entry1.ID, entry2.ID)
	require.Equal(t, entry1.AccountID, entry2.AccountID)
	require.Equal(t, entry1.Amount, entry2.Amount)
	require.Equal(t, entry1.CreatedAt, entry2.CreatedAt)
}

func TestList(t *testing.T) {
	testUser1 := createRandomUser(t)
	testAccount1 := createRandomAccount(t, testUser1)

	for i := 0; i < 10; i++ {
		createRandomEntry(t, testAccount1)
	}

	entries, err := testEntryRepo.List(context.Background(), testAccount1.ID, 5, 5)
	require.NoError(t, err)
	require.Len(t, entries, 5)

	for _, e := range entries {
		require.NotEmpty(t, e)
		require.Equal(t, testAccount1.ID, e.AccountID)
	}
}
