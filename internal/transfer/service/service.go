package service

import (
	"context"

	"github.com/go-petr/pet-bank/internal/account/delivery"
	"github.com/go-petr/pet-bank/internal/transfer"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
)

//go:generate mockgen -source service.go -destination service_mock.go -package service
type transferRepoInterface interface {
	TransferTx(ctx context.Context, arg transfer.CreateTransferParams) (transfer.TransferTxResult, error)
}

type transferService struct {
	transferRepo   transferRepoInterface
	accountService delivery.AccountServiceInterface
}

func NewTransferService(tr transferRepoInterface, as delivery.AccountServiceInterface) *transferService {
	return &transferService{
		transferRepo:   tr,
		accountService: as,
	}
}

func (s *transferService) validTransferRequest(ctx context.Context, fromUsername string, fromAccountID, toAccountID int32, amount string) error {

	l := zerolog.Ctx(ctx)

	amountDecimal, err := decimal.NewFromString(amount)
	if err != nil {
		l.Info().Err(err).Send()
		return transfer.ErrInvalidAmount
	}

	if amountDecimal.LessThanOrEqual(decimal.Zero) {
		l.Info().Err(err).Send()
		return transfer.ErrNegativeAmount
	}

	FromAccount, err := s.accountService.GetAccount(ctx, fromAccountID)
	if err != nil {
		l.Error().Err(err).Send()
		return err
	}

	if FromAccount.Owner != fromUsername {
		l.Info().Err(err).Send()
		return transfer.ErrInvalidOwner
	}

	currentFromAccountBalance, err := decimal.NewFromString(FromAccount.Balance)
	if err != nil {
		l.Error().Err(err).Send()
		return err
	}

	if currentFromAccountBalance.LessThan(amountDecimal) {
		return transfer.ErrInsufficientBalance
	}

	ToAccount, err := s.accountService.GetAccount(ctx, toAccountID)
	if err != nil {
		l.Info().Err(err).Send()
		return err
	}

	if FromAccount.Currency != ToAccount.Currency {
		return transfer.ErrCurrencyMismatch
	}

	return nil
}

func (s transferService) TransferTx(ctx context.Context, fromUsername string, arg transfer.CreateTransferParams) (transfer.TransferTxResult, error) {

	if err := s.validTransferRequest(ctx, fromUsername, arg.FromAccountID, arg.ToAccountID, arg.Amount); err != nil {
		return transfer.TransferTxResult{}, err
	}

	return s.transferRepo.TransferTx(ctx, arg)
}
