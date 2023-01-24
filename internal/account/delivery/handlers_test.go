package delivery

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"github.com/go-petr/pet-bank/internal/account"
	"github.com/go-petr/pet-bank/internal/middleware"
	"github.com/go-petr/pet-bank/pkg/token"
	"github.com/go-petr/pet-bank/pkg/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func randomAccount(owner string) account.Account {
	return account.Account{
		ID:        util.RandomInt(1, 100),
		Owner:     owner,
		Balance:   util.RandomMoneyAmountBetween(1000, 10_000),
		Currency:  util.RandomCurrency(),
		CreatedAt: time.Now().Truncate(time.Second).UTC(),
	}
}

func requireBodyMatchAccount(t *testing.T, body *bytes.Buffer, acc account.Account) {

	data, err := ioutil.ReadAll(body)
	require.NoError(t, err)

	var gotAccount account.Account
	err = json.Unmarshal(data, &gotAccount)
	require.NoError(t, err)
	require.Equal(t, acc, gotAccount)
}

func TestCreateAccountAPI(t *testing.T) {

	testUsername := util.RandomOwner()
	testAccount := randomAccount(testUsername)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	accountService := NewMockaccountServiceInterface(ctrl)
	accountHandler := NewAccountHandler(accountService)

	tokenMaker, err := token.NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	url := "/accounts"
	server := gin.Default()
	server.Use(middleware.AuthMiddleware(tokenMaker))
	server.POST(url, accountHandler.CreateAccount)
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("currency", ValidCurrency)
	}

	testCases := []struct {
		name          string
		requestBody   gin.H
		setupAuth     func(t *testing.T, request *http.Request, TokenMaker token.Maker)
		buildStubs    func(accountService *MockaccountServiceInterface)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "NoAuthorization",
			requestBody: gin.H{
				"currency": testAccount.Currency,
			},
			setupAuth: func(t *testing.T, request *http.Request, TokenMaker token.Maker) {
			},
			buildStubs: func(accountService *MockaccountServiceInterface) {

				accountService.EXPECT().
					CreateAccount(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},

		{
			name: "InvalidCurrency",
			requestBody: gin.H{
				"currency": "RUB",
			},
			setupAuth: func(t *testing.T, request *http.Request, TokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, TokenMaker,
					middleware.AuthorizationTypeBearer, testUsername, time.Minute)
			},
			buildStubs: func(accountService *MockaccountServiceInterface) {

				accountService.EXPECT().
					CreateAccount(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},

		{
			name: "ErrNoOwnerExists",
			requestBody: gin.H{
				"currency": testAccount.Currency,
			},
			setupAuth: func(t *testing.T, request *http.Request, TokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, TokenMaker,
					middleware.AuthorizationTypeBearer, testUsername, time.Minute)
			},
			buildStubs: func(accountService *MockaccountServiceInterface) {

				accountService.EXPECT().
					CreateAccount(gomock.Any(),
						gomock.Eq(testAccount.Owner),
						gomock.Eq(testAccount.Currency)).
					Times(1).
					Return(account.Account{}, account.ErrNoOwnerExists)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},

		{
			name: "ErrCurrencyAlreadyExists",
			requestBody: gin.H{
				"currency": testAccount.Currency,
			},
			setupAuth: func(t *testing.T, request *http.Request, TokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, TokenMaker,
					middleware.AuthorizationTypeBearer, testUsername, time.Minute)
			},
			buildStubs: func(accountService *MockaccountServiceInterface) {

				accountService.EXPECT().
					CreateAccount(gomock.Any(),
						gomock.Eq(testAccount.Owner),
						gomock.Eq(testAccount.Currency)).
					Times(1).
					Return(account.Account{}, account.ErrCurrencyAlreadyExists)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, recorder.Code)
			},
		},

		{
			name: "InternalServerError",
			requestBody: gin.H{
				"currency": testAccount.Currency,
			},
			setupAuth: func(t *testing.T, request *http.Request, TokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, TokenMaker,
					middleware.AuthorizationTypeBearer, testUsername, time.Minute)
			},
			buildStubs: func(accountService *MockaccountServiceInterface) {

				accountService.EXPECT().
					CreateAccount(gomock.Any(),
						gomock.Eq(testAccount.Owner),
						gomock.Eq(testAccount.Currency)).
					Times(1).
					Return(account.Account{}, util.ErrInternal)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},

		{
			name: "OK",
			requestBody: gin.H{
				"currency": testAccount.Currency,
			},
			setupAuth: func(t *testing.T, request *http.Request, TokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, TokenMaker,
					middleware.AuthorizationTypeBearer, testUsername, time.Minute)
			},
			buildStubs: func(accountService *MockaccountServiceInterface) {
				accountService.EXPECT().
					CreateAccount(gomock.Any(),
						gomock.Eq(testAccount.Owner),
						gomock.Eq(testAccount.Currency)).
					Times(1).
					Return(testAccount, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchAccount(t, recorder.Body, testAccount)
			},
		},
	}

	for i := range testCases {

		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {

			tc.buildStubs(accountService)

			recorder := httptest.NewRecorder()

			body, err := json.Marshal(tc.requestBody)
			require.NoError(t, err)

			req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
			require.NoError(t, err)

			tc.setupAuth(t, req, tokenMaker)
			server.ServeHTTP(recorder, req)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestGetAccountAPI(t *testing.T) {

	testUsername := util.RandomOwner()
	testAccount := randomAccount(testUsername)
	tokenMaker, err := token.NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	accountService := NewMockaccountServiceInterface(ctrl)
	accountHandler := NewAccountHandler(accountService)

	// start test server and send request
	server := gin.Default()
	server.Use(middleware.AuthMiddleware(tokenMaker))
	server.GET("/accounts/:id", accountHandler.GetAccount)

	testCases := []struct {
		name          string
		accountID     int32
		setupAuth     func(t *testing.T, request *http.Request, TokenMaker token.Maker)
		buildStubs    func(accountService *MockaccountServiceInterface)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:      "NoAuthorization",
			accountID: testAccount.ID,
			setupAuth: func(t *testing.T, request *http.Request, TokenMaker token.Maker) {
			},
			buildStubs: func(accountService *MockaccountServiceInterface) {

				accountService.EXPECT().
					GetAccount(gomock.Any(), gomock.Any()).
					Times(0)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:      "InvalidID",
			accountID: 0,
			setupAuth: func(t *testing.T, request *http.Request, TokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, TokenMaker,
					middleware.AuthorizationTypeBearer, testUsername, time.Minute)
			},
			buildStubs: func(accountService *MockaccountServiceInterface) {

				accountService.EXPECT().
					GetAccount(gomock.Any(), gomock.Any()).
					Times(0)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:      "NotFound",
			accountID: testAccount.ID,
			setupAuth: func(t *testing.T, request *http.Request, TokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, TokenMaker,
					middleware.AuthorizationTypeBearer, testUsername, time.Minute)
			},
			buildStubs: func(accountService *MockaccountServiceInterface) {

				accountService.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(testAccount.ID)).
					Times(1).
					Return(account.Account{}, account.ErrAccountNotFound)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:      "InternalError",
			accountID: testAccount.ID,
			setupAuth: func(t *testing.T, request *http.Request, TokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, TokenMaker,
					middleware.AuthorizationTypeBearer, testUsername, time.Minute)
			},
			buildStubs: func(accountService *MockaccountServiceInterface) {
				accountService.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(testAccount.ID)).
					Times(1).
					Return(account.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:      "UnauthorizedUser",
			accountID: testAccount.ID,
			setupAuth: func(t *testing.T, request *http.Request, TokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, TokenMaker,
					middleware.AuthorizationTypeBearer, "UnauthorizedUser", time.Minute)
			},
			buildStubs: func(accountService *MockaccountServiceInterface) {

				accountService.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(testAccount.ID)).
					Times(1).
					Return(testAccount, nil)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:      "OK",
			accountID: testAccount.ID,
			setupAuth: func(t *testing.T, request *http.Request, TokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, TokenMaker,
					middleware.AuthorizationTypeBearer, testUsername, time.Minute)
			},
			buildStubs: func(accountService *MockaccountServiceInterface) {
				accountService.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(testAccount.ID)).
					Times(1).
					Return(testAccount, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchAccount(t, recorder.Body, testAccount)
			},
		},
	}

	for i := range testCases {

		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {

			tc.buildStubs(accountService)

			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/accounts/%d", tc.accountID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, tokenMaker)
			server.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestListAccountsAPI(t *testing.T) {

	testUsername := util.RandomOwner()

	// n specifies number of account in DB
	n := 10
	accounts := make([]account.Account, n)
	for i := 0; i < n; i++ {
		accounts[i] = randomAccount(testUsername)
	}

	testCases := []struct {
		name          string
		pageID        int32
		pageSize      int32
		setupAuth     func(t *testing.T, request *http.Request, TokenMaker token.Maker)
		buildStubs    func(accountService *MockaccountServiceInterface)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:     "NoAuthorization",
			pageID:   1,
			pageSize: 5,
			setupAuth: func(t *testing.T, request *http.Request, TokenMaker token.Maker) {
			},
			buildStubs: func(accountService *MockaccountServiceInterface) {
				accountService.EXPECT().
					ListAccounts(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:     "InvalidPageID",
			pageID:   -1,
			pageSize: 5,
			setupAuth: func(t *testing.T, request *http.Request, TokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, TokenMaker,
					middleware.AuthorizationTypeBearer, testUsername, time.Minute)
			},
			buildStubs: func(accountService *MockaccountServiceInterface) {
				accountService.EXPECT().
					ListAccounts(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:     "InvalidPageSize",
			pageID:   1,
			pageSize: 500,
			setupAuth: func(t *testing.T, request *http.Request, TokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, TokenMaker,
					middleware.AuthorizationTypeBearer, testUsername, time.Minute)
			},
			buildStubs: func(accountService *MockaccountServiceInterface) {
				accountService.EXPECT().
					ListAccounts(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:     "InternalError",
			pageID:   1,
			pageSize: 5,
			setupAuth: func(t *testing.T, request *http.Request, TokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, TokenMaker,
					middleware.AuthorizationTypeBearer, testUsername, time.Minute)
			},
			buildStubs: func(accountService *MockaccountServiceInterface) {
				accountService.EXPECT().
					ListAccounts(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1).
					Return([]account.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:     "InternalError",
			pageID:   1,
			pageSize: 5,
			setupAuth: func(t *testing.T, request *http.Request, TokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, TokenMaker,
					middleware.AuthorizationTypeBearer, testUsername, time.Minute)
			},
			buildStubs: func(accountService *MockaccountServiceInterface) {
				accountService.EXPECT().
					ListAccounts(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1).
					Return(accounts, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var gotAccounts []account.Account
				err := json.Unmarshal(recorder.Body.Bytes(), &gotAccounts)
				require.NoError(t, err)

				require.Equal(t, accounts, gotAccounts)

			},
		},
	}

	tokenMaker, err := token.NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	for i := range testCases {

		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			accountService := NewMockaccountServiceInterface(ctrl)
			accountHandler := NewAccountHandler(accountService)

			tc.buildStubs(accountService)

			server := gin.Default()
			server.Use(middleware.AuthMiddleware(tokenMaker))
			server.GET("/accounts", accountHandler.ListAccounts)

			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/accounts?page_id=%v&page_size=%v", tc.pageID, tc.pageSize)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, req, tokenMaker)
			server.ServeHTTP(recorder, req)
			tc.checkResponse(t, recorder)
		})
	}

}
