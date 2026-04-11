package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"agent-message/server/models"
	"agent-message/server/push"
	"agent-message/server/realtime"
	"agent-message/server/store"

	"github.com/google/uuid"
)

type messagesHandler struct {
	store     store.Store
	hub       *realtime.Hub
	notifier  *push.Service
	nowFn     func() time.Time
	uploadDir string
}

func newMessagesHandler(s store.Store, hub *realtime.Hub) *messagesHandler {
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
		writeError(w, http.StatusNotFound, "message not found")
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

	switch result.Action {
	case models.ReactionMutationAdded:
		h.broadcastReactionEventForMessage(r, user.ID, messageID, realtime.EventTypeReactionAdded, result.Reaction)
	case models.ReactionMutationRemoved:
		h.broadcastReactionEventForMessage(r, user.ID, messageID, realtime.EventTypeReactionRemoved, reactionRemovedEventData(result.Reaction))
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
		writeError(w, http.StatusNotFound, "reaction not found")
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

	h.broadcastReactionEventForMessage(r, user.ID, messageID, realtime.EventTypeReactionRemoved, reactionRemovedEventData(reaction))
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
		writeError(w, http.StatusNotFound, "conversation not found")
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
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}

	content, kind, jsonRenderSpec, attachments, attachmentURL, attachmentType, err := h.parseCreateMessagePayload(w, r)
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
		Kind:           kind,
		JSONRenderSpec: jsonRenderSpec,
		Attachments:    attachments,
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

	log.Printf("message created conversation=%s message=%s sender=%s attachment=%t", conversationID, message.ID, user.ID, message.AttachmentURL != nil)
	h.broadcastConversationEvent(conversationID, realtime.EventTypeMessageNew, message)
	h.notifyConversationMessage(conversationID, user, message)
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
		writeError(w, http.StatusNotFound, "message not found")
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

	log.Printf("message edited conversation=%s message=%s actor=%s", message.ConversationID, message.ID, user.ID)
	h.broadcastConversationEvent(message.ConversationID, realtime.EventTypeMessageEdited, message)
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
		writeError(w, http.StatusNotFound, "message not found")
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

	log.Printf("message deleted conversation=%s message=%s actor=%s", message.ConversationID, message.ID, user.ID)
	h.broadcastConversationEvent(message.ConversationID, realtime.EventTypeMessageDeleted, map[string]string{"id": message.ID})
	writeJSON(w, http.StatusOK, message)
}

func (h *messagesHandler) broadcastConversationEvent(conversationID, eventType string, data any) {
	if h.hub == nil {
		return
	}
	result, err := h.hub.BroadcastToConversation(conversationID, realtime.Event{
		Type: eventType,
		Data: data,
	})
	if err != nil {
		log.Printf("broadcast failed conversation=%s event=%s: %v", conversationID, eventType, err)
		return
	}
	log.Printf("broadcast conversation=%s event=%s delivered=%d dropped=%d", conversationID, eventType, result.Delivered, result.Dropped)
}

func (h *messagesHandler) broadcastReactionEventForMessage(r *http.Request, userID, messageID, eventType string, data any) {
	if h.hub == nil {
		return
	}

	message, err := h.store.GetMessageByIDForUser(r.Context(), models.GetMessageForUserParams{
		MessageID: messageID,
		UserID:    userID,
	})
	if err != nil {
		log.Printf("reaction broadcast lookup failed actor=%s message=%s event=%s: %v", userID, messageID, eventType, err)
		return
	}
	log.Printf("reaction event conversation=%s message=%s actor=%s event=%s", message.ConversationID, messageID, userID, eventType)
	h.broadcastConversationEvent(message.ConversationID, eventType, data)
}

func reactionRemovedEventData(reaction models.Reaction) map[string]string {
	return map[string]string{
		"message_id": reaction.MessageID,
		"emoji":      reaction.Emoji,
		"user_id":    reaction.UserID,
	}
}

func (h *messagesHandler) notifyConversationMessage(conversationID string, sender models.User, message models.Message) {
	if h.notifier == nil || !h.notifier.Enabled() {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		details, err := h.store.GetConversationByIDForUser(ctx, models.GetConversationForUserParams{
			ConversationID: conversationID,
			UserID:         sender.ID,
		})
		if err != nil {
			log.Printf("push conversation lookup failed conversation=%s sender=%s: %v", conversationID, sender.ID, err)
			return
		}

		recipientID := details.ParticipantA.ID
		if recipientID == sender.ID {
			recipientID = details.ParticipantB.ID
		}

		if err := h.notifier.NotifyMessage(ctx, recipientID, push.MessageNotification{
			ConversationID: conversationID,
			MessageID:      message.ID,
			SenderID:       sender.ID,
			SenderName:     sender.EffectiveUsername(),
			Preview:        messageNotificationPreview(message),
			URL:            "/dm/" + conversationID,
		}); err != nil {
			log.Printf("push notify failed conversation=%s message=%s recipient=%s: %v", conversationID, message.ID, recipientID, err)
		}
	}()
}

func messageNotificationPreview(message models.Message) string {
	message.ApplyAttachmentFallbacks()
	attachmentCount := len(message.Attachments)

	switch {
	case message.Deleted:
		return "Deleted message"
	case message.Content != nil && strings.TrimSpace(*message.Content) != "":
		return truncateNotificationText(strings.TrimSpace(*message.Content), 140)
	case message.Kind == models.MessageKindJSONRender:
		return "Sent an interactive card"
	case attachmentCount > 1 && allMessageAttachmentsAreType(message.Attachments, models.AttachmentTypeImage):
		return "Sent images"
	case attachmentCount > 1:
		return "Sent attachments"
	case message.AttachmentType != nil && *message.AttachmentType == models.AttachmentTypeImage:
		return "Sent an image"
	case message.AttachmentType != nil && *message.AttachmentType == models.AttachmentTypeFile:
		return "Sent a file"
	default:
		return "New message"
	}
}

func truncateNotificationText(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return strings.TrimSpace(string(runes[:limit])) + "…"
}

func (h *messagesHandler) parseCreateMessagePayload(
	w http.ResponseWriter,
	r *http.Request,
) (*string, models.MessageKind, json.RawMessage, []models.MessageAttachment, *string, *models.AttachmentType, error) {
	contentType := strings.TrimSpace(r.Header.Get("Content-Type"))
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, "", nil, nil, nil, nil, errors.New("invalid content type")
	}

	switch mediaType {
	case "application/json":
		var req models.SendMessageRequest
		if err := decodeJSONBody(r, &req); err != nil {
			return nil, "", nil, nil, nil, nil, errors.New("invalid JSON body")
		}
		if err := req.Validate(); err != nil {
			return nil, "", nil, nil, nil, nil, err
		}
		if req.Kind == models.MessageKindJSONRender {
			return nil, models.MessageKindJSONRender, req.JSONRenderSpec, nil, nil, nil, nil
		}
		content := strings.TrimSpace(*req.Content)
		return &content, models.MessageKindText, nil, nil, nil, nil, nil
	case "multipart/form-data":
		content, attachments, attachmentURL, attachmentType, err := h.parseMultipartMessagePayload(w, r)
		return content, models.MessageKindText, nil, attachments, attachmentURL, attachmentType, err
	default:
		return nil, "", nil, nil, nil, nil, errors.New("content type must be application/json or multipart/form-data")
	}
}

func (h *messagesHandler) parseMultipartMessagePayload(
	w http.ResponseWriter,
	r *http.Request,
) (*string, []models.MessageAttachment, *string, *models.AttachmentType, error) {
	maxBodyBytes := int64(maxUploadBytes + multipartBodySizeBuffer)
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := r.ParseMultipartForm(maxBodyBytes); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return nil, nil, nil, nil, errRequestEntityTooLarge
		}
		return nil, nil, nil, nil, errors.New("invalid multipart form")
	}

	contentText := strings.TrimSpace(r.FormValue("content"))
	var content *string
	if contentText != "" {
		content = &contentText
	}

	attachments := make([]models.MessageAttachment, 0)
	if r.MultipartForm != nil {
		files := r.MultipartForm.File["attachment"]
		for _, header := range files {
			uploadedFile, err := header.Open()
			if err != nil {
				return nil, nil, nil, nil, errors.New("invalid attachment payload")
			}

			url, attachmentType, saveErr := saveUploadedFile(h.uploadDir, uploadedFile, header)
			_ = uploadedFile.Close()
			if saveErr != nil {
				return nil, nil, nil, nil, saveErr
			}

			attachments = append(attachments, models.MessageAttachment{
				URL:  url,
				Type: attachmentType,
			})
		}
	}

	if len(attachments) == 0 && r.MultipartForm != nil {
		attachmentURLValues := r.MultipartForm.Value["attachment_url"]
		attachmentTypeValues := r.MultipartForm.Value["attachment_type"]
		for index, rawURL := range attachmentURLValues {
			attachmentURLText := strings.TrimSpace(rawURL)
			if attachmentURLText == "" {
				continue
			}

			rawType := ""
			if index < len(attachmentTypeValues) {
				rawType = attachmentTypeValues[index]
			}
			attachmentType, err := parseAttachmentType(rawType)
			if err != nil {
				return nil, nil, nil, nil, err
			}
			attachments = append(attachments, models.MessageAttachment{
				URL:  attachmentURLText,
				Type: attachmentType,
			})
		}
	}

	if len(attachments) == 0 && content == nil {
		return nil, nil, nil, nil, models.ErrMessageContentRequired
	}

	firstAttachmentURL, firstAttachmentType := firstMessageAttachmentPointers(attachments)
	return content, attachments, firstAttachmentURL, firstAttachmentType, nil
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

func firstMessageAttachmentPointers(
	attachments []models.MessageAttachment,
) (*string, *models.AttachmentType) {
	if len(attachments) == 0 {
		return nil, nil
	}

	firstAttachment := attachments[0]
	return &firstAttachment.URL, &firstAttachment.Type
}

func allMessageAttachmentsAreType(
	attachments []models.MessageAttachment,
	attachmentType models.AttachmentType,
) bool {
	if len(attachments) == 0 {
		return false
	}

	for _, attachment := range attachments {
		if attachment.Type != attachmentType {
			return false
		}
	}

	return true
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
