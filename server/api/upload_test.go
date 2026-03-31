package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"agent-message/server/models"
)

func TestUploadEndpointAndStaticServing(t *testing.T) {
	router, _ := newTestRouter(t)
	alice := registerAndLoginUser(t, router, "alice", "1234")

	t.Run("requires auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/upload", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected %d, got %d", http.StatusUnauthorized, resp.Code)
		}
	})

	t.Run("rejects missing file", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		if err := writer.Close(); err != nil {
			t.Fatalf("close multipart writer: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/upload", &body)
		req.Header.Set("Authorization", "Bearer "+alice.Token)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d", http.StatusBadRequest, resp.Code)
		}
	})

	t.Run("rejects oversized file", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("file", "too-large.bin")
		if err != nil {
			t.Fatalf("create file form part: %v", err)
		}
		if _, err := part.Write(bytes.Repeat([]byte("a"), maxUploadBytes+1)); err != nil {
			t.Fatalf("write upload payload: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("close multipart writer: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/upload", &body)
		req.Header.Set("Authorization", "Bearer "+alice.Token)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected %d, got %d", http.StatusRequestEntityTooLarge, resp.Code)
		}
	})

	t.Run("uploads file and serves statically", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("file", "sample.txt")
		if err != nil {
			t.Fatalf("create file form part: %v", err)
		}
		if _, err := part.Write([]byte("hello-upload")); err != nil {
			t.Fatalf("write upload payload: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("close multipart writer: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/upload", &body)
		req.Header.Set("Authorization", "Bearer "+alice.Token)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusCreated {
			t.Fatalf("expected upload status %d, got %d body=%s", http.StatusCreated, resp.Code, resp.Body.String())
		}

		var uploadResp models.UploadResponse
		if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
			t.Fatalf("decode upload response: %v", err)
		}
		if !strings.HasPrefix(uploadResp.URL, staticUploadsPrefix) {
			t.Fatalf("expected upload URL to start with %q, got %q", staticUploadsPrefix, uploadResp.URL)
		}

		staticReq := httptest.NewRequest(http.MethodGet, uploadResp.URL, nil)
		staticResp := httptest.NewRecorder()
		router.ServeHTTP(staticResp, staticReq)

		if staticResp.Code != http.StatusOK {
			t.Fatalf("expected static file status %d, got %d body=%s", http.StatusOK, staticResp.Code, staticResp.Body.String())
		}
		if body := staticResp.Body.String(); body != "hello-upload" {
			t.Fatalf("unexpected static file body %q", body)
		}
	})

	t.Run("rejects unsupported file type", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("file", "script.sh")
		if err != nil {
			t.Fatalf("create file form part: %v", err)
		}
		if _, err := part.Write([]byte("echo hello")); err != nil {
			t.Fatalf("write upload payload: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("close multipart writer: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/upload", &body)
		req.Header.Set("Authorization", "Bearer "+alice.Token)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d", http.StatusBadRequest, resp.Code)
		}
		assertErrorBody(t, resp.Body, "unsupported file type")
	})
}
