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
	"github.com/go-petr/pet-bank/internal/user"
	"github.com/go-petr/pet-bank/internal/user/service"
	"github.com/go-petr/pet-bank/pkg/token"
	"github.com/go-petr/pet-bank/pkg/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

var (
	testConfig util.Config
)

func TestMain(m *testing.M) {
	testConfig = util.Config{
		TokenSymmetricKey:   util.RandomString(32),
		AccessTokenDuration: time.Minute,
	}
	os.Exit(m.Run())
}

func randomUser(t *testing.T) (user.User, string) {

	password := util.RandomString(10)

	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)

	user := user.User{
		Username:       util.RandomOwner(),
		HashedPassword: hashedPassword,
		FullName:       util.RandomOwner(),
		Email:          util.RandomEmail(),
	}

	return user, password
}

func TestCreateUserAPI(t *testing.T) {

	testUser, password := randomUser(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tokenMaker := token.NewMockMaker(ctrl)
	userService := NewMockuserServiceInterface(ctrl)
	userHandler := NewUserHandler(userService, tokenMaker, time.Minute)
	server := gin.Default()
	url := "/users"
	server.POST(url, userHandler.CreateUser)

	testCases := []struct {
		name          string
		requestBody   gin.H
		buildStubs    func(userService *MockuserServiceInterface, tokenMaker *token.MockMaker)
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
			buildStubs: func(userService *MockuserServiceInterface, tokenMaker *token.MockMaker) {

				userService.EXPECT().
					CreateUser(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)

				tokenMaker.EXPECT().
					CreateToken(gomock.Any(), gomock.Any()).
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
			buildStubs: func(userService *MockuserServiceInterface, tokenMaker *token.MockMaker) {

				userService.EXPECT().
					CreateUser(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)

				tokenMaker.EXPECT().
					CreateToken(gomock.Any(), gomock.Any()).
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
			buildStubs: func(userService *MockuserServiceInterface, tokenMaker *token.MockMaker) {

				userService.EXPECT().
					CreateUser(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)

				tokenMaker.EXPECT().
					CreateToken(gomock.Any(), gomock.Any()).
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
				"fullname": testUser.FullName,
				"email":    testUser.Email,
			},
			buildStubs: func(userService *MockuserServiceInterface, tokenMaker *token.MockMaker) {

				userService.EXPECT().
					CreateUser(gomock.Any(),
						gomock.Eq("NotFound"),
						gomock.Eq(password),
						gomock.Eq(testUser.FullName),
						gomock.Eq(testUser.Email)).
					Times(1).
					Return(user.UserWihtoutPassword{}, user.ErrUserNotFound)

				tokenMaker.EXPECT().
					CreateToken(gomock.Any(), gomock.Any()).
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
			buildStubs: func(userService *MockuserServiceInterface, tokenMaker *token.MockMaker) {

				userService.EXPECT().
					CreateUser(gomock.Any(),
						gomock.Eq(testUser.Username),
						gomock.Eq(password),
						gomock.Eq(testUser.FullName),
						gomock.Eq(testUser.Email)).
					Times(1).
					Return(user.UserWihtoutPassword{}, user.ErrUsernameAlreadyExists)

				tokenMaker.EXPECT().
					CreateToken(gomock.Any(), gomock.Any()).
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
			buildStubs: func(userService *MockuserServiceInterface, tokenMaker *token.MockMaker) {

				userService.EXPECT().
					CreateUser(gomock.Any(),
						gomock.Eq(testUser.Username),
						gomock.Eq(password),
						gomock.Eq(testUser.FullName),
						gomock.Eq(testUser.Email)).
					Times(1).
					Return(user.UserWihtoutPassword{}, user.ErrEmailALreadyExists)

				tokenMaker.EXPECT().
					CreateToken(gomock.Any(), gomock.Any()).
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
			buildStubs: func(userService *MockuserServiceInterface, tokenMaker *token.MockMaker) {

				userService.EXPECT().
					CreateUser(gomock.Any(),
						gomock.Eq(testUser.Username),
						gomock.Eq(password),
						gomock.Eq(testUser.FullName),
						gomock.Eq(testUser.Email)).
					Times(1).
					Return(user.UserWihtoutPassword{}, util.ErrInternal)

				tokenMaker.EXPECT().
					CreateToken(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "CreateTokenInternalError",
			requestBody: gin.H{
				"username": testUser.Username,
				"password": password,
				"fullname": testUser.FullName,
				"email":    testUser.Email,
			},
			buildStubs: func(userService *MockuserServiceInterface, tokenMaker *token.MockMaker) {

				userService.EXPECT().
					CreateUser(gomock.Any(),
						gomock.Eq(testUser.Username),
						gomock.Eq(password),
						gomock.Eq(testUser.FullName),
						gomock.Eq(testUser.Email)).
					Times(1).
					Return(user.UserWihtoutPassword{}, nil)

				tokenMaker.EXPECT().
					CreateToken(gomock.Eq(testUser.Username), gomock.Eq(userHandler.tokenDuration)).
					Times(1).
					Return("", util.ErrInternal)
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
			buildStubs: func(userService *MockuserServiceInterface, tokenMaker *token.MockMaker) {

				userService.EXPECT().
					CreateUser(gomock.Any(),
						gomock.Eq(testUser.Username),
						gomock.Eq(password),
						gomock.Eq(testUser.FullName),
						gomock.Eq(testUser.Email)).
					Times(1).
					Return(service.NewUserWihtoutPassword(testUser), nil)

				tokenMaker.EXPECT().
					CreateToken(gomock.Eq(testUser.Username), gomock.Eq(userHandler.tokenDuration)).
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

			tc.buildStubs(userService, tokenMaker)

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

	tokenMaker := token.NewMockMaker(ctrl)
	userService := NewMockuserServiceInterface(ctrl)
	userHandler := NewUserHandler(userService, tokenMaker, time.Minute)
	server := gin.Default()
	url := "/users/login"
	server.POST(url, userHandler.LoginUser)

	testCases := []struct {
		name          string
		requestBody   gin.H
		buildStubs    func(userService *MockuserServiceInterface, tokenMaker *token.MockMaker)
		checkResponse func(recoder *httptest.ResponseRecorder)
	}{
		{
			name: "InvalidUsernameRequest",
			requestBody: gin.H{
				"username": "invalid-%user#1",
				"password": password,
			},
			buildStubs: func(userService *MockuserServiceInterface, tokenMaker *token.MockMaker) {

				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)

				tokenMaker.EXPECT().
					CreateToken(gomock.Eq(testUser.Username), gomock.AssignableToTypeOf(time.Minute)).
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
			buildStubs: func(userService *MockuserServiceInterface, tokenMaker *token.MockMaker) {

				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)

				tokenMaker.EXPECT().
					CreateToken(gomock.Eq(testUser.Username), gomock.AssignableToTypeOf(time.Minute)).
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
			buildStubs: func(userService *MockuserServiceInterface, tokenMaker *token.MockMaker) {

				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1).
					Return(user.UserWihtoutPassword{}, user.ErrUserNotFound)

				tokenMaker.EXPECT().
					CreateToken(gomock.Eq(testUser.Username), gomock.AssignableToTypeOf(time.Minute)).
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
			buildStubs: func(userService *MockuserServiceInterface, tokenMaker *token.MockMaker) {

				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Eq(testUser.Username), gomock.Eq("incorrect")).
					Times(1).
					Return(user.UserWihtoutPassword{}, user.ErrWrongPassword)

				tokenMaker.EXPECT().
					CreateToken(gomock.Eq(testUser.Username), gomock.AssignableToTypeOf(time.Minute)).
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
			buildStubs: func(userService *MockuserServiceInterface, tokenMaker *token.MockMaker) {

				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Eq(testUser.Username), gomock.Eq(password)).
					Times(1).
					Return(user.UserWihtoutPassword{}, errors.New("Internal"))

				tokenMaker.EXPECT().
					CreateToken(gomock.Eq(testUser.Username), gomock.AssignableToTypeOf(time.Minute)).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},

		{
			name: "CreateTokenInternalError",
			requestBody: gin.H{
				"username": testUser.Username,
				"password": password,
			},
			buildStubs: func(userService *MockuserServiceInterface, tokenMaker *token.MockMaker) {

				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Eq(testUser.Username), gomock.Eq(password)).
					Times(1).
					Return(service.NewUserWihtoutPassword(testUser), nil)

				tokenMaker.EXPECT().
					CreateToken(gomock.Eq(testUser.Username), gomock.AssignableToTypeOf(time.Minute)).
					Times(1).
					Return("", errors.New("Internal"))
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
			buildStubs: func(userService *MockuserServiceInterface, tokenMaker *token.MockMaker) {

				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Eq(testUser.Username), gomock.Eq(password)).
					Times(1).
					Return(service.NewUserWihtoutPassword(testUser), nil)

				tokenMaker.EXPECT().
					CreateToken(gomock.Eq(testUser.Username), gomock.AssignableToTypeOf(time.Minute)).
					Times(1).
					Return("token", nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
	}

	for i := range testCases {

		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {

			tc.buildStubs(userService, tokenMaker)

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
