package realtime

import (
	"errors"
	"strings"
	"sync"
)

const (
	EventTypeMessageNew      = "message.new"
	EventTypeMessageEdited   = "message.edited"
	EventTypeMessageDeleted  = "message.deleted"
	EventTypeReactionAdded   = "reaction.added"
	EventTypeReactionRemoved = "reaction.removed"
	EventTypePresenceUpdated = "presence.updated"

	ClientKindUnknown = "unknown"
	ClientKindWeb     = "web"
	ClientKindWatcher = "watcher"
)

var (
	ErrClientNil             = errors.New("client is required")
	ErrClientSendChannelNil  = errors.New("client send channel is required")
	ErrClientUserIDRequired  = errors.New("client user id is required")
	ErrClientNotRegistered   = errors.New("client is not registered")
	ErrConversationIDMissing = errors.New("conversation id is required")
	ErrEventTypeMissing      = errors.New("event type is required")
)

type Event struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

type BroadcastResult struct {
	Delivered int
	Dropped   int
}

type Client struct {
	UserID string
	Kind   string
	Send   chan Event
}

type clientState struct {
	userID        string
	kind          string
	conversations map[string]struct{}
}

type Hub struct {
	mu                  sync.RWMutex
	clients             map[*Client]clientState
	userClients         map[string]map[*Client]struct{}
	conversationClients map[string]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{
		clients:             make(map[*Client]clientState),
		userClients:         make(map[string]map[*Client]struct{}),
		conversationClients: make(map[string]map[*Client]struct{}),
	}
}

func (h *Hub) Register(client *Client, conversationIDs []string) error {
	if client == nil {
		return ErrClientNil
	}
	if client.Send == nil {
		return ErrClientSendChannelNil
	}
	userID := strings.TrimSpace(client.UserID)
	if userID == "" {
		return ErrClientUserIDRequired
	}
	clientKind := NormalizeClientKind(client.Kind)

	conversationSet := normalizeConversationSet(conversationIDs)

	h.mu.Lock()
	defer h.mu.Unlock()

	if existing, ok := h.clients[client]; ok {
		h.removeClientLocked(client, existing)
	}

	state := clientState{
		userID:        userID,
		kind:          clientKind,
		conversations: conversationSet,
	}
	h.clients[client] = state
	h.addUserClientLocked(userID, client)
	for conversationID := range conversationSet {
		h.addConversationClientLocked(conversationID, client)
	}

	return nil
}

func (h *Hub) Unregister(client *Client) {
	if client == nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	existing, ok := h.clients[client]
	if !ok {
		return
	}
	h.removeClientLocked(client, existing)
}

func (h *Hub) SubscribeUser(userID, conversationID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ErrClientUserIDRequired
	}
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return ErrConversationIDMissing
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.userClients[userID] {
		state, ok := h.clients[client]
		if !ok {
			continue
		}
		if _, exists := state.conversations[conversationID]; exists {
			continue
		}

		state.conversations[conversationID] = struct{}{}
		h.clients[client] = state
		h.addConversationClientLocked(conversationID, client)
	}

	return nil
}

func (h *Hub) UnsubscribeUser(userID, conversationID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ErrClientUserIDRequired
	}
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return ErrConversationIDMissing
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.userClients[userID] {
		state, ok := h.clients[client]
		if !ok {
			continue
		}
		if _, exists := state.conversations[conversationID]; !exists {
			continue
		}

		delete(state.conversations, conversationID)
		h.clients[client] = state
		h.removeConversationClientLocked(conversationID, client)
	}

	return nil
}

func (h *Hub) BroadcastToConversation(conversationID string, event Event) (BroadcastResult, error) {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return BroadcastResult{}, ErrConversationIDMissing
	}
	if strings.TrimSpace(event.Type) == "" {
		return BroadcastResult{}, ErrEventTypeMissing
	}

	h.mu.RLock()
	recipients := h.copyConversationRecipientsLocked(conversationID)
	h.mu.RUnlock()

	result := BroadcastResult{}
	for _, client := range recipients {
		select {
		case client.Send <- event:
			result.Delivered++
		default:
			result.Dropped++
		}
	}

	return result, nil
}

func (h *Hub) removeClientLocked(client *Client, state clientState) {
	for conversationID := range state.conversations {
		h.removeConversationClientLocked(conversationID, client)
	}
	h.removeUserClientLocked(state.userID, client)
	delete(h.clients, client)
}

func (h *Hub) addUserClientLocked(userID string, client *Client) {
	if _, ok := h.userClients[userID]; !ok {
		h.userClients[userID] = make(map[*Client]struct{})
	}
	h.userClients[userID][client] = struct{}{}
}

func (h *Hub) removeUserClientLocked(userID string, client *Client) {
	userSet, ok := h.userClients[userID]
	if !ok {
		return
	}
	delete(userSet, client)
	if len(userSet) == 0 {
		delete(h.userClients, userID)
	}
}

func (h *Hub) addConversationClientLocked(conversationID string, client *Client) {
	if _, ok := h.conversationClients[conversationID]; !ok {
		h.conversationClients[conversationID] = make(map[*Client]struct{})
	}
	h.conversationClients[conversationID][client] = struct{}{}
}

func (h *Hub) removeConversationClientLocked(conversationID string, client *Client) {
	conversationSet, ok := h.conversationClients[conversationID]
	if !ok {
		return
	}
	delete(conversationSet, client)
	if len(conversationSet) == 0 {
		delete(h.conversationClients, conversationID)
	}
}

func (h *Hub) copyConversationRecipientsLocked(conversationID string) []*Client {
	recipients := h.conversationClients[conversationID]
	out := make([]*Client, 0, len(recipients))
	for client := range recipients {
		out = append(out, client)
	}
	return out
}

func normalizeConversationSet(conversationIDs []string) map[string]struct{} {
	conversationSet := make(map[string]struct{}, len(conversationIDs))
	for _, rawConversationID := range conversationIDs {
		conversationID := strings.TrimSpace(rawConversationID)
		if conversationID == "" {
			continue
		}
		conversationSet[conversationID] = struct{}{}
	}
	return conversationSet
}

func NormalizeClientKind(clientKind string) string {
	switch strings.TrimSpace(clientKind) {
	case ClientKindWeb:
		return ClientKindWeb
	case ClientKindWatcher:
		return ClientKindWatcher
	default:
		return ClientKindUnknown
	}
}
