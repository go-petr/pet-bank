//go:build integration

package userrepo_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"
	"time"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/internal/test"
	"github.com/go-petr/pet-bank/internal/userrepo"
	"github.com/go-petr/pet-bank/pkg/configpkg"
	"github.com/go-petr/pet-bank/pkg/dbpkg/integrationtest"
	"github.com/go-petr/pet-bank/pkg/randompkg"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

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

func TestCreate(t *testing.T) {

	testCases := []struct {
		name    string
		arg     func(tx *sql.Tx) domain.CreateUserParams
		wantErr error
	}{
		{
			name: "OK",
			arg: func(tx *sql.Tx) domain.CreateUserParams {
				return domain.CreateUserParams{
					Username:       randompkg.Owner(),
					HashedPassword: randompkg.String(32),
					FullName:       randompkg.String(10),
					Email:          randompkg.Email(),
				}
			},
		},
		{
			name: "ErrUsernameAlreadyExists",
			arg: func(tx *sql.Tx) domain.CreateUserParams {
				arg := domain.CreateUserParams{
					Username:       randompkg.Owner(),
					HashedPassword: randompkg.String(32),
					FullName:       randompkg.String(10),
					Email:          randompkg.Email(),
				}

				userRepo := userrepo.NewRepoPGS(tx)
				_, err := userRepo.Create(context.Background(), arg)
				if err != nil {
					t.Fatalf(`userRepo.Create(context.Background(), %v) returned error: %v`,
						arg, err.Error())
				}

				arg.Email = randompkg.Email()

				return arg
			},
			wantErr: domain.ErrUsernameAlreadyExists,
		},
		{
			name: "ErrEmailALreadyExists",
			arg: func(tx *sql.Tx) domain.CreateUserParams {
				arg := domain.CreateUserParams{
					Username:       randompkg.Owner(),
					HashedPassword: randompkg.String(32),
					FullName:       randompkg.String(10),
					Email:          randompkg.Email(),
				}

				userRepo := userrepo.NewRepoPGS(tx)
				_, err := userRepo.Create(context.Background(), arg)
				if err != nil {
					t.Fatalf(`userRepo.Create(context.Background(), %v) returned error: %v`,
						arg, err.Error())
				}

				arg.Username = randompkg.Owner()

				return arg
			},
			wantErr: domain.ErrEmailALreadyExists,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Prepare test transaction
			tx := integrationtest.SetupTX(t, dbDriver, dbSource)
			userRepo := userrepo.NewRepoPGS(tx)

			// Run test
			arg := tc.arg(tx)
			got, err := userRepo.Create(context.Background(), arg)
			if err != nil {
				if err == tc.wantErr {
					return
				}
				t.Fatalf(`userRepo.Create(context.Background(), %v) returned error: %v`,
					arg, err.Error())
			}

			want := domain.User{
				Username:          arg.Username,
				HashedPassword:    arg.HashedPassword,
				FullName:          arg.FullName,
				Email:             arg.Email,
				PasswordChangedAt: time.Now().UTC().Truncate(time.Second),
				CreatedAt:         time.Now().UTC().Truncate(time.Second),
			}

			ignoreFields := cmpopts.IgnoreFields(domain.User{}, "PasswordChangedAt", "CreatedAt")
			if diff := cmp.Diff(want, got, ignoreFields); diff != "" {
				t.Errorf(`userRepo.Create(context.Background(), %v) returned unexpected difference (-want +got):\n%s"`,
					arg, diff)
			}

			if !cmp.Equal(got.CreatedAt, want.CreatedAt, cmpopts.EquateApproxTime(time.Second)) {
				t.Errorf("got.CreatedAt = %v, want %v +- minute",
					got.CreatedAt.Truncate(time.Second), want.CreatedAt)
			}
		})
	}
}

func TestGet(t *testing.T) {

	testCases := []struct {
		name    string
		want    func(tx *sql.Tx) domain.User
		wantErr error
	}{
		{
			name: "OK",
			want: func(tx *sql.Tx) domain.User {
				return test.SeedUser(t, tx)
			},
		},
		{
			name: "ErrUserNotFound",
			want: func(tx *sql.Tx) domain.User {
				return domain.User{Username: "notfound"}
			},
			wantErr: domain.ErrUserNotFound,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			// t.Parallel()

			// Prepare test transaction and seed database
			tx := integrationtest.SetupTX(t, dbDriver, dbSource)
			want := tc.want(tx)
			userRepo := userrepo.NewRepoPGS(tx)

			// Run test
			got, err := userRepo.Get(context.Background(), want.Username)
			if err != nil {
				if err == tc.wantErr {
					return
				}

				t.Errorf(`userRepo.Get(context.Background(), %v) returned unexpected error: %v`,
					want.Username, err)
				return
			}

			ignoreFields := cmpopts.IgnoreFields(domain.User{}, "CreatedAt")
			if diff := cmp.Diff(want, got, ignoreFields); diff != "" {
				t.Errorf(`userRepo.Get(context.Background(), %v) returned unexpected difference (-want +got):\n%s"`,
					want.Username, diff)
			}

			if !cmp.Equal(got.CreatedAt, want.CreatedAt, cmpopts.EquateApproxTime(time.Second)) {
				t.Errorf("got.CreatedAt = %v, want %v +- minute",
					got.CreatedAt.Truncate(time.Second), want.CreatedAt.Truncate(time.Second))
			}
		})
	}
}
