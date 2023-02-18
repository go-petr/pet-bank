package accountdelivery

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/internal/middleware"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/go-petr/pet-bank/pkg/randompkg"
	"github.com/go-petr/pet-bank/pkg/tokenpkg"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func randomAccount(owner string) domain.Account {
	return domain.Account{
		ID:        randompkg.IntBetween(1, 100),
		Owner:     owner,
		Balance:   randompkg.MoneyAmountBetween(1000, 10_000),
		Currency:  randompkg.Currency(),
		CreatedAt: time.Now().Truncate(time.Second).UTC(),
	}
}

func requireBodyMatchAccount(t *testing.T, body *bytes.Buffer, res response) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotRes response
	err = json.Unmarshal(data, &gotRes)
	require.NoError(t, err)
	require.Equal(t, res, gotRes)
}

func TestCreateAPI(t *testing.T) {
	username := randompkg.Owner()
	account := randomAccount(username)

	tokenMaker, err := tokenpkg.NewPasetoMaker(randompkg.String(32))
	require.NoError(t, err)

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		require.NoError(t, v.RegisterValidation("currency", ValidCurrency))
	}

	testCases := []struct {
		name          string
		requestBody   gin.H
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker)
		buildStubs    func(accountService *MockService)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "NoAuthorization",
			requestBody: gin.H{
				"currency": account.Currency,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker) {
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					Create(gomock.Any(), gomock.Any(), gomock.Any()).
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
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker) {
				err := middleware.AddAuthorization(request, tokenMaker, middleware.AuthTypeBearer, username, time.Minute)
				require.NoError(t, err)
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					Create(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},

		{
			name: "ErrOwnerNotFound",
			requestBody: gin.H{
				"currency": account.Currency,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker) {
				err := middleware.AddAuthorization(request, tokenMaker, middleware.AuthTypeBearer, username, time.Minute)
				require.NoError(t, err)
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					Create(gomock.Any(),
						gomock.Eq(account.Owner),
						gomock.Eq(account.Currency)).
					Times(1).
					Return(domain.Account{}, domain.ErrOwnerNotFound)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},

		{
			name: "ErrCurrencyAlreadyExists",
			requestBody: gin.H{
				"currency": account.Currency,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker) {
				err := middleware.AddAuthorization(request, tokenMaker, middleware.AuthTypeBearer, username, time.Minute)
				require.NoError(t, err)
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					Create(gomock.Any(),
						gomock.Eq(account.Owner),
						gomock.Eq(account.Currency)).
					Times(1).
					Return(domain.Account{}, domain.ErrCurrencyAlreadyExists)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, recorder.Code)
			},
		},

		{
			name: "InternalServerError",
			requestBody: gin.H{
				"currency": account.Currency,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker) {
				err := middleware.AddAuthorization(request, tokenMaker, middleware.AuthTypeBearer, username, time.Minute)
				require.NoError(t, err)
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					Create(gomock.Any(),
						gomock.Eq(account.Owner),
						gomock.Eq(account.Currency)).
					Times(1).
					Return(domain.Account{}, errorspkg.ErrInternal)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},

		{
			name: "OK",
			requestBody: gin.H{
				"currency": account.Currency,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker) {
				err := middleware.AddAuthorization(request, tokenMaker, middleware.AuthTypeBearer, username, time.Minute)
				require.NoError(t, err)
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					Create(gomock.Any(),
						gomock.Eq(account.Owner),
						gomock.Eq(account.Currency)).
					Times(1).
					Return(account, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				res := response{
					Data: data{account},
				}
				requireBodyMatchAccount(t, recorder.Body, res)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			accountService := NewMockService(ctrl)
			accountHandler := NewHandler(accountService)

			url := "/accounts"
			server := gin.Default()
			server.Use(middleware.AuthMiddleware(tokenMaker))
			server.POST(url, accountHandler.Create)

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

func TestGetAPI(t *testing.T) {
	username := randompkg.Owner()
	account := randomAccount(username)
	tokenMaker, err := tokenpkg.NewPasetoMaker(randompkg.String(32))
	require.NoError(t, err)

	testCases := []struct {
		name          string
		accountID     int32
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker)
		buildStubs    func(accountService *MockService)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:      "NoAuthorization",
			accountID: account.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker) {
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:      "InvalidID",
			accountID: 0,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker) {
				err := middleware.AddAuthorization(request, tokenMaker, middleware.AuthTypeBearer, username, time.Minute)
				require.NoError(t, err)
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:      "NotFound",
			accountID: account.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker) {
				err := middleware.AddAuthorization(request, tokenMaker, middleware.AuthTypeBearer, username, time.Minute)
				require.NoError(t, err)
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					Get(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(domain.Account{}, domain.ErrAccountNotFound)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:      "InternalError",
			accountID: account.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker) {
				err := middleware.AddAuthorization(request, tokenMaker, middleware.AuthTypeBearer, username, time.Minute)
				require.NoError(t, err)
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					Get(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(domain.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:      "UnauthorizedUser",
			accountID: account.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker) {
				err := middleware.AddAuthorization(request, tokenMaker, middleware.AuthTypeBearer, "UnauthorizedUser", time.Minute)
				require.NoError(t, err)
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					Get(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(account, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:      "OK",
			accountID: account.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker) {
				err := middleware.AddAuthorization(request, tokenMaker, middleware.AuthTypeBearer, username, time.Minute)
				require.NoError(t, err)
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					Get(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(account, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				res := response{
					Data: data{account},
				}
				requireBodyMatchAccount(t, recorder.Body, res)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			accountService := NewMockService(ctrl)
			accountHandler := NewHandler(accountService)

			server := gin.Default()
			server.Use(middleware.AuthMiddleware(tokenMaker))
			server.GET("/accounts/:id", accountHandler.Get)

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

func TestListAPI(t *testing.T) {
	username := randompkg.Owner()

	// n specifies number of account in DB
	n := 10
	accounts := make([]domain.Account, n)

	for i := 0; i < n; i++ {
		accounts[i] = randomAccount(username)
	}

	testCases := []struct {
		name          string
		pageID        int32
		pageSize      int32
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker)
		buildStubs    func(accountService *MockService)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:     "NoAuthorization",
			pageID:   1,
			pageSize: 5,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker) {
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
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
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker) {
				err := middleware.AddAuthorization(request, tokenMaker, middleware.AuthTypeBearer, username, time.Minute)
				require.NoError(t, err)
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
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
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker) {
				err := middleware.AddAuthorization(request, tokenMaker, middleware.AuthTypeBearer, username, time.Minute)
				require.NoError(t, err)
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
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
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker) {
				err := middleware.AddAuthorization(request, tokenMaker, middleware.AuthTypeBearer, username, time.Minute)
				require.NoError(t, err)
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1).
					Return([]domain.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:     "InternalError",
			pageID:   1,
			pageSize: 5,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker) {
				err := middleware.AddAuthorization(request, tokenMaker, middleware.AuthTypeBearer, username, time.Minute)
				require.NoError(t, err)
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1).
					Return(accounts, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				res := responseAccounts{
					Data: dataAccounts{accounts},
				}

				var gotRes responseAccounts
				err := json.Unmarshal(recorder.Body.Bytes(), &gotRes)
				require.NoError(t, err)

				require.Equal(t, res, gotRes)
			},
		},
	}

	tokenMaker, err := tokenpkg.NewPasetoMaker(randompkg.String(32))
	require.NoError(t, err)

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			accountService := NewMockService(ctrl)
			accountHandler := NewHandler(accountService)

			tc.buildStubs(accountService)

			server := gin.Default()
			server.Use(middleware.AuthMiddleware(tokenMaker))
			server.GET("/accounts", accountHandler.List)

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
