package transferdelivery

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/internal/middleware"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/go-petr/pet-bank/pkg/randompkg"
	"github.com/go-petr/pet-bank/pkg/token"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func randomAccount(owner string) domain.Account {
	return domain.Account{
		ID:        randompkg.IntBetween(1, 100),
		Owner:     owner,
		Balance:   randompkg.MoneyAmountBetween(1000, 10_000),
		Currency:  randompkg.Currency(),
		CreatedAt: time.Now().Truncate(time.Second).UTC(),
	}
}

func TestCreateTranferAPI(t *testing.T) {
	testUsername1 := randompkg.Owner()
	testUsername2 := randompkg.Owner()

	testAccount1 := randomAccount(testUsername1)
	testAccount2 := randomAccount(testUsername2)
	amount := "100"

	tokenMaker, err := token.NewPasetoMaker(randompkg.String(32))
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	transferService := NewMockService(ctrl)
	transferHandler := NewHandler(transferService)

	server := gin.Default()
	url := "/transfers"

	server.Use(middleware.AuthMiddleware(tokenMaker))
	server.POST(url, transferHandler.Create)

	testCases := []struct {
		name          string
		requestBody   gin.H
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(transferService *MockService)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "NoAuthorization",
			requestBody: gin.H{
				"from_account_id": testAccount1.ID,
				"to_account_id":   testAccount2.ID,
				"amount":          amount,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			},
			buildStubs: func(transferService *MockService) {
				transferService.EXPECT().Transfer(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "InvalidBindFromAccountID",
			requestBody: gin.H{
				"from_account_id": 0,
				"to_account_id":   testAccount2.ID,
				"amount":          amount,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, tokenMaker, middleware.AuthorizationTypeBearer, testUsername1, time.Minute)
			},
			buildStubs: func(transferService *MockService) {
				transferService.EXPECT().Transfer(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidBindToAccountID",
			requestBody: gin.H{
				"from_account_id": testAccount1.ID,
				"to_account_id":   0,
				"amount":          amount,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, tokenMaker, middleware.AuthorizationTypeBearer, testUsername1, time.Minute)
			},
			buildStubs: func(transferService *MockService) {
				transferService.EXPECT().Transfer(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidAmount",
			requestBody: gin.H{
				"from_account_id": testAccount1.ID,
				"to_account_id":   testAccount2.ID,
				"amount":          "",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, tokenMaker, middleware.AuthorizationTypeBearer, testUsername1, time.Minute)
			},
			buildStubs: func(transferService *MockService) {
				transferService.EXPECT().Transfer(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "Invalid owner",
			requestBody: gin.H{
				"from_account_id": testAccount1.ID,
				"to_account_id":   testAccount2.ID,
				"amount":          amount,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, tokenMaker, middleware.AuthorizationTypeBearer, testUsername1, time.Minute)
			},
			buildStubs: func(transferService *MockService) {
				arg := domain.CreateTransferParams{
					FromAccountID: testAccount1.ID,
					ToAccountID:   testAccount2.ID,
					Amount:        amount,
				}

				transferService.EXPECT().
					Transfer(gomock.Any(), gomock.Eq(testUsername1), gomock.Eq(arg)).
					Times(1).
					Return(domain.TransferTxResult{}, domain.ErrInvalidOwner)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "InvalidTransferRequestError",
			requestBody: gin.H{
				"from_account_id": testAccount1.ID,
				"to_account_id":   testAccount2.ID,
				"amount":          amount,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, tokenMaker, middleware.AuthorizationTypeBearer, testUsername1, time.Minute)
			},
			buildStubs: func(transferService *MockService) {
				arg := domain.CreateTransferParams{
					FromAccountID: testAccount1.ID,
					ToAccountID:   testAccount2.ID,
					Amount:        amount,
				}

				transferService.EXPECT().
					Transfer(gomock.Any(), gomock.Eq(testUsername1), gomock.Eq(arg)).
					Times(1).
					Return(domain.TransferTxResult{}, domain.ErrCurrencyMismatch)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidTransferInternalError",
			requestBody: gin.H{
				"from_account_id": testAccount1.ID,
				"to_account_id":   testAccount2.ID,
				"amount":          amount,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, tokenMaker, middleware.AuthorizationTypeBearer, testUsername1, time.Minute)
			},
			buildStubs: func(transferService *MockService) {
				arg := domain.CreateTransferParams{
					FromAccountID: testAccount1.ID,
					ToAccountID:   testAccount2.ID,
					Amount:        amount,
				}

				transferService.EXPECT().
					Transfer(gomock.Any(), gomock.Eq(testUsername1), gomock.Eq(arg)).
					Times(1).
					Return(domain.TransferTxResult{}, errorspkg.ErrInternal)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "OK",
			requestBody: gin.H{
				"from_account_id": testAccount1.ID,
				"to_account_id":   testAccount2.ID,
				"amount":          amount,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, tokenMaker, middleware.AuthorizationTypeBearer, testUsername1, time.Minute)
			},
			buildStubs: func(transferService *MockService) {
				arg := domain.CreateTransferParams{
					FromAccountID: testAccount1.ID,
					ToAccountID:   testAccount2.ID,
					Amount:        amount,
				}

				transferService.EXPECT().
					Transfer(gomock.Any(), gomock.Eq(testUsername1), gomock.Eq(arg)).
					Times(1).
					Return(domain.TransferTxResult{}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			tc.buildStubs(transferService)

			recorder := httptest.NewRecorder()

			body, err := json.Marshal(tc.requestBody)
			require.NoError(t, err)

			req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
			require.NoError(t, err)

			tc.setupAuth(t, req, tokenMaker)
			server.ServeHTTP(recorder, req)
			tc.checkResponse(recorder)
		})
	}
}
