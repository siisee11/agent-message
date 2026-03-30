package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

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
