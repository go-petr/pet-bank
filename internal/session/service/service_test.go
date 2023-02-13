package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-petr/pet-bank/internal/session"
	"github.com/go-petr/pet-bank/pkg/configpkg"
	"github.com/go-petr/pet-bank/pkg/apperrors"
	"github.com/go-petr/pet-bank/pkg/apprandom"
	"github.com/go-petr/pet-bank/pkg/token"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

var (
	testConfig     configpkg.Config
	testTokenMaker token.Maker
)

func TestMain(m *testing.M) {
	testConfig = configpkg.Config{
		TokenSymmetricKey:    apprandom.String(32),
		AccessTokenDuration:  time.Minute,
		RefreshTokenDuration: time.Minute,
	}
	os.Exit(m.Run())
}

func TestCreate(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var err error
	testTokenMaker, err = token.NewPasetoMaker(testConfig.TokenSymmetricKey)
	require.NoError(t, err)

	sessionRepoMock := NewMockSessionRepoInterface(ctrl)
	testSessionService, err := NewSessionService(sessionRepoMock, testConfig, testTokenMaker)
	require.NoError(t, err)
	require.NotEmpty(t, testSessionService)

	testUsername := apprandom.Owner()
	testSession := session.Session{
		Username: testUsername,
	}

	testCases := []struct {
		name          string
		arg           session.CreateSessionParams
		buildStubs    func(repo *MockSessionRepoInterface)
		checkResponse func(accessToken string, accessTokenExpiresAt time.Time, sess session.Session, err error)
	}{
		{
			name: "repo.CreateSession error",
			arg: session.CreateSessionParams{
				Username: testUsername,
			},
			buildStubs: func(repo *MockSessionRepoInterface) {

				repo.EXPECT().
					CreateSession(gomock.Any(), gomock.AssignableToTypeOf(session.CreateSessionParams{})).
					Times(1).
					Return(session.Session{}, apperrors.ErrInternal)
			},
			checkResponse: func(accessToken string, accessTokenExpiresAt time.Time, sess session.Session, err error) {

				require.Empty(t, accessToken)
				require.Empty(t, accessTokenExpiresAt)
				require.Empty(t, sess)
				require.EqualError(t, err, apperrors.ErrInternal.Error())
			},
		},
		{
			name: "OK",
			arg: session.CreateSessionParams{
				Username: testUsername,
			},
			buildStubs: func(repo *MockSessionRepoInterface) {

				repo.EXPECT().
					CreateSession(gomock.Any(), gomock.AssignableToTypeOf(session.CreateSessionParams{})).
					Times(1).
					Return(testSession, nil)
			},
			checkResponse: func(accessToken string, accessTokenExpiresAt time.Time, sess session.Session, err error) {

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

	testTokenMaker, err := token.NewPasetoMaker(testConfig.TokenSymmetricKey)
	require.NoError(t, err)

	sessionRepoMock := NewMockSessionRepoInterface(ctrl)
	testSessionService, err := NewSessionService(sessionRepoMock, testConfig, testTokenMaker)
	require.NoError(t, err)
	require.NotEmpty(t, testSessionService)

	testUsername := apprandom.Owner()
	testRefreshToken, testTokenPayload, err := testTokenMaker.CreateToken(testUsername, testConfig.RefreshTokenDuration)
	require.NoError(t, err)

	testUnauthorizedUsername := apprandom.Owner()
	testUnauthorizedRefreshToken, testUnauthorizedRefreshTokenPayload, err := testTokenMaker.CreateToken(testUnauthorizedUsername, testConfig.RefreshTokenDuration)
	require.NoError(t, err)

	testCases := []struct {
		name          string
		refreshToken  string
		buildStubs    func(repo *MockSessionRepoInterface)
		checkResponse func(accessToken string, accessTokenExpiresAt time.Time, err error)
	}{
		{
			name:         "Ivalid refresh token",
			refreshToken: "invalid",
			buildStubs: func(repo *MockSessionRepoInterface) {

				repo.EXPECT().
					GetSession(gomock.Any(), gomock.Any()).
					Times(0)

			},
			checkResponse: func(accessToken string, accessTokenExpiresAt time.Time, err error) {

				require.Empty(t, accessToken)
				require.Empty(t, accessTokenExpiresAt)
				require.EqualError(t, err, apperrors.ErrInternal.Error())
			},
		},
		{
			name:         "repo.GetSession error",
			refreshToken: testRefreshToken,
			buildStubs: func(repo *MockSessionRepoInterface) {

				repo.EXPECT().
					GetSession(gomock.Any(), gomock.Eq(testTokenPayload.ID)).
					Times(1).
					Return(session.Session{}, session.ErrSessionNotFound)
			},
			checkResponse: func(accessToken string, accessTokenExpiresAt time.Time, err error) {

				require.Empty(t, accessToken)
				require.Empty(t, accessTokenExpiresAt)
				require.EqualError(t, err, session.ErrSessionNotFound.Error())
			},
		},

		{
			name:         "repo.GetSession blocked session",
			refreshToken: testRefreshToken,
			buildStubs: func(repo *MockSessionRepoInterface) {

				repo.EXPECT().
					GetSession(gomock.Any(), gomock.Eq(testTokenPayload.ID)).
					Times(1).
					Return(session.Session{IsBlocked: true}, nil)
			},
			checkResponse: func(accessToken string, accessTokenExpiresAt time.Time, err error) {

				require.Empty(t, accessToken)
				require.Empty(t, accessTokenExpiresAt)
				require.EqualError(t, err, session.ErrBlockedSession.Error())
			},
		},
		{
			name:         "repo.GetSession testUnauthorizedRefreshToken",
			refreshToken: testUnauthorizedRefreshToken,
			buildStubs: func(repo *MockSessionRepoInterface) {

				repo.EXPECT().
					GetSession(gomock.Any(), gomock.Eq(testUnauthorizedRefreshTokenPayload.ID)).
					Times(1).
					Return(session.Session{
						Username: testUsername,
					}, nil)
			},
			checkResponse: func(accessToken string, accessTokenExpiresAt time.Time, err error) {

				require.Empty(t, accessToken)
				require.Empty(t, accessTokenExpiresAt)
				require.EqualError(t, err, session.ErrInvalidUser.Error())
			},
		},
		{
			name:         "repo.GetSession blocked session",
			refreshToken: testRefreshToken,
			buildStubs: func(repo *MockSessionRepoInterface) {

				repo.EXPECT().
					GetSession(gomock.Any(), gomock.Eq(testTokenPayload.ID)).
					Times(1).
					Return(session.Session{
						Username:     testUsername,
						RefreshToken: testUnauthorizedRefreshToken,
					}, nil)
			},
			checkResponse: func(accessToken string, accessTokenExpiresAt time.Time, err error) {

				require.Empty(t, accessToken)
				require.Empty(t, accessTokenExpiresAt)
				require.EqualError(t, err, session.ErrMismatchedRefreshToken.Error())
			},
		},
		{
			name:         "expired session",
			refreshToken: testRefreshToken,
			buildStubs: func(repo *MockSessionRepoInterface) {

				repo.EXPECT().
					GetSession(gomock.Any(), gomock.Eq(testTokenPayload.ID)).
					Times(1).
					Return(session.Session{
						Username:     testUsername,
						RefreshToken: testRefreshToken,
						ExpiresAt:    time.Now().Add(-time.Hour),
					}, nil)
			},
			checkResponse: func(accessToken string, accessTokenExpiresAt time.Time, err error) {

				require.Empty(t, accessToken)
				require.Empty(t, accessTokenExpiresAt)
				require.EqualError(t, err, session.ErrExpiredSession.Error())
			},
		},
		{
			name:         "OK",
			refreshToken: testRefreshToken,
			buildStubs: func(repo *MockSessionRepoInterface) {

				repo.EXPECT().
					GetSession(gomock.Any(), gomock.Eq(testTokenPayload.ID)).
					Times(1).
					Return(session.Session{
						Username:     testUsername,
						RefreshToken: testRefreshToken,
						ExpiresAt:    testTokenPayload.ExpiredAt,
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
