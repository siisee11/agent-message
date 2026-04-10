package api

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"agent-message/server/models"
	"agent-message/server/realtime"
	"agent-message/server/store"

	"github.com/google/uuid"
)

const (
	defaultUserSearchLimit   = 20
	defaultConversationLimit = 50
)

type usersHandler struct {
	store store.Store
}

func newUsersHandler(s store.Store) *usersHandler {
	return &usersHandler{store: s}
}

func (h *usersHandler) handleUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	authUser, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("username"))
	if query == "" {
		writeJSON(w, http.StatusOK, make([]models.UserProfile, 0))
		return
	}
	if err := models.ValidateUsernameQuery(query); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	limit, err := parsePositiveIntQuery(r, "limit")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if limit == 0 {
		limit = defaultUserSearchLimit
	}

	users, err := h.store.SearchUsersByUsername(r.Context(), models.SearchUsersParams{
		Query: query,
		Limit: limit,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to search users")
		return
	}

	profiles := make([]models.UserProfile, 0, len(users))
	for _, user := range users {
		if user.ID == authUser.ID {
			continue
		}
		profiles = append(profiles, user.Profile())
	}

	writeJSON(w, http.StatusOK, profiles)
}

func (h *usersHandler) handleMe(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGetMe(w, r)
	case http.MethodPatch:
		h.handlePatchMe(w, r)
	default:
		w.Header().Set("Allow", "GET, PATCH")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *usersHandler) handleGetMe(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
		return
	}

	writeJSON(w, http.StatusOK, user.Profile())
}

func (h *usersHandler) handlePatchMe(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
		return
	}

	var req models.UpdateUsernameRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	desiredUsername := strings.TrimSpace(req.Username)
	if desiredUsername != "" {
		existing, err := h.store.GetUserByUsername(r.Context(), desiredUsername)
		if err == nil && existing.ID != user.ID {
			writeError(w, http.StatusConflict, "username already exists")
			return
		}
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusInternalServerError, "failed to update username")
			return
		}
	}

	updatedUser, err := h.store.UpdateUsername(r.Context(), models.UpdateUsernameParams{
		UserID:    user.ID,
		Username:  desiredUsername,
		UpdatedAt: time.Now().UTC(),
	})
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update username")
		return
	}

	writeJSON(w, http.StatusOK, updatedUser.Profile())
}

type conversationsHandler struct {
	store           store.Store
	hub             *realtime.Hub
	watcherPresence *realtime.WatcherPresence
	nowFn           func() time.Time
}

func newConversationsHandler(s store.Store, hub *realtime.Hub, watcherPresence *realtime.WatcherPresence) *conversationsHandler {
	return &conversationsHandler{
		store:           s,
		hub:             hub,
		watcherPresence: watcherPresence,
		nowFn:           time.Now,
	}
}

func (h *conversationsHandler) handleConversationsCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleListConversations(w, r)
	case http.MethodPost:
		h.handleStartConversation(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *conversationsHandler) handleConversationDetail(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGetConversationDetail(w, r)
	case http.MethodPatch:
		h.handlePatchConversationDetail(w, r)
	default:
		w.Header().Set("Allow", "GET, PATCH")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *conversationsHandler) handleGetConversationDetail(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
		return
	}

	conversationID, valid := conversationIDFromPath(r.URL.Path)
	if !valid {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}

	details, err := h.store.GetConversationByIDForUser(r.Context(), models.GetConversationForUserParams{
		ConversationID: conversationID,
		UserID:         user.ID,
	})
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "conversation not found")
		case errors.Is(err, store.ErrForbidden):
			writeError(w, http.StatusForbidden, "forbidden")
		default:
			writeError(w, http.StatusInternalServerError, "failed to fetch conversation")
		}
		return
	}

	h.decorateWatcherPresence(&details, user.ID)
	writeJSON(w, http.StatusOK, details)
}

func (h *conversationsHandler) handlePatchConversationDetail(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
		return
	}

	conversationID, valid := conversationIDFromPath(r.URL.Path)
	if !valid {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}

	var req models.UpdateConversationRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	updatedConversation, err := h.store.UpdateConversationTitle(r.Context(), models.UpdateConversationTitleParams{
		ConversationID: conversationID,
		ActorUserID:    user.ID,
		Title:          strings.TrimSpace(req.Title),
	})
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "conversation not found")
		case errors.Is(err, store.ErrForbidden):
			writeError(w, http.StatusForbidden, "forbidden")
		default:
			writeError(w, http.StatusInternalServerError, "failed to update conversation")
		}
		return
	}

	details, err := h.store.GetConversationByIDForUser(r.Context(), models.GetConversationForUserParams{
		ConversationID: updatedConversation.ID,
		UserID:         user.ID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch conversation")
		return
	}

	h.decorateWatcherPresence(&details, user.ID)
	writeJSON(w, http.StatusOK, details)
}

func (h *conversationsHandler) handleListConversations(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
		return
	}

	limit, err := parsePositiveIntQuery(r, "limit")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if limit == 0 {
		limit = defaultConversationLimit
	}

	conversations, err := h.store.ListConversationsByUser(r.Context(), models.ListUserConversationsParams{
		UserID: user.ID,
		Limit:  limit,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list conversations")
		return
	}
	if conversations == nil {
		conversations = make([]models.ConversationSummary, 0)
	}

	writeJSON(w, http.StatusOK, conversations)
}

func (h *conversationsHandler) handleStartConversation(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
		return
	}

	var req models.StartConversationRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.EqualFold(strings.TrimSpace(req.Username), user.EffectiveUsername()) {
		writeError(w, http.StatusBadRequest, "cannot start a conversation with yourself")
		return
	}

	targetUser, err := h.store.GetUserByUsername(r.Context(), strings.TrimSpace(req.Username))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to start conversation")
		return
	}
	if targetUser.ID == user.ID {
		writeError(w, http.StatusBadRequest, "cannot start a conversation with yourself")
		return
	}

	newConversationID := uuid.NewString()
	conversation, err := h.store.GetOrCreateDirectConversation(r.Context(), models.GetOrCreateDirectConversationParams{
		ConversationID: newConversationID,
		CurrentUserID:  user.ID,
		TargetUserID:   targetUser.ID,
		CreatedAt:      h.nowFn().UTC(),
	})
	if err != nil {
		switch {
		case errors.Is(err, store.ErrForbidden):
			writeError(w, http.StatusForbidden, "forbidden")
		default:
			writeError(w, http.StatusInternalServerError, "failed to start conversation")
		}
		return
	}

	details, err := h.store.GetConversationByIDForUser(r.Context(), models.GetConversationForUserParams{
		ConversationID: conversation.ID,
		UserID:         user.ID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start conversation")
		return
	}

	status := http.StatusOK
	if conversation.ID == newConversationID {
		status = http.StatusCreated
	}
	log.Printf(
		"conversation ready id=%s actor=%s target=%s created=%t",
		conversation.ID,
		user.ID,
		targetUser.ID,
		conversation.ID == newConversationID,
	)
	h.subscribeConversationParticipants(conversation)
	h.decorateWatcherPresence(&details, user.ID)
	writeJSON(w, status, details)
}

func (h *conversationsHandler) decorateWatcherPresence(details *models.ConversationDetails, currentUserID string) {
	if details == nil {
		return
	}

	otherParticipant := details.ParticipantA
	if otherParticipant.ID == currentUserID {
		otherParticipant = details.ParticipantB
	} else if details.ParticipantB.ID != currentUserID {
		otherParticipant = details.ParticipantA
	}

	details.WatcherPresence = &models.WatcherPresence{
		UserID:     otherParticipant.ID,
		ClientKind: realtime.ClientKindWatcher,
		Online:     h.watcherPresence != nil && h.watcherPresence.IsOnline(otherParticipant.ID),
	}
}

func (h *conversationsHandler) subscribeConversationParticipants(conversation models.Conversation) {
	if h.hub == nil {
		return
	}

	if err := h.hub.SubscribeUser(conversation.ParticipantA, conversation.ID); err != nil {
		log.Printf("conversation subscribe failed user=%s conversation=%s: %v", conversation.ParticipantA, conversation.ID, err)
	} else {
		log.Printf("conversation subscribed user=%s conversation=%s source=create", conversation.ParticipantA, conversation.ID)
	}
	if h.watcherPresence != nil {
		if err := h.watcherPresence.SubscribeUser(conversation.ParticipantA, conversation.ID); err != nil {
			log.Printf("watcher presence subscribe failed user=%s conversation=%s: %v", conversation.ParticipantA, conversation.ID, err)
		}
	}
	if err := h.hub.SubscribeUser(conversation.ParticipantB, conversation.ID); err != nil {
		log.Printf("conversation subscribe failed user=%s conversation=%s: %v", conversation.ParticipantB, conversation.ID, err)
	} else {
		log.Printf("conversation subscribed user=%s conversation=%s source=create", conversation.ParticipantB, conversation.ID)
	}
	if h.watcherPresence != nil {
		if err := h.watcherPresence.SubscribeUser(conversation.ParticipantB, conversation.ID); err != nil {
			log.Printf("watcher presence subscribe failed user=%s conversation=%s: %v", conversation.ParticipantB, conversation.ID, err)
		}
	}
}

func parsePositiveIntQuery(r *http.Request, key string) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return 0, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, errors.New("limit must be a positive integer")
	}
	return value, nil
}

func conversationIDFromPath(path string) (string, bool) {
	const prefix = "/api/conversations/"

	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(path, prefix))
	if rest == "" || strings.Contains(rest, "/") {
		return "", false
	}
	return rest, true
}
