package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"agent-message/cli/internal/api"

	"github.com/spf13/cobra"
)

func newListConversationsCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List your direct-message conversations",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runListConversations(rt)
		},
	}
}

func runListConversations(rt *Runtime) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}
	if err := ensureLoggedIn(rt); err != nil {
		return err
	}

	summaries, err := rt.Client.ListConversations(context.Background(), 0)
	if err != nil {
		return err
	}
	if rt.JSONOutput {
		return writeJSON(rt.Stdout, map[string]any{
			"conversations": summaries,
		})
	}

	for _, summary := range summaries {
		_, _ = fmt.Fprintf(rt.Stdout, "%s %s\n", summary.Conversation.ID, summary.OtherUser.Username)
	}
	return nil
}

func newOpenConversationCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "open <username>",
		Short: "Open a conversation with a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runOpenConversation(rt, args[0])
		},
	}
}

func runOpenConversation(rt *Runtime, username string) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}

	details, err := resolveConversationByUsername(context.Background(), rt, username)
	if err != nil {
		return err
	}

	otherUsername := resolveOtherUsername(details, strings.TrimSpace(username))
	return writeTextOrJSON(rt, fmt.Sprintf("%s %s", details.Conversation.ID, otherUsername), map[string]any{
		"conversation":   details.Conversation,
		"participant_a":  details.ParticipantA,
		"participant_b":  details.ParticipantB,
		"other_username": otherUsername,
	})
}

// resolveConversationByUsername gets or creates a DM conversation for the provided username.
// This helper is shared by open and later message/watch commands.
func resolveConversationByUsername(ctx context.Context, rt *Runtime, username string) (api.ConversationDetails, error) {
	if err := ensureRuntime(rt); err != nil {
		return api.ConversationDetails{}, err
	}
	if err := ensureLoggedIn(rt); err != nil {
		return api.ConversationDetails{}, err
	}

	trimmed := strings.TrimSpace(username)
	if trimmed == "" {
		return api.ConversationDetails{}, errors.New("username is required")
	}

	details, err := rt.Client.OpenConversation(ctx, trimmed)
	if err != nil {
		return api.ConversationDetails{}, err
	}
	if strings.TrimSpace(details.Conversation.ID) == "" {
		return api.ConversationDetails{}, errors.New("server returned empty conversation ID")
	}
	return details, nil
}

func resolveConversationIDByUsername(ctx context.Context, rt *Runtime, username string) (string, error) {
	details, err := resolveConversationByUsername(ctx, rt, username)
	if err != nil {
		return "", err
	}
	return details.Conversation.ID, nil
}

func ensureLoggedIn(rt *Runtime) error {
	if strings.TrimSpace(rt.Config.Token) != "" {
		return nil
	}
	return errors.New("not logged in; run `agent-message login <username> <password>`")
}

func resolveOtherUsername(details api.ConversationDetails, requestedUsername string) string {
	requested := strings.TrimSpace(requestedUsername)
	if requested != "" {
		if strings.EqualFold(details.ParticipantA.Username, requested) {
			return details.ParticipantA.Username
		}
		if strings.EqualFold(details.ParticipantB.Username, requested) {
			return details.ParticipantB.Username
		}
		return requested
	}

	if strings.TrimSpace(details.ParticipantA.Username) != "" {
		return details.ParticipantA.Username
	}
	return details.ParticipantB.Username
}
