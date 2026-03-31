package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"agent-messenger/cli/internal/config"
)

func TestRunConfigPathPrintsStorePath(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "", nil)

	if err := runConfigPath(rt); err != nil {
		t.Fatalf("runConfigPath: %v", err)
	}
	if got, want := strings.TrimSpace(stdout.String()), rt.ConfigStore.Path(); got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}

func TestRunConfigGetPrintsSingleValue(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "", nil)

	if err := runConfigGet(rt, "server_url"); err != nil {
		t.Fatalf("runConfigGet(server_url): %v", err)
	}
	if got, want := strings.TrimSpace(stdout.String()), "http://example.test"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}

func TestRunConfigGetPrintsJSON(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "tok-config", nil)

	if err := runConfigGet(rt, ""); err != nil {
		t.Fatalf("runConfigGet: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode stdout json: %v", err)
	}
	if got, want := payload["server_url"], "http://example.test"; got != want {
		t.Fatalf("server_url mismatch: got %v want %q", got, want)
	}
	if got, want := payload["token"], "tok-config"; got != want {
		t.Fatalf("token mismatch: got %v want %q", got, want)
	}
}

func TestRunConfigSetPersistsServerURL(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "tok-config", nil)

	if err := runConfigSet(rt, "server_url", " https://chat.example.test/api/ "); err != nil {
		t.Fatalf("runConfigSet: %v", err)
	}
	if got, want := strings.TrimSpace(stdout.String()), "https://chat.example.test/api"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
	if got, want := rt.Config.ServerURL, "https://chat.example.test/api"; got != want {
		t.Fatalf("runtime server_url mismatch: got %q want %q", got, want)
	}
	if got, want := rt.Client.ServerURL(), "https://chat.example.test/api"; got != want {
		t.Fatalf("client server_url mismatch: got %q want %q", got, want)
	}

	persisted, err := rt.ConfigStore.Load()
	if err != nil {
		t.Fatalf("load persisted config: %v", err)
	}
	if got, want := persisted.ServerURL, "https://chat.example.test/api"; got != want {
		t.Fatalf("persisted server_url mismatch: got %q want %q", got, want)
	}
}

func TestRunConfigUnsetResetsServerURLToDefault(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "https://chat.example.test/api", "tok-config", nil)

	if err := runConfigUnset(rt, "server_url"); err != nil {
		t.Fatalf("runConfigUnset: %v", err)
	}
	if got, want := strings.TrimSpace(stdout.String()), config.DefaultServerURL(); got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
	if got, want := rt.Config.ServerURL, config.DefaultServerURL(); got != want {
		t.Fatalf("runtime server_url mismatch: got %q want %q", got, want)
	}
}

func TestRunConfigSetRejectsUnsupportedKey(t *testing.T) {
	t.Parallel()

	rt, _, _ := newTestRuntime(t, "http://example.test", "", nil)

	err := runConfigSet(rt, "token", "abc")
	if err == nil {
		t.Fatalf("expected error")
	}
	if got := err.Error(); !strings.Contains(got, "supported keys: server_url") {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestDefaultPathUsesAgentMessengerDirectory(t *testing.T) {
	t.Parallel()

	path := config.DefaultPath()
	if got, want := filepath.Base(filepath.Dir(path)), ".agent-messenger"; got != want {
		t.Fatalf("config directory mismatch: got %q want %q", got, want)
	}
	if got, want := filepath.Base(path), "config"; got != want {
		t.Fatalf("config file mismatch: got %q want %q", got, want)
	}
}
