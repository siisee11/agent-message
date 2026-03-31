package cmd

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"agent-message/cli/internal/config"

	"github.com/spf13/cobra"
)

func newProfileCommand(rt *Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage saved login profiles",
	}

	cmd.AddCommand(
		newProfileListCommand(rt),
		newProfileCurrentCommand(rt),
		newProfileSwitchCommand(rt),
	)

	return cmd
}

func newProfileListCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List saved profiles",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runProfileList(rt)
		},
	}
}

func newProfileCurrentCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Print the active profile name",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runProfileCurrent(rt)
		},
	}
}

func newProfileSwitchCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "switch <profile>",
		Short: "Switch to a saved profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runProfileSwitch(rt, args[0])
		},
	}
}

func runProfileList(rt *Runtime) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}

	names := sortedProfileNames(rt.Config.Profiles)
	for _, name := range names {
		profile := rt.Config.Profiles[name]
		marker := " "
		if name == rt.Config.ActiveProfile {
			marker = "*"
		}

		status := ""
		if strings.TrimSpace(profile.Token) == "" {
			status = " logged_out"
		}
		_, _ = fmt.Fprintf(rt.Stdout, "%s %s%s\n", marker, name, status)
	}
	return nil
}

func runProfileCurrent(rt *Runtime) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}

	if strings.TrimSpace(rt.Config.ActiveProfile) == "" {
		return errors.New("no active profile")
	}
	_, _ = fmt.Fprintln(rt.Stdout, rt.Config.ActiveProfile)
	return nil
}

func runProfileSwitch(rt *Runtime, rawName string) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}

	name, profile, err := resolveStoredProfile(rt.Config, rawName)
	if err != nil {
		return err
	}

	cfg := rt.Config
	cfg.ActiveProfile = name
	cfg.ServerURL = profile.ServerURL
	cfg.Token = profile.Token
	cfg.ReadSessions = cloneReadSessionsMap(profile.ReadSessions)
	cfg.LastReadConversationID = profile.LastReadConversationID

	if err := saveRuntimeConfig(rt, cfg); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(rt.Stdout, "switched to %s\n", name)
	return nil
}

func resolveStoredProfile(cfg config.Config, rawName string) (string, config.Profile, error) {
	name := strings.TrimSpace(rawName)
	if name == "" {
		return "", config.Profile{}, errors.New("profile name is required")
	}

	if profile, ok := cfg.Profiles[name]; ok {
		return name, profile, nil
	}

	for existingName, profile := range cfg.Profiles {
		if strings.EqualFold(existingName, name) {
			return existingName, profile, nil
		}
	}

	return "", config.Profile{}, fmt.Errorf("profile %q not found; run `agent-message login <username> <pin>` first", name)
}

func sortedProfileNames(profiles map[string]config.Profile) []string {
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func cloneReadSessionsMap(readSessions map[string]config.ReadSession) map[string]config.ReadSession {
	cloned := make(map[string]config.ReadSession, len(readSessions))
	for key, session := range readSessions {
		indexToMessage := make(map[int]string, len(session.IndexToMessage))
		for index, messageID := range session.IndexToMessage {
			indexToMessage[index] = messageID
		}
		cloned[key] = config.ReadSession{
			ConversationID:  session.ConversationID,
			Username:        session.Username,
			IndexToMessage:  indexToMessage,
			LastReadMessage: session.LastReadMessage,
		}
	}
	return cloned
}
