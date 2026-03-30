package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"agent-messenger/cli/internal/api"
	"agent-messenger/cli/internal/config"
)

func TestRunRegisterStoresTokenInConfig(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "", func(req *http.Request, body []byte) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if req.URL.Path != "/api/auth/register" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}

		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode register payload: %v", err)
		}
		if payload["username"] != "alice" || payload["pin"] != "1234" {
			t.Fatalf("unexpected register payload: %+v", payload)
		}

		return jsonResponse(http.StatusCreated, `{"token":"reg-token","user":{"id":"u1","username":"alice","created_at":"2026-01-01T00:00:00Z"}}`), nil
	})

	if err := runRegister(rt, "alice", "1234"); err != nil {
		t.Fatalf("runRegister: %v", err)
	}

	if got, want := rt.Config.Token, "reg-token"; got != want {
		t.Fatalf("token mismatch: got %q want %q", got, want)
	}
	if got, want := strings.TrimSpace(stdout.String()), "registered alice"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}

	persisted, err := rt.ConfigStore.Load()
	if err != nil {
		t.Fatalf("load persisted config: %v", err)
	}
	if got, want := persisted.Token, "reg-token"; got != want {
		t.Fatalf("persisted token mismatch: got %q want %q", got, want)
	}
}

func TestRunLoginStoresTokenInConfig(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "", func(req *http.Request, body []byte) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if req.URL.Path != "/api/auth/login" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}

		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode login payload: %v", err)
		}
		if payload["username"] != "alice" || payload["pin"] != "1234" {
			t.Fatalf("unexpected login payload: %+v", payload)
		}

		return jsonResponse(http.StatusOK, `{"token":"login-token","user":{"id":"u1","username":"alice","created_at":"2026-01-01T00:00:00Z"}}`), nil
	})

	if err := runLogin(rt, "alice", "1234"); err != nil {
		t.Fatalf("runLogin: %v", err)
	}

	if got, want := rt.Config.Token, "login-token"; got != want {
		t.Fatalf("token mismatch: got %q want %q", got, want)
	}
	if got, want := strings.TrimSpace(stdout.String()), "logged in as alice"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}

	persisted, err := rt.ConfigStore.Load()
	if err != nil {
		t.Fatalf("load persisted config: %v", err)
	}
	if got, want := persisted.Token, "login-token"; got != want {
		t.Fatalf("persisted token mismatch: got %q want %q", got, want)
	}
}

func TestRunLogoutClearsTokenEvenWhenServerFails(t *testing.T) {
	t.Parallel()

	rt, stdout, stderr := newTestRuntime(t, "http://example.test", "stale-token", func(req *http.Request, _ []byte) (*http.Response, error) {
		if req.Method != http.MethodDelete {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if req.URL.Path != "/api/auth/logout" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		if got, want := req.Header.Get("Authorization"), "Bearer stale-token"; got != want {
			t.Fatalf("authorization mismatch: got %q want %q", got, want)
		}
		return jsonResponse(http.StatusInternalServerError, `{"error":"failed to logout"}`), nil
	})

	if err := runLogout(rt); err != nil {
		t.Fatalf("runLogout: %v", err)
	}

	if got := rt.Config.Token; got != "" {
		t.Fatalf("expected runtime token to be cleared, got %q", got)
	}
	if got := strings.TrimSpace(stdout.String()); got != "logged out" {
		t.Fatalf("stdout mismatch: got %q", got)
	}
	if got := stderr.String(); !strings.Contains(got, "warning: server logout failed") {
		t.Fatalf("expected logout warning, got %q", got)
	}

	persisted, err := rt.ConfigStore.Load()
	if err != nil {
		t.Fatalf("load persisted config: %v", err)
	}
	if got := persisted.Token; got != "" {
		t.Fatalf("expected persisted token to be cleared, got %q", got)
	}
}

func TestRunWhoAmIReturnsCurrentUsername(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "whoami-token", func(req *http.Request, _ []byte) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if req.URL.Path != "/api/users/me" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		if got, want := req.Header.Get("Authorization"), "Bearer whoami-token"; got != want {
			t.Fatalf("authorization mismatch: got %q want %q", got, want)
		}

		return jsonResponse(http.StatusOK, `{"id":"u1","username":"alice","created_at":"2026-01-01T00:00:00Z"}`), nil
	})

	if err := runWhoAmI(rt); err != nil {
		t.Fatalf("runWhoAmI: %v", err)
	}
	if got, want := strings.TrimSpace(stdout.String()), "alice"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}

func TestRunWhoAmIRequiresLogin(t *testing.T) {
	t.Parallel()

	rt, _, _ := newTestRuntime(t, "http://example.test", "", func(req *http.Request, _ []byte) (*http.Response, error) {
		t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		return nil, nil
	})

	err := runWhoAmI(rt)
	if err == nil {
		t.Fatalf("expected error")
	}
	if got := err.Error(); !strings.Contains(got, "not logged in") {
		t.Fatalf("unexpected error: %q", got)
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestRuntime(
	t *testing.T,
	serverURL string,
	token string,
	transport func(req *http.Request, body []byte) (*http.Response, error),
) (*Runtime, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()

	store := config.NewStore(filepath.Join(t.TempDir(), "config"))
	cfg := config.Config{
		ServerURL: serverURL,
		Token:     token,
	}

	if err := store.Save(cfg); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	client, err := api.NewClient(serverURL, token)
	if err != nil {
		t.Fatalf("create api client: %v", err)
	}
	client.SetHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var body []byte
		if req.Body != nil {
			var err error
			body, err = io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}
			_ = req.Body.Close()
		}
		return transport(req, body)
	})})

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	return &Runtime{
		ConfigStore: store,
		Config:      cfg,
		Client:      client,
		Stdout:      stdout,
		Stderr:      stderr,
	}, stdout, stderr
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}
}
