package api

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"agent-messenger/server/store"
	"agent-messenger/server/ws"
)

const websocketSendBufferSize = 16

type websocketHandler struct {
	store store.Store
	hub   *ws.Hub
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
	if err := h.hub.Register(client, nil); err != nil {
		return
	}
	defer h.hub.Unregister(client)

	_, _ = io.Copy(io.Discard, conn)
}
