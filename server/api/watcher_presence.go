package api

import (
	"errors"
	"net/http"
	"strings"
	"unicode"

	"agent-message/server/realtime"
)

type watcherPresenceHandler struct {
	presence *realtime.WatcherPresence
	hub      *realtime.Hub
}

func newWatcherPresenceHandler(presence *realtime.WatcherPresence, hub *realtime.Hub) *watcherPresenceHandler {
	if presence == nil {
		presence = realtime.NewWatcherPresence(realtime.DefaultWatcherPresenceTTL)
	}
	return &watcherPresenceHandler{presence: presence, hub: hub}
}

type watcherHeartbeatRequest struct {
	SessionID string `json:"session_id"`
}

func (h *watcherPresenceHandler) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
		return
	}

	var req watcherHeartbeatRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	sessionID, err := validateWatcherSessionID(req.SessionID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	transition, ok := h.presence.Heartbeat(user.ID, sessionID)
	if !ok {
		writeError(w, http.StatusNotFound, "watcher session not found")
		return
	}
	h.broadcastWatcherTransition(transition)

	w.WriteHeader(http.StatusNoContent)
}

func (h *watcherPresenceHandler) handleSessionByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeMethodNotAllowed(w, http.MethodDelete)
		return
	}

	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
		return
	}

	sessionID, valid := watcherSessionIDFromPath(r.URL.Path)
	if !valid {
		writeError(w, http.StatusNotFound, "watcher session not found")
		return
	}

	h.broadcastWatcherTransition(h.presence.Unregister(user.ID, sessionID))
	w.WriteHeader(http.StatusNoContent)
}

func validateWatcherSessionID(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	switch {
	case trimmed == "":
		return "", realtime.ErrWatcherPresenceSessionIDRequired
	case strings.ContainsAny(trimmed, `/\?#`):
		return "", errors.New("watcher session id must not contain path or query syntax")
	case hasWatcherSessionControlCharacters(trimmed):
		return "", errors.New("watcher session id must not contain control characters")
	default:
		return trimmed, nil
	}
}

func watcherSessionIDFromPath(path string) (string, bool) {
	const prefix = "/api/watchers/sessions/"
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(path, prefix))
	if rest == "" || strings.Contains(rest, "/") {
		return "", false
	}
	sessionID, err := validateWatcherSessionID(rest)
	if err != nil {
		return "", false
	}
	return sessionID, true
}

func hasWatcherSessionControlCharacters(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func (h *watcherPresenceHandler) broadcastWatcherTransition(transition *realtime.WatcherPresenceTransition) {
	if transition == nil || h.hub == nil {
		return
	}

	seen := make(map[string]struct{}, len(transition.ConversationIDs))
	for _, rawConversationID := range transition.ConversationIDs {
		conversationID := strings.TrimSpace(rawConversationID)
		if conversationID == "" {
			continue
		}
		if _, ok := seen[conversationID]; ok {
			continue
		}
		seen[conversationID] = struct{}{}

		_, _ = h.hub.BroadcastToConversation(conversationID, realtime.Event{
			Type: realtime.EventTypePresenceUpdated,
			Data: map[string]any{
				"conversation_id": conversationID,
				"user_id":         transition.UserID,
				"client_kind":     realtime.ClientKindWatcher,
				"online":          transition.Online,
			},
		})
	}
}
