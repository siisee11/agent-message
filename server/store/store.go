package store

import (
	"context"
	"errors"

	"agent-messenger/server/models"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrForbidden      = errors.New("forbidden")
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
	SearchUsersByUsername(ctx context.Context, params models.SearchUsersParams) ([]models.User, error)
	ListConversationsByUser(ctx context.Context, params models.ListUserConversationsParams) ([]models.ConversationSummary, error)
	GetOrCreateDirectConversation(ctx context.Context, params models.GetOrCreateDirectConversationParams) (models.Conversation, error)
	GetConversationByIDForUser(ctx context.Context, params models.GetConversationForUserParams) (models.ConversationDetails, error)
	ListMessagesByConversation(ctx context.Context, params models.ListConversationMessagesParams) ([]models.MessageDetails, error)
	GetMessageByIDForUser(ctx context.Context, params models.GetMessageForUserParams) (models.Message, error)
	CreateMessage(ctx context.Context, params models.CreateMessageParams) (models.Message, error)
	UpdateMessage(ctx context.Context, params models.UpdateMessageParams) (models.Message, error)
	SoftDeleteMessage(ctx context.Context, params models.SoftDeleteMessageParams) (models.Message, error)
	ToggleMessageReaction(ctx context.Context, params models.ToggleMessageReactionParams) (models.ToggleReactionResult, error)
	RemoveMessageReaction(ctx context.Context, params models.RemoveMessageReactionParams) (models.Reaction, error)
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

func (s *NoopStore) SearchUsersByUsername(_ context.Context, _ models.SearchUsersParams) ([]models.User, error) {
	return nil, ErrNotImplemented
}

func (s *NoopStore) ListConversationsByUser(_ context.Context, _ models.ListUserConversationsParams) ([]models.ConversationSummary, error) {
	return nil, ErrNotImplemented
}

func (s *NoopStore) GetOrCreateDirectConversation(_ context.Context, _ models.GetOrCreateDirectConversationParams) (models.Conversation, error) {
	return models.Conversation{}, ErrNotImplemented
}

func (s *NoopStore) GetConversationByIDForUser(_ context.Context, _ models.GetConversationForUserParams) (models.ConversationDetails, error) {
	return models.ConversationDetails{}, ErrNotImplemented
}

func (s *NoopStore) ListMessagesByConversation(_ context.Context, _ models.ListConversationMessagesParams) ([]models.MessageDetails, error) {
	return nil, ErrNotImplemented
}

func (s *NoopStore) GetMessageByIDForUser(_ context.Context, _ models.GetMessageForUserParams) (models.Message, error) {
	return models.Message{}, ErrNotImplemented
}

func (s *NoopStore) CreateMessage(_ context.Context, _ models.CreateMessageParams) (models.Message, error) {
	return models.Message{}, ErrNotImplemented
}

func (s *NoopStore) UpdateMessage(_ context.Context, _ models.UpdateMessageParams) (models.Message, error) {
	return models.Message{}, ErrNotImplemented
}

func (s *NoopStore) SoftDeleteMessage(_ context.Context, _ models.SoftDeleteMessageParams) (models.Message, error) {
	return models.Message{}, ErrNotImplemented
}

func (s *NoopStore) ToggleMessageReaction(_ context.Context, _ models.ToggleMessageReactionParams) (models.ToggleReactionResult, error) {
	return models.ToggleReactionResult{}, ErrNotImplemented
}

func (s *NoopStore) RemoveMessageReaction(_ context.Context, _ models.RemoveMessageReactionParams) (models.Reaction, error) {
	return models.Reaction{}, ErrNotImplemented
}
