package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newTitleCommand(rt *Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "title",
		Short: "Manage conversation titles",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "set <username> <title>",
		Short: "Set a conversation title",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return runSetConversationTitle(rt, args[0], args[1])
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "clear <username>",
		Short: "Clear a conversation title",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runSetConversationTitle(rt, args[0], "")
		},
	})

	return cmd
}

func runSetConversationTitle(rt *Runtime, username, title string) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}
	if err := ensureLoggedIn(rt); err != nil {
		return err
	}

	details, err := resolveConversationByUsername(context.Background(), rt, username)
	if err != nil {
		return err
	}

	updated, err := rt.Client.UpdateConversationTitle(context.Background(), details.Conversation.ID, title)
	if err != nil {
		return err
	}

	otherUsername := resolveOtherUsername(updated, strings.TrimSpace(username))
	trimmedTitle := strings.TrimSpace(updated.Conversation.Title)

	message := fmt.Sprintf("title set for %s", otherUsername)
	if trimmedTitle == "" {
		message = fmt.Sprintf("title cleared for %s", otherUsername)
	}

	return writeTextOrJSON(rt, message, map[string]any{
		"status":        "conversation_title_updated",
		"title":         trimmedTitle,
		"conversation":  updated.Conversation,
		"participant_a": updated.ParticipantA,
		"participant_b": updated.ParticipantB,
		"username":      otherUsername,
		"message":       message,
	})
}
