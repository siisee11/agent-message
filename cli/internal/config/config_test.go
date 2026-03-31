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
	if cfg.Profiles == nil {
		t.Fatalf("expected profiles map to be initialized")
	}
	if cfg.ReadSessions == nil {
		t.Fatalf("expected read_sessions map to be initialized")
	}
}

func TestSaveLoadRoundTripNormalizesServerURL(t *testing.T) {
	t.Parallel()

	store := NewStore(filepath.Join(t.TempDir(), "config"))
	input := Config{
		ServerURL:              " https://example.com/api/ ",
		Token:                  "  abc123  ",
		LastReadConversationID: " conv-1 ",
		ReadSessions: map[string]ReadSession{
			" conv-1 ": {
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
	if got, want := cfg.LastReadConversationID, "conv-1"; got != want {
		t.Fatalf("last read conversation mismatch: got %q want %q", got, want)
	}
}

func TestSaveLoadRoundTripWithActiveProfile(t *testing.T) {
	t.Parallel()

	store := NewStore(filepath.Join(t.TempDir(), "config"))
	input := Config{
		ServerURL:     " https://chat.example.test/api/ ",
		Token:         " active-token ",
		ActiveProfile: "alice",
		Profiles: map[string]Profile{
			"alice": {
				Username:  "alice",
				ServerURL: "http://localhost:8080",
				Token:     "stale-token",
				ReadSessions: map[string]ReadSession{
					" conv-1 ": {
						ConversationID: "conv-1",
						IndexToMessage: map[int]string{1: "msg-1"},
					},
				},
				LastReadConversationID: " conv-1 ",
			},
			"bob": {
				Username:  "bob",
				ServerURL: "http://localhost:9090/",
				Token:     "bob-token",
			},
		},
		ReadSessions: map[string]ReadSession{
			"conv-1": {
				ConversationID: "conv-1",
				IndexToMessage: map[int]string{1: "msg-1"},
			},
		},
		LastReadConversationID: "conv-1",
	}

	if err := store.Save(input); err != nil {
		t.Fatalf("save config: %v", err)
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if got, want := cfg.ActiveProfile, "alice"; got != want {
		t.Fatalf("active profile mismatch: got %q want %q", got, want)
	}
	if got, want := cfg.ServerURL, "https://chat.example.test/api"; got != want {
		t.Fatalf("server_url mismatch: got %q want %q", got, want)
	}
	if got, want := cfg.Token, "active-token"; got != want {
		t.Fatalf("token mismatch: got %q want %q", got, want)
	}
	if got, want := cfg.ReadSessions["conv-1"].IndexToMessage[1], "msg-1"; got != want {
		t.Fatalf("active read session mismatch: got %q want %q", got, want)
	}
	if got, want := cfg.Profiles["bob"].ServerURL, "http://localhost:9090"; got != want {
		t.Fatalf("inactive profile server_url mismatch: got %q want %q", got, want)
	}
}
