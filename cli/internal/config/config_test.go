package config

import (
	"encoding/json"
	"os"
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
	if cfg.Master != "" {
		t.Fatalf("expected empty master, got %q", cfg.Master)
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
		Master:                 "  jay  ",
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
	if got, want := cfg.Master, "jay"; got != want {
		t.Fatalf("master mismatch: got %q want %q", got, want)
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
		Master:        "  jay  ",
		ActiveProfile: "alice",
		Profiles: map[string]Profile{
			"alice": {
				Username:  "alice",
				ServerURL: DefaultServerURL(),
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
	if got, want := cfg.ActiveProfileServerURL, "https://chat.example.test/api"; got != want {
		t.Fatalf("active profile server_url mismatch: got %q want %q", got, want)
	}
	if got, want := cfg.Token, "active-token"; got != want {
		t.Fatalf("token mismatch: got %q want %q", got, want)
	}
	if got, want := cfg.Master, "jay"; got != want {
		t.Fatalf("master mismatch: got %q want %q", got, want)
	}
	if got, want := cfg.ReadSessions["conv-1"].IndexToMessage[1], "msg-1"; got != want {
		t.Fatalf("active read session mismatch: got %q want %q", got, want)
	}
	if got, want := cfg.Profiles["bob"].ServerURL, "http://localhost:9090"; got != want {
		t.Fatalf("inactive profile server_url mismatch: got %q want %q", got, want)
	}
}

func TestLoadPreservesConfiguredServerURLSeparatelyFromActiveProfileServerURL(t *testing.T) {
	t.Parallel()

	store := NewStore(filepath.Join(t.TempDir(), "config"))
	input := Config{
		ServerURL:              " https://am.namjaeyoun.com/ ",
		Token:                  " local-token ",
		Master:                 " jay ",
		ActiveProfile:          "alice",
		ActiveProfileServerURL: " http://127.0.0.1:45180/ ",
		Profiles: map[string]Profile{
			"alice": {
				Username:  "alice",
				ServerURL: "http://127.0.0.1:45180/",
				Token:     "local-token",
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

	if got, want := cfg.ServerURL, "https://am.namjaeyoun.com"; got != want {
		t.Fatalf("configured server_url mismatch: got %q want %q", got, want)
	}
	if got, want := cfg.ActiveProfileServerURL, "http://127.0.0.1:45180"; got != want {
		t.Fatalf("active profile server_url mismatch: got %q want %q", got, want)
	}
	if got, want := cfg.Master, "jay"; got != want {
		t.Fatalf("master mismatch: got %q want %q", got, want)
	}
	if got, want := cfg.ActiveServerURL(), "http://127.0.0.1:45180"; got != want {
		t.Fatalf("active server resolution mismatch: got %q want %q", got, want)
	}
}

func TestLoadPromotesLegacyActiveProfileMasterToGlobalMaster(t *testing.T) {
	t.Parallel()

	store := NewStore(filepath.Join(t.TempDir(), "config"))
	rawLegacyConfig := `{
  "server_url": "https://chat.example.test/api/",
  "active_profile": "alice",
  "profiles": {
    "alice": {
      "username": "alice",
      "server_url": "https://chat.example.test/api/",
      "token": "alice-token",
      "master": "jay"
    },
    "bob": {
      "username": "bob",
      "server_url": "https://chat.example.test/api/",
      "token": "bob-token",
      "master": "boss"
    }
  }
}`
	if err := os.WriteFile(store.Path(), []byte(rawLegacyConfig), 0o600); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if got, want := cfg.Master, "jay"; got != want {
		t.Fatalf("global master mismatch: got %q want %q", got, want)
	}
	if err := store.Save(cfg); err != nil {
		t.Fatalf("save migrated config: %v", err)
	}

	rawSaved, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatalf("read migrated config: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(rawSaved, &decoded); err != nil {
		t.Fatalf("decode migrated config: %v", err)
	}
	profiles, ok := decoded["profiles"].(map[string]any)
	if !ok {
		t.Fatalf("expected profiles object in migrated config")
	}
	alice, ok := profiles["alice"].(map[string]any)
	if !ok {
		t.Fatalf("expected alice profile object in migrated config")
	}
	if _, exists := alice["master"]; exists {
		t.Fatalf("expected migrated alice profile to omit master")
	}
	bob, ok := profiles["bob"].(map[string]any)
	if !ok {
		t.Fatalf("expected bob profile object in migrated config")
	}
	if _, exists := bob["master"]; exists {
		t.Fatalf("expected migrated bob profile to omit master")
	}
}
