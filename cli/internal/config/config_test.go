package config

import (
	"path/filepath"
	"testing"
)

func TestLoadMissingConfigUsesDefaults(t *testing.T) {
	t.Parallel()

	store := NewStore(filepath.Join(t.TempDir(), "missing-config"))
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("load missing config: %v", err)
	}

	if cfg.ServerURL != DefaultServerURL() {
		t.Fatalf("expected default server URL %q, got %q", DefaultServerURL(), cfg.ServerURL)
	}
	if cfg.Token != "" {
		t.Fatalf("expected empty token, got %q", cfg.Token)
	}
	if cfg.ReadSessions == nil {
		t.Fatalf("expected read_sessions map to be initialized")
	}
}

func TestSaveLoadRoundTripNormalizesServerURL(t *testing.T) {
	t.Parallel()

	store := NewStore(filepath.Join(t.TempDir(), "config"))
	input := Config{
		ServerURL: " https://example.com/api/ ",
		Token:     "  abc123  ",
		ReadSessions: map[string]ReadSession{
			"conv-1": {
				ConversationID: "conv-1",
				Username:       "alice",
				IndexToMessage: map[int]string{1: "msg-1"},
			},
		},
	}

	if err := store.Save(input); err != nil {
		t.Fatalf("save config: %v", err)
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if got, want := cfg.ServerURL, "https://example.com/api"; got != want {
		t.Fatalf("server_url mismatch: got %q want %q", got, want)
	}
	if got, want := cfg.Token, "abc123"; got != want {
		t.Fatalf("token mismatch: got %q want %q", got, want)
	}
	if got, ok := cfg.ReadSessions["conv-1"]; !ok || got.IndexToMessage[1] != "msg-1" {
		t.Fatalf("expected read session mapping to survive round trip")
	}
}
