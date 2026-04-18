package store

import (
	"context"
	"errors"

	"agent-message/server/models"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrForbidden      = errors.New("forbidden")
	ErrNotImplemented = errors.New("not implemented")
)

type Store interface {
	Close() error
	CreateUser(ctx context.Context, params models.CreateUserParams) (models.User, error)
	GetUserByAccountID(ctx context.Context, accountID string) (models.User, error)
	GetUserByUsername(ctx context.Context, username string) (models.User, error)
	GetUserByID(ctx context.Context, userID string) (models.User, error)
	UpdateUsername(ctx context.Context, params models.UpdateUsernameParams) (models.User, error)
	UpdatePasswordHash(ctx context.Context, params models.UpdatePasswordHashParams) (models.User, error)
	CreateSession(ctx context.Context, params models.CreateSessionParams) (models.Session, error)
	GetSessionByToken(ctx context.Context, token string) (models.Session, error)
	DeleteSessionByToken(ctx context.Context, token string) error
	GetUserBySessionToken(ctx context.Context, token string) (models.User, error)
	UpsertPushSubscription(ctx context.Context, params models.UpsertPushSubscriptionParams) (models.PushSubscription, error)
	DeletePushSubscriptionByEndpoint(ctx context.Context, endpoint string) error
	DeletePushSubscriptionByEndpointForUser(ctx context.Context, userID, endpoint string) error
	ListPushSubscriptionsByUser(ctx context.Context, userID string) ([]models.PushSubscription, error)
	SearchUsersByUsername(ctx context.Context, params models.SearchUsersParams) ([]models.User, error)
	ListConversationsByUser(ctx context.Context, params models.ListUserConversationsParams) ([]models.ConversationSummary, error)
	GetOrCreateDirectConversation(ctx context.Context, params models.GetOrCreateDirectConversationParams) (models.Conversation, error)
	GetConversationByIDForUser(ctx context.Context, params models.GetConversationForUserParams) (models.ConversationDetails, error)
	UpdateConversationTitle(ctx context.Context, params models.UpdateConversationTitleParams) (models.Conversation, error)
	DeleteConversationForUser(ctx context.Context, params models.DeleteConversationForUserParams) error
	ListMessagesByConversation(ctx context.Context, params models.ListConversationMessagesParams) ([]models.MessageDetails, error)
	GetMessageByIDForUser(ctx context.Context, params models.GetMessageForUserParams) (models.Message, error)
	CreateMessage(ctx context.Context, params models.CreateMessageParams) (models.Message, error)
	UpdateMessage(ctx context.Context, params models.UpdateMessageParams) (models.Message, error)
	SoftDeleteMessage(ctx context.Context, params models.SoftDeleteMessageParams) (models.Message, error)
	ToggleMessageReaction(ctx context.Context, params models.ToggleMessageReactionParams) (models.ToggleReactionResult, error)
	RemoveMessageReaction(ctx context.Context, params models.RemoveMessageReactionParams) (models.Reaction, error)
}
