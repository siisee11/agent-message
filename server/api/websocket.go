package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"agent-messenger/server/models"
	"agent-messenger/server/store"
	"agent-messenger/server/ws"
)

const websocketSendBufferSize = 16
const websocketConversationBootstrapLimit = 1000

type websocketHandler struct {
	store store.Store
	hub   *ws.Hub
}

type inboundWebSocketEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type readEventData struct {
	ConversationID string `json:"conversation_id"`
}

func newWebSocketHandler(s store.Store, hub *ws.Hub) *websocketHandler {
	if hub == nil {
		hub = ws.NewHub()
	}

	return &websocketHandler{
		store: s,
		hub:   hub,
	}
}

func (h *websocketHandler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
		return
	}

	user, err := h.store.GetUserBySessionToken(r.Context(), token)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusUnauthorized, "invalid session token")
		default:
			writeError(w, http.StatusInternalServerError, "failed to validate bearer token")
		}
		return
	}

	conn, err := ws.Upgrade(w, r)
	if err != nil {
		switch {
		case errors.Is(err, ws.ErrWebSocketUpgradeRequired), errors.Is(err, ws.ErrWebSocketVersionRequired), errors.Is(err, ws.ErrWebSocketKeyInvalid):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "failed to upgrade websocket connection")
		}
		return
	}
	defer conn.Close()

	client := &ws.Client{
		UserID: user.ID,
		Send:   make(chan ws.Event, websocketSendBufferSize),
	}
	conversationIDs, err := h.listConversationIDsForUser(r, user.ID)
	if err != nil {
		return
	}
	if err := h.hub.Register(client, conversationIDs); err != nil {
		return
	}
	defer h.hub.Unregister(client)

	done := make(chan struct{})
	defer close(done)

	writeErr := make(chan error, 1)
	go func() {
		for {
			select {
			case <-done:
				return
			case event := <-client.Send:
				if err := ws.WriteJSON(conn, event); err != nil {
					select {
					case writeErr <- err:
					default:
					}
					return
				}
			}
		}
	}()

	for {
		select {
		case <-writeErr:
			return
		default:
		}

		var event inboundWebSocketEvent
		if err := ws.ReadJSON(conn, &event); err != nil {
			return
		}
		h.handleInboundEvent(r, user.ID, client, event)
	}
}

func (h *websocketHandler) listConversationIDsForUser(r *http.Request, userID string) ([]string, error) {
	summaries, err := h.store.ListConversationsByUser(r.Context(), models.ListUserConversationsParams{
		UserID: userID,
		Limit:  websocketConversationBootstrapLimit,
	})
	if err != nil {
		return nil, err
	}

	conversationIDs := make([]string, 0, len(summaries))
	seen := make(map[string]struct{}, len(summaries))
	for _, summary := range summaries {
		conversationID := strings.TrimSpace(summary.Conversation.ID)
		if conversationID == "" {
			continue
		}
		if _, ok := seen[conversationID]; ok {
			continue
		}
		seen[conversationID] = struct{}{}
		conversationIDs = append(conversationIDs, conversationID)
	}
	return conversationIDs, nil
}

func (h *websocketHandler) handleInboundEvent(r *http.Request, userID string, client *ws.Client, event inboundWebSocketEvent) {
	if strings.TrimSpace(event.Type) != "read" {
		return
	}

	var data readEventData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return
	}

	conversationID := strings.TrimSpace(data.ConversationID)
	if conversationID == "" {
		return
	}

	if _, err := h.store.GetConversationByIDForUser(r.Context(), models.GetConversationForUserParams{
		ConversationID: conversationID,
		UserID:         userID,
	}); err != nil {
		return
	}

	_ = h.hub.Subscribe(client, conversationID)
}
