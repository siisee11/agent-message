package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"

	"agent-message/server/models"
	"agent-message/server/realtime"
	"agent-message/server/store"
)

type eventStreamHandler struct {
	store store.Store
	hub   *realtime.Hub
}

const eventStreamHeartbeatInterval = 25 * time.Second

func newEventStreamHandler(s store.Store, hub *realtime.Hub) *eventStreamHandler {
	if hub == nil {
		hub = realtime.NewHub()
	}

	return &eventStreamHandler{
		store: s,
		hub:   hub,
	}
}

func (h *eventStreamHandler) handleEventStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
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

	client := &realtime.Client{
		UserID: user.ID,
		Kind:   eventStreamClientKind(r),
		Send:   make(chan realtime.Event, realtimeSendBufferSize),
	}
	conversationIDs, err := listConversationIDsForUser(r, h.store, user.ID)
	if err != nil {
		log.Printf("sse bootstrap failed user=%s: %v", user.ID, err)
		writeError(w, http.StatusInternalServerError, "failed to list conversations")
		return
	}
	previousWatcherConnections := 0
	if client.Kind == realtime.ClientKindWatcher {
		previousWatcherConnections = h.hub.ConnectionsForUserKind(user.ID, realtime.ClientKindWatcher)
	}

	if err := h.hub.Register(client, conversationIDs); err != nil {
		log.Printf("sse register failed user=%s conversations=%d: %v", user.ID, len(conversationIDs), err)
		writeError(w, http.StatusInternalServerError, "failed to start event stream")
		return
	}
	if client.Kind == realtime.ClientKindWatcher && previousWatcherConnections == 0 {
		h.broadcastWatcherPresence(conversationIDs, user.ID, true)
	}
	defer func() {
		h.hub.Unregister(client)
		if client.Kind == realtime.ClientKindWatcher && h.hub.ConnectionsForUserKind(user.ID, realtime.ClientKindWatcher) == 0 {
			h.broadcastWatcherPresence(conversationIDs, user.ID, false)
		}
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	if err := writeSSEComment(w, "connected"); err != nil {
		if !isExpectedSSEDisconnect(err) {
			log.Printf("sse bootstrap write failed user=%s: %v", user.ID, err)
		}
		return
	}
	flusher.Flush()
	heartbeatTicker := time.NewTicker(eventStreamHeartbeatInterval)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeatTicker.C:
			if err := writeSSEComment(w, "keep-alive"); err != nil {
				if !isExpectedSSEDisconnect(err) {
					log.Printf("sse heartbeat failed user=%s: %v", user.ID, err)
				}
				return
			}
			flusher.Flush()
		case event := <-client.Send:
			if err := writeSSEEvent(w, event); err != nil {
				if !isExpectedSSEDisconnect(err) {
					log.Printf("sse write failed user=%s event=%s: %v", user.ID, event.Type, err)
				}
				return
			}
			flusher.Flush()
		}
	}
}

func eventStreamClientKind(r *http.Request) string {
	clientKind := strings.TrimSpace(r.URL.Query().Get("client_kind"))
	if clientKind == "" {
		return realtime.ClientKindWeb
	}
	return realtime.NormalizeClientKind(clientKind)
}

func (h *eventStreamHandler) broadcastWatcherPresence(conversationIDs []string, userID string, online bool) {
	if h.hub == nil {
		return
	}

	seen := make(map[string]struct{}, len(conversationIDs))
	for _, rawConversationID := range conversationIDs {
		conversationID := strings.TrimSpace(rawConversationID)
		if conversationID == "" {
			continue
		}
		if _, ok := seen[conversationID]; ok {
			continue
		}
		seen[conversationID] = struct{}{}

		if _, err := h.hub.BroadcastToConversation(conversationID, realtime.Event{
			Type: realtime.EventTypePresenceUpdated,
			Data: models.WatcherPresenceEvent{
				ConversationID: conversationID,
				UserID:         strings.TrimSpace(userID),
				ClientKind:     realtime.ClientKindWatcher,
				Online:         online,
			},
		}); err != nil {
			log.Printf("presence broadcast failed conversation=%s user=%s online=%t: %v", conversationID, userID, online, err)
		}
	}
}

func isExpectedSSEDisconnect(err error) bool {
	return errors.Is(err, context.Canceled) ||
		errors.Is(err, net.ErrClosed) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ECONNRESET)
}

func writeSSEComment(w http.ResponseWriter, comment string) error {
	_, err := fmt.Fprintf(w, ": %s\n\n", comment)
	return err
}

func writeSSEEvent(w http.ResponseWriter, event realtime.Event) error {
	payload, err := json.Marshal(map[string]any{
		"type": event.Type,
		"data": event.Data,
	})
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "event: %s\n", event.Type); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
		return err
	}
	return nil
}
