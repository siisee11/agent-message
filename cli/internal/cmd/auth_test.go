package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"agent-message/cli/internal/api"
	"agent-message/cli/internal/config"
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
		if payload["account_id"] != "alice" || payload["password"] != "secret123" {
			t.Fatalf("unexpected register payload: %+v", payload)
		}

		return jsonResponse(http.StatusCreated, `{"token":"reg-token","user":{"id":"u1","account_id":"alice","username":"alice","created_at":"2026-01-01T00:00:00Z"}}`), nil
	})

	if err := runRegister(rt, "alice", "secret123"); err != nil {
		t.Fatalf("runRegister: %v", err)
	}

	if got, want := rt.Config.Token, "reg-token"; got != want {
		t.Fatalf("token mismatch: got %q want %q", got, want)
	}
	if got, want := rt.Config.ActiveProfile, "alice"; got != want {
		t.Fatalf("active profile mismatch: got %q want %q", got, want)
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
	if got, want := persisted.ActiveProfile, "alice"; got != want {
		t.Fatalf("persisted active profile mismatch: got %q want %q", got, want)
	}
	if got, want := persisted.ServerURL, "http://example.test"; got != want {
		t.Fatalf("persisted configured server_url mismatch: got %q want %q", got, want)
	}
	if got, want := persisted.ActiveProfileServerURL, "http://example.test"; got != want {
		t.Fatalf("persisted active profile server_url mismatch: got %q want %q", got, want)
	}
	if got, want := persisted.Profiles["alice"].Token, "reg-token"; got != want {
		t.Fatalf("persisted profile token mismatch: got %q want %q", got, want)
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
		if payload["account_id"] != "alice" || payload["password"] != "secret123" {
			t.Fatalf("unexpected login payload: %+v", payload)
		}

		return jsonResponse(http.StatusOK, `{"token":"login-token","user":{"id":"u1","account_id":"alice","username":"alice","created_at":"2026-01-01T00:00:00Z"}}`), nil
	})

	if err := runLogin(rt, "alice", "secret123"); err != nil {
		t.Fatalf("runLogin: %v", err)
	}

	if got, want := rt.Config.Token, "login-token"; got != want {
		t.Fatalf("token mismatch: got %q want %q", got, want)
	}
	if got, want := rt.Config.ActiveProfile, "alice"; got != want {
		t.Fatalf("active profile mismatch: got %q want %q", got, want)
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
	if got, want := persisted.ActiveProfile, "alice"; got != want {
		t.Fatalf("persisted active profile mismatch: got %q want %q", got, want)
	}
	if got, want := persisted.ServerURL, "http://example.test"; got != want {
		t.Fatalf("persisted configured server_url mismatch: got %q want %q", got, want)
	}
	if got, want := persisted.ActiveProfileServerURL, "http://example.test"; got != want {
		t.Fatalf("persisted active profile server_url mismatch: got %q want %q", got, want)
	}
	if got, want := persisted.Profiles["alice"].Token, "login-token"; got != want {
		t.Fatalf("persisted profile token mismatch: got %q want %q", got, want)
	}
}

func TestRegisterCommandSupportsRawPayload(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "", func(req *http.Request, body []byte) (*http.Response, error) {
		if req.URL.Path != "/api/auth/register" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode register payload: %v", err)
		}
		if got, want := payload["account_id"], "alice"; got != want {
			t.Fatalf("account_id mismatch: got %q want %q", got, want)
		}
		if got, want := payload["password"], "secret123"; got != want {
			t.Fatalf("password mismatch: got %q want %q", got, want)
		}
		return jsonResponse(http.StatusCreated, `{"token":"reg-token","user":{"id":"u1","account_id":"alice","username":"alice","created_at":"2026-01-01T00:00:00Z"}}`), nil
	})

	command := newRegisterCommand(rt)
	command.SetArgs([]string{"--payload", `{"account_id":"alice","password":"secret123"}`})

	if err := command.Execute(); err != nil {
		t.Fatalf("execute register command: %v", err)
	}
	if got, want := strings.TrimSpace(stdout.String()), "registered alice"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}

func TestRunLoginPreservesGlobalMaster(t *testing.T) {
	t.Parallel()

	rt, _, _ := newTestRuntime(t, "http://example.test", "", func(req *http.Request, body []byte) (*http.Response, error) {
		if req.URL.Path != "/api/auth/login" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode login payload: %v", err)
		}
		if got, want := payload["account_id"], "alice"; got != want {
			t.Fatalf("account_id mismatch: got %q want %q", got, want)
		}
		return jsonResponse(http.StatusOK, `{"token":"login-token","user":{"id":"u1","account_id":"alice","username":"alice","created_at":"2026-01-01T00:00:00Z"}}`), nil
	})
	rt.Config.Master = "jay"
	rt.Config.Profiles = map[string]config.Profile{
		"alice": {
			Username:  "alice",
			ServerURL: "http://example.test",
		},
	}
	if err := rt.ConfigStore.Save(rt.Config); err != nil {
		t.Fatalf("seed profiles: %v", err)
	}

	if err := runLogin(rt, "alice", "secret123"); err != nil {
		t.Fatalf("runLogin: %v", err)
	}
	if got, want := rt.Config.Master, "jay"; got != want {
		t.Fatalf("master mismatch: got %q want %q", got, want)
	}
}

func TestRunOnboardLogsInAndSetsMaster(t *testing.T) {
	t.Parallel()

	requests := make([]string, 0, 2)
	rt, stdout, _ := newTestRuntime(t, "http://example.test", "", func(req *http.Request, body []byte) (*http.Response, error) {
		requests = append(requests, req.URL.Path)
		if req.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", req.Method)
		}

		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if got, want := payload["account_id"], "alice"; got != want {
			t.Fatalf("account_id mismatch: got %q want %q", got, want)
		}
		if got, want := payload["password"], "secret123"; got != want {
			t.Fatalf("password mismatch: got %q want %q", got, want)
		}

		switch req.URL.Path {
		case "/api/auth/login":
			return jsonResponse(http.StatusOK, `{"token":"login-token","user":{"id":"u1","account_id":"alice","username":"alice","created_at":"2026-01-01T00:00:00Z"}}`), nil
		default:
			t.Fatalf("unexpected path: %s", req.URL.Path)
			return nil, nil
		}
	})
	rt.Config.Profiles = map[string]config.Profile{
		"alice": {
			Username:  "alice",
			ServerURL: "http://example.test",
		},
	}
	if err := rt.ConfigStore.Save(rt.Config); err != nil {
		t.Fatalf("seed profiles: %v", err)
	}
	rt.Stdin = strings.NewReader("alice\nsecret123\n")

	if err := runOnboard(rt); err != nil {
		t.Fatalf("runOnboard: %v", err)
	}

	if got, want := requests, []string{"/api/auth/login"}; !slices.Equal(got, want) {
		t.Fatalf("request path sequence mismatch: got %v want %v", got, want)
	}
	if got, want := rt.Config.Token, "login-token"; got != want {
		t.Fatalf("token mismatch: got %q want %q", got, want)
	}
	if got, want := rt.Config.ActiveProfile, "alice"; got != want {
		t.Fatalf("active profile mismatch: got %q want %q", got, want)
	}
	if got, want := rt.Config.Master, "alice"; got != want {
		t.Fatalf("master mismatch: got %q want %q", got, want)
	}
	if got, want := stdout.String(), "account_id: password: onboarded alice\n"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}

	persisted, err := rt.ConfigStore.Load()
	if err != nil {
		t.Fatalf("load persisted config: %v", err)
	}
	if got, want := persisted.Master, "alice"; got != want {
		t.Fatalf("persisted master mismatch: got %q want %q", got, want)
	}
}

func TestRunOnboardRegistersWhenLoginReturnsUnauthorized(t *testing.T) {
	t.Parallel()

	requests := make([]string, 0, 2)
	rt, stdout, _ := newTestRuntime(t, "http://example.test", "", func(req *http.Request, body []byte) (*http.Response, error) {
		requests = append(requests, req.URL.Path)
		if req.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", req.Method)
		}

		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if got, want := payload["account_id"], "alice"; got != want {
			t.Fatalf("account_id mismatch: got %q want %q", got, want)
		}
		if got, want := payload["password"], "secret123"; got != want {
			t.Fatalf("password mismatch: got %q want %q", got, want)
		}

		switch req.URL.Path {
		case "/api/auth/login":
			return jsonResponse(http.StatusUnauthorized, `{"error":"invalid credentials"}`), nil
		case "/api/auth/register":
			return jsonResponse(http.StatusCreated, `{"token":"reg-token","user":{"id":"u1","account_id":"alice","username":"alice","created_at":"2026-01-01T00:00:00Z"}}`), nil
		default:
			t.Fatalf("unexpected path: %s", req.URL.Path)
			return nil, nil
		}
	})
	rt.Stdin = strings.NewReader("alice\nsecret123\n")

	if err := runOnboard(rt); err != nil {
		t.Fatalf("runOnboard: %v", err)
	}

	if got, want := requests, []string{"/api/auth/login", "/api/auth/register"}; !slices.Equal(got, want) {
		t.Fatalf("request path sequence mismatch: got %v want %v", got, want)
	}
	if got, want := rt.Config.Token, "reg-token"; got != want {
		t.Fatalf("token mismatch: got %q want %q", got, want)
	}
	if got, want := rt.Config.Master, "alice"; got != want {
		t.Fatalf("master mismatch: got %q want %q", got, want)
	}
	if got, want := stdout.String(), "account_id: password: onboarded alice\n"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}

func TestRunRegisterUsesConfiguredServerURLInsteadOfActiveProfileServerURL(t *testing.T) {
	t.Parallel()

	rt, _, _ := newTestRuntime(t, "https://am.namjaeyoun.com", "", func(req *http.Request, body []byte) (*http.Response, error) {
		if got, want := req.URL.Scheme, "https"; got != want {
			t.Fatalf("scheme mismatch: got %q want %q", got, want)
		}
		if got, want := req.URL.Host, "am.namjaeyoun.com"; got != want {
			t.Fatalf("host mismatch: got %q want %q", got, want)
		}
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode register payload: %v", err)
		}
		if got, want := payload["account_id"], "alice"; got != want {
			t.Fatalf("account_id mismatch: got %q want %q", got, want)
		}
		return jsonResponse(http.StatusCreated, `{"token":"reg-token","user":{"id":"u1","account_id":"alice","username":"alice","created_at":"2026-01-01T00:00:00Z"}}`), nil
	})
	rt.Config.ActiveProfile = "local-alice"
	rt.Config.ActiveProfileServerURL = "http://127.0.0.1:45180"
	rt.Config.Profiles = map[string]config.Profile{
		"local-alice": {
			Username:  "local-alice",
			ServerURL: "http://127.0.0.1:45180",
			Token:     "local-token",
		},
	}
	var err error
	rt.Client, err = api.NewClient("http://127.0.0.1:45180", "")
	if err != nil {
		t.Fatalf("create local api client: %v", err)
	}
	rt.Client.SetHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var body []byte
		if req.Body != nil {
			var err error
			body, err = io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}
			_ = req.Body.Close()
		}
		if got, want := req.URL.Scheme, "https"; got != want {
			t.Fatalf("scheme mismatch: got %q want %q", got, want)
		}
		if got, want := req.URL.Host, "am.namjaeyoun.com"; got != want {
			t.Fatalf("host mismatch: got %q want %q", got, want)
		}
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode register payload: %v", err)
		}
		if got, want := payload["account_id"], "alice"; got != want {
			t.Fatalf("account_id mismatch: got %q want %q", got, want)
		}
		return jsonResponse(http.StatusCreated, `{"token":"reg-token","user":{"id":"u1","account_id":"alice","username":"alice","created_at":"2026-01-01T00:00:00Z"}}`), nil
	})})

	if err := runRegister(rt, "alice", "1234"); err != nil {
		t.Fatalf("runRegister: %v", err)
	}

	persisted, err := rt.ConfigStore.Load()
	if err != nil {
		t.Fatalf("load persisted config: %v", err)
	}
	if got, want := persisted.ServerURL, "https://am.namjaeyoun.com"; got != want {
		t.Fatalf("configured server_url mismatch: got %q want %q", got, want)
	}
	if got, want := persisted.ActiveProfileServerURL, "https://am.namjaeyoun.com"; got != want {
		t.Fatalf("active profile server_url mismatch: got %q want %q", got, want)
	}
	if got, want := persisted.Profiles["alice"].ServerURL, "https://am.namjaeyoun.com"; got != want {
		t.Fatalf("registered profile server_url mismatch: got %q want %q", got, want)
	}
}

func TestRunOnboardUsesConfiguredServerURLInsteadOfActiveProfileServerURL(t *testing.T) {
	t.Parallel()

	rt, _, _ := newTestRuntime(t, "https://am.namjaeyoun.com", "", func(req *http.Request, body []byte) (*http.Response, error) {
		if got, want := req.URL.Scheme, "https"; got != want {
			t.Fatalf("scheme mismatch: got %q want %q", got, want)
		}
		if got, want := req.URL.Host, "am.namjaeyoun.com"; got != want {
			t.Fatalf("host mismatch: got %q want %q", got, want)
		}
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode onboard payload: %v", err)
		}
		if got, want := payload["account_id"], "alice"; got != want {
			t.Fatalf("account_id mismatch: got %q want %q", got, want)
		}
		return jsonResponse(http.StatusOK, `{"token":"login-token","user":{"id":"u1","account_id":"alice","username":"alice","created_at":"2026-01-01T00:00:00Z"}}`), nil
	})
	rt.Config.ActiveProfile = "local-alice"
	rt.Config.ActiveProfileServerURL = "http://127.0.0.1:45180"
	rt.Config.Profiles = map[string]config.Profile{
		"local-alice": {
			Username:  "local-alice",
			ServerURL: "http://127.0.0.1:45180",
			Token:     "local-token",
		},
	}
	var err error
	rt.Client, err = api.NewClient("http://127.0.0.1:45180", "")
	if err != nil {
		t.Fatalf("create local api client: %v", err)
	}
	rt.Client.SetHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var body []byte
		if req.Body != nil {
			var err error
			body, err = io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}
			_ = req.Body.Close()
		}
		if got, want := req.URL.Scheme, "https"; got != want {
			t.Fatalf("scheme mismatch: got %q want %q", got, want)
		}
		if got, want := req.URL.Host, "am.namjaeyoun.com"; got != want {
			t.Fatalf("host mismatch: got %q want %q", got, want)
		}
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode onboard payload: %v", err)
		}
		if got, want := payload["account_id"], "alice"; got != want {
			t.Fatalf("account_id mismatch: got %q want %q", got, want)
		}
		return jsonResponse(http.StatusOK, `{"token":"login-token","user":{"id":"u1","account_id":"alice","username":"alice","created_at":"2026-01-01T00:00:00Z"}}`), nil
	})})

	rt.Stdin = strings.NewReader("alice\n1234\n")

	if err := runOnboard(rt); err != nil {
		t.Fatalf("runOnboard: %v", err)
	}

	persisted, err := rt.ConfigStore.Load()
	if err != nil {
		t.Fatalf("load persisted config: %v", err)
	}
	if got, want := persisted.ServerURL, "https://am.namjaeyoun.com"; got != want {
		t.Fatalf("configured server_url mismatch: got %q want %q", got, want)
	}
	if got, want := persisted.ActiveProfileServerURL, "https://am.namjaeyoun.com"; got != want {
		t.Fatalf("active profile server_url mismatch: got %q want %q", got, want)
	}
	if got, want := persisted.Profiles["alice"].ServerURL, "https://am.namjaeyoun.com"; got != want {
		t.Fatalf("onboarded profile server_url mismatch: got %q want %q", got, want)
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

func TestRunLogoutClearsActiveProfileToken(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "active-token", func(req *http.Request, _ []byte) (*http.Response, error) {
		if req.URL.Path != "/api/auth/logout" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		return jsonResponse(http.StatusNoContent, ``), nil
	})
	rt.Config.ActiveProfile = "alice"
	rt.Config.Profiles = map[string]config.Profile{
		"alice": {
			Username:  "alice",
			ServerURL: "http://example.test",
			Token:     "active-token",
		},
		"bob": {
			Username:  "bob",
			ServerURL: "http://example.test",
			Token:     "bob-token",
		},
	}
	if err := rt.ConfigStore.Save(rt.Config); err != nil {
		t.Fatalf("seed profiles: %v", err)
	}

	if err := runLogout(rt); err != nil {
		t.Fatalf("runLogout: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "logged out" {
		t.Fatalf("stdout mismatch: got %q", got)
	}

	persisted, err := rt.ConfigStore.Load()
	if err != nil {
		t.Fatalf("load persisted config: %v", err)
	}
	if got := persisted.Profiles["alice"].Token; got != "" {
		t.Fatalf("expected alice token to be cleared, got %q", got)
	}
	if got, want := persisted.Profiles["bob"].Token, "bob-token"; got != want {
		t.Fatalf("expected bob token to stay intact, got %q want %q", got, want)
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

		return jsonResponse(http.StatusOK, `{"id":"u1","account_id":"alice","username":"alice","created_at":"2026-01-01T00:00:00Z"}`), nil
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
		Stdin:       strings.NewReader(""),
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
