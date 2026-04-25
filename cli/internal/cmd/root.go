package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"agent-message/cli/internal/api"
	"agent-message/cli/internal/config"

	"github.com/spf13/cobra"
)

// Runtime holds shared dependencies for command execution.
type Runtime struct {
	ConfigStore *config.Store
	Config      config.Config
	Client      *api.Client
	JSONOutput  bool
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	RunExternal func(ctx context.Context, stdout, stderr io.Writer, name string, args ...string) error
}

func Execute() error {
	return NewRootCommand().Execute()
}

func NewRootCommand() *cobra.Command {
	rt := &Runtime{
		Stdin:       os.Stdin,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		RunExternal: runExternalCommand,
	}

	var configPath string
	var serverURLOverride string
	var fromProfile string
	var jsonOutput bool
	var root *cobra.Command

	cmd := &cobra.Command{
		Use:   "agent-message",
		Short: "CLI client for Agent Message",
		PersistentPreRunE: func(command *cobra.Command, _ []string) error {
			store := config.NewStore(configPath)
			cfg, err := store.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			if trimmedFrom := strings.TrimSpace(fromProfile); trimmedFrom != "" {
				_, profile, profileErr := resolveStoredProfile(cfg, trimmedFrom)
				if profileErr != nil {
					return fmt.Errorf("--from profile: %w", profileErr)
				}
				cfg.Token = profile.Token
				if strings.TrimSpace(profile.ServerURL) != "" {
					cfg.ActiveProfileServerURL = profile.ServerURL
				}
			}

			serverURL := resolvedClientServerURL(cfg, command, serverURLOverride)
			client, err := api.NewClient(serverURL, cfg.Token)
			if err != nil {
				return fmt.Errorf("initialize API client: %w", err)
			}

			rt.ConfigStore = store
			rt.Config = cfg
			rt.Client = client
			rt.JSONOutput = jsonOutput
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&configPath, "config", config.DefaultPath(), "Path to config file")
	cmd.PersistentFlags().StringVar(&serverURLOverride, "server-url", "", "Override server URL for this command")
	cmd.PersistentFlags().StringVar(&fromProfile, "from", "", "Use a specific profile for this command without switching the active profile")
	cmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output command results as JSON when supported")

	cmd.AddCommand(
		newSchemaCommand(rt, func() *cobra.Command { return root }),
		newCatalogCommand(rt),
		newHowtoCommand(rt),
		newConfigCommand(rt),
		newProfileCommand(rt),
		newOnboardCommand(rt),
		newRegisterCommand(rt),
		newLoginCommand(rt),
		newLogoutCommand(rt),
		newWhoAmICommand(rt),
		newUsernameCommand(rt),
		newTitleCommand(rt),
		newListConversationsCommand(rt),
		newOpenConversationCommand(rt),
		newUploadCommand(rt),
		newSendMessageCommand(rt),
		newReadMessagesCommand(rt),
		newEditMessageCommand(rt),
		newDeleteMessageCommand(rt),
		newReactCommand(rt),
		newUnreactCommand(rt),
		newWatchCommand(rt),
		newWaitCommand(rt),
	)
	root = cmd

	return cmd
}

func runExternalCommand(ctx context.Context, stdout, stderr io.Writer, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func resolvedClientServerURL(cfg config.Config, command *cobra.Command, serverURLOverride string) string {
	if strings.TrimSpace(serverURLOverride) != "" {
		return strings.TrimSpace(serverURLOverride)
	}
	if commandUsesConfiguredServerURL(command) {
		return cfg.ServerURL
	}
	return cfg.ActiveServerURL()
}

func commandUsesConfiguredServerURL(command *cobra.Command) bool {
	if command == nil {
		return false
	}
	switch strings.TrimSpace(command.Name()) {
	case "register", "login", "onboard":
		return true
	default:
		return false
	}
}
