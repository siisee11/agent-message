package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"agent-messenger/cli/internal/api"
	"agent-messenger/cli/internal/config"

	"github.com/spf13/cobra"
)

const configKeyServerURL = "server_url"

func newConfigCommand(rt *Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and update local CLI configuration",
	}

	cmd.AddCommand(
		newConfigPathCommand(rt),
		newConfigGetCommand(rt),
		newConfigSetCommand(rt),
		newConfigUnsetCommand(rt),
	)

	return cmd
}

func newConfigPathCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the current config file path",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runConfigPath(rt)
		},
	}
}

func newConfigGetCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "get [key]",
		Short: "Print the full config or a single key",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 1 {
				return errors.New("accepts at most 1 arg(s)")
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			key := ""
			if len(args) == 1 {
				key = args[0]
			}
			return runConfigGet(rt, key)
		},
	}
}

func newConfigSetCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Persist a config value",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return runConfigSet(rt, args[0], args[1])
		},
	}
}

func newConfigUnsetCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "unset <key>",
		Short: "Clear a config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runConfigUnset(rt, args[0])
		},
	}
}

func runConfigPath(rt *Runtime) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(rt.Stdout, rt.ConfigStore.Path())
	return nil
}

func runConfigGet(rt *Runtime, key string) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}

	normalizedKey := normalizeConfigKey(key)
	switch normalizedKey {
	case "":
		payload, err := json.MarshalIndent(rt.Config, "", "  ")
		if err != nil {
			return fmt.Errorf("encode config: %w", err)
		}
		_, _ = fmt.Fprintln(rt.Stdout, string(payload))
		return nil
	case configKeyServerURL:
		_, _ = fmt.Fprintln(rt.Stdout, rt.Config.ServerURL)
		return nil
	default:
		return unsupportedConfigKeyError(normalizedKey)
	}
}

func runConfigSet(rt *Runtime, key string, value string) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}

	normalizedKey := normalizeConfigKey(key)
	switch normalizedKey {
	case configKeyServerURL:
		cfg := rt.Config
		cfg.ServerURL = value
		if err := saveRuntimeConfig(rt, cfg); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(rt.Stdout, rt.Config.ServerURL)
		return nil
	default:
		return unsupportedConfigKeyError(normalizedKey)
	}
}

func runConfigUnset(rt *Runtime, key string) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}

	normalizedKey := normalizeConfigKey(key)
	switch normalizedKey {
	case configKeyServerURL:
		cfg := rt.Config
		cfg.ServerURL = ""
		if err := saveRuntimeConfig(rt, cfg); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(rt.Stdout, rt.Config.ServerURL)
		return nil
	default:
		return unsupportedConfigKeyError(normalizedKey)
	}
}

func saveRuntimeConfig(rt *Runtime, cfg config.Config) error {
	if err := rt.ConfigStore.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	loaded, err := rt.ConfigStore.Load()
	if err != nil {
		return fmt.Errorf("reload config: %w", err)
	}

	client, err := api.NewClient(loaded.ServerURL, loaded.Token)
	if err != nil {
		return fmt.Errorf("initialize API client: %w", err)
	}

	rt.Config = loaded
	rt.Client = client
	return nil
}

func normalizeConfigKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

func unsupportedConfigKeyError(key string) error {
	if key == "" {
		return errors.New("config key is required")
	}
	return fmt.Errorf("unsupported config key %q; supported keys: %s", key, configKeyServerURL)
}
