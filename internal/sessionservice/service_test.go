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
	"github.com/stretchr/testify/require"
)

var (
	config     configpkg.Config
	tokenMaker tokenpkg.Maker
)

func TestMain(m *testing.M) {
	config = configpkg.Config{
		TokenSymmetricKey:    randompkg.String(32),
		AccessTokenDuration:  time.Minute,
		RefreshTokenDuration: time.Minute,
	}

	os.Exit(m.Run())
}

func TestCreate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var err error
	tokenMaker, err = tokenpkg.NewPasetoMaker(config.TokenSymmetricKey)
	require.NoError(t, err)

	sessionRepoMock := NewMockRepo(ctrl)
	testSessionService, err := New(sessionRepoMock, config, tokenMaker)
	require.NoError(t, err)
	require.NotEmpty(t, testSessionService)

	username := randompkg.Owner()
	testSession := domain.Session{
		Username: username,
	}

	testCases := []struct {
		name          string
		arg           domain.CreateSessionParams
		buildStubs    func(repo *MockRepo)
		checkResponse func(accessToken string, accessTokenExpiresAt time.Time, sess domain.Session, err error)
	}{
		{
			name: "repo.CreateSession error",
			arg: domain.CreateSessionParams{
				Username: username,
			},
			buildStubs: func(repo *MockRepo) {
				repo.EXPECT().
					Create(gomock.Any(), gomock.AssignableToTypeOf(domain.CreateSessionParams{})).
					Times(1).
					Return(domain.Session{}, errorspkg.ErrInternal)
			},
			checkResponse: func(accessToken string, accessTokenExpiresAt time.Time, sess domain.Session, err error) {
				require.Empty(t, accessToken)
				require.Empty(t, accessTokenExpiresAt)
				require.Empty(t, sess)
				require.EqualError(t, err, errorspkg.ErrInternal.Error())
			},
		},
		{
			name: "OK",
			arg: domain.CreateSessionParams{
				Username: username,
			},
			buildStubs: func(repo *MockRepo) {
				repo.EXPECT().
					Create(gomock.Any(), gomock.AssignableToTypeOf(domain.CreateSessionParams{})).
					Times(1).
					Return(testSession, nil)
			},
			checkResponse: func(accessToken string, accessTokenExpiresAt time.Time, sess domain.Session, err error) {
				require.NotEmpty(t, accessToken)
				require.NotEmpty(t, accessTokenExpiresAt)
				require.Equal(t, testSession.ID, sess.ID)
				require.Equal(t, testSession.Username, sess.Username)
				require.NoError(t, err)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			tc.buildStubs(sessionRepoMock)

			accessToken, accessTokenExpiresAt, sess, err := testSessionService.Create(
				context.Background(),
				tc.arg,
			)

			tc.checkResponse(accessToken, accessTokenExpiresAt, sess, err)
		})
	}
}

func TestRenewAccessToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tokenMaker, err := tokenpkg.NewPasetoMaker(config.TokenSymmetricKey)
	require.NoError(t, err)

	sessionRepoMock := NewMockRepo(ctrl)
	testSessionService, err := New(sessionRepoMock, config, tokenMaker)
	require.NoError(t, err)
	require.NotEmpty(t, testSessionService)

	username := randompkg.Owner()
	refreshToken, tokenPayload, err := tokenMaker.CreateToken(username, config.RefreshTokenDuration)
	require.NoError(t, err)

	unauthUsername := randompkg.Owner()
	unauthRefreshToken, unauthRefreshTokenPayload, err := tokenMaker.CreateToken(unauthUsername, config.RefreshTokenDuration)
	require.NoError(t, err)

	testCases := []struct {
		name          string
		refreshToken  string
		buildStubs    func(repo *MockRepo)
		checkResponse func(accessToken string, accessTokenExpiresAt time.Time, err error)
	}{
		{
			name:         "Ivalid refresh token",
			refreshToken: "invalid",
			buildStubs: func(repo *MockRepo) {
				repo.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(accessToken string, accessTokenExpiresAt time.Time, err error) {
				require.Empty(t, accessToken)
				require.Empty(t, accessTokenExpiresAt)
				require.EqualError(t, err, errorspkg.ErrInternal.Error())
			},
		},
		{
			name:         "repo.GetSession error",
			refreshToken: refreshToken,
			buildStubs: func(repo *MockRepo) {
				repo.EXPECT().
					Get(gomock.Any(), gomock.Eq(tokenPayload.ID)).
					Times(1).
					Return(domain.Session{}, domain.ErrSessionNotFound)
			},
			checkResponse: func(accessToken string, accessTokenExpiresAt time.Time, err error) {
				require.Empty(t, accessToken)
				require.Empty(t, accessTokenExpiresAt)
				require.EqualError(t, err, domain.ErrSessionNotFound.Error())
			},
		},

		{
			name:         "repo.GetSession blocked session",
			refreshToken: refreshToken,
			buildStubs: func(repo *MockRepo) {
				repo.EXPECT().
					Get(gomock.Any(), gomock.Eq(tokenPayload.ID)).
					Times(1).
					Return(domain.Session{IsBlocked: true}, nil)
			},
			checkResponse: func(accessToken string, accessTokenExpiresAt time.Time, err error) {
				require.Empty(t, accessToken)
				require.Empty(t, accessTokenExpiresAt)
				require.EqualError(t, err, domain.ErrBlockedSession.Error())
			},
		},
		{
			name:         "repo.GetSession unauthRefreshToken",
			refreshToken: unauthRefreshToken,
			buildStubs: func(repo *MockRepo) {
				repo.EXPECT().
					Get(gomock.Any(), gomock.Eq(unauthRefreshTokenPayload.ID)).
					Times(1).
					Return(domain.Session{
						Username: username,
					}, nil)
			},
			checkResponse: func(accessToken string, accessTokenExpiresAt time.Time, err error) {
				require.Empty(t, accessToken)
				require.Empty(t, accessTokenExpiresAt)
				require.EqualError(t, err, domain.ErrInvalidUser.Error())
			},
		},
		{
			name:         "repo.GetSession blocked session",
			refreshToken: refreshToken,
			buildStubs: func(repo *MockRepo) {
				repo.EXPECT().
					Get(gomock.Any(), gomock.Eq(tokenPayload.ID)).
					Times(1).
					Return(domain.Session{
						Username:     username,
						RefreshToken: unauthRefreshToken,
					}, nil)
			},
			checkResponse: func(accessToken string, accessTokenExpiresAt time.Time, err error) {
				require.Empty(t, accessToken)
				require.Empty(t, accessTokenExpiresAt)
				require.EqualError(t, err, domain.ErrMismatchedRefreshToken.Error())
			},
		},
		{
			name:         "expired session",
			refreshToken: refreshToken,
			buildStubs: func(repo *MockRepo) {
				repo.EXPECT().
					Get(gomock.Any(), gomock.Eq(tokenPayload.ID)).
					Times(1).
					Return(domain.Session{
						Username:     username,
						RefreshToken: refreshToken,
						ExpiresAt:    time.Now().Add(-time.Hour),
					}, nil)
			},
			checkResponse: func(accessToken string, accessTokenExpiresAt time.Time, err error) {
				require.Empty(t, accessToken)
				require.Empty(t, accessTokenExpiresAt)
				require.EqualError(t, err, domain.ErrExpiredSession.Error())
			},
		},
		{
			name:         "OK",
			refreshToken: refreshToken,
			buildStubs: func(repo *MockRepo) {
				repo.EXPECT().
					Get(gomock.Any(), gomock.Eq(tokenPayload.ID)).
					Times(1).
					Return(domain.Session{
						Username:     username,
						RefreshToken: refreshToken,
						ExpiresAt:    tokenPayload.ExpiredAt,
					}, nil)
			},
			checkResponse: func(accessToken string, accessTokenExpiresAt time.Time, err error) {
				require.NotEmpty(t, accessToken)
				require.NotEmpty(t, accessTokenExpiresAt)
				require.NoError(t, err)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			tc.buildStubs(sessionRepoMock)

			accessToken, accessTokenExpiresAt, err := testSessionService.RenewAccessToken(
				context.Background(),
				tc.refreshToken,
			)

			tc.checkResponse(accessToken, accessTokenExpiresAt, err)
		})
	}
}
