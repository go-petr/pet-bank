package service

import (
	"context"
	"time"

	"github.com/go-petr/pet-bank/internal/session"
	"github.com/go-petr/pet-bank/internal/session/repo"
	"github.com/go-petr/pet-bank/pkg/token"
	"github.com/go-petr/pet-bank/pkg/util"
)

type SessionService struct {
	repo       *repo.SessionRepo
	TokenMaker token.Maker
	config     util.Config
}

func NewSessionService(sr *repo.SessionRepo, config util.Config) (*SessionService, error) {

	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, err
	}

	return &SessionService{
		repo:       sr,
		TokenMaker: tokenMaker,
		config:     config,
	}, nil
}

func (s *SessionService) Create(ctx context.Context, arg session.CreateSessionParams) (string, time.Time, session.Session, error) {

	var sess session.Session

	accessToken, accessPayload, err := s.TokenMaker.CreateToken(arg.Username, s.config.AccessTokenDuration)
	if err != nil {
		return "", accessPayload.ExpiredAt, sess, err
	}

	refreshToken, refreshPayload, err := s.TokenMaker.CreateToken(arg.Username, s.config.RefreshTokenDuration)
	if err != nil {
		return "", accessPayload.ExpiredAt, sess, err
	}

	arg.ID = refreshPayload.ID
	arg.RefreshToken = refreshToken
	arg.ExpiresAt = refreshPayload.ExpiredAt

	sess, err = s.repo.CreateSession(ctx, arg)
	if err != nil {
		return "", accessPayload.ExpiredAt, sess, err
	}

	return accessToken, accessPayload.ExpiredAt, sess, nil
}
