package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	var payload string
	var payloadFile string
	var payloadStdin bool
	cmd := &cobra.Command{
		Use:   "open <username>",
		Short: "Open a conversation with a user",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 1 {
				return fmt.Errorf("accepts at most 1 arg(s), received %d", len(args))
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			request, err := resolveOpenConversationRequest(rt.Stdin, rawPayloadOptions{
				Payload:      payload,
				PayloadFile:  payloadFile,
				PayloadStdin: payloadStdin,
			}, args)
			if err != nil {
				return err
			}
			return runOpenConversationRequest(rt, request)
		},
	}
	cmd.Flags().StringVar(&payload, "payload", "", "Raw JSON payload matching the open conversation request body")
	cmd.Flags().StringVar(&payloadFile, "payload-file", "", "Read the raw open conversation JSON payload from a file")
	cmd.Flags().BoolVar(&payloadStdin, "payload-stdin", false, "Read the raw open conversation JSON payload from stdin")
	return cmd
}

func runOpenConversation(rt *Runtime, username string) error {
	return runOpenConversationRequest(rt, api.OpenConversationRequest{
		Username: username,
	})
}

func runOpenConversationRequest(rt *Runtime, request api.OpenConversationRequest) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}

	details, err := resolveConversationByUsernameRequest(context.Background(), rt, request)
	if err != nil {
		return err
	}

	otherUsername := resolveOtherUsername(details, strings.TrimSpace(request.Username))
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
	return resolveConversationByUsernameRequest(ctx, rt, api.OpenConversationRequest{
		Username: username,
	})
}

func resolveConversationByUsernameRequest(ctx context.Context, rt *Runtime, request api.OpenConversationRequest) (api.ConversationDetails, error) {
	if err := ensureRuntime(rt); err != nil {
		return api.ConversationDetails{}, err
	}
	if err := ensureLoggedIn(rt); err != nil {
		return api.ConversationDetails{}, err
	}

	trimmed := strings.TrimSpace(request.Username)
	if trimmed == "" {
		return api.ConversationDetails{}, errors.New("username is required")
	}

	request.Username = trimmed
	details, err := rt.Client.OpenConversationWithRequest(ctx, request)
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

func resolveOpenConversationRequest(stdin io.Reader, payloadOptions rawPayloadOptions, args []string) (api.OpenConversationRequest, error) {
	rawPayload, err := resolveRawPayload(stdin, payloadOptions)
	if err != nil {
		return api.OpenConversationRequest{}, err
	}
	if rawPayload != nil {
		if len(args) != 0 {
			return api.OpenConversationRequest{}, errors.New("open accepts no positional args when raw payload flags are used")
		}
		return decodeStrictJSONObject[api.OpenConversationRequest](rawPayload, "open payload")
	}
	if len(args) != 1 {
		return api.OpenConversationRequest{}, errors.New("open requires <username>")
	}
	return api.OpenConversationRequest{Username: args[0]}, nil
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
