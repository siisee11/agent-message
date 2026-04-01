package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"agent-message/server/models"
	"agent-message/server/push"
	"agent-message/server/store"

	"github.com/google/uuid"
)

type pushHandler struct {
	store    store.Store
	notifier *push.Service
	nowFn    func() time.Time
}

func newPushHandler(s store.Store, notifier *push.Service) *pushHandler {
	return &pushHandler{
		store:    s,
		notifier: notifier,
		nowFn:    time.Now,
	}
}

func (h *pushHandler) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	if h.notifier == nil {
		writeJSON(w, http.StatusOK, models.PushConfigResponse{Enabled: false})
		return
	}
	writeJSON(w, http.StatusOK, h.notifier.PublicConfig())
}

func (h *pushHandler) handleSubscriptions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.handleUpsertSubscription(w, r)
	case http.MethodDelete:
		h.handleDeleteSubscription(w, r)
	default:
		w.Header().Set("Allow", "POST, DELETE")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *pushHandler) handleUpsertSubscription(w http.ResponseWriter, r *http.Request) {
	if h.notifier == nil || !h.notifier.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "push notifications are not configured")
		return
	}

	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
		return
	}

	var req models.UpsertPushSubscriptionRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	now := h.nowFn().UTC()
	subscription, err := h.store.UpsertPushSubscription(r.Context(), models.UpsertPushSubscriptionParams{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		Endpoint:  strings.TrimSpace(req.Endpoint),
		P256DH:    strings.TrimSpace(req.Keys.P256DH),
		Auth:      strings.TrimSpace(req.Keys.Auth),
		UserAgent: strings.TrimSpace(r.Header.Get("User-Agent")),
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save push subscription")
		return
	}

	writeJSON(w, http.StatusCreated, subscription)
}

func (h *pushHandler) handleDeleteSubscription(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
		return
	}

	var req models.DeletePushSubscriptionRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.store.DeletePushSubscriptionByEndpointForUser(r.Context(), user.ID, strings.TrimSpace(req.Endpoint)); err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			w.WriteHeader(http.StatusNoContent)
		default:
			writeError(w, http.StatusInternalServerError, "failed to delete push subscription")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
