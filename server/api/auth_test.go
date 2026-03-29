package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"agent-messenger/server/models"
	"agent-messenger/server/store"
	"agent-messenger/server/ws"

	"golang.org/x/crypto/bcrypt"
)

func TestAuthRegisterLoginLogoutFlow(t *testing.T) {
	ctx := context.Background()
	router, sqliteStore := newTestRouter(t)

	registerBody := `{"username":"alice","pin":"1234"}`
	registerReq := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerResp := httptest.NewRecorder()

	router.ServeHTTP(registerResp, registerReq)
	if registerResp.Code != http.StatusCreated {
		t.Fatalf("expected register status %d, got %d, body=%s", http.StatusCreated, registerResp.Code, registerResp.Body.String())
	}

	var registerResult models.AuthResponse
	if err := json.NewDecoder(registerResp.Body).Decode(&registerResult); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	if registerResult.Token == "" {
		t.Fatalf("register token should not be empty")
	}
	if registerResult.User.Username != "alice" {
		t.Fatalf("expected username alice, got %q", registerResult.User.Username)
	}

	storedUser, err := sqliteStore.GetUserByUsername(ctx, "alice")
	if err != nil {
		t.Fatalf("GetUserByUsername() error = %v", err)
	}
	if storedUser.PINHash == "1234" {
		t.Fatalf("expected stored pin hash to be hashed")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(storedUser.PINHash), []byte("1234")); err != nil {
		t.Fatalf("expected stored hash to match pin: %v", err)
	}

	invalidLoginBody := `{"username":"alice","pin":"9999"}`
	invalidLoginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(invalidLoginBody))
	invalidLoginReq.Header.Set("Content-Type", "application/json")
	invalidLoginResp := httptest.NewRecorder()
	router.ServeHTTP(invalidLoginResp, invalidLoginReq)
	if invalidLoginResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected invalid login status %d, got %d", http.StatusUnauthorized, invalidLoginResp.Code)
	}

	loginBody := `{"username":"alice","pin":"1234"}`
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp := httptest.NewRecorder()
	router.ServeHTTP(loginResp, loginReq)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected login status %d, got %d, body=%s", http.StatusOK, loginResp.Code, loginResp.Body.String())
	}

	var loginResult models.AuthResponse
	if err := json.NewDecoder(loginResp.Body).Decode(&loginResult); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if loginResult.Token == "" {
		t.Fatalf("login token should not be empty")
	}

	logoutWithoutBearerReq := httptest.NewRequest(http.MethodDelete, "/api/auth/logout", nil)
	logoutWithoutBearerResp := httptest.NewRecorder()
	router.ServeHTTP(logoutWithoutBearerResp, logoutWithoutBearerReq)
	if logoutWithoutBearerResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected logout without bearer status %d, got %d", http.StatusUnauthorized, logoutWithoutBearerResp.Code)
	}

	logoutReq := httptest.NewRequest(http.MethodDelete, "/api/auth/logout", nil)
	logoutReq.Header.Set("Authorization", "Bearer "+loginResult.Token)
	logoutResp := httptest.NewRecorder()
	router.ServeHTTP(logoutResp, logoutReq)
	if logoutResp.Code != http.StatusNoContent {
		t.Fatalf("expected logout status %d, got %d", http.StatusNoContent, logoutResp.Code)
	}

	logoutAgainReq := httptest.NewRequest(http.MethodDelete, "/api/auth/logout", nil)
	logoutAgainReq.Header.Set("Authorization", "Bearer "+loginResult.Token)
	logoutAgainResp := httptest.NewRecorder()
	router.ServeHTTP(logoutAgainResp, logoutAgainReq)
	if logoutAgainResp.Code != http.StatusNoContent {
		t.Fatalf("expected idempotent logout status %d, got %d", http.StatusNoContent, logoutAgainResp.Code)
	}
}

func TestAuthRegisterDuplicateUsername(t *testing.T) {
	router, _ := newTestRouter(t)

	body := `{"username":"alice","pin":"1234"}`
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if i == 0 && resp.Code != http.StatusCreated {
			t.Fatalf("first register expected %d, got %d", http.StatusCreated, resp.Code)
		}
		if i == 1 && resp.Code != http.StatusConflict {
			t.Fatalf("second register expected %d, got %d", http.StatusConflict, resp.Code)
		}
	}
}

func TestAuthRegisterValidation(t *testing.T) {
	router, _ := newTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(`{"username":"alice","pin":"12"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.Code)
	}
}

func newTestRouter(t *testing.T) (http.Handler, *store.SQLiteStore) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "api.sqlite")
	sqliteStore, err := store.NewSQLiteStore(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	t.Cleanup(func() {
		_ = sqliteStore.Close()
	})

	router := NewRouter(Dependencies{
		Store: sqliteStore,
		Hub:   ws.NewHub(),
	})
	return router, sqliteStore
}
