package service

import (
	"context"
	"time"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/pkg/configpkg"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/go-petr/pet-bank/pkg/token"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

//go:generate mockgen -source service.go -destination service_mock.go -package service
type SessionRepoInterface interface {
	CreateSession(ctx context.Context, arg domain.CreateSessionParams) (domain.Session, error)
	GetSession(ctx context.Context, id uuid.UUID) (domain.Session, error)
}

type SessionService struct {
	repo       SessionRepoInterface
	TokenMaker token.Maker
	config     configpkg.Config
}

func NewSessionService(sr SessionRepoInterface, config configpkg.Config, tm token.Maker) (*SessionService, error) {
	return &SessionService{
		repo:       sr,
		TokenMaker: tm,
		config:     config,
	}, nil
}

func (s *SessionService) Create(ctx context.Context, arg domain.CreateSessionParams) (string, time.Time, domain.Session, error) {

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

	sess, err = s.repo.CreateSession(ctx, arg)
	if err != nil {
		l.Error().Err(err).Send()
		return "", time.Time{}, sess, errorspkg.ErrInternal
	}

	return accessToken, accessPayload.ExpiredAt, sess, nil
}

func (s *SessionService) RenewAccessToken(ctx context.Context, refreshToken string) (string, time.Time, error) {

	l := zerolog.Ctx(ctx)

	refreshPayload, err := s.TokenMaker.VerifyToken(refreshToken)
	if err != nil {
		l.Error().Err(err).Send()
		return "", time.Time{}, errorspkg.ErrInternal
	}

	sess, err := s.repo.GetSession(ctx, refreshPayload.ID)
	if err != nil {
		l.Error().Err(err).Send()
		return "", time.Time{}, domain.ErrSessionNotFound
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
