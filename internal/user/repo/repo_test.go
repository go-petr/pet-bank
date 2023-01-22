package repo

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"
	"time"

	"github.com/go-petr/pet-bank/internal/user"
	"github.com/go-petr/pet-bank/pkg/util"
	"github.com/stretchr/testify/require"

	_ "github.com/lib/pq"
)

var (
	testUserRepo *UserRepo
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

	testUserRepo = NewUserRepo(testDB)

	os.Exit(m.Run())
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

	user, err := testUserRepo.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.Equal(t, arg.Username, user.Username)
	require.Equal(t, arg.HashedPassword, user.HashedPassword)
	require.Equal(t, arg.FullName, user.FullName)
	require.Equal(t, arg.Email, user.Email)

	require.NotZero(t, user.CreatedAt)

	return user
}

func TestCreateUser(t *testing.T) {
	createRandomUser(t)
}

func TestCreateUserUniqueViolation(t *testing.T) {

	user1 := createRandomUser(t)

	hashedPassword, err := util.HashPassword(util.RandomString(10))
	require.NoError(t, err)

	arg := user.CreateUserParams{
		Username:       user1.Username, // Username duplicate
		HashedPassword: hashedPassword,
		FullName:       util.RandomOwner(),
		Email:          util.RandomEmail(),
	}

	user2, err := testUserRepo.CreateUser(context.Background(), arg)
	require.EqualError(t, err, user.ErrUsernameAlreadyExists.Error())
	require.Empty(t, user2)

	arg = user.CreateUserParams{
		Username:       util.RandomOwner(),
		HashedPassword: hashedPassword,
		FullName:       util.RandomOwner(),
		Email:          user1.Email, // Email duplicate
	}

	user2, err = testUserRepo.CreateUser(context.Background(), arg)
	require.EqualError(t, err, user.ErrEmailALreadyExists.Error())
	require.Empty(t, user2)
}

func TestGetUser(t *testing.T) {

	user1 := createRandomUser(t)

	user2, err := testUserRepo.GetUser(context.Background(), user1.Username)
	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, user1.Username, user2.Username)
	require.Equal(t, user1.HashedPassword, user2.HashedPassword)
	require.Equal(t, user1.FullName, user2.FullName)
	require.Equal(t, user1.Email, user2.Email)
	require.WithinDuration(t, user1.PasswordChangedAt, user2.PasswordChangedAt, time.Second)
	require.WithinDuration(t, user1.CreatedAt, user2.CreatedAt, time.Second)

	// Not found
	_, err = testUserRepo.GetUser(context.Background(), "non-existent")
	require.EqualError(t, err, user.ErrUserNotFound.Error())

}
