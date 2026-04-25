package cmd

import (
	"context"
	"errors"
	"strings"

	"agent-message/cli/internal/api"

	"github.com/spf13/cobra"
)

func newUploadCommand(rt *Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upload <path>",
		Short: "Upload a file and print its static URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runUploadFile(rt, args[0])
		},
	}
	return cmd
}

func runUploadFile(rt *Runtime, path string) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}
	if err := ensureLoggedIn(rt); err != nil {
		return err
	}

	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return errors.New("upload path is required")
	}

	response, err := rt.Client.UploadFile(context.Background(), trimmedPath)
	if err != nil {
		return err
	}

	return writeUploadResponse(rt, response)
}

func writeUploadResponse(rt *Runtime, response api.UploadResponse) error {
	return writeTextOrJSON(rt, response.URL, map[string]any{
		"url": response.URL,
	})
}
