package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

func newRegisterCommand(_ *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "register <username> <pin>",
		Short: "Register a new account",
		Args:  cobra.ExactArgs(2),
		RunE:  notImplemented("register"),
	}
}

func newLoginCommand(_ *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "login <username> <pin>",
		Short: "Log in with username and PIN",
		Args:  cobra.ExactArgs(2),
		RunE:  notImplemented("login"),
	}
}

func newLogoutCommand(_ *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out and clear local token",
		Args:  cobra.NoArgs,
		RunE:  notImplemented("logout"),
	}
}

func newListConversationsCommand(_ *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List your direct-message conversations",
		Args:  cobra.NoArgs,
		RunE:  notImplemented("ls"),
	}
}

func newOpenConversationCommand(_ *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "open <username>",
		Short: "Open a conversation with a user",
		Args:  cobra.ExactArgs(1),
		RunE:  notImplemented("open"),
	}
}

func newSendMessageCommand(_ *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "send <username> <text>",
		Short: "Send a message to a user",
		Args:  cobra.ExactArgs(2),
		RunE:  notImplemented("send"),
	}
}

func newReadMessagesCommand(_ *Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read <username>",
		Short: "Read recent messages from a conversation",
		Args:  cobra.ExactArgs(1),
		RunE:  notImplemented("read"),
	}
	cmd.Flags().IntP("n", "n", 20, "Number of most recent messages to fetch")
	return cmd
}

func newEditMessageCommand(_ *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "edit <index> <text>",
		Short: "Edit a message by index from last read",
		Args:  cobra.ExactArgs(2),
		RunE:  notImplemented("edit"),
	}
}

func newDeleteMessageCommand(_ *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <index>",
		Short: "Delete a message by index from last read",
		Args:  cobra.ExactArgs(1),
		RunE:  notImplemented("delete"),
	}
}

func newReactCommand(_ *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "react <index> <emoji>",
		Short: "React to a message by index from last read",
		Args:  cobra.ExactArgs(2),
		RunE:  notImplemented("react"),
	}
}

func newUnreactCommand(_ *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "unreact <index> <emoji>",
		Short: "Remove a reaction from a message by index from last read",
		Args:  cobra.ExactArgs(2),
		RunE:  notImplemented("unreact"),
	}
}

func newWatchCommand(_ *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "watch <username>",
		Short: "Watch incoming messages in real time",
		Args:  cobra.ExactArgs(1),
		RunE:  notImplemented("watch"),
	}
}

func notImplemented(commandName string) func(*cobra.Command, []string) error {
	return func(_ *cobra.Command, _ []string) error {
		return errors.New(commandName + " not implemented yet")
	}
}
