package models

import (
	"errors"
	"strings"
)

const (
	// DefaultMessagePageLimit is the default page size for message history reads.
	DefaultMessagePageLimit = 20
	// MaxMessagePageLimit caps message page size for cursor pagination.
	MaxMessagePageLimit = 100
)

var (
	ErrMessageContentRequired = errors.New("message content is required")
	ErrReactionEmojiRequired  = errors.New("emoji is required")
	ErrPageLimitOutOfRange    = errors.New("limit must be between 1 and 100")
)

// StartConversationRequest is the JSON body for POST /api/conversations.
type StartConversationRequest struct {
	Username string `json:"username"`
}

func (r StartConversationRequest) Validate() error {
	return ValidateUsername(r.Username)
}

// SendMessageRequest is the JSON body for POST /api/conversations/:id/messages text sends.
type SendMessageRequest struct {
	Content string `json:"content"`
}

func (r SendMessageRequest) Validate() error {
	if strings.TrimSpace(r.Content) == "" {
		return ErrMessageContentRequired
	}
	return nil
}

// EditMessageRequest is the JSON body for PATCH /api/messages/:id.
type EditMessageRequest struct {
	Content string `json:"content"`
}

func (r EditMessageRequest) Validate() error {
	if strings.TrimSpace(r.Content) == "" {
		return ErrMessageContentRequired
	}
	return nil
}

// ToggleReactionRequest is the JSON body for POST /api/messages/:id/reactions.
type ToggleReactionRequest struct {
	Emoji string `json:"emoji"`
}

func (r ToggleReactionRequest) Validate() error {
	if strings.TrimSpace(r.Emoji) == "" {
		return ErrReactionEmojiRequired
	}
	return nil
}

// ListMessagesQuery defines cursor pagination inputs for message history.
type ListMessagesQuery struct {
	Before string `json:"before"`
	Limit  int    `json:"limit"`
}

// Normalize sets defaults and validates bounds.
func (q *ListMessagesQuery) Normalize() error {
	q.Before = strings.TrimSpace(q.Before)
	if q.Limit == 0 {
		q.Limit = DefaultMessagePageLimit
	}
	if q.Limit < 1 || q.Limit > MaxMessagePageLimit {
		return ErrPageLimitOutOfRange
	}
	return nil
}

// ConversationSummary is a list projection for GET /api/conversations.
type ConversationSummary struct {
	Conversation Conversation `json:"conversation"`
	OtherUser    UserProfile  `json:"other_user"`
	LastMessage  *Message     `json:"last_message,omitempty"`
}

// ConversationDetails is the detail projection for GET /api/conversations/:id.
type ConversationDetails struct {
	Conversation Conversation `json:"conversation"`
	ParticipantA UserProfile  `json:"participant_a"`
	ParticipantB UserProfile  `json:"participant_b"`
}

// MessageDetails enriches messages with sender profile information.
type MessageDetails struct {
	Message Message     `json:"message"`
	Sender  UserProfile `json:"sender"`
}

// UploadResponse is returned by POST /api/upload.
type UploadResponse struct {
	URL string `json:"url"`
}
