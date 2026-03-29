package models

import "time"

// CreateUserParams is a persistence boundary shape for inserting users.
type CreateUserParams struct {
	ID        string
	Username  string
	PINHash   string
	CreatedAt time.Time
}

// CreateSessionParams is a persistence boundary shape for inserting sessions.
type CreateSessionParams struct {
	Token     string
	UserID    string
	CreatedAt time.Time
}

// SearchUsersParams defines a username-prefix search query.
type SearchUsersParams struct {
	Query string
	Limit int
}

// ListUserConversationsParams defines conversation list retrieval for a user.
type ListUserConversationsParams struct {
	UserID string
	Limit  int
}

// GetOrCreateDirectConversationParams defines DM creation/lookup between two users.
type GetOrCreateDirectConversationParams struct {
	ConversationID string
	CurrentUserID  string
	TargetUserID   string
	CreatedAt      time.Time
}

// GetConversationForUserParams defines participant-scoped conversation lookup.
type GetConversationForUserParams struct {
	ConversationID string
	UserID         string
}

// ListConversationMessagesParams defines participant-scoped message pagination.
type ListConversationMessagesParams struct {
	ConversationID  string
	UserID          string
	BeforeMessageID *string
	Limit           int
}

// GetMessageForUserParams defines participant-scoped message lookup by ID.
type GetMessageForUserParams struct {
	MessageID string
	UserID    string
}

// CreateMessageParams is the persistence boundary for creating a message.
type CreateMessageParams struct {
	ID             string
	ConversationID string
	SenderID       string
	Content        *string
	AttachmentURL  *string
	AttachmentType *AttachmentType
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// UpdateMessageParams is the persistence boundary for editing own message content.
type UpdateMessageParams struct {
	MessageID   string
	ActorUserID string
	Content     string
	UpdatedAt   time.Time
}

// SoftDeleteMessageParams is the persistence boundary for soft-deleting own message.
type SoftDeleteMessageParams struct {
	MessageID   string
	ActorUserID string
	UpdatedAt   time.Time
}

// ToggleMessageReactionParams defines add/toggle behavior for message reactions.
type ToggleMessageReactionParams struct {
	ReactionID  string
	MessageID   string
	ActorUserID string
	Emoji       string
	CreatedAt   time.Time
}

// RemoveMessageReactionParams defines explicit removal of caller-owned reactions.
type RemoveMessageReactionParams struct {
	MessageID   string
	ActorUserID string
	Emoji       string
}
