package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"agent-message/server/models"
)

func TestCatalogPromptEndpointReturnsPromptWithoutAuth(t *testing.T) {
	t.Parallel()

	router, _ := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/catalog/prompt", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if got, want := resp.Code, http.StatusOK; got != want {
		t.Fatalf("status mismatch: got %d want %d body=%s", got, want, resp.Body.String())
	}
	if contentType := resp.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("expected application/json content type, got %q", contentType)
	}

	var payload models.CatalogPromptResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if strings.TrimSpace(payload.Prompt) == "" {
		t.Fatalf("expected non-empty prompt")
	}
	if !strings.Contains(payload.Prompt, "OUTPUT FORMAT") {
		t.Fatalf("expected generated catalog prompt content, got %q", payload.Prompt)
	}
}

func TestCatalogPromptEndpointRejectsNonGetMethods(t *testing.T) {
	t.Parallel()

	router, _ := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/api/catalog/prompt", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if got, want := resp.Code, http.StatusMethodNotAllowed; got != want {
		t.Fatalf("status mismatch: got %d want %d body=%s", got, want, resp.Body.String())
	}
	if allow := resp.Header().Get("Allow"); allow != http.MethodGet {
		t.Fatalf("allow header mismatch: got %q want %q", allow, http.MethodGet)
	}
	assertErrorBody(t, resp.Body, "method not allowed")
}
