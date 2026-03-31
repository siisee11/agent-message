package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"agent-messenger/cli/internal/api"
	"agent-messenger/cli/internal/config"

	"github.com/spf13/cobra"
)

// Runtime holds shared dependencies for command execution.
type Runtime struct {
	ConfigStore *config.Store
	Config      config.Config
	Client      *api.Client
	Stdout      io.Writer
	Stderr      io.Writer
}

func Execute() error {
	return NewRootCommand().Execute()
}

func NewRootCommand() *cobra.Command {
	rt := &Runtime{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	var configPath string
	var serverURLOverride string

	cmd := &cobra.Command{
		Use:   "agent-messenger",
		Short: "CLI client for Agent Messenger",
		PersistentPreRunE: func(command *cobra.Command, _ []string) error {
			store := config.NewStore(configPath)
			cfg, err := store.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			if strings.TrimSpace(serverURLOverride) != "" {
				cfg.ServerURL = serverURLOverride
			}

			client, err := api.NewClient(cfg.ServerURL, cfg.Token)
			if err != nil {
				return fmt.Errorf("initialize API client: %w", err)
			}

			rt.ConfigStore = store
			rt.Config = cfg
			rt.Client = client
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&configPath, "config", config.DefaultPath(), "Path to config file")
	cmd.PersistentFlags().StringVar(&serverURLOverride, "server-url", "", "Override server URL for this command")

	cmd.AddCommand(
		newConfigCommand(rt),
		newProfileCommand(rt),
		newRegisterCommand(rt),
		newLoginCommand(rt),
		newLogoutCommand(rt),
		newWhoAmICommand(rt),
		newListConversationsCommand(rt),
		newOpenConversationCommand(rt),
		newSendMessageCommand(rt),
		newReadMessagesCommand(rt),
		newEditMessageCommand(rt),
		newDeleteMessageCommand(rt),
		newReactCommand(rt),
		newUnreactCommand(rt),
		newWatchCommand(rt),
	)

	return cmd
}
