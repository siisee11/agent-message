package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newCatalogCommand(rt *Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "catalog",
		Short: "Inspect the server json-render catalog metadata",
	}

	cmd.AddCommand(
		newCatalogPromptCommand(rt),
	)

	return cmd
}

func newCatalogPromptCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "prompt",
		Short: "Print the server json-render catalog.prompt() output",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runCatalogPrompt(rt)
		},
	}
}

func runCatalogPrompt(rt *Runtime) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}

	response, err := rt.Client.GetCatalogPrompt(context.Background())
	if err != nil {
		return err
	}

	prompt := response.Prompt
	if strings.HasSuffix(prompt, "\n") {
		_, _ = fmt.Fprint(rt.Stdout, prompt)
		return nil
	}

	_, _ = fmt.Fprintln(rt.Stdout, prompt)
	return nil
}
