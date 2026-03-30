package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

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
