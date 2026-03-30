package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"agent-messenger/server/models"
	"agent-messenger/server/store"

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
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
		return
	}

	writeJSON(w, http.StatusOK, user.Profile())
}

type conversationsHandler struct {
	store store.Store
	nowFn func() time.Time
}

func newConversationsHandler(s store.Store) *conversationsHandler {
	return &conversationsHandler{
		store: s,
		nowFn: time.Now,
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
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

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
	if strings.EqualFold(strings.TrimSpace(req.Username), user.Username) {
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
	writeJSON(w, status, details)
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
