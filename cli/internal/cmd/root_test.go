package cmd

import (
	"testing"

	"agent-message/cli/internal/config"

	"github.com/spf13/cobra"
)

func TestResolvedClientServerURLUsesConfiguredServerURLForAuthCommands(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		ServerURL:              "https://am.namjaeyoun.com",
		ActiveProfile:          "alice",
		ActiveProfileServerURL: "http://127.0.0.1:45180",
		Profiles: map[string]config.Profile{
			"alice": {
				Username:  "alice",
				ServerURL: "http://127.0.0.1:45180",
			},
		},
	}

	for _, name := range []string{"register", "login", "onboard"} {
		command := &cobra.Command{Use: name}
		if got, want := resolvedClientServerURL(cfg, command, ""), "https://am.namjaeyoun.com"; got != want {
			t.Fatalf("%s server_url mismatch: got %q want %q", name, got, want)
		}
	}
}

func TestResolvedClientServerURLUsesActiveProfileServerURLForMessagingCommands(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		ServerURL:              "https://am.namjaeyoun.com",
		ActiveProfile:          "alice",
		ActiveProfileServerURL: "http://127.0.0.1:45180",
		Profiles: map[string]config.Profile{
			"alice": {
				Username:  "alice",
				ServerURL: "http://127.0.0.1:45180",
			},
		},
	}

	command := &cobra.Command{Use: "send"}
	if got, want := resolvedClientServerURL(cfg, command, ""), "http://127.0.0.1:45180"; got != want {
		t.Fatalf("send server_url mismatch: got %q want %q", got, want)
	}
}
