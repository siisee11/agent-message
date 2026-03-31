package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"agent-message/server/models"
)

func TestMessagesEndpoints(t *testing.T) {
	router, _ := newTestRouter(t)

	alice := registerAndLoginUser(t, router, "alice", "1234")
	bob := registerAndLoginUser(t, router, "bob", "1234")
	charlie := registerAndLoginUser(t, router, "charlie", "1234")

	conversationID := mustStartConversation(t, router, alice.Token, "bob")

	msg1 := mustSendJSONMessage(t, router, alice.Token, conversationID, `{"content":"hello bob"}`)
	msg2 := mustSendJSONMessage(t, router, bob.Token, conversationID, `{"content":"hi alice"}`)
	msg3 := mustSendMultipartMessage(t, router, alice.Token, conversationID, "photo attached", "photo.png", []byte("fake-image-bytes"))

	if msg3.AttachmentURL == nil || msg3.AttachmentType == nil {
		t.Fatalf("expected multipart message attachment metadata, got %+v", msg3)
	}
	if *msg3.AttachmentType != models.AttachmentTypeFile {
		t.Fatalf("expected attachment type file from multipart upload, got %q", *msg3.AttachmentType)
	}

	t.Run("list messages first page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/conversations/"+conversationID+"/messages?limit=2", nil)
		req.Header.Set("Authorization", "Bearer "+alice.Token)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d body=%s", http.StatusOK, resp.Code, resp.Body.String())
		}

		var messages []models.MessageDetails
		if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
			t.Fatalf("decode message list: %v", err)
		}
		if len(messages) != 2 || messages[0].Message.ID != msg3.ID || messages[1].Message.ID != msg2.ID {
			t.Fatalf("unexpected first page messages: %+v", messages)
		}
	})

	t.Run("list messages with before cursor", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/conversations/"+conversationID+"/messages?before="+msg2.ID+"&limit=20", nil)
		req.Header.Set("Authorization", "Bearer "+bob.Token)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d body=%s", http.StatusOK, resp.Code, resp.Body.String())
		}

		var messages []models.MessageDetails
		if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
			t.Fatalf("decode paginated messages: %v", err)
		}
		if len(messages) != 1 || messages[0].Message.ID != msg1.ID {
			t.Fatalf("unexpected paginated messages: %+v", messages)
		}
	})

	t.Run("list messages rejects outsider", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/conversations/"+conversationID+"/messages", nil)
		req.Header.Set("Authorization", "Bearer "+charlie.Token)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected %d, got %d", http.StatusForbidden, resp.Code)
		}
	})

	t.Run("list messages validates limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/conversations/"+conversationID+"/messages?limit=0", nil)
		req.Header.Set("Authorization", "Bearer "+alice.Token)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d", http.StatusBadRequest, resp.Code)
		}
	})

	t.Run("reject json message with empty content", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/conversations/"+conversationID+"/messages", bytes.NewBufferString(`{"content":"   "}`))
		req.Header.Set("Authorization", "Bearer "+alice.Token)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d", http.StatusBadRequest, resp.Code)
		}
	})

	t.Run("send json_render message with inline spec", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/conversations/"+conversationID+"/messages",
			bytes.NewBufferString(`{"kind":"json_render","json_render_spec":{"root":"stack-1","elements":{"stack-1":{"type":"Stack"}}}}`),
		)
		req.Header.Set("Authorization", "Bearer "+alice.Token)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusCreated {
			t.Fatalf("expected %d, got %d body=%s", http.StatusCreated, resp.Code, resp.Body.String())
		}

		var message models.Message
		if err := json.NewDecoder(resp.Body).Decode(&message); err != nil {
			t.Fatalf("decode json_render message: %v", err)
		}
		if message.Kind != models.MessageKindJSONRender {
			t.Fatalf("expected json_render kind, got %q", message.Kind)
		}
		if string(message.JSONRenderSpec) == "" {
			t.Fatalf("expected json_render_spec in response, got %+v", message)
		}
	})

	t.Run("reject unsupported content type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/conversations/"+conversationID+"/messages", bytes.NewBufferString("hello"))
		req.Header.Set("Authorization", "Bearer "+alice.Token)
		req.Header.Set("Content-Type", "text/plain")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d", http.StatusBadRequest, resp.Code)
		}
	})

	t.Run("reject multipart with unsupported attachment type", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("attachment", "script.sh")
		if err != nil {
			t.Fatalf("create multipart file part: %v", err)
		}
		if _, err := part.Write([]byte("echo hello")); err != nil {
			t.Fatalf("write multipart attachment: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("close multipart writer: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/conversations/"+conversationID+"/messages", &body)
		req.Header.Set("Authorization", "Bearer "+alice.Token)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d", http.StatusBadRequest, resp.Code)
		}
		assertErrorBody(t, resp.Body, "unsupported file type")
	})

	t.Run("toggle reaction add then remove", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/messages/"+msg1.ID+"/reactions", bytes.NewBufferString(`{"emoji":"👍"}`))
		req.Header.Set("Authorization", "Bearer "+alice.Token)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d body=%s", http.StatusOK, resp.Code, resp.Body.String())
		}

		var addResult models.ToggleReactionResult
		if err := json.NewDecoder(resp.Body).Decode(&addResult); err != nil {
			t.Fatalf("decode toggle add response: %v", err)
		}
		if addResult.Action != models.ReactionMutationAdded {
			t.Fatalf("expected add action, got %q", addResult.Action)
		}
		if addResult.Reaction.MessageID != msg1.ID || addResult.Reaction.UserID != alice.User.ID || addResult.Reaction.Emoji != "👍" {
			t.Fatalf("unexpected add reaction payload: %+v", addResult.Reaction)
		}

		req = httptest.NewRequest(http.MethodPost, "/api/messages/"+msg1.ID+"/reactions", bytes.NewBufferString(`{"emoji":"👍"}`))
		req.Header.Set("Authorization", "Bearer "+alice.Token)
		req.Header.Set("Content-Type", "application/json")
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d body=%s", http.StatusOK, resp.Code, resp.Body.String())
		}

		var removeResult models.ToggleReactionResult
		if err := json.NewDecoder(resp.Body).Decode(&removeResult); err != nil {
			t.Fatalf("decode toggle remove response: %v", err)
		}
		if removeResult.Action != models.ReactionMutationRemoved {
			t.Fatalf("expected remove action, got %q", removeResult.Action)
		}
		if removeResult.Reaction.MessageID != msg1.ID || removeResult.Reaction.UserID != alice.User.ID || removeResult.Reaction.Emoji != "👍" {
			t.Fatalf("unexpected remove reaction payload: %+v", removeResult.Reaction)
		}
	})

	t.Run("remove reaction by emoji path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/messages/"+msg1.ID+"/reactions", bytes.NewBufferString(`{"emoji":"🔥"}`))
		req.Header.Set("Authorization", "Bearer "+alice.Token)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d body=%s", http.StatusOK, resp.Code, resp.Body.String())
		}

		req = httptest.NewRequest(http.MethodDelete, "/api/messages/"+msg1.ID+"/reactions/"+url.PathEscape("🔥"), nil)
		req.Header.Set("Authorization", "Bearer "+alice.Token)
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d body=%s", http.StatusOK, resp.Code, resp.Body.String())
		}

		var removed models.Reaction
		if err := json.NewDecoder(resp.Body).Decode(&removed); err != nil {
			t.Fatalf("decode removed reaction: %v", err)
		}
		if removed.MessageID != msg1.ID || removed.UserID != alice.User.ID || removed.Emoji != "🔥" {
			t.Fatalf("unexpected removed reaction payload: %+v", removed)
		}
	})

	t.Run("reject reaction with empty emoji", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/messages/"+msg1.ID+"/reactions", bytes.NewBufferString(`{"emoji":"   "}`))
		req.Header.Set("Authorization", "Bearer "+alice.Token)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d", http.StatusBadRequest, resp.Code)
		}
	})

	t.Run("reject reaction toggle by outsider", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/messages/"+msg1.ID+"/reactions", bytes.NewBufferString(`{"emoji":"🔥"}`))
		req.Header.Set("Authorization", "Bearer "+charlie.Token)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected %d, got %d", http.StatusForbidden, resp.Code)
		}
	})

	t.Run("remove reaction missing returns not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/messages/"+msg1.ID+"/reactions/"+url.PathEscape("💥"), nil)
		req.Header.Set("Authorization", "Bearer "+alice.Token)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected %d, got %d", http.StatusNotFound, resp.Code)
		}
	})

	t.Run("edit own message", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/api/messages/"+msg1.ID, bytes.NewBufferString(`{"content":"hello edited"}`))
		req.Header.Set("Authorization", "Bearer "+alice.Token)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d body=%s", http.StatusOK, resp.Code, resp.Body.String())
		}

		var edited models.Message
		if err := json.NewDecoder(resp.Body).Decode(&edited); err != nil {
			t.Fatalf("decode edited message: %v", err)
		}
		if edited.Content == nil || *edited.Content != "hello edited" || !edited.Edited {
			t.Fatalf("unexpected edited message: %+v", edited)
		}
	})

	t.Run("reject edit of others message", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/api/messages/"+msg1.ID, bytes.NewBufferString(`{"content":"nope"}`))
		req.Header.Set("Authorization", "Bearer "+bob.Token)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected %d, got %d", http.StatusForbidden, resp.Code)
		}
	})

	t.Run("soft delete own message", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/messages/"+msg2.ID, nil)
		req.Header.Set("Authorization", "Bearer "+bob.Token)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d body=%s", http.StatusOK, resp.Code, resp.Body.String())
		}

		var deleted models.Message
		if err := json.NewDecoder(resp.Body).Decode(&deleted); err != nil {
			t.Fatalf("decode deleted message: %v", err)
		}
		if !deleted.Deleted || deleted.Content != nil || deleted.AttachmentURL != nil || deleted.AttachmentType != nil {
			t.Fatalf("unexpected deleted message payload: %+v", deleted)
		}
	})

	t.Run("reject delete of others message", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/messages/"+msg2.ID, nil)
		req.Header.Set("Authorization", "Bearer "+alice.Token)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected %d, got %d", http.StatusForbidden, resp.Code)
		}
	})

	t.Run("edit not found message", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/api/messages/does-not-exist", bytes.NewBufferString(`{"content":"x"}`))
		req.Header.Set("Authorization", "Bearer "+alice.Token)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected %d, got %d", http.StatusNotFound, resp.Code)
		}
	})

	t.Run("delete not found message", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/messages/does-not-exist", nil)
		req.Header.Set("Authorization", "Bearer "+alice.Token)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected %d, got %d", http.StatusNotFound, resp.Code)
		}
	})
}

func mustStartConversation(t *testing.T, router http.Handler, token, username string) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/conversations", bytes.NewBufferString(`{"username":"`+username+`"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated && resp.Code != http.StatusOK {
		t.Fatalf("start conversation expected %d or %d, got %d body=%s", http.StatusCreated, http.StatusOK, resp.Code, resp.Body.String())
	}

	var details models.ConversationDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		t.Fatalf("decode conversation details: %v", err)
	}
	if details.Conversation.ID == "" {
		t.Fatalf("expected conversation id in response")
	}
	return details.Conversation.ID
}

func mustSendJSONMessage(t *testing.T, router http.Handler, token, conversationID, body string) models.Message {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/conversations/"+conversationID+"/messages", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("send json message expected %d, got %d body=%s", http.StatusCreated, resp.Code, resp.Body.String())
	}

	var message models.Message
	if err := json.NewDecoder(resp.Body).Decode(&message); err != nil {
		t.Fatalf("decode message: %v", err)
	}
	return message
}

func mustSendMultipartMessage(t *testing.T, router http.Handler, token, conversationID, content, filename string, data []byte) models.Message {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("content", content); err != nil {
		t.Fatalf("write multipart content field: %v", err)
	}
	part, err := writer.CreateFormFile("attachment", filename)
	if err != nil {
		t.Fatalf("create multipart file part: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("write multipart attachment: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/conversations/"+conversationID+"/messages", &body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("send multipart message expected %d, got %d body=%s", http.StatusCreated, resp.Code, resp.Body.String())
	}

	var message models.Message
	if err := json.NewDecoder(resp.Body).Decode(&message); err != nil {
		t.Fatalf("decode multipart message: %v", err)
	}
	return message
}
