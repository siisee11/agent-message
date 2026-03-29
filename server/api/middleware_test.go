package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddlewarePreflight(t *testing.T) {
	router, _ := newTestRouter(t)

	req := httptest.NewRequest(http.MethodOptions, "/api/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization,Content-Type")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected preflight status %d, got %d", http.StatusNoContent, resp.Code)
	}
	if got := resp.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected allow origin '*', got %q", got)
	}
}

func TestBearerAuthMiddlewareProtectsLogout(t *testing.T) {
	router, _ := newTestRouter(t)

	registerBody := `{"username":"bob","pin":"1234"}`
	registerReq := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerResp := httptest.NewRecorder()
	router.ServeHTTP(registerResp, registerReq)

	if registerResp.Code != http.StatusCreated {
		t.Fatalf("register expected %d, got %d", http.StatusCreated, registerResp.Code)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer not-a-valid-session")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected logout status %d for invalid session, got %d", http.StatusUnauthorized, resp.Code)
	}
}
