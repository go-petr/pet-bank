// Package userdelivery manages delivery layer of users.
package userdelivery

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/internal/userservice"
	"github.com/go-petr/pet-bank/pkg/configpkg"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/go-petr/pet-bank/pkg/passpkg"
	"github.com/go-petr/pet-bank/pkg/randompkg"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

var (
	testConfig configpkg.Config
)

func TestMain(m *testing.M) {
	testConfig = configpkg.Config{
		TokenSymmetricKey:   randompkg.String(32),
		AccessTokenDuration: time.Minute,
	}

	gin.SetMode(gin.ReleaseMode)
	os.Exit(m.Run())
}

func randomUser(t *testing.T) (domain.User, string) {
	password := randompkg.String(10)

	hashedPassword, err := passpkg.Hash(password)
	require.NoError(t, err)

	user := domain.User{
		Username:       randompkg.Owner(),
		HashedPassword: hashedPassword,
		FullName:       randompkg.Owner(),
		Email:          randompkg.Email(),
	}

	return user, password
}

func TestCreateAPI(t *testing.T) {
	testUser, password := randomUser(t)

	testCases := []struct {
		name          string
		requestBody   gin.H
		buildStubs    func(userService *MockService, sessionMaker *MockSessionMaker)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "InvalidUsername",
			requestBody: gin.H{
				"username": "user&%",
				"password": password,
				"fullname": testUser.FullName,
				"email":    testUser.Email,
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "ShortPassword",
			requestBody: gin.H{
				"username": testUser.Username,
				"password": "xyz",
				"fullname": testUser.FullName,
				"email":    testUser.Email,
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidEmail",
			requestBody: gin.H{
				"username": testUser.Username,
				"password": password,
				"fullname": testUser.FullName,
				"email":    "user%email.com",
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "UniqueViolationUsername",
			requestBody: gin.H{
				"username": testUser.Username,
				"password": password,
				"fullname": testUser.FullName,
				"email":    testUser.Email,
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					Create(gomock.Any(),
						gomock.Eq(testUser.Username),
						gomock.Eq(password),
						gomock.Eq(testUser.FullName),
						gomock.Eq(testUser.Email)).
					Times(1).
					Return(domain.UserWihtoutPassword{}, domain.ErrUsernameAlreadyExists)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, recorder.Code)
			},
		},
		{
			name: "UniqueViolationEmail",
			requestBody: gin.H{
				"username": testUser.Username,
				"password": password,
				"fullname": testUser.FullName,
				"email":    testUser.Email,
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					Create(gomock.Any(),
						gomock.Eq(testUser.Username),
						gomock.Eq(password),
						gomock.Eq(testUser.FullName),
						gomock.Eq(testUser.Email)).
					Times(1).
					Return(domain.UserWihtoutPassword{}, domain.ErrEmailALreadyExists)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, recorder.Code)
			},
		},
		{
			name: "CreateInternalError",
			requestBody: gin.H{
				"username": testUser.Username,
				"password": password,
				"fullname": testUser.FullName,
				"email":    testUser.Email,
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					Create(gomock.Any(),
						gomock.Eq(testUser.Username),
						gomock.Eq(password),
						gomock.Eq(testUser.FullName),
						gomock.Eq(testUser.Email)).
					Times(1).
					Return(domain.UserWihtoutPassword{}, errorspkg.ErrInternal)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "CreateSessionInternalError",
			requestBody: gin.H{
				"username": testUser.Username,
				"password": password,
				"fullname": testUser.FullName,
				"email":    testUser.Email,
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					Create(gomock.Any(),
						gomock.Eq(testUser.Username),
						gomock.Eq(password),
						gomock.Eq(testUser.FullName),
						gomock.Eq(testUser.Email)).
					Times(1).
					Return(domain.UserWihtoutPassword{}, nil)

				arg := domain.CreateSessionParams{
					Username: testUser.Username,
				}

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return("", time.Now(), domain.Session{}, errorspkg.ErrInternal)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "OK",
			requestBody: gin.H{
				"username": testUser.Username,
				"password": password,
				"fullname": testUser.FullName,
				"Email":    testUser.Email,
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					Create(gomock.Any(),
						gomock.Eq(testUser.Username),
						gomock.Eq(password),
						gomock.Eq(testUser.FullName),
						gomock.Eq(testUser.Email)).
					Times(1).
					Return(userservice.NewUserWihtoutPassword(testUser), nil)

				arg := domain.CreateSessionParams{
					Username: testUser.Username,
				}

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Eq(arg)).
					Times(1)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				data, err := ioutil.ReadAll(recorder.Body)
				require.NoError(t, err)

				var response response
				err = json.Unmarshal(data, &response)
				require.NoError(t, err)

				require.Equal(t, testUser.Username, response.Data.User.Username)
				require.Equal(t, testUser.FullName, response.Data.User.FullName)
				require.Equal(t, testUser.Email, response.Data.User.Email)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			sessionMaker := NewMockSessionMaker(ctrl)
			userService := NewMockService(ctrl)
			userHandler := NewHandler(userService, sessionMaker)

			server := gin.Default()
			url := "/users"
			server.POST(url, userHandler.Create)

			tc.buildStubs(userService, sessionMaker)

			body, err := json.Marshal(tc.requestBody)
			require.NoError(t, err)

			req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
			require.NoError(t, err)

			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, req)

			tc.checkResponse(recorder)
		})
	}
}

func TestLoginAPI(t *testing.T) {
	testUser, password := randomUser(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sessionMaker := NewMockSessionMaker(ctrl)
	userService := NewMockService(ctrl)
	userHandler := NewHandler(userService, sessionMaker)
	server := gin.Default()
	url := "/users/login"
	server.POST(url, userHandler.Login)

	testCases := []struct {
		name          string
		requestBody   gin.H
		buildStubs    func(userService *MockService, sessionMaker *MockSessionMaker)
		checkResponse func(recoder *httptest.ResponseRecorder)
	}{
		{
			name: "InvalidUsernameRequest",
			requestBody: gin.H{
				"username": "invalid-%user#1",
				"password": password,
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},

		{
			name: "ShortPasswordRequest",
			requestBody: gin.H{
				"username": testUser.Username,
				"password": "xyz",
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},

		{
			name: "UserNotFound",
			requestBody: gin.H{
				"username": "NotFound",
				"password": password,
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1).
					Return(domain.UserWihtoutPassword{}, domain.ErrUserNotFound)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},

		{
			name: "IncorrectPassword",
			requestBody: gin.H{
				"username": testUser.Username,
				"password": "incorrect",
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Eq(testUser.Username), gomock.Eq("incorrect")).
					Times(1).
					Return(domain.UserWihtoutPassword{}, domain.ErrWrongPassword)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},

		{
			name: "CheckPasswordInternalError",
			requestBody: gin.H{
				"username": testUser.Username,
				"password": password,
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Eq(testUser.Username), gomock.Eq(password)).
					Times(1).
					Return(domain.UserWihtoutPassword{}, errorspkg.ErrInternal)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},

		{
			name: "CreateSessionInternalError",
			requestBody: gin.H{
				"username": testUser.Username,
				"password": password,
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Eq(testUser.Username), gomock.Eq(password)).
					Times(1).
					Return(userservice.NewUserWihtoutPassword(testUser), nil)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(1).
					Return("", time.Now(), domain.Session{}, errorspkg.ErrInternal)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},

		{
			name: "OK",
			requestBody: gin.H{
				"username": testUser.Username,
				"password": password,
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Eq(testUser.Username), gomock.Eq(password)).
					Times(1).
					Return(userservice.NewUserWihtoutPassword(testUser), nil)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(1).
					Return("token", time.Now(), domain.Session{}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			tc.buildStubs(userService, sessionMaker)

			data, err := json.Marshal(tc.requestBody)
			require.NoError(t, err)

			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, request)

			tc.checkResponse(recorder)
		})
	}
}
