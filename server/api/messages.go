package api

import (
	"errors"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"agent-messenger/server/models"
	"agent-messenger/server/store"
	"agent-messenger/server/ws"

	"github.com/google/uuid"
)

type messagesHandler struct {
	store     store.Store
	hub       *ws.Hub
	nowFn     func() time.Time
	uploadDir string
}

func newMessagesHandler(s store.Store, hub *ws.Hub) *messagesHandler {
	return &messagesHandler{
		store:     s,
		hub:       hub,
		nowFn:     time.Now,
		uploadDir: defaultUploadDir,
	}
}

func (h *messagesHandler) handleConversationMessages(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleListMessages(w, r)
	case http.MethodPost:
		h.handleCreateMessage(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *messagesHandler) handleMessageByID(w http.ResponseWriter, r *http.Request) {
	if _, isReactionsPath := messageReactionsPath(r.URL.Path); isReactionsPath {
		h.handleMessageReactions(w, r)
		return
	}
	if _, _, isReactionByEmojiPath := messageReactionByEmojiPath(r.URL.Path); isReactionByEmojiPath {
		h.handleMessageReactionByEmoji(w, r)
		return
	}

	switch r.Method {
	case http.MethodPatch:
		h.handleEditMessage(w, r)
	case http.MethodDelete:
		h.handleDeleteMessage(w, r)
	default:
		w.Header().Set("Allow", "PATCH, DELETE")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *messagesHandler) handleMessageReactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
		return
	}

	messageID, valid := messageReactionsPath(r.URL.Path)
	if !valid {
		http.NotFound(w, r)
		return
	}

	var req models.ToggleReactionRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.store.ToggleMessageReaction(r.Context(), models.ToggleMessageReactionParams{
		ReactionID:  uuid.NewString(),
		MessageID:   messageID,
		ActorUserID: user.ID,
		Emoji:       strings.TrimSpace(req.Emoji),
		CreatedAt:   h.nowFn().UTC(),
	})
	if err != nil {
		switch {
		case errors.Is(err, store.ErrForbidden):
			writeError(w, http.StatusForbidden, "forbidden")
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "message not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to toggle reaction")
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *messagesHandler) handleMessageReactionByEmoji(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeMethodNotAllowed(w, http.MethodDelete)
		return
	}

	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
		return
	}

	messageID, emoji, valid := messageReactionByEmojiPath(r.URL.Path)
	if !valid {
		http.NotFound(w, r)
		return
	}

	reaction, err := h.store.RemoveMessageReaction(r.Context(), models.RemoveMessageReactionParams{
		MessageID:   messageID,
		ActorUserID: user.ID,
		Emoji:       emoji,
	})
	if err != nil {
		switch {
		case errors.Is(err, store.ErrForbidden):
			writeError(w, http.StatusForbidden, "forbidden")
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "reaction or message not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to remove reaction")
		}
		return
	}

	writeJSON(w, http.StatusOK, reaction)
}

func (h *messagesHandler) handleListMessages(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
		return
	}

	conversationID, valid := conversationMessagesPath(r.URL.Path)
	if !valid {
		http.NotFound(w, r)
		return
	}

	limit, err := parseMessageLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	beforeRaw := strings.TrimSpace(r.URL.Query().Get("before"))
	var before *string
	if beforeRaw != "" {
		before = &beforeRaw
	}

	messages, err := h.store.ListMessagesByConversation(r.Context(), models.ListConversationMessagesParams{
		ConversationID:  conversationID,
		UserID:          user.ID,
		BeforeMessageID: before,
		Limit:           limit,
	})
	if err != nil {
		switch {
		case errors.Is(err, store.ErrForbidden):
			writeError(w, http.StatusForbidden, "forbidden")
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "conversation or message cursor not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to list messages")
		}
		return
	}
	if messages == nil {
		messages = make([]models.MessageDetails, 0)
	}

	writeJSON(w, http.StatusOK, messages)
}

func (h *messagesHandler) handleCreateMessage(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
		return
	}

	conversationID, valid := conversationMessagesPath(r.URL.Path)
	if !valid {
		http.NotFound(w, r)
		return
	}

	content, attachmentURL, attachmentType, err := h.parseCreateMessagePayload(w, r)
	if err != nil {
		switch {
		case errors.Is(err, errRequestEntityTooLarge):
			writeError(w, http.StatusRequestEntityTooLarge, "attachment exceeds 20 MB")
		case errors.Is(err, models.ErrMessageContentRequired):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	now := h.nowFn().UTC()
	message, err := h.store.CreateMessage(r.Context(), models.CreateMessageParams{
		ID:             uuid.NewString(),
		ConversationID: conversationID,
		SenderID:       user.ID,
		Content:        content,
		AttachmentURL:  attachmentURL,
		AttachmentType: attachmentType,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if err != nil {
		switch {
		case errors.Is(err, store.ErrForbidden):
			writeError(w, http.StatusForbidden, "forbidden")
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "conversation not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to send message")
		}
		return
	}

	h.broadcastConversationEvent(conversationID, ws.EventTypeMessageNew, message)
	writeJSON(w, http.StatusCreated, message)
}

func (h *messagesHandler) handleEditMessage(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
		return
	}

	messageID, valid := messageIDFromPath(r.URL.Path)
	if !valid {
		http.NotFound(w, r)
		return
	}

	var req models.EditMessageRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	message, err := h.store.UpdateMessage(r.Context(), models.UpdateMessageParams{
		MessageID:   messageID,
		ActorUserID: user.ID,
		Content:     req.Content,
		UpdatedAt:   h.nowFn().UTC(),
	})
	if err != nil {
		switch {
		case errors.Is(err, store.ErrForbidden):
			writeError(w, http.StatusForbidden, "forbidden")
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "message not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to edit message")
		}
		return
	}

	h.broadcastConversationEvent(message.ConversationID, ws.EventTypeMessageEdited, message)
	writeJSON(w, http.StatusOK, message)
}

func (h *messagesHandler) handleDeleteMessage(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
		return
	}

	messageID, valid := messageIDFromPath(r.URL.Path)
	if !valid {
		http.NotFound(w, r)
		return
	}

	message, err := h.store.SoftDeleteMessage(r.Context(), models.SoftDeleteMessageParams{
		MessageID:   messageID,
		ActorUserID: user.ID,
		UpdatedAt:   h.nowFn().UTC(),
	})
	if err != nil {
		switch {
		case errors.Is(err, store.ErrForbidden):
			writeError(w, http.StatusForbidden, "forbidden")
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "message not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to delete message")
		}
		return
	}

	h.broadcastConversationEvent(message.ConversationID, ws.EventTypeMessageDeleted, map[string]string{"id": message.ID})
	writeJSON(w, http.StatusOK, message)
}

func (h *messagesHandler) broadcastConversationEvent(conversationID, eventType string, data any) {
	if h.hub == nil {
		return
	}
	_, _ = h.hub.BroadcastToConversation(conversationID, ws.Event{
		Type: eventType,
		Data: data,
	})
}

func (h *messagesHandler) parseCreateMessagePayload(w http.ResponseWriter, r *http.Request) (*string, *string, *models.AttachmentType, error) {
	contentType := strings.TrimSpace(r.Header.Get("Content-Type"))
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, nil, nil, errors.New("invalid content type")
	}

	switch mediaType {
	case "application/json":
		var req models.SendMessageRequest
		if err := decodeJSONBody(r, &req); err != nil {
			return nil, nil, nil, errors.New("invalid JSON body")
		}
		if err := req.Validate(); err != nil {
			return nil, nil, nil, err
		}
		content := strings.TrimSpace(req.Content)
		return &content, nil, nil, nil
	case "multipart/form-data":
		return h.parseMultipartMessagePayload(w, r)
	default:
		return nil, nil, nil, errors.New("content type must be application/json or multipart/form-data")
	}
}

func (h *messagesHandler) parseMultipartMessagePayload(w http.ResponseWriter, r *http.Request) (*string, *string, *models.AttachmentType, error) {
	maxBodyBytes := int64(maxUploadBytes + multipartBodySizeBuffer)
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := r.ParseMultipartForm(maxBodyBytes); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return nil, nil, nil, errRequestEntityTooLarge
		}
		return nil, nil, nil, errors.New("invalid multipart form")
	}

	contentText := strings.TrimSpace(r.FormValue("content"))
	var content *string
	if contentText != "" {
		content = &contentText
	}

	uploadedFile, header, err := r.FormFile("attachment")
	if err == nil {
		defer uploadedFile.Close()
		url, attachmentType, saveErr := saveUploadedFile(h.uploadDir, uploadedFile, header)
		if saveErr != nil {
			return nil, nil, nil, saveErr
		}
		return content, &url, &attachmentType, nil
	}
	if !errors.Is(err, http.ErrMissingFile) {
		return nil, nil, nil, errors.New("invalid attachment payload")
	}

	attachmentURLText := strings.TrimSpace(r.FormValue("attachment_url"))
	if attachmentURLText != "" {
		attachmentType, err := parseAttachmentType(r.FormValue("attachment_type"))
		if err != nil {
			return nil, nil, nil, err
		}
		return content, &attachmentURLText, &attachmentType, nil
	}

	if content == nil {
		return nil, nil, nil, models.ErrMessageContentRequired
	}
	return content, nil, nil, nil
}

func parseAttachmentType(raw string) (models.AttachmentType, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "", string(models.AttachmentTypeFile):
		return models.AttachmentTypeFile, nil
	case string(models.AttachmentTypeImage):
		return models.AttachmentTypeImage, nil
	default:
		return "", errors.New("attachment_type must be image or file")
	}
}

func parseMessageLimit(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return models.DefaultMessagePageLimit, nil
	}

	limit, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.New("limit must be a positive integer")
	}
	if limit < 1 {
		return 0, errors.New("limit must be a positive integer")
	}
	if limit > models.MaxMessagePageLimit {
		return 0, models.ErrPageLimitOutOfRange
	}

	return limit, nil
}

func conversationMessagesPath(path string) (string, bool) {
	const prefix = "/api/conversations/"
	const suffix = "/messages"

	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return "", false
	}

	trimmed := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" || strings.Contains(trimmed, "/") {
		return "", false
	}
	return trimmed, true
}

func messageIDFromPath(path string) (string, bool) {
	const prefix = "/api/messages/"

	if !strings.HasPrefix(path, prefix) {
		return "", false
	}

	trimmed := strings.TrimSpace(strings.TrimPrefix(path, prefix))
	if trimmed == "" || strings.Contains(trimmed, "/") {
		return "", false
	}
	return trimmed, true
}

func messageReactionsPath(path string) (string, bool) {
	const (
		prefix = "/api/messages/"
		suffix = "/reactions"
	)

	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return "", false
	}

	messageID := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix))
	if messageID == "" || strings.Contains(messageID, "/") {
		return "", false
	}
	return messageID, true
}

func messageReactionByEmojiPath(path string) (string, string, bool) {
	const (
		prefix = "/api/messages/"
		infix  = "/reactions/"
	)

	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}

	rest := strings.TrimSpace(strings.TrimPrefix(path, prefix))
	messageID, encodedEmoji, found := strings.Cut(rest, infix)
	if !found || messageID == "" || encodedEmoji == "" || strings.Contains(messageID, "/") || strings.Contains(encodedEmoji, "/") {
		return "", "", false
	}

	emoji, err := url.PathUnescape(encodedEmoji)
	if err != nil {
		return "", "", false
	}
	emoji = strings.TrimSpace(emoji)
	if emoji == "" {
		return "", "", false
	}
	return messageID, emoji, true
}
