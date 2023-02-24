//go:build integration

package sessionrepo_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	"testing"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/internal/sessionrepo"
	"github.com/go-petr/pet-bank/internal/test"
	"github.com/go-petr/pet-bank/pkg/configpkg"
	"github.com/go-petr/pet-bank/pkg/dbpkg/integrationtest"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/go-petr/pet-bank/pkg/randompkg"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

var (
	dbDriver string
	dbSource string
)

func TestMain(m *testing.M) {
	config, err := configpkg.Load("../../configs")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	dbDriver = config.DBDriver
	dbSource = config.DBSource

	os.Exit(m.Run())
}

func SeedSession(t *testing.T, tx *sql.Tx, username string) domain.Session {

	arg := domain.CreateSessionParams{
		ID:           uuid.New(),
		Username:     username,
		RefreshToken: randompkg.String(10),
		UserAgent:    randompkg.String(10),
		ClientIP:     randompkg.String(10),
		ExpiresAt:    time.Now().Truncate(time.Second).UTC(),
	}

	sessionRepo := sessionrepo.NewRepoPGS(tx)

	session, err := sessionRepo.Create(context.Background(), arg)
	if err != nil {
		t.Fatalf("sessionRepo.Create(context.Background(), %+v) returned error: %v", arg, err)
	}

	return session
}

func TestCreate(t *testing.T) {

	testCases := []struct {
		name        string
		wantSession func(tx *sql.Tx) domain.Session
		wantErr     error
	}{
		{
			name: "OK",
			wantSession: func(tx *sql.Tx) domain.Session {
				user := test.SeedUser(t, tx)
				return domain.Session{
					ID:           uuid.New(),
					Username:     user.Username,
					RefreshToken: randompkg.String(10),
					UserAgent:    randompkg.String(10),
					ClientIP:     randompkg.String(10),
					ExpiresAt:    time.Now().Truncate(time.Second).UTC(),
					CreatedAt:    time.Now().Truncate(time.Second).UTC(),
				}
			},
		},
		{
			name: "ErrUserNotFound",
			wantSession: func(tx *sql.Tx) domain.Session {
				return domain.Session{
					ID:           uuid.New(),
					Username:     randompkg.Owner(),
					RefreshToken: randompkg.String(10),
					UserAgent:    randompkg.String(10),
					ClientIP:     randompkg.String(10),
					ExpiresAt:    time.Now().Truncate(time.Second).UTC(),
					CreatedAt:    time.Now().Truncate(time.Second).UTC(),
				}
			},
			wantErr: domain.ErrUserNotFound,
		},
		{
			name: "PubKeyDublicate",
			wantSession: func(tx *sql.Tx) domain.Session {
				user := test.SeedUser(t, tx)
				s := SeedSession(t, tx, user.Username)
				return domain.Session{
					ID:           s.ID,
					Username:     randompkg.Owner(),
					RefreshToken: randompkg.String(10),
					UserAgent:    randompkg.String(10),
					ClientIP:     randompkg.String(10),
					ExpiresAt:    time.Now().Truncate(time.Second).UTC(),
					CreatedAt:    time.Now().Truncate(time.Second).UTC(),
				}
			},
			wantErr: errorspkg.ErrInternal,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tx := integrationtest.SetupTX(t, dbDriver, dbSource)

			want := tc.wantSession(tx)

			sessionRepo := sessionrepo.NewRepoPGS(tx)

			arg := domain.CreateSessionParams{
				ID:           want.ID,
				Username:     want.Username,
				RefreshToken: want.RefreshToken,
				UserAgent:    want.UserAgent,
				ClientIP:     want.ClientIP,
				ExpiresAt:    want.ExpiresAt,
			}

			got, err := sessionRepo.Create(context.Background(), arg)
			if err != nil {
				if err == tc.wantErr {
					return
				}
				t.Fatalf("sessionRepo.Create(context.Background(), %+v) returned error: %v", arg, err)
			}

			if diff := cmp.Diff(want, got, cmpopts.EquateApproxTime(time.Second)); diff != "" {
				t.Errorf(`sessionRepo.Create(context.Background(), %+v) returned unexpected difference (-want +got):\n%s"`,
					arg, diff)
			}
		})
	}
}

func TestGetSession(t *testing.T) {

	testCases := []struct {
		name        string
		wantSession func(tx *sql.Tx) domain.Session
		wantErr     error
	}{
		{
			name: "OK",
			wantSession: func(tx *sql.Tx) domain.Session {
				user := test.SeedUser(t, tx)
				s := SeedSession(t, tx, user.Username)
				return s
			},
		},
		{
			name: "ErrSessionNotFound",
			wantSession: func(tx *sql.Tx) domain.Session {
				return domain.Session{ID: uuid.New()}
			},
			wantErr: domain.ErrSessionNotFound,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tx := integrationtest.SetupTX(t, dbDriver, dbSource)
			want := tc.wantSession(tx)
			sessionRepo := sessionrepo.NewRepoPGS(tx)

			got, err := sessionRepo.Get(context.Background(), want.ID)
			if err != nil {
				if err == tc.wantErr {
					return
				}
				t.Fatalf("sessionRepo.Create(context.Background(), %+v) returned error: %v", want.ID, err)
			}

			if diff := cmp.Diff(want, got, cmpopts.EquateApproxTime(time.Second)); diff != "" {
				t.Errorf(`sessionRepo.Get(context.Background(), %v) returned unexpected difference (-want +got):\n%s"`,
					want.ID, diff)
			}
		})
	}
}
