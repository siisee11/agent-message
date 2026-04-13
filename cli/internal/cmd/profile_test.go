package cmd

import (
	"strings"
	"testing"

	"agent-message/cli/internal/config"
)

func TestRunProfileSwitchActivatesStoredProfile(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "alice-token", nil)
	rt.Config.ActiveProfile = "alice"
	rt.Config.Master = "jay"
	rt.Config.Profiles = map[string]config.Profile{
		"alice": {
			Username:  "alice",
			ServerURL: "http://example.test",
			Token:     "alice-token",
			ReadSessions: map[string]config.ReadSession{
				"conv-a": {
					ConversationID: "conv-a",
					IndexToMessage: map[int]string{1: "msg-a"},
				},
			},
			LastReadConversationID: "conv-a",
		},
		"bob": {
			Username:  "bob",
			ServerURL: "https://chat.example.test/api",
			Token:     "bob-token",
			ReadSessions: map[string]config.ReadSession{
				"conv-b": {
					ConversationID: "conv-b",
					IndexToMessage: map[int]string{1: "msg-b"},
				},
			},
			LastReadConversationID: "conv-b",
		},
	}
	if err := rt.ConfigStore.Save(rt.Config); err != nil {
		t.Fatalf("seed profiles: %v", err)
	}

	if err := runProfileSwitch(rt, "bob"); err != nil {
		t.Fatalf("runProfileSwitch: %v", err)
	}

	if got, want := rt.Config.ActiveProfile, "bob"; got != want {
		t.Fatalf("active profile mismatch: got %q want %q", got, want)
	}
	if got, want := rt.Config.Token, "bob-token"; got != want {
		t.Fatalf("token mismatch: got %q want %q", got, want)
	}
	if got, want := rt.Config.Master, "jay"; got != want {
		t.Fatalf("master mismatch: got %q want %q", got, want)
	}
	if got, want := rt.Config.ServerURL, "http://example.test"; got != want {
		t.Fatalf("configured server url mismatch: got %q want %q", got, want)
	}
	if got, want := rt.Config.ActiveProfileServerURL, "https://chat.example.test/api"; got != want {
		t.Fatalf("active profile server url mismatch: got %q want %q", got, want)
	}
	if got, want := rt.Client.ServerURL(), "https://chat.example.test/api"; got != want {
		t.Fatalf("client server url mismatch: got %q want %q", got, want)
	}
	if got, want := rt.Config.LastReadConversationID, "conv-b"; got != want {
		t.Fatalf("last read conversation mismatch: got %q want %q", got, want)
	}
	if got, want := rt.Config.ReadSessions["conv-b"].IndexToMessage[1], "msg-b"; got != want {
		t.Fatalf("read session mismatch: got %q want %q", got, want)
	}
	if got, want := strings.TrimSpace(stdout.String()), "switched to bob"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}

func TestRunProfileListPrintsActiveAndLoggedOutProfiles(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "", nil)
	rt.Config.ActiveProfile = "alice"
	rt.Config.Profiles = map[string]config.Profile{
		"alice": {
			Username:  "alice",
			ServerURL: "http://example.test",
			Token:     "alice-token",
		},
		"bob": {
			Username:  "bob",
			ServerURL: "http://example.test",
		},
	}

	if err := runProfileList(rt); err != nil {
		t.Fatalf("runProfileList: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), stdout.String())
	}
	if got, want := lines[0], "* alice"; got != want {
		t.Fatalf("first line mismatch: got %q want %q", got, want)
	}
	if got, want := lines[1], "  bob logged_out"; got != want {
		t.Fatalf("second line mismatch: got %q want %q", got, want)
	}
}

func TestRunProfileCurrentPrintsActiveProfile(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "", nil)
	rt.Config.ActiveProfile = "alice"

	if err := runProfileCurrent(rt); err != nil {
		t.Fatalf("runProfileCurrent: %v", err)
	}
	if got, want := strings.TrimSpace(stdout.String()), "alice"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}
