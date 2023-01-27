package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-petr/pet-bank/internal/account"
	"github.com/go-petr/pet-bank/internal/account/delivery"
	"github.com/go-petr/pet-bank/internal/entry"
	"github.com/go-petr/pet-bank/internal/transfer"
	"github.com/go-petr/pet-bank/pkg/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func randomAccount(id int32, balance, currency string) account.Account {
	return account.Account{
		ID:        id,
		Owner:     util.RandomOwner(),
		Balance:   balance,
		Currency:  currency,
		CreatedAt: time.Now().Truncate(time.Second).UTC(),
	}
}

func TestTransferTx(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tranferRepo := NewMocktransferRepoInterface(ctrl)
	accountService := delivery.NewMockAccountServiceInterface(ctrl)
	transferService := NewTransferService(tranferRepo, accountService)

	testAccount1 := randomAccount(1, "1000", util.USD)
	testAccount2 := randomAccount(2, "1000", util.USD)
	testAccount3 := randomAccount(1, "1000", util.EUR)
	testAmount := "100"

	testTxResult := transfer.TransferTxResult{
		Transfer: transfer.Transfer{
			FromAccountID: testAccount1.ID,
			ToAccountID:   testAccount2.ID,
			Amount:        testAmount,
		},
		FromAccount: testAccount1,
		ToAccount:   testAccount2,
		FromEntry: entry.Entry{
			AccountID: testAccount1.ID,
			Amount:    "-" + testAmount,
		},
		ToEntry: entry.Entry{
			AccountID: testAccount2.ID,
			Amount:    testAmount,
		},
	}

	testCases := []struct {
		name  string
		input struct {
			fromUsername string
			arg          transfer.CreateTransferParams
		}
		buildStubs    func(repo *MocktransferRepoInterface, accountService *delivery.MockAccountServiceInterface)
		checkResponse func(res transfer.TransferTxResult, err error)
	}{
		{
			name: "Invalid amount",
			input: struct {
				fromUsername string
				arg          transfer.CreateTransferParams
			}{
				fromUsername: testAccount1.Owner,
				arg: transfer.CreateTransferParams{
					FromAccountID: testAccount1.ID,
					ToAccountID:   testAccount2.ID,
					Amount:        "!@#$",
				},
			},
			buildStubs: func(repo *MocktransferRepoInterface, accountService *delivery.MockAccountServiceInterface) {

				repo.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
				accountService.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(res transfer.TransferTxResult, err error) {
				require.Empty(t, res)
				require.EqualError(t, err, transfer.ErrInvalidAmount.Error())
			},
		},
		{
			name: "Negative amount",
			input: struct {
				fromUsername string
				arg          transfer.CreateTransferParams
			}{
				fromUsername: testAccount1.Owner,
				arg: transfer.CreateTransferParams{
					FromAccountID: testAccount1.ID,
					ToAccountID:   testAccount2.ID,
					Amount:        "-100",
				},
			},
			buildStubs: func(repo *MocktransferRepoInterface, accountService *delivery.MockAccountServiceInterface) {

				repo.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
				accountService.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(res transfer.TransferTxResult, err error) {
				require.Empty(t, res)
				require.EqualError(t, err, transfer.ErrNegativeAmount.Error())
			},
		},
		{
			name: "Account service err",
			input: struct {
				fromUsername string
				arg          transfer.CreateTransferParams
			}{
				fromUsername: testAccount1.Owner,
				arg: transfer.CreateTransferParams{
					FromAccountID: testAccount1.ID,
					ToAccountID:   testAccount2.ID,
					Amount:        testAmount,
				},
			},
			buildStubs: func(repo *MocktransferRepoInterface, accountService *delivery.MockAccountServiceInterface) {

				repo.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
				accountService.EXPECT().GetAccount(gomock.Any(), gomock.Eq(testAccount1.ID)).
					Times(1).
					Return(account.Account{}, util.ErrInternal)
			},
			checkResponse: func(res transfer.TransferTxResult, err error) {
				require.Empty(t, res)
				require.EqualError(t, err, util.ErrInternal.Error())
			},
		},
		{
			name: "Invalid owner",
			input: struct {
				fromUsername string
				arg          transfer.CreateTransferParams
			}{
				fromUsername: testAccount1.Owner,
				arg: transfer.CreateTransferParams{
					FromAccountID: testAccount2.ID,
					ToAccountID:   testAccount1.ID,
					Amount:        testAmount,
				},
			},
			buildStubs: func(repo *MocktransferRepoInterface, accountService *delivery.MockAccountServiceInterface) {

				repo.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
				accountService.EXPECT().GetAccount(gomock.Any(), gomock.Eq(testAccount2.ID)).
					Times(1).
					Return(account.Account{
						Owner: testAccount2.Owner,
					}, nil)
			},
			checkResponse: func(res transfer.TransferTxResult, err error) {
				require.Empty(t, res)
				require.EqualError(t, err, transfer.ErrInvalidOwner.Error())
			},
		},
		{
			name: "From account internal balance error",
			input: struct {
				fromUsername string
				arg          transfer.CreateTransferParams
			}{
				fromUsername: testAccount1.Owner,
				arg: transfer.CreateTransferParams{
					FromAccountID: testAccount1.ID,
					ToAccountID:   testAccount2.ID,
					Amount:        testAmount,
				},
			},
			buildStubs: func(repo *MocktransferRepoInterface, accountService *delivery.MockAccountServiceInterface) {

				repo.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
				accountService.EXPECT().GetAccount(gomock.Any(), gomock.Eq(testAccount1.ID)).
					Times(1).
					Return(account.Account{
						Owner:   testAccount1.Owner,
						Balance: "invalid",
					}, nil)
			},
			checkResponse: func(res transfer.TransferTxResult, err error) {
				require.Empty(t, res)
				require.EqualError(t, err, errors.New("can't convert invalid to decimal").Error())
			},
		},
		{
			name: "Insufficient balance",
			input: struct {
				fromUsername string
				arg          transfer.CreateTransferParams
			}{
				fromUsername: testAccount1.Owner,
				arg: transfer.CreateTransferParams{
					FromAccountID: testAccount1.ID,
					ToAccountID:   testAccount2.ID,
					Amount:        "10000",
				},
			},
			buildStubs: func(repo *MocktransferRepoInterface, accountService *delivery.MockAccountServiceInterface) {

				repo.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
				accountService.EXPECT().GetAccount(gomock.Any(), gomock.Eq(testAccount1.ID)).
					Times(1).
					Return(account.Account{
						ID:       testAccount1.ID,
						Owner:    testAccount1.Owner,
						Balance:  testAccount1.Balance,
						Currency: testAccount1.Currency,
					}, nil)
			},
			checkResponse: func(res transfer.TransferTxResult, err error) {
				require.Empty(t, res)
				require.EqualError(t, err, transfer.ErrInsufficientBalance.Error())
			},
		},
		{
			name: "ToAccount service err",
			input: struct {
				fromUsername string
				arg          transfer.CreateTransferParams
			}{
				fromUsername: testAccount1.Owner,
				arg: transfer.CreateTransferParams{
					FromAccountID: testAccount1.ID,
					ToAccountID:   testAccount2.ID,
					Amount:        testAmount,
				},
			},
			buildStubs: func(repo *MocktransferRepoInterface, accountService *delivery.MockAccountServiceInterface) {

				repo.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
				accountService.EXPECT().GetAccount(gomock.Any(), gomock.Eq(testAccount1.ID)).
					Times(1).
					Return(account.Account{
						ID:       testAccount1.ID,
						Owner:    testAccount1.Owner,
						Balance:  testAccount1.Balance,
						Currency: testAccount1.Currency,
					}, nil)
				accountService.EXPECT().GetAccount(gomock.Any(), gomock.Eq(testAccount2.ID)).
					Times(1).
					Return(account.Account{}, util.ErrInternal)
			},
			checkResponse: func(res transfer.TransferTxResult, err error) {
				require.Empty(t, res)
				require.EqualError(t, err, util.ErrInternal.Error())
			},
		},
		{
			name: "Accounts currency mismatch",
			input: struct {
				fromUsername string
				arg          transfer.CreateTransferParams
			}{
				fromUsername: testAccount1.Owner,
				arg: transfer.CreateTransferParams{
					FromAccountID: testAccount1.ID,
					ToAccountID:   testAccount3.ID,
					Amount:        testAmount,
				},
			},
			buildStubs: func(repo *MocktransferRepoInterface, accountService *delivery.MockAccountServiceInterface) {

				repo.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
				accountService.EXPECT().GetAccount(gomock.Any(), gomock.Eq(testAccount1.ID)).
					Times(1).
					Return(account.Account{
						ID:       testAccount1.ID,
						Owner:    testAccount1.Owner,
						Balance:  testAccount1.Balance,
						Currency: testAccount1.Currency,
					}, nil)
				accountService.EXPECT().GetAccount(gomock.Any(), gomock.Eq(testAccount3.ID)).
					Times(1).
					Return(account.Account{
						Currency: testAccount3.Currency,
					}, nil)
			},
			checkResponse: func(res transfer.TransferTxResult, err error) {
				require.Empty(t, res)
				require.EqualError(t, err, transfer.ErrCurrencyMismatch.Error())
			},
		},
		{
			name: "OK",
			input: struct {
				fromUsername string
				arg          transfer.CreateTransferParams
			}{
				fromUsername: testAccount1.Owner,
				arg: transfer.CreateTransferParams{
					FromAccountID: testAccount1.ID,
					ToAccountID:   testAccount2.ID,
					Amount:        testAmount,
				},
			},
			buildStubs: func(repo *MocktransferRepoInterface, accountService *delivery.MockAccountServiceInterface) {

				repo.EXPECT().TransferTx(gomock.Any(), gomock.Any()).
					Times(1).
					Return(testTxResult, nil)
				accountService.EXPECT().GetAccount(gomock.Any(), gomock.Eq(testAccount1.ID)).
					Times(1).
					Return(account.Account{
						ID:       testAccount1.ID,
						Owner:    testAccount1.Owner,
						Balance:  testAccount1.Balance,
						Currency: testAccount1.Currency,
					}, nil)
				accountService.EXPECT().GetAccount(gomock.Any(), gomock.Eq(testAccount2.ID)).
					Times(1).
					Return(account.Account{
						Currency: testAccount2.Currency,
					}, nil)
			},
			checkResponse: func(res transfer.TransferTxResult, err error) {

				require.Equal(t, testTxResult, res)
				require.NoError(t, err)
			},
		},
	}

	for i := range testCases {

		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {

			tc.buildStubs(tranferRepo, accountService)

			tc.checkResponse(transferService.TransferTx(
				context.Background(),
				tc.input.fromUsername,
				tc.input.arg))

		})
	}
}
