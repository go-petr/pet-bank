package sessionrepo

import (
	"context"
	"database/sql"
	"log"
	"os"

	"testing"
	"time"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/internal/userrepo"
	"github.com/go-petr/pet-bank/pkg/configpkg"
	"github.com/go-petr/pet-bank/pkg/passpkg"
	"github.com/go-petr/pet-bank/pkg/randompkg"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

var (
	testSessionRepo *RepoPGS
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

	testSessionRepo = NewRepoPGS(testDB)
	testUserRepo = userrepo.NewRepoPGS(testDB)

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

func createRandomSession(t *testing.T, username string) domain.Session {
	arg := domain.CreateSessionParams{
		ID:           uuid.New(),
		Username:     username,
		RefreshToken: randompkg.String(10),
		UserAgent:    randompkg.String(10),
		ClientIP:     randompkg.String(10),
		ExpiresAt:    time.Now().Truncate(time.Second).UTC(),
	}

	testSession, err := testSessionRepo.Create(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, testSession)

	require.Equal(t, arg.ID, testSession.ID)
	require.Equal(t, arg.Username, testSession.Username)
	require.Equal(t, arg.RefreshToken, testSession.RefreshToken)
	require.Equal(t, arg.UserAgent, testSession.UserAgent)
	require.Equal(t, arg.ClientIP, testSession.ClientIP)
	require.Equal(t, arg.IsBlocked, testSession.IsBlocked)
	require.Equal(t, arg.ExpiresAt, testSession.ExpiresAt.UTC())

	require.NotZero(t, testSession.CreatedAt)

	return testSession
}

func TestCreateAccount(t *testing.T) {
	testUser := createRandomUser(t)
	createRandomSession(t, testUser.Username)
}

func TestCreateAccountConstraintViolation(t *testing.T) {
	arg := domain.CreateSessionParams{
		ID:           uuid.New(),
		Username:     "invalid",
		RefreshToken: randompkg.String(10),
		UserAgent:    randompkg.String(10),
		ClientIP:     randompkg.String(10),
		ExpiresAt:    time.Now().Truncate(time.Second).UTC(),
	}

	testSession, err := testSessionRepo.Create(context.Background(), arg)
	require.EqualError(t, err, domain.ErrUserNotFound.Error())
	require.Empty(t, testSession)
}

func TestGetSession(t *testing.T) {
	testUser := createRandomUser(t)
	testSession := createRandomSession(t, testUser.Username)

	gotSession, err := testSessionRepo.Get(context.Background(), testSession.ID)
	require.NoError(t, err)
	require.NotEmpty(t, gotSession)

	require.Equal(t, testSession.ID, gotSession.ID)
	require.Equal(t, testSession.Username, gotSession.Username)
	require.Equal(t, testSession.RefreshToken, gotSession.RefreshToken)
	require.Equal(t, testSession.UserAgent, gotSession.UserAgent)
	require.Equal(t, testSession.ClientIP, gotSession.ClientIP)
	require.Equal(t, testSession.IsBlocked, gotSession.IsBlocked)
	require.Equal(t, testSession.ExpiresAt, gotSession.ExpiresAt)
	require.NotEmpty(t, gotSession.ExpiresAt.UTC())
}
