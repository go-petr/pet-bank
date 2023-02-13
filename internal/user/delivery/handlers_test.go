package delivery

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-petr/pet-bank/internal/session"
	"github.com/go-petr/pet-bank/internal/user"
	"github.com/go-petr/pet-bank/internal/user/service"
	"github.com/go-petr/pet-bank/pkg/configpkg"
	"github.com/go-petr/pet-bank/pkg/apperrors"
	"github.com/go-petr/pet-bank/pkg/passpkg"
	"github.com/go-petr/pet-bank/pkg/apprandom"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

var (
	testConfig configpkg.Config
)

func TestMain(m *testing.M) {
	testConfig = configpkg.Config{
		TokenSymmetricKey:   apprandom.String(32),
		AccessTokenDuration: time.Minute,
	}
	gin.SetMode(gin.ReleaseMode)
	os.Exit(m.Run())
}

func randomUser(t *testing.T) (user.User, string) {

	password := apprandom.String(10)

	hashedPassword, err := passpkg.Hash(password)
	require.NoError(t, err)

	user := user.User{
		Username:       apprandom.Owner(),
		HashedPassword: hashedPassword,
		FullName:       apprandom.Owner(),
		Email:          apprandom.Email(),
	}

	return user, password
}

func TestCreateUserAPI(t *testing.T) {

	testUser, password := randomUser(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sessionMaker := NewMockSessionMakerInterface(ctrl)
	userService := NewMockuserServiceInterface(ctrl)
	userHandler := NewUserHandler(userService, sessionMaker)
	server := gin.Default()
	url := "/users"
	server.POST(url, userHandler.CreateUser)

	testCases := []struct {
		name          string
		requestBody   gin.H
		buildStubs    func(userService *MockuserServiceInterface, sessionMaker *MockSessionMakerInterface)
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
			buildStubs: func(userService *MockuserServiceInterface, sessionMaker *MockSessionMakerInterface) {

				userService.EXPECT().
					CreateUser(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
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
			buildStubs: func(userService *MockuserServiceInterface, sessionMaker *MockSessionMakerInterface) {

				userService.EXPECT().
					CreateUser(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
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
			buildStubs: func(userService *MockuserServiceInterface, sessionMaker *MockSessionMakerInterface) {

				userService.EXPECT().
					CreateUser(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
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
			buildStubs: func(userService *MockuserServiceInterface, sessionMaker *MockSessionMakerInterface) {

				userService.EXPECT().
					CreateUser(gomock.Any(),
						gomock.Eq(testUser.Username),
						gomock.Eq(password),
						gomock.Eq(testUser.FullName),
						gomock.Eq(testUser.Email)).
					Times(1).
					Return(user.UserWihtoutPassword{}, user.ErrUsernameAlreadyExists)

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
			buildStubs: func(userService *MockuserServiceInterface, sessionMaker *MockSessionMakerInterface) {

				userService.EXPECT().
					CreateUser(gomock.Any(),
						gomock.Eq(testUser.Username),
						gomock.Eq(password),
						gomock.Eq(testUser.FullName),
						gomock.Eq(testUser.Email)).
					Times(1).
					Return(user.UserWihtoutPassword{}, user.ErrEmailALreadyExists)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, recorder.Code)
			},
		},
		{
			name: "CreateUserInternalError",
			requestBody: gin.H{
				"username": testUser.Username,
				"password": password,
				"fullname": testUser.FullName,
				"email":    testUser.Email,
			},
			buildStubs: func(userService *MockuserServiceInterface, sessionMaker *MockSessionMakerInterface) {

				userService.EXPECT().
					CreateUser(gomock.Any(),
						gomock.Eq(testUser.Username),
						gomock.Eq(password),
						gomock.Eq(testUser.FullName),
						gomock.Eq(testUser.Email)).
					Times(1).
					Return(user.UserWihtoutPassword{}, apperrors.ErrInternal)

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
			buildStubs: func(userService *MockuserServiceInterface, sessionMaker *MockSessionMakerInterface) {

				userService.EXPECT().
					CreateUser(gomock.Any(),
						gomock.Eq(testUser.Username),
						gomock.Eq(password),
						gomock.Eq(testUser.FullName),
						gomock.Eq(testUser.Email)).
					Times(1).
					Return(user.UserWihtoutPassword{}, nil)

				arg := session.CreateSessionParams{
					Username: testUser.Username,
				}

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return("", time.Now(), session.Session{}, apperrors.ErrInternal)
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
			buildStubs: func(userService *MockuserServiceInterface, sessionMaker *MockSessionMakerInterface) {

				userService.EXPECT().
					CreateUser(gomock.Any(),
						gomock.Eq(testUser.Username),
						gomock.Eq(password),
						gomock.Eq(testUser.FullName),
						gomock.Eq(testUser.Email)).
					Times(1).
					Return(service.NewUserWihtoutPassword(testUser), nil)

				arg := session.CreateSessionParams{
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

				var response userResponse
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

func TestLoginUserAPI(t *testing.T) {

	testUser, password := randomUser(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sessionMaker := NewMockSessionMakerInterface(ctrl)
	userService := NewMockuserServiceInterface(ctrl)
	userHandler := NewUserHandler(userService, sessionMaker)
	server := gin.Default()
	url := "/users/login"
	server.POST(url, userHandler.LoginUser)

	testCases := []struct {
		name          string
		requestBody   gin.H
		buildStubs    func(userService *MockuserServiceInterface, sessionMaker *MockSessionMakerInterface)
		checkResponse func(recoder *httptest.ResponseRecorder)
	}{
		{
			name: "InvalidUsernameRequest",
			requestBody: gin.H{
				"username": "invalid-%user#1",
				"password": password,
			},
			buildStubs: func(userService *MockuserServiceInterface, sessionMaker *MockSessionMakerInterface) {

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
			buildStubs: func(userService *MockuserServiceInterface, sessionMaker *MockSessionMakerInterface) {

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
			buildStubs: func(userService *MockuserServiceInterface, sessionMaker *MockSessionMakerInterface) {

				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1).
					Return(user.UserWihtoutPassword{}, user.ErrUserNotFound)

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
			buildStubs: func(userService *MockuserServiceInterface, sessionMaker *MockSessionMakerInterface) {

				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Eq(testUser.Username), gomock.Eq("incorrect")).
					Times(1).
					Return(user.UserWihtoutPassword{}, user.ErrWrongPassword)

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
			buildStubs: func(userService *MockuserServiceInterface, sessionMaker *MockSessionMakerInterface) {

				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Eq(testUser.Username), gomock.Eq(password)).
					Times(1).
					Return(user.UserWihtoutPassword{}, errors.New("Internal"))

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
			buildStubs: func(userService *MockuserServiceInterface, sessionMaker *MockSessionMakerInterface) {

				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Eq(testUser.Username), gomock.Eq(password)).
					Times(1).
					Return(service.NewUserWihtoutPassword(testUser), nil)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(1).
					Return("", time.Now(), session.Session{}, errors.New("Internal"))
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
			buildStubs: func(userService *MockuserServiceInterface, sessionMaker *MockSessionMakerInterface) {

				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Eq(testUser.Username), gomock.Eq(password)).
					Times(1).
					Return(service.NewUserWihtoutPassword(testUser), nil)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(1).
					Return("token", time.Now(), session.Session{}, nil)
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
