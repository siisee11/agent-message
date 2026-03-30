package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"agent-messenger/server/realtime"
	"agent-messenger/server/store"
)

type eventStreamHandler struct {
	store store.Store
	hub   *realtime.Hub
}

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
		Send:   make(chan realtime.Event, realtimeSendBufferSize),
	}
	conversationIDs, err := listConversationIDsForUser(r, h.store, user.ID)
	if err != nil {
		log.Printf("sse bootstrap failed user=%s: %v", user.ID, err)
		writeError(w, http.StatusInternalServerError, "failed to list conversations")
		return
	}
	if err := h.hub.Register(client, conversationIDs); err != nil {
		log.Printf("sse register failed user=%s conversations=%d: %v", user.ID, len(conversationIDs), err)
		writeError(w, http.StatusInternalServerError, "failed to start event stream")
		return
	}
	defer func() {
		h.hub.Unregister(client)
		log.Printf("sse disconnected user=%s remote=%s", user.ID, r.RemoteAddr)
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	log.Printf("sse connected user=%s conversations=%d remote=%s", user.ID, len(conversationIDs), r.RemoteAddr)

	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-client.Send:
			if err := writeSSEEvent(w, event); err != nil {
				log.Printf("sse write failed user=%s event=%s: %v", user.ID, event.Type, err)
				return
			}
			flusher.Flush()
		}
	}
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
