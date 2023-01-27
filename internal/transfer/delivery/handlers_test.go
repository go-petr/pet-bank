package delivery

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/go-petr/pet-bank/internal/account"
	"github.com/go-petr/pet-bank/internal/middleware"
	"github.com/go-petr/pet-bank/internal/transfer"
	"github.com/go-petr/pet-bank/pkg/token"
	"github.com/go-petr/pet-bank/pkg/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func randomAccount(owner string) account.Account {
	return account.Account{
		ID:        util.RandomInt(1, 100),
		Owner:     owner,
		Balance:   util.RandomMoneyAmountBetween(1000, 10_000),
		Currency:  util.RandomCurrency(),
		CreatedAt: time.Now().Truncate(time.Second).UTC(),
	}
}

func TestCreateTranferAPI(t *testing.T) {

	testUsername1 := util.RandomOwner()
	testUsername2 := util.RandomOwner()

	testAccount1 := randomAccount(testUsername1)
	testAccount2 := randomAccount(testUsername2)
	amount := "100"

	tokenMaker, err := token.NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	transferService := NewMocktransferServiceInterface(ctrl)
	transferHandler := NewTransferHandler(transferService)

	server := gin.Default()
	url := "/transfers"
	server.Use(middleware.AuthMiddleware(tokenMaker))
	server.POST(url, transferHandler.CreateTransfer)

	testCases := []struct {
		name          string
		requestBody   gin.H
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(transferService *MocktransferServiceInterface)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "NoAuthorization",
			requestBody: gin.H{
				"from_account_id": testAccount1.ID,
				"to_account_id":   testAccount2.ID,
				"amount":          amount,
			},
			setupAuth: func(t *testing.T, request *http.Request, TokenMaker token.Maker) {
			},
			buildStubs: func(transferService *MocktransferServiceInterface) {
				transferService.EXPECT().TransferTx(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
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
			buildStubs: func(transferService *MocktransferServiceInterface) {
				transferService.EXPECT().TransferTx(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
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
			buildStubs: func(transferService *MocktransferServiceInterface) {

				transferService.EXPECT().TransferTx(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
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
			buildStubs: func(transferService *MocktransferServiceInterface) {

				transferService.EXPECT().TransferTx(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
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
			buildStubs: func(transferService *MocktransferServiceInterface) {

				arg := transfer.CreateTransferParams{
					FromAccountID: testAccount1.ID,
					ToAccountID:   testAccount2.ID,
					Amount:        amount,
				}

				transferService.EXPECT().
					TransferTx(gomock.Any(), gomock.Eq(testUsername1), gomock.Eq(arg)).
					Times(1).
					Return(transfer.TransferTxResult{}, transfer.ErrInvalidOwner)
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
			buildStubs: func(transferService *MocktransferServiceInterface) {

				arg := transfer.CreateTransferParams{
					FromAccountID: testAccount1.ID,
					ToAccountID:   testAccount2.ID,
					Amount:        amount,
				}

				transferService.EXPECT().
					TransferTx(gomock.Any(), gomock.Eq(testUsername1), gomock.Eq(arg)).
					Times(1).
					Return(transfer.TransferTxResult{}, transfer.ErrCurrencyMismatch)
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
			buildStubs: func(transferService *MocktransferServiceInterface) {

				arg := transfer.CreateTransferParams{
					FromAccountID: testAccount1.ID,
					ToAccountID:   testAccount2.ID,
					Amount:        amount,
				}

				transferService.EXPECT().
					TransferTx(gomock.Any(), gomock.Eq(testUsername1), gomock.Eq(arg)).
					Times(1).
					Return(transfer.TransferTxResult{}, util.ErrInternal)
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
			buildStubs: func(transferService *MocktransferServiceInterface) {

				arg := transfer.CreateTransferParams{
					FromAccountID: testAccount1.ID,
					ToAccountID:   testAccount2.ID,
					Amount:        amount,
				}

				transferService.EXPECT().
					TransferTx(gomock.Any(), gomock.Eq(testUsername1), gomock.Eq(arg)).
					Times(1).
					Return(transfer.TransferTxResult{}, nil)
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