package models

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"
)

const (
	// DefaultMessagePageLimit is the default page size for message history reads.
	DefaultMessagePageLimit = 20
	// MaxMessagePageLimit caps message page size for cursor pagination.
	MaxMessagePageLimit = 100
)

var (
	ErrMessageContentRequired        = errors.New("message content is required")
	ErrMessageKindInvalid            = errors.New("message kind must be text or json_render")
	ErrMessageJSONRenderSpecRequired = errors.New("json_render_spec object is required for json_render messages")
	ErrReactionEmojiRequired         = errors.New("emoji is required")
	ErrPageLimitOutOfRange           = errors.New("limit must be between 1 and 100")
	ErrConversationTitleTooLong      = errors.New("title must be at most 120 characters")
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
	Content        *string         `json:"content,omitempty"`
	Kind           MessageKind     `json:"kind,omitempty"`
	JSONRenderSpec json.RawMessage `json:"json_render_spec,omitempty"`
}

func (r SendMessageRequest) Validate() error {
	kind := r.Kind
	if kind == "" {
		kind = MessageKindText
	}

	switch kind {
	case MessageKindText:
		if r.Content == nil || strings.TrimSpace(*r.Content) == "" {
			return ErrMessageContentRequired
		}
		return nil
	case MessageKindJSONRender:
		if !isJSONObject(r.JSONRenderSpec) {
			return ErrMessageJSONRenderSpecRequired
		}
		return nil
	default:
		return ErrMessageKindInvalid
	}
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
	Conversation    Conversation `json:"conversation"`
	OtherUser       UserProfile  `json:"other_user"`
	LastMessage     *Message     `json:"last_message,omitempty"`
	SessionFolder   string       `json:"session_folder,omitempty"`
	SessionHostname string       `json:"session_hostname,omitempty"`
}

// ConversationDetails is the detail projection for GET /api/conversations/:id.
type ConversationDetails struct {
	Conversation    Conversation     `json:"conversation"`
	ParticipantA    UserProfile      `json:"participant_a"`
	ParticipantB    UserProfile      `json:"participant_b"`
	WatcherPresence *WatcherPresence `json:"watcher_presence,omitempty"`
}

// UpdateConversationRequest is the JSON body for PATCH /api/conversations/:id.
type UpdateConversationRequest struct {
	Title string `json:"title"`
}

func (r UpdateConversationRequest) Validate() error {
	title := strings.TrimSpace(r.Title)
	if title == "" {
		return nil
	}
	if utf8.RuneCountInString(title) > 120 {
		return ErrConversationTitleTooLong
	}
	return nil
}

// WatcherPresence reports whether the other participant currently has an active watcher connection.
type WatcherPresence struct {
	UserID     string `json:"user_id"`
	ClientKind string `json:"client_kind"`
	Online     bool   `json:"online"`
}

// WatcherPresenceEvent is broadcast to conversation subscribers when watcher presence changes.
type WatcherPresenceEvent struct {
	ConversationID string `json:"conversation_id"`
	UserID         string `json:"user_id"`
	ClientKind     string `json:"client_kind"`
	Online         bool   `json:"online"`
}

// MessageDetails enriches messages with sender profile information.
type MessageDetails struct {
	Message   Message     `json:"message"`
	Sender    UserProfile `json:"sender"`
	Reactions []Reaction  `json:"reactions"`
}

// UploadResponse is returned by POST /api/upload.
type UploadResponse struct {
	URL string `json:"url"`
}

func isJSONObject(value json.RawMessage) bool {
	trimmed := bytes.TrimSpace(value)
	if len(trimmed) == 0 {
		return false
	}
	if trimmed[0] != '{' {
		return false
	}

	var decoded map[string]any
	if err := json.Unmarshal(trimmed, &decoded); err != nil {
		return false
	}
	return true
}
