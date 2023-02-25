package transferservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-petr/pet-bank/internal/accountdelivery"
	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/pkg/currencypkg"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/go-petr/pet-bank/pkg/randompkg"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func randomAccount(id int32, balance, currency string) domain.Account {
	return domain.Account{
		ID:        id,
		Owner:     randompkg.Owner(),
		Balance:   balance,
		Currency:  currency,
		CreatedAt: time.Now().Truncate(time.Second).UTC(),
	}
}

func TestTransfer(t *testing.T) {
	testAccount1 := randomAccount(1, "1000", currencypkg.USD)
	testAccount2 := randomAccount(2, "1000", currencypkg.USD)
	testAccount3 := randomAccount(1, "1000", currencypkg.EUR)
	testAmount := "100"

	testTxResult := domain.TransferTxResult{
		Transfer: domain.Transfer{
			FromAccountID: testAccount1.ID,
			ToAccountID:   testAccount2.ID,
			Amount:        testAmount,
		},
		FromAccount: testAccount1,
		ToAccount:   testAccount2,
		FromEntry: domain.Entry{
			AccountID: testAccount1.ID,
			Amount:    "-" + testAmount,
		},
		ToEntry: domain.Entry{
			AccountID: testAccount2.ID,
			Amount:    testAmount,
		},
	}

	type input struct {
		fromUsername string
		arg          domain.CreateTransferParams
	}

	testCases := []struct {
		name          string
		input         input
		buildStubs    func(repo *MockRepo, accountService *accountdelivery.MockService)
		checkResponse func(res domain.TransferTxResult, err error)
	}{
		{
			name: "Invalid amount",
			input: input{
				fromUsername: testAccount1.Owner,
				arg: domain.CreateTransferParams{
					FromAccountID: testAccount1.ID,
					ToAccountID:   testAccount2.ID,
					Amount:        "!@#$",
				},
			},
			buildStubs: func(repo *MockRepo, accountService *accountdelivery.MockService) {
				repo.EXPECT().Transfer(gomock.Any(), gomock.Any()).Times(0)
				accountService.EXPECT().Get(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(res domain.TransferTxResult, err error) {
				require.Empty(t, res)
				require.EqualError(t, err, domain.ErrInvalidAmount.Error())
			},
		},
		{
			name: "Negative amount",
			input: input{
				fromUsername: testAccount1.Owner,
				arg: domain.CreateTransferParams{
					FromAccountID: testAccount1.ID,
					ToAccountID:   testAccount2.ID,
					Amount:        "-100",
				},
			},
			buildStubs: func(repo *MockRepo, accountService *accountdelivery.MockService) {
				repo.EXPECT().Transfer(gomock.Any(), gomock.Any()).Times(0)
				accountService.EXPECT().Get(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(res domain.TransferTxResult, err error) {
				require.Empty(t, res)
				require.EqualError(t, err, domain.ErrNegativeAmount.Error())
			},
		},
		{
			name: "Account service err",
			input: input{
				fromUsername: testAccount1.Owner,
				arg: domain.CreateTransferParams{
					FromAccountID: testAccount1.ID,
					ToAccountID:   testAccount2.ID,
					Amount:        testAmount,
				},
			},
			buildStubs: func(repo *MockRepo, accountService *accountdelivery.MockService) {
				repo.EXPECT().Transfer(gomock.Any(), gomock.Any()).Times(0)
				accountService.EXPECT().Get(gomock.Any(), gomock.Eq(testAccount1.ID)).
					Times(1).
					Return(domain.Account{}, errorspkg.ErrInternal)
			},
			checkResponse: func(res domain.TransferTxResult, err error) {
				require.Empty(t, res)
				require.EqualError(t, err, errorspkg.ErrInternal.Error())
			},
		},
		{
			name: "Invalid owner",
			input: input{
				fromUsername: testAccount1.Owner,
				arg: domain.CreateTransferParams{
					FromAccountID: testAccount2.ID,
					ToAccountID:   testAccount1.ID,
					Amount:        testAmount,
				},
			},
			buildStubs: func(repo *MockRepo, accountService *accountdelivery.MockService) {
				repo.EXPECT().Transfer(gomock.Any(), gomock.Any()).Times(0)
				accountService.EXPECT().Get(gomock.Any(), gomock.Eq(testAccount2.ID)).
					Times(1).
					Return(domain.Account{
						Owner: testAccount2.Owner,
					}, nil)
			},
			checkResponse: func(res domain.TransferTxResult, err error) {
				require.Empty(t, res)
				require.EqualError(t, err, domain.ErrInvalidOwner.Error())
			},
		},
		{
			name: "From account internal balance error",
			input: input{
				fromUsername: testAccount1.Owner,
				arg: domain.CreateTransferParams{
					FromAccountID: testAccount1.ID,
					ToAccountID:   testAccount2.ID,
					Amount:        testAmount,
				},
			},
			buildStubs: func(repo *MockRepo, accountService *accountdelivery.MockService) {
				repo.EXPECT().Transfer(gomock.Any(), gomock.Any()).Times(0)
				accountService.EXPECT().Get(gomock.Any(), gomock.Eq(testAccount1.ID)).
					Times(1).
					Return(domain.Account{
						Owner:   testAccount1.Owner,
						Balance: "invalid",
					}, nil)
			},
			checkResponse: func(res domain.TransferTxResult, err error) {
				require.Empty(t, res)
				require.EqualError(t, err, errors.New("can't convert invalid to decimal").Error())
			},
		},
		{
			name: "Insufficient balance",
			input: input{
				fromUsername: testAccount1.Owner,
				arg: domain.CreateTransferParams{
					FromAccountID: testAccount1.ID,
					ToAccountID:   testAccount2.ID,
					Amount:        "10000",
				},
			},
			buildStubs: func(repo *MockRepo, accountService *accountdelivery.MockService) {
				repo.EXPECT().Transfer(gomock.Any(), gomock.Any()).Times(0)
				accountService.EXPECT().Get(gomock.Any(), gomock.Eq(testAccount1.ID)).
					Times(1).
					Return(domain.Account{
						ID:       testAccount1.ID,
						Owner:    testAccount1.Owner,
						Balance:  testAccount1.Balance,
						Currency: testAccount1.Currency,
					}, nil)
			},
			checkResponse: func(res domain.TransferTxResult, err error) {
				require.Empty(t, res)
				require.EqualError(t, err, domain.ErrInsufficientBalance.Error())
			},
		},
		{
			name: "ToAccount service err",
			input: input{
				fromUsername: testAccount1.Owner,
				arg: domain.CreateTransferParams{
					FromAccountID: testAccount1.ID,
					ToAccountID:   testAccount2.ID,
					Amount:        testAmount,
				},
			},
			buildStubs: func(repo *MockRepo, accountService *accountdelivery.MockService) {
				repo.EXPECT().Transfer(gomock.Any(), gomock.Any()).Times(0)
				accountService.EXPECT().Get(gomock.Any(), gomock.Eq(testAccount1.ID)).
					Times(1).
					Return(domain.Account{
						ID:       testAccount1.ID,
						Owner:    testAccount1.Owner,
						Balance:  testAccount1.Balance,
						Currency: testAccount1.Currency,
					}, nil)
				accountService.EXPECT().Get(gomock.Any(), gomock.Eq(testAccount2.ID)).
					Times(1).
					Return(domain.Account{}, errorspkg.ErrInternal)
			},
			checkResponse: func(res domain.TransferTxResult, err error) {
				require.Empty(t, res)
				require.EqualError(t, err, errorspkg.ErrInternal.Error())
			},
		},
		{
			name: "Accounts currency mismatch",
			input: input{
				fromUsername: testAccount1.Owner,
				arg: domain.CreateTransferParams{
					FromAccountID: testAccount1.ID,
					ToAccountID:   testAccount3.ID,
					Amount:        testAmount,
				},
			},
			buildStubs: func(repo *MockRepo, accountService *accountdelivery.MockService) {
				repo.EXPECT().Transfer(gomock.Any(), gomock.Any()).Times(0)
				accountService.EXPECT().Get(gomock.Any(), gomock.Eq(testAccount1.ID)).
					Times(1).
					Return(domain.Account{
						ID:       testAccount1.ID,
						Owner:    testAccount1.Owner,
						Balance:  testAccount1.Balance,
						Currency: testAccount1.Currency,
					}, nil)
				accountService.EXPECT().Get(gomock.Any(), gomock.Eq(testAccount3.ID)).
					Times(1).
					Return(domain.Account{
						Currency: testAccount3.Currency,
					}, nil)
			},
			checkResponse: func(res domain.TransferTxResult, err error) {
				require.Empty(t, res)
				require.EqualError(t, err, domain.ErrCurrencyMismatch.Error())
			},
		},
		{
			name: "OK",
			input: input{
				fromUsername: testAccount1.Owner,
				arg: domain.CreateTransferParams{
					FromAccountID: testAccount1.ID,
					ToAccountID:   testAccount2.ID,
					Amount:        testAmount,
				},
			},
			buildStubs: func(repo *MockRepo, accountService *accountdelivery.MockService) {
				repo.EXPECT().Transfer(gomock.Any(), gomock.Any()).
					Times(1).
					Return(testTxResult, nil)
				accountService.EXPECT().Get(gomock.Any(), gomock.Eq(testAccount1.ID)).
					Times(1).
					Return(domain.Account{
						ID:       testAccount1.ID,
						Owner:    testAccount1.Owner,
						Balance:  testAccount1.Balance,
						Currency: testAccount1.Currency,
					}, nil)
				accountService.EXPECT().Get(gomock.Any(), gomock.Eq(testAccount2.ID)).
					Times(1).
					Return(domain.Account{
						Currency: testAccount2.Currency,
					}, nil)
			},
			checkResponse: func(res domain.TransferTxResult, err error) {
				require.Equal(t, testTxResult, res)
				require.NoError(t, err)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			tranferRepo := NewMockRepo(ctrl)
			accountService := accountdelivery.NewMockService(ctrl)
			transferService := New(tranferRepo, accountService)

			tc.buildStubs(tranferRepo, accountService)

			tc.checkResponse(transferService.Transfer(
				context.Background(),
				tc.input.fromUsername,
				tc.input.arg))
		})
	}
}
