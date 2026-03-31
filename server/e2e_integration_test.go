package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"agent-message/server/api"
	"agent-message/server/models"
	"agent-message/server/realtime"
	"agent-message/server/store"
)

func TestServerStackE2EHappyPath(t *testing.T) {
	testServer := newE2EServer(t)
	defer testServer.Close()

	alice := mustRegisterViaAPI(t, testServer.URL, "alice", "1234")
	bob := mustRegisterViaAPI(t, testServer.URL, "bob", "5678")

	conversation := mustStartConversationViaAPI(t, testServer.URL, alice.Token, "bob")
	message := mustSendMessageViaAPI(t, testServer.URL, alice.Token, conversation.Conversation.ID, "hello bob from e2e")
	mustToggleReactionViaAPI(t, testServer.URL, bob.Token, message.ID, "👍")

	messages := mustListMessagesViaAPI(t, testServer.URL, bob.Token, conversation.Conversation.ID)
	if len(messages) == 0 {
		t.Fatalf("expected at least one message in conversation %q", conversation.Conversation.ID)
	}
	if messages[0].Message.ID != message.ID {
		t.Fatalf("expected latest message %q, got %q", message.ID, messages[0].Message.ID)
	}

	uploadURL := mustUploadFileViaAPI(t, testServer.URL, alice.Token, "note.txt", []byte("hello upload"))
	resp, err := http.Get(testServer.URL + uploadURL)
	if err != nil {
		t.Fatalf("get uploaded file: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected uploaded file status %d, got %d", http.StatusOK, resp.StatusCode)
	}
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read uploaded file body: %v", err)
	}
	if string(payload) != "hello upload" {
		t.Fatalf("unexpected uploaded file payload %q", string(payload))
	}
}

func TestServerStackE2EValidationErrors(t *testing.T) {
	testServer := newE2EServer(t)
	defer testServer.Close()

	t.Run("register rejects invalid username", func(t *testing.T) {
		status, body := doJSONRequest(t, http.MethodPost, testServer.URL+"/api/auth/register", "", map[string]string{
			"username": "ab",
			"pin":      "1234",
		})
		if status != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d", http.StatusBadRequest, status)
		}
		assertErrorEnvelope(t, body, models.ErrUsernameLength.Error())
	})

	t.Run("register rejects invalid pin", func(t *testing.T) {
		status, body := doJSONRequest(t, http.MethodPost, testServer.URL+"/api/auth/register", "", map[string]string{
			"username": "charlie",
			"pin":      "12x4",
		})
		if status != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d", http.StatusBadRequest, status)
		}
		assertErrorEnvelope(t, body, models.ErrPINInvalid.Error())
	})

	alice := mustRegisterViaAPI(t, testServer.URL, "alice", "1234")
	bob := mustRegisterViaAPI(t, testServer.URL, "bob", "5678")
	conversation := mustStartConversationViaAPI(t, testServer.URL, alice.Token, "bob")

	t.Run("user search rejects invalid query", func(t *testing.T) {
		status, body := doJSONRequest(t, http.MethodGet, testServer.URL+"/api/users?username=bo%20x", alice.Token, nil)
		if status != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d", http.StatusBadRequest, status)
		}
		assertErrorEnvelope(t, body, models.ErrUsernameInvalid.Error())
	})

	t.Run("upload rejects unsupported file type", func(t *testing.T) {
		status, body := doMultipartUploadRequest(t, testServer.URL+"/api/upload", alice.Token, "script.sh", []byte("echo hello"))
		if status != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d", http.StatusBadRequest, status)
		}
		assertErrorEnvelope(t, body, "unsupported file type")
	})

	t.Run("message multipart rejects unsupported attachment type", func(t *testing.T) {
		endpoint := testServer.URL + "/api/conversations/" + conversation.Conversation.ID + "/messages"
		status, body := doMultipartAttachmentMessageRequest(t, endpoint, alice.Token, "script.sh", []byte("echo hello"))
		if status != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d", http.StatusBadRequest, status)
		}
		assertErrorEnvelope(t, body, "unsupported file type")
	})

	_ = bob
}

func newE2EServer(t *testing.T) *httptest.Server {
	t.Helper()

	sqliteStore, err := store.NewSQLiteStore(context.Background(), filepath.Join(t.TempDir(), "e2e.sqlite"))
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	t.Cleanup(func() {
		_ = sqliteStore.Close()
	})

	router := api.NewRouter(api.Dependencies{
		Store:     sqliteStore,
		Hub:       realtime.NewHub(),
		UploadDir: filepath.Join(t.TempDir(), "uploads"),
	})
	return httptest.NewServer(router)
}

func mustRegisterViaAPI(t *testing.T, baseURL, username, pin string) models.AuthResponse {
	t.Helper()

	status, body := doJSONRequest(t, http.MethodPost, baseURL+"/api/auth/register", "", map[string]string{
		"username": username,
		"pin":      pin,
	})
	if status != http.StatusCreated {
		t.Fatalf("register %q expected %d, got %d body=%s", username, http.StatusCreated, status, body)
	}

	var resp models.AuthResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	if resp.Token == "" {
		t.Fatalf("register token empty for user %q", username)
	}
	return resp
}

func mustStartConversationViaAPI(t *testing.T, baseURL, token, username string) models.ConversationDetails {
	t.Helper()

	status, body := doJSONRequest(t, http.MethodPost, baseURL+"/api/conversations", token, map[string]string{
		"username": username,
	})
	if status != http.StatusCreated && status != http.StatusOK {
		t.Fatalf("start conversation expected %d or %d, got %d body=%s", http.StatusCreated, http.StatusOK, status, body)
	}

	var details models.ConversationDetails
	if err := json.Unmarshal([]byte(body), &details); err != nil {
		t.Fatalf("decode conversation response: %v", err)
	}
	if details.Conversation.ID == "" {
		t.Fatal("conversation id empty")
	}
	return details
}

func mustSendMessageViaAPI(t *testing.T, baseURL, token, conversationID, content string) models.Message {
	t.Helper()

	status, body := doJSONRequest(t, http.MethodPost, baseURL+"/api/conversations/"+conversationID+"/messages", token, map[string]string{
		"content": content,
	})
	if status != http.StatusCreated {
		t.Fatalf("send message expected %d, got %d body=%s", http.StatusCreated, status, body)
	}

	var message models.Message
	if err := json.Unmarshal([]byte(body), &message); err != nil {
		t.Fatalf("decode message response: %v", err)
	}
	if message.ID == "" {
		t.Fatal("message id empty")
	}
	return message
}

func mustToggleReactionViaAPI(t *testing.T, baseURL, token, messageID, emoji string) {
	t.Helper()

	status, body := doJSONRequest(t, http.MethodPost, baseURL+"/api/messages/"+messageID+"/reactions", token, map[string]string{
		"emoji": emoji,
	})
	if status != http.StatusOK {
		t.Fatalf("toggle reaction expected %d, got %d body=%s", http.StatusOK, status, body)
	}

	var result models.ToggleReactionResult
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Fatalf("decode reaction response: %v", err)
	}
	if result.Action != models.ReactionMutationAdded {
		t.Fatalf("expected reaction action %q, got %q", models.ReactionMutationAdded, result.Action)
	}
}

func mustListMessagesViaAPI(t *testing.T, baseURL, token, conversationID string) []models.MessageDetails {
	t.Helper()

	status, body := doJSONRequest(t, http.MethodGet, baseURL+"/api/conversations/"+conversationID+"/messages?limit=20", token, nil)
	if status != http.StatusOK {
		t.Fatalf("list messages expected %d, got %d body=%s", http.StatusOK, status, body)
	}

	var messages []models.MessageDetails
	if err := json.Unmarshal([]byte(body), &messages); err != nil {
		t.Fatalf("decode messages response: %v", err)
	}
	return messages
}

func mustUploadFileViaAPI(t *testing.T, baseURL, token, filename string, payload []byte) string {
	t.Helper()

	status, body := doMultipartUploadRequest(t, baseURL+"/api/upload", token, filename, payload)
	if status != http.StatusCreated {
		t.Fatalf("upload expected %d, got %d body=%s", http.StatusCreated, status, body)
	}

	var resp models.UploadResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}
	if !strings.HasPrefix(resp.URL, "/static/uploads/") {
		t.Fatalf("expected upload URL prefix /static/uploads/, got %q", resp.URL)
	}
	return resp.URL
}

func doJSONRequest(t *testing.T, method, endpoint, token string, body any) (int, string) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, endpoint, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return resp.StatusCode, string(payload)
}

func doMultipartUploadRequest(t *testing.T, endpoint, token, filename string, payload []byte) (int, string) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(payload); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, &body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return resp.StatusCode, string(respBody)
}

func doMultipartAttachmentMessageRequest(t *testing.T, endpoint, token, filename string, payload []byte) (int, string) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("attachment", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(payload); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, &body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return resp.StatusCode, string(respBody)
}

func assertErrorEnvelope(t *testing.T, body, expected string) {
	t.Helper()

	var payload map[string]string
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("decode error envelope: %v (body=%s)", err, body)
	}
	if payload["error"] != expected {
		t.Fatalf("expected error %q, got %q (body=%s)", expected, payload["error"], body)
	}
}
