package store

import (
	"context"
	"errors"

	"agent-messenger/server/models"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrNotImplemented = errors.New("not implemented")
)

type Store interface {
	Close() error
	CreateUser(ctx context.Context, params models.CreateUserParams) (models.User, error)
	GetUserByUsername(ctx context.Context, username string) (models.User, error)
	GetUserByID(ctx context.Context, userID string) (models.User, error)
	CreateSession(ctx context.Context, params models.CreateSessionParams) (models.Session, error)
	GetSessionByToken(ctx context.Context, token string) (models.Session, error)
	DeleteSessionByToken(ctx context.Context, token string) error
	GetUserBySessionToken(ctx context.Context, token string) (models.User, error)
}

type NoopStore struct{}

func NewNoopStore() *NoopStore {
	return &NoopStore{}
}

func (s *NoopStore) Close() error {
	return nil
}

func (s *NoopStore) CreateUser(_ context.Context, _ models.CreateUserParams) (models.User, error) {
	return models.User{}, ErrNotImplemented
}

func (s *NoopStore) GetUserByUsername(_ context.Context, _ string) (models.User, error) {
	return models.User{}, ErrNotImplemented
}

func (s *NoopStore) GetUserByID(_ context.Context, _ string) (models.User, error) {
	return models.User{}, ErrNotImplemented
}

func (s *NoopStore) CreateSession(_ context.Context, _ models.CreateSessionParams) (models.Session, error) {
	return models.Session{}, ErrNotImplemented
}

func (s *NoopStore) GetSessionByToken(_ context.Context, _ string) (models.Session, error) {
	return models.Session{}, ErrNotImplemented
}

func (s *NoopStore) DeleteSessionByToken(_ context.Context, _ string) error {
	return ErrNotImplemented
}

func (s *NoopStore) GetUserBySessionToken(_ context.Context, _ string) (models.User, error) {
	return models.User{}, ErrNotImplemented
}
