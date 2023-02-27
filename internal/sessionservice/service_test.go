package sessionservice

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/pkg/configpkg"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/go-petr/pet-bank/pkg/randompkg"
	"github.com/go-petr/pet-bank/pkg/tokenpkg"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
)

var config configpkg.Config

func TestMain(m *testing.M) {
	config = configpkg.Config{
		TokenSymmetricKey:    randompkg.String(32),
		AccessTokenDuration:  time.Minute,
		RefreshTokenDuration: time.Minute,
	}

	os.Exit(m.Run())
}

func TestCreate(t *testing.T) {
	t.Parallel()

	tokenMaker, err := tokenpkg.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		t.Fatalf("tokenpkg.NewPasetoMaker(%v) failed: %v", config.TokenSymmetricKey, err)
	}

	username := randompkg.Owner()
	want := domain.Session{
		Username: username,
	}

	testCases := []struct {
		name          string
		arg           domain.CreateSessionParams
		buildStubs    func(repo *MockRepo)
		checkResponse func(accessToken string, accessTokenExpiresAt time.Time, sess domain.Session)
		wantError     error
	}{
		{
			name: "OK",
			arg: domain.CreateSessionParams{
				Username: username,
			},
			buildStubs: func(repo *MockRepo) {
				repo.EXPECT().
					Create(gomock.Any(), gomock.AssignableToTypeOf(domain.CreateSessionParams{})).
					Times(1).
					Return(want, nil)
			},
			checkResponse: func(accessToken string, accessTokenExpiresAt time.Time, got domain.Session) {
				if accessToken == "" {
					t.Error(`accessToken = "", want non empty`)
				}

				if accessTokenExpiresAt.IsZero() {
					t.Error(`accessTokenExpiresAt is zero, want non zero`)
				}

				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("session returned unexpected diff: %s", diff)
				}
			},
		},
		{
			name: "RepoInternalError",
			arg: domain.CreateSessionParams{
				Username: username,
			},
			buildStubs: func(repo *MockRepo) {
				repo.EXPECT().
					Create(gomock.Any(), gomock.AssignableToTypeOf(domain.CreateSessionParams{})).
					Times(1).
					Return(domain.Session{}, errorspkg.ErrInternal)
			},
			wantError: errorspkg.ErrInternal,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			sessionRepoMock := NewMockRepo(ctrl)
			sessionService, err := New(sessionRepoMock, config, tokenMaker)
			if err != nil {
				t.Fatalf("New(%v, %v, %v) failed: %v", sessionRepoMock, config, tokenMaker, err)
			}

			tc.buildStubs(sessionRepoMock)

			accessToken, accessTokenExpiresAt, sess, err := sessionService.Create(context.Background(), tc.arg)
			if err != nil {
				if err == tc.wantError {
					return
				}

				t.Fatalf("sessionService.Create(context.Background(), %v) returned unexpected error: %v",
					tc.arg, err)
			}

			tc.checkResponse(accessToken, accessTokenExpiresAt, sess)
		})
	}
}

func TestRenewAccessToken(t *testing.T) {
	t.Parallel()

	tokenMaker, err := tokenpkg.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		t.Fatalf("tokenpkg.NewPasetoMaker(%v) failed: %v", config.TokenSymmetricKey, err)
	}

	username := randompkg.Owner()

	token1, payload1, err := tokenMaker.CreateToken(username, config.RefreshTokenDuration)
	if err != nil {
		t.Fatalf("tokenpkg.CreateToken(%v, %v) failed: %v",
			username, config.RefreshTokenDuration, err)
	}

	expired, _, err := tokenMaker.CreateToken(username, time.Nanosecond)
	if err != nil {
		t.Fatalf("tokenpkg.CreateToken(%v, %v) failed: %v",
			username, time.Nanosecond, err)
	}

	unauthUsername := randompkg.Owner()

	token2, payload2, err := tokenMaker.CreateToken(unauthUsername, config.RefreshTokenDuration)
	if err != nil {
		t.Fatalf("tokenpkg.CreateToken(%v, %v) failed: %v",
			username, config.RefreshTokenDuration, err)
	}

	testCases := []struct {
		name          string
		token         string
		buildStubs    func(repo *MockRepo)
		checkResponse func(t *testing.T, accessToken string, accessTokenExpiresAt time.Time)
		wantError     error
	}{
		{
			name:  "OK",
			token: token1,
			buildStubs: func(repo *MockRepo) {
				s := domain.Session{
					Username:     username,
					RefreshToken: token1,
					ExpiresAt:    payload1.ExpiredAt,
				}
				repo.EXPECT().
					Get(gomock.Any(), gomock.Eq(payload1.ID)).
					Times(1).
					Return(s, nil)
			},
			checkResponse: func(t *testing.T, accessToken string, accessTokenExpiresAt time.Time) {
				if accessToken == "" {
					t.Error(`accessToken = "", want non empty`)
				}

				if accessTokenExpiresAt.IsZero() {
					t.Error(`accessTokenExpiresAt is zero, want non zero`)
				}
			},
		},
		{
			name:  "ErrExpiredToken",
			token: expired,
			buildStubs: func(repo *MockRepo) {
				repo.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantError: tokenpkg.ErrExpiredToken,
		},
		{
			name:  "ErrInvalidToken",
			token: "invalid",
			buildStubs: func(repo *MockRepo) {
				repo.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantError: tokenpkg.ErrInvalidToken,
		},
		{
			name:  "ErrSessionNotFound",
			token: token1,
			buildStubs: func(repo *MockRepo) {
				repo.EXPECT().
					Get(gomock.Any(), gomock.Eq(payload1.ID)).
					Times(1).
					Return(domain.Session{}, domain.ErrSessionNotFound)
			},
			wantError: domain.ErrSessionNotFound,
		},
		{
			name:  "ErrBlockedSession",
			token: token1,
			buildStubs: func(repo *MockRepo) {
				repo.EXPECT().
					Get(gomock.Any(), gomock.Eq(payload1.ID)).
					Times(1).
					Return(domain.Session{IsBlocked: true}, nil)
			},
			wantError: domain.ErrBlockedSession,
		},
		{
			name:  "sess.Username!=refreshPayload.Username",
			token: token2,
			buildStubs: func(repo *MockRepo) {
				repo.EXPECT().
					Get(gomock.Any(), gomock.Eq(payload2.ID)).
					Times(1).
					Return(domain.Session{Username: username}, nil)
			},
			wantError: domain.ErrInvalidUser,
		},
		{
			name:  "ErrMismatchedRefreshToken",
			token: token1,
			buildStubs: func(repo *MockRepo) {
				repo.EXPECT().
					Get(gomock.Any(), gomock.Eq(payload1.ID)).
					Times(1).
					Return(domain.Session{Username: username, RefreshToken: token2}, nil)
			},
			wantError: domain.ErrMismatchedRefreshToken,
		},
		{
			name:  "ErrExpiredSession",
			token: token1,
			buildStubs: func(repo *MockRepo) {
				repo.EXPECT().
					Get(gomock.Any(), gomock.Eq(payload1.ID)).
					Times(1).
					Return(domain.Session{
						Username:     username,
						RefreshToken: token1,
						ExpiresAt:    time.Now().Add(-time.Hour),
					}, nil)
			},
			wantError: domain.ErrExpiredSession,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			sessionRepoMock := NewMockRepo(ctrl)
			sessionService, err := New(sessionRepoMock, config, tokenMaker)
			if err != nil {
				t.Fatalf("New(%v, %v, %v) failed: %v", sessionRepoMock, config, tokenMaker, err)
			}

			tc.buildStubs(sessionRepoMock)

			accessToken, expires, err := sessionService.RenewAccessToken(context.Background(), tc.token)
			if err != nil {
				if err == tc.wantError {
					return
				}
				t.Fatalf("sessionService.RenewAccessToken(context.Background(),  %v) failed: %v", tc.token, err)
			}

			tc.checkResponse(t, accessToken, expires)
		})
	}
}
