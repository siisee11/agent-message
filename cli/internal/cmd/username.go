package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newUsernameCommand(rt *Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "username",
		Short: "Manage your public username",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "set <username>",
		Short: "Set or replace your public username",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runSetUsername(rt, args[0])
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "clear",
		Short: "Clear your public username and fall back to your account ID",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runSetUsername(rt, "")
		},
	})

	return cmd
}

func runSetUsername(rt *Runtime, username string) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}
	if err := ensureLoggedIn(rt); err != nil {
		return err
	}

	profile, err := rt.Client.UpdateUsername(context.Background(), username)
	if err != nil {
		return err
	}

	message := fmt.Sprintf("username set to %s", profile.Username)
	if strings.TrimSpace(username) == "" {
		message = fmt.Sprintf("username cleared; using %s", profile.Username)
	}

	cfg := rt.Config
	if strings.TrimSpace(cfg.Master) == strings.TrimSpace(username) || strings.TrimSpace(username) == "" {
		cfg.Master = profile.Username
		if err := saveRuntimeConfig(rt, cfg); err != nil {
			return err
		}
	}

	return writeTextOrJSON(rt, message, map[string]any{
		"status":  "username_updated",
		"user":    profile,
		"message": message,
	})
}
