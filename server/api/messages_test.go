package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-messenger/server/models"
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
