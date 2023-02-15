// Package transferservice manages business logic layer of transfers.
package transferservice

import (
	"context"

	"github.com/go-petr/pet-bank/internal/accountdelivery"
	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
)

// Repo provides data access layer interface needed by transfer service layer.
//
//go:generate mockgen -source service.go -destination service_mock.go -package transferservice
type Repo interface {
	Transfer(ctx context.Context, arg domain.CreateTransferParams) (domain.TransferTxResult, error)
}

// Service facilitates transfer service layer logic.
type Service struct {
	repo           Repo
	accountService accountdelivery.Service
}

// New return transfer service struct to manage transfer bussines logic.
func New(tr Repo, as accountdelivery.Service) *Service {
	return &Service{
		repo:           tr,
		accountService: as,
	}
}

func (s *Service) validRequest(ctx context.Context, fromUsername string, fromAccountID, toAccountID int32, amount string) error {
	l := zerolog.Ctx(ctx)

	amountDecimal, err := decimal.NewFromString(amount)
	if err != nil {
		l.Info().Err(err).Send()
		return domain.ErrInvalidAmount
	}

	if amountDecimal.LessThanOrEqual(decimal.Zero) {
		l.Info().Err(err).Send()
		return domain.ErrNegativeAmount
	}

	fromAccount, err := s.accountService.Get(ctx, fromAccountID)
	if err != nil {
		l.Error().Err(err).Send()
		return err
	}

	if fromAccount.Owner != fromUsername {
		l.Info().Err(err).Send()
		return domain.ErrInvalidOwner
	}

	currentFromAccountBalance, err := decimal.NewFromString(fromAccount.Balance)
	if err != nil {
		l.Error().Err(err).Send()
		return err
	}

	if currentFromAccountBalance.LessThan(amountDecimal) {
		return domain.ErrInsufficientBalance
	}

	toAccount, err := s.accountService.Get(ctx, toAccountID)
	if err != nil {
		l.Info().Err(err).Send()
		return err
	}

	if fromAccount.Currency != toAccount.Currency {
		return domain.ErrCurrencyMismatch
	}

	return nil
}

// Transfer checks if transfer request is valid and then executes transfer.
func (s Service) Transfer(ctx context.Context, fromUsername string, arg domain.CreateTransferParams) (domain.TransferTxResult, error) {
	if err := s.validRequest(ctx, fromUsername, arg.FromAccountID, arg.ToAccountID, arg.Amount); err != nil {
		return domain.TransferTxResult{}, err
	}

	result, err := s.repo.Transfer(ctx, arg)
	if err != nil {
		return result, err
	}

	return result, nil
}
