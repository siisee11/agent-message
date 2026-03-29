package ws

import (
	"errors"
	"sort"
	"strings"
	"sync"
)

const (
	EventTypeMessageNew      = "message.new"
	EventTypeMessageEdited   = "message.edited"
	EventTypeMessageDeleted  = "message.deleted"
	EventTypeReactionAdded   = "reaction.added"
	EventTypeReactionRemoved = "reaction.removed"
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
	Send   chan Event
}

type clientState struct {
	userID        string
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

	conversationSet := normalizeConversationSet(conversationIDs)

	h.mu.Lock()
	defer h.mu.Unlock()

	if existing, ok := h.clients[client]; ok {
		h.removeClientLocked(client, existing)
	}

	state := clientState{
		userID:        userID,
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

func (h *Hub) Subscribe(client *Client, conversationID string) error {
	if client == nil {
		return ErrClientNil
	}
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return ErrConversationIDMissing
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	state, ok := h.clients[client]
	if !ok {
		return ErrClientNotRegistered
	}
	if _, exists := state.conversations[conversationID]; exists {
		return nil
	}

	state.conversations[conversationID] = struct{}{}
	h.clients[client] = state
	h.addConversationClientLocked(conversationID, client)
	return nil
}

func (h *Hub) Unsubscribe(client *Client, conversationID string) error {
	if client == nil {
		return ErrClientNil
	}
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return ErrConversationIDMissing
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	state, ok := h.clients[client]
	if !ok {
		return ErrClientNotRegistered
	}
	if _, exists := state.conversations[conversationID]; !exists {
		return nil
	}

	delete(state.conversations, conversationID)
	h.clients[client] = state
	h.removeConversationClientLocked(conversationID, client)
	return nil
}

func (h *Hub) SetConversations(client *Client, conversationIDs []string) error {
	if client == nil {
		return ErrClientNil
	}

	conversationSet := normalizeConversationSet(conversationIDs)

	h.mu.Lock()
	defer h.mu.Unlock()

	state, ok := h.clients[client]
	if !ok {
		return ErrClientNotRegistered
	}

	for conversationID := range state.conversations {
		if _, keep := conversationSet[conversationID]; keep {
			continue
		}
		h.removeConversationClientLocked(conversationID, client)
	}
	for conversationID := range conversationSet {
		if _, exists := state.conversations[conversationID]; exists {
			continue
		}
		h.addConversationClientLocked(conversationID, client)
	}

	state.conversations = conversationSet
	h.clients[client] = state
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

func (h *Hub) ConnectionsForUser(userID string) int {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return 0
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.userClients[userID])
}

func (h *Hub) ConversationIDs(client *Client) ([]string, error) {
	if client == nil {
		return nil, ErrClientNil
	}

	h.mu.RLock()
	state, ok := h.clients[client]
	h.mu.RUnlock()
	if !ok {
		return nil, ErrClientNotRegistered
	}

	out := make([]string, 0, len(state.conversations))
	for conversationID := range state.conversations {
		out = append(out, conversationID)
	}
	sort.Strings(out)
	return out, nil
}

func (h *Hub) ConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
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
	clientSet, ok := h.conversationClients[conversationID]
	if !ok {
		return
	}
	delete(clientSet, client)
	if len(clientSet) == 0 {
		delete(h.conversationClients, conversationID)
	}
}

func (h *Hub) copyConversationRecipientsLocked(conversationID string) []*Client {
	clientSet := h.conversationClients[conversationID]
	out := make([]*Client, 0, len(clientSet))
	for client := range clientSet {
		out = append(out, client)
	}
	return out
}

func normalizeConversationSet(ids []string) map[string]struct{} {
	out := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		out[id] = struct{}{}
	}
	return out
}
