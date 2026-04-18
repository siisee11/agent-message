package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"agent-message/server/models"
	"agent-message/server/realtime"
	"agent-message/server/store"

	"golang.org/x/crypto/bcrypt"
)

func TestAuthRegisterLoginLogoutFlow(t *testing.T) {
	ctx := context.Background()
	router, sqliteStore := newTestRouter(t)

	registerBody := `{"username":"alice","password":"secret123"}`
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
	registerCookies := registerResp.Result().Cookies()
	if len(registerCookies) == 0 || registerCookies[0].Name != defaultSessionCookieName {
		t.Fatalf("expected auth session cookie on register, got %+v", registerCookies)
	}
	if registerResult.User.Username != "alice" {
		t.Fatalf("expected username alice, got %q", registerResult.User.Username)
	}

	storedUser, err := sqliteStore.GetUserByUsername(ctx, "alice")
	if err != nil {
		t.Fatalf("GetUserByUsername() error = %v", err)
	}
	if storedUser.PasswordHash == "secret123" {
		t.Fatalf("expected stored password hash to be hashed")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(storedUser.PasswordHash), []byte("secret123")); err != nil {
		t.Fatalf("expected stored hash to match password: %v", err)
	}

	invalidLoginBody := `{"username":"alice","password":"wrongpass"}`
	invalidLoginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(invalidLoginBody))
	invalidLoginReq.Header.Set("Content-Type", "application/json")
	invalidLoginResp := httptest.NewRecorder()
	router.ServeHTTP(invalidLoginResp, invalidLoginReq)
	if invalidLoginResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected invalid login status %d, got %d", http.StatusUnauthorized, invalidLoginResp.Code)
	}

	loginBody := `{"username":"alice","password":"secret123"}`
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
	loginCookies := loginResp.Result().Cookies()
	if len(loginCookies) == 0 || loginCookies[0].Name != defaultSessionCookieName {
		t.Fatalf("expected auth session cookie on login, got %+v", loginCookies)
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	meReq.AddCookie(loginCookies[0])
	meResp := httptest.NewRecorder()
	router.ServeHTTP(meResp, meReq)
	if meResp.Code != http.StatusOK {
		t.Fatalf("expected cookie-auth me status %d, got %d body=%s", http.StatusOK, meResp.Code, meResp.Body.String())
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
	logoutCookies := logoutResp.Result().Cookies()
	if len(logoutCookies) == 0 || logoutCookies[0].Name != defaultSessionCookieName || logoutCookies[0].MaxAge != -1 {
		t.Fatalf("expected session cookie clear on logout, got %+v", logoutCookies)
	}

	logoutAgainReq := httptest.NewRequest(http.MethodDelete, "/api/auth/logout", nil)
	logoutAgainReq.Header.Set("Authorization", "Bearer "+loginResult.Token)
	logoutAgainResp := httptest.NewRecorder()
	router.ServeHTTP(logoutAgainResp, logoutAgainReq)
	if logoutAgainResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected second logout status %d, got %d", http.StatusUnauthorized, logoutAgainResp.Code)
	}
}

func TestAuthRegisterDuplicateUsername(t *testing.T) {
	router, _ := newTestRouter(t)

	body := `{"username":"alice","password":"secret123"}`
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

func TestAuthChangePasswordFlow(t *testing.T) {
	ctx := context.Background()
	router, sqliteStore := newTestRouter(t)

	registerReq := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(`{"username":"alice","password":"secret123"}`))
	registerReq.Header.Set("Content-Type", "application/json")
	registerResp := httptest.NewRecorder()
	router.ServeHTTP(registerResp, registerReq)
	if registerResp.Code != http.StatusCreated {
		t.Fatalf("expected register status %d, got %d body=%s", http.StatusCreated, registerResp.Code, registerResp.Body.String())
	}

	var registerResult models.AuthResponse
	if err := json.NewDecoder(registerResp.Body).Decode(&registerResult); err != nil {
		t.Fatalf("decode register response: %v", err)
	}

	changeReq := httptest.NewRequest(
		http.MethodPatch,
		"/api/users/me/password",
		bytes.NewBufferString(`{"current_password":"secret123","new_password":"newsecret123"}`),
	)
	changeReq.Header.Set("Authorization", "Bearer "+registerResult.Token)
	changeReq.Header.Set("Content-Type", "application/json")
	changeResp := httptest.NewRecorder()
	router.ServeHTTP(changeResp, changeReq)
	if changeResp.Code != http.StatusNoContent {
		t.Fatalf("expected change password status %d, got %d body=%s", http.StatusNoContent, changeResp.Code, changeResp.Body.String())
	}

	storedUser, err := sqliteStore.GetUserByUsername(ctx, "alice")
	if err != nil {
		t.Fatalf("GetUserByUsername() error = %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(storedUser.PasswordHash), []byte("newsecret123")); err != nil {
		t.Fatalf("expected updated hash to match new password: %v", err)
	}

	oldLoginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"alice","password":"secret123"}`))
	oldLoginReq.Header.Set("Content-Type", "application/json")
	oldLoginResp := httptest.NewRecorder()
	router.ServeHTTP(oldLoginResp, oldLoginReq)
	if oldLoginResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected old password login status %d, got %d", http.StatusUnauthorized, oldLoginResp.Code)
	}

	newLoginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"alice","password":"newsecret123"}`))
	newLoginReq.Header.Set("Content-Type", "application/json")
	newLoginResp := httptest.NewRecorder()
	router.ServeHTTP(newLoginResp, newLoginReq)
	if newLoginResp.Code != http.StatusOK {
		t.Fatalf("expected new password login status %d, got %d body=%s", http.StatusOK, newLoginResp.Code, newLoginResp.Body.String())
	}
}

func TestAuthChangePasswordRejectsWrongCurrentPassword(t *testing.T) {
	router, _ := newTestRouter(t)

	registerReq := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(`{"username":"alice","password":"secret123"}`))
	registerReq.Header.Set("Content-Type", "application/json")
	registerResp := httptest.NewRecorder()
	router.ServeHTTP(registerResp, registerReq)
	if registerResp.Code != http.StatusCreated {
		t.Fatalf("expected register status %d, got %d body=%s", http.StatusCreated, registerResp.Code, registerResp.Body.String())
	}

	var registerResult models.AuthResponse
	if err := json.NewDecoder(registerResp.Body).Decode(&registerResult); err != nil {
		t.Fatalf("decode register response: %v", err)
	}

	changeReq := httptest.NewRequest(
		http.MethodPatch,
		"/api/users/me/password",
		bytes.NewBufferString(`{"current_password":"wrongpass","new_password":"newsecret123"}`),
	)
	changeReq.Header.Set("Authorization", "Bearer "+registerResult.Token)
	changeReq.Header.Set("Content-Type", "application/json")
	changeResp := httptest.NewRecorder()
	router.ServeHTTP(changeResp, changeReq)
	if changeResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected wrong current password status %d, got %d body=%s", http.StatusUnauthorized, changeResp.Code, changeResp.Body.String())
	}
}

func TestAuthRegisterValidation(t *testing.T) {
	router, _ := newTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(`{"username":"alice","password":"123"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(`{"username":"ab","password":"1234"}`))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(`{"username":"alice smith","password":"1234"}`))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
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
		Store:     sqliteStore,
		Hub:       realtime.NewHub(),
		UploadDir: filepath.Join(t.TempDir(), "uploads"),
	})
	return router, sqliteStore
}
