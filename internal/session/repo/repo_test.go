package repo

import (
	"context"
	"database/sql"
	"log"
	"os"

	"testing"
	"time"

	"github.com/go-petr/pet-bank/internal/session"
	"github.com/go-petr/pet-bank/internal/user"
	"github.com/go-petr/pet-bank/internal/user/repo"
	"github.com/go-petr/pet-bank/pkg/configpkg"
	"github.com/go-petr/pet-bank/pkg/apppass"
	"github.com/go-petr/pet-bank/pkg/apprandom"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

var (
	testSessionRepo *SessionRepo
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

	testSessionRepo = NewSessionRepo(testDB)
	testUserRepo = repo.NewUserRepo(testDB)

	os.Exit(m.Run())
}

func createRandomUser(t *testing.T) user.User {

	hashedPassword, err := apppass.Hash(apprandom.String(10))
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

func createRandomSession(t *testing.T, username string) session.Session {

	arg := session.CreateSessionParams{
		ID:           uuid.New(),
		Username:     username,
		RefreshToken: apprandom.String(10),
		UserAgent:    apprandom.String(10),
		ClientIP:     apprandom.String(10),
		ExpiresAt:    time.Now().Truncate(time.Second).UTC(),
	}

	testSession, err := testSessionRepo.CreateSession(context.Background(), arg)
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

	arg := session.CreateSessionParams{
		ID:           uuid.New(),
		Username:     "invalid",
		RefreshToken: apprandom.String(10),
		UserAgent:    apprandom.String(10),
		ClientIP:     apprandom.String(10),
		ExpiresAt:    time.Now().Truncate(time.Second).UTC(),
	}

	testSession, err := testSessionRepo.CreateSession(context.Background(), arg)
	require.EqualError(t, err, user.ErrUserNotFound.Error())
	require.Empty(t, testSession)
}

func TestGetSession(t *testing.T) {

	testUser := createRandomUser(t)
	testSession := createRandomSession(t, testUser.Username)

	gotSession, err := testSessionRepo.GetSession(context.Background(), testSession.ID)
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
