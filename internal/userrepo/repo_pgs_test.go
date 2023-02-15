package userrepo

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"
	"time"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/pkg/configpkg"
	"github.com/go-petr/pet-bank/pkg/passpkg"
	"github.com/go-petr/pet-bank/pkg/randompkg"
	"github.com/stretchr/testify/require"

	_ "github.com/lib/pq"
)

var (
	testUserRepo *RepoPGS
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

	testUserRepo = NewRepoPGS(testDB)

	os.Exit(m.Run())
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

	user, err := testUserRepo.Create(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.Equal(t, arg.Username, user.Username)
	require.Equal(t, arg.HashedPassword, user.HashedPassword)
	require.Equal(t, arg.FullName, user.FullName)
	require.Equal(t, arg.Email, user.Email)

	require.NotZero(t, user.CreatedAt)

	return user
}

func TestCreate(t *testing.T) {
	createRandomUser(t)
}

func TestCreateUserUniqueViolation(t *testing.T) {
	user1 := createRandomUser(t)

	hashedPassword, err := passpkg.Hash(randompkg.String(10))
	require.NoError(t, err)

	arg := domain.CreateUserParams{
		Username:       user1.Username, // Username duplicate
		HashedPassword: hashedPassword,
		FullName:       randompkg.Owner(),
		Email:          randompkg.Email(),
	}

	user2, err := testUserRepo.Create(context.Background(), arg)
	require.EqualError(t, err, domain.ErrUsernameAlreadyExists.Error())
	require.Empty(t, user2)

	arg = domain.CreateUserParams{
		Username:       randompkg.Owner(),
		HashedPassword: hashedPassword,
		FullName:       randompkg.Owner(),
		Email:          user1.Email, // Email duplicate
	}

	user2, err = testUserRepo.Create(context.Background(), arg)
	require.EqualError(t, err, domain.ErrEmailALreadyExists.Error())
	require.Empty(t, user2)
}

func TestGetUser(t *testing.T) {
	user1 := createRandomUser(t)

	user2, err := testUserRepo.Get(context.Background(), user1.Username)
	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, user1.Username, user2.Username)
	require.Equal(t, user1.HashedPassword, user2.HashedPassword)
	require.Equal(t, user1.FullName, user2.FullName)
	require.Equal(t, user1.Email, user2.Email)
	require.WithinDuration(t, user1.PasswordChangedAt, user2.PasswordChangedAt, time.Second)
	require.WithinDuration(t, user1.CreatedAt, user2.CreatedAt, time.Second)

	// Not found
	_, err = testUserRepo.Get(context.Background(), "non-existent")
	require.EqualError(t, err, domain.ErrUserNotFound.Error())
}
