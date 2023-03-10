// Package sessionservice manages business logic layer of sessions.
package sessionservice

import (
	"context"
	"time"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/pkg/configpkg"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/go-petr/pet-bank/pkg/tokenpkg"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Repo provides data access layer interface needed by session service layer.
//
//go:generate mockgen -source service.go -destination service_mock.go -package sessionservice
type Repo interface {
	Create(ctx context.Context, arg domain.CreateSessionParams) (domain.Session, error)
	Get(ctx context.Context, id uuid.UUID) (domain.Session, error)
}

// Service facilitates session service layer logic.
type Service struct {
	repo       Repo
	TokenMaker tokenpkg.Maker
	config     configpkg.Config
}

// New returns session service struct to manage session bussines logic.
func New(sr Repo, config configpkg.Config, tm tokenpkg.Maker) (*Service, error) {
	return &Service{
		repo:       sr,
		TokenMaker: tm,
		config:     config,
	}, nil
}

// Create session and access token.
func (s *Service) Create(ctx context.Context, arg domain.CreateSessionParams) (string, time.Time, domain.Session, error) {
	l := zerolog.Ctx(ctx)

	var sess domain.Session

	accessToken, accessPayload, err := s.TokenMaker.CreateToken(arg.Username, s.config.AccessTokenDuration)
	if err != nil {
		l.Error().Err(err).Send()
		return "", time.Time{}, sess, errorspkg.ErrInternal
	}

	refreshToken, refreshPayload, err := s.TokenMaker.CreateToken(arg.Username, s.config.RefreshTokenDuration)
	if err != nil {
		l.Error().Err(err).Send()
		return "", time.Time{}, sess, errorspkg.ErrInternal
	}

	arg.ID = refreshPayload.ID
	arg.RefreshToken = refreshToken
	arg.ExpiresAt = refreshPayload.ExpiredAt

	sess, err = s.repo.Create(ctx, arg)
	if err != nil {
		l.Error().Err(err).Send()
		return "", time.Time{}, sess, errorspkg.ErrInternal
	}

	return accessToken, accessPayload.ExpiredAt, sess, nil
}

// RenewAccessToken verifies refresh token, renews access token and returns it.
func (s *Service) RenewAccessToken(ctx context.Context, refreshToken string) (string, time.Time, error) {
	l := zerolog.Ctx(ctx)

	refreshPayload, err := s.TokenMaker.VerifyToken(refreshToken)
	if err != nil {
		if err == tokenpkg.ErrExpiredToken || err == tokenpkg.ErrInvalidToken {
			return "", time.Time{}, err
		}

		l.Error().Err(err).Send()

		return "", time.Time{}, errorspkg.ErrInternal
	}

	sess, err := s.repo.Get(ctx, refreshPayload.ID)
	if err != nil {
		return "", time.Time{}, err
	}

	if sess.IsBlocked {
		l.Info().Err(err).Send()
		return "", time.Time{}, domain.ErrBlockedSession
	}

	if sess.Username != refreshPayload.Username {
		l.Info().Err(err).Send()
		return "", time.Time{}, domain.ErrInvalidUser
	}

	if sess.RefreshToken != refreshToken {
		l.Info().Err(err).Send()
		return "", time.Time{}, domain.ErrMismatchedRefreshToken
	}

	if time.Now().After(sess.ExpiresAt) {
		l.Info().Err(domain.ErrExpiredSession).Send()
		return "", time.Time{}, domain.ErrExpiredSession
	}

	accessToken, accessPayload, err := s.TokenMaker.CreateToken(
		refreshPayload.Username,
		s.config.AccessTokenDuration,
	)
	if err != nil {
		l.Error().Err(err).Send()
		return "", time.Time{}, errorspkg.ErrInternal
	}

	return accessToken, accessPayload.ExpiredAt, nil
}
