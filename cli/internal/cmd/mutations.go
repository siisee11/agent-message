package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"agent-message/cli/internal/api"
	"agent-message/cli/internal/config"

	"github.com/spf13/cobra"
)

func newEditMessageCommand(rt *Runtime) *cobra.Command {
	var explicitMessageID string
	var payload string
	var payloadFile string
	var payloadStdin bool
	cmd := &cobra.Command{
		Use:   "edit [message-id-or-index] <text>",
		Short: "Edit a message by explicit message ID or by index from the last read",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 2 {
				return fmt.Errorf("accepts at most 2 arg(s), received %d", len(args))
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			return nil
		},
	}
	cmd.Flags().StringVar(&explicitMessageID, "message-id", "", "Edit by explicit message ID")
	cmd.Flags().StringVar(&payload, "payload", "", "Raw JSON payload matching the edit request body")
	cmd.Flags().StringVar(&payloadFile, "payload-file", "", "Read the raw edit JSON payload from a file")
	cmd.Flags().BoolVar(&payloadStdin, "payload-stdin", false, "Read the raw edit JSON payload from stdin")
	indexPtr := cmd.Flags().IntP("index", "", 0, "Edit by index from the last read")
	cmd.RunE = func(_ *cobra.Command, args []string) error {
		selector, request, err := resolveEditInput(rt.Stdin, rawPayloadOptions{
			Payload:      payload,
			PayloadFile:  payloadFile,
			PayloadStdin: payloadStdin,
		}, args, explicitMessageID, *indexPtr)
		if err != nil {
			return err
		}
		return runEditMessageRequest(rt, selector, request, explicitMessageID, *indexPtr)
	}
	return cmd
}

func runEditMessage(rt *Runtime, selectorArg string, text string, explicitMessageID string, explicitIndex int) error {
	return runEditMessageRequest(rt, selectorArg, api.EditMessageRequest{
		Content: text,
	}, explicitMessageID, explicitIndex)
}

func runEditMessageRequest(rt *Runtime, selectorArg string, request api.EditMessageRequest, explicitMessageID string, explicitIndex int) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}
	if err := ensureLoggedIn(rt); err != nil {
		return err
	}

	trimmedText := strings.TrimSpace(request.Content)
	if trimmedText == "" {
		return errors.New("message text is required")
	}
	request.Content = trimmedText

	messageID, source, err := resolveMessageTarget(rt, selectorArg, explicitMessageID, explicitIndex)
	if err != nil {
		return err
	}

	message, err := rt.Client.EditMessageWithRequest(context.Background(), messageID, request)
	if err != nil {
		return err
	}

	return writeTextOrJSON(rt, fmt.Sprintf("edited %s", message.ID), map[string]any{
		"action":     "edited",
		"message":    message,
		"message_id": message.ID,
		"source":     source,
	})
}

func newDeleteMessageCommand(rt *Runtime) *cobra.Command {
	var explicitMessageID string
	index := 0
	cmd := &cobra.Command{
		Use:   "delete [message-id-or-index]",
		Short: "Delete a message by explicit message ID or by index from the last read",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 1 {
				return fmt.Errorf("accepts at most 1 arg(s), received %d", len(args))
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			selector, err := resolveDeleteArgs(args, explicitMessageID, index)
			if err != nil {
				return err
			}
			return runDeleteMessage(rt, selector, explicitMessageID, index)
		},
	}
	cmd.Flags().StringVar(&explicitMessageID, "message-id", "", "Delete by explicit message ID")
	indexPtr := cmd.Flags().IntP("index", "", 0, "Delete by index from the last read")
	cmd.RunE = func(_ *cobra.Command, args []string) error {
		selector, err := resolveDeleteArgs(args, explicitMessageID, *indexPtr)
		if err != nil {
			return err
		}
		return runDeleteMessage(rt, selector, explicitMessageID, *indexPtr)
	}
	return cmd
}

func runDeleteMessage(rt *Runtime, selectorArg string, explicitMessageID string, explicitIndex int) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}
	if err := ensureLoggedIn(rt); err != nil {
		return err
	}

	messageID, source, err := resolveMessageTarget(rt, selectorArg, explicitMessageID, explicitIndex)
	if err != nil {
		return err
	}

	message, err := rt.Client.DeleteMessage(context.Background(), messageID)
	if err != nil {
		return err
	}

	return writeTextOrJSON(rt, fmt.Sprintf("deleted %s", message.ID), map[string]any{
		"action":     "deleted",
		"message":    message,
		"message_id": message.ID,
		"source":     source,
	})
}

func newReactCommand(rt *Runtime) *cobra.Command {
	var explicitMessageID string
	var payload string
	var payloadFile string
	var payloadStdin bool
	cmd := &cobra.Command{
		Use:   "react [message-id-or-index] <emoji>",
		Short: "React to a message by explicit message ID or by index from the last read",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 2 {
				return fmt.Errorf("accepts at most 2 arg(s), received %d", len(args))
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			selector, emoji, err := resolveReactionArgs(args, explicitMessageID, 0)
			if err != nil {
				return err
			}
			return runReact(rt, selector, emoji, explicitMessageID, 0)
		},
	}
	cmd.Flags().StringVar(&explicitMessageID, "message-id", "", "React by explicit message ID")
	cmd.Flags().StringVar(&payload, "payload", "", "Raw JSON payload matching the react request body")
	cmd.Flags().StringVar(&payloadFile, "payload-file", "", "Read the raw react JSON payload from a file")
	cmd.Flags().BoolVar(&payloadStdin, "payload-stdin", false, "Read the raw react JSON payload from stdin")
	indexPtr := cmd.Flags().IntP("index", "", 0, "React by index from the last read")
	cmd.RunE = func(_ *cobra.Command, args []string) error {
		selector, request, err := resolveReactionInput(rt.Stdin, rawPayloadOptions{
			Payload:      payload,
			PayloadFile:  payloadFile,
			PayloadStdin: payloadStdin,
		}, args, explicitMessageID, *indexPtr, "react")
		if err != nil {
			return err
		}
		return runReactRequest(rt, selector, request, explicitMessageID, *indexPtr)
	}
	return cmd
}

func runReact(rt *Runtime, selectorArg string, emoji string, explicitMessageID string, explicitIndex int) error {
	return runReactRequest(rt, selectorArg, api.ToggleReactionRequest{
		Emoji: emoji,
	}, explicitMessageID, explicitIndex)
}

func runReactRequest(rt *Runtime, selectorArg string, request api.ToggleReactionRequest, explicitMessageID string, explicitIndex int) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}
	if err := ensureLoggedIn(rt); err != nil {
		return err
	}

	trimmedEmoji := strings.TrimSpace(request.Emoji)
	if trimmedEmoji == "" {
		return errors.New("emoji is required")
	}
	request.Emoji = trimmedEmoji

	messageID, source, err := resolveMessageTarget(rt, selectorArg, explicitMessageID, explicitIndex)
	if err != nil {
		return err
	}
	result, err := rt.Client.AddReactionWithRequest(context.Background(), messageID, request)
	if err != nil {
		return err
	}

	action := strings.TrimSpace(result.Action)
	if action == "" {
		action = "added"
	}
	return writeTextOrJSON(rt, fmt.Sprintf("reaction %s %s %s", action, result.Reaction.MessageID, result.Reaction.Emoji), map[string]any{
		"action":     action,
		"reaction":   result.Reaction,
		"message_id": result.Reaction.MessageID,
		"source":     source,
	})
}

func newUnreactCommand(rt *Runtime) *cobra.Command {
	var explicitMessageID string
	cmd := &cobra.Command{
		Use:   "unreact [message-id-or-index] <emoji>",
		Short: "Remove a reaction by explicit message ID or by index from the last read",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 2 {
				return fmt.Errorf("accepts at most 2 arg(s), received %d", len(args))
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			selector, emoji, err := resolveReactionArgs(args, explicitMessageID, 0)
			if err != nil {
				return err
			}
			return runUnreact(rt, selector, emoji, explicitMessageID, 0)
		},
	}
	cmd.Flags().StringVar(&explicitMessageID, "message-id", "", "Unreact by explicit message ID")
	indexPtr := cmd.Flags().IntP("index", "", 0, "Unreact by index from the last read")
	cmd.RunE = func(_ *cobra.Command, args []string) error {
		selector, emoji, err := resolveReactionArgs(args, explicitMessageID, *indexPtr)
		if err != nil {
			return err
		}
		return runUnreact(rt, selector, emoji, explicitMessageID, *indexPtr)
	}
	return cmd
}

func runUnreact(rt *Runtime, selectorArg string, emoji string, explicitMessageID string, explicitIndex int) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}
	if err := ensureLoggedIn(rt); err != nil {
		return err
	}

	trimmedEmoji := strings.TrimSpace(emoji)
	if trimmedEmoji == "" {
		return errors.New("emoji is required")
	}

	messageID, source, err := resolveMessageTarget(rt, selectorArg, explicitMessageID, explicitIndex)
	if err != nil {
		return err
	}
	reaction, err := rt.Client.RemoveReaction(context.Background(), messageID, trimmedEmoji)
	if err != nil {
		return err
	}

	return writeTextOrJSON(rt, fmt.Sprintf("reaction removed %s %s", reaction.MessageID, reaction.Emoji), map[string]any{
		"action":     "removed",
		"reaction":   reaction,
		"message_id": reaction.MessageID,
		"source":     source,
	})
}

func resolveEditInput(stdin io.Reader, payloadOptions rawPayloadOptions, args []string, explicitMessageID string, explicitIndex int) (string, api.EditMessageRequest, error) {
	rawPayload, err := resolveRawPayload(stdin, payloadOptions)
	if err != nil {
		return "", api.EditMessageRequest{}, err
	}
	if rawPayload != nil {
		selector, resolveErr := resolveSelectorForRawPayload("edit", args, explicitMessageID, explicitIndex)
		if resolveErr != nil {
			return "", api.EditMessageRequest{}, resolveErr
		}
		request, decodeErr := decodeStrictJSONObject[api.EditMessageRequest](rawPayload, "edit payload")
		if decodeErr != nil {
			return "", api.EditMessageRequest{}, decodeErr
		}
		return selector, request, nil
	}
	selector, text, err := resolveEditArgs(args, explicitMessageID, explicitIndex)
	if err != nil {
		return "", api.EditMessageRequest{}, err
	}
	return selector, api.EditMessageRequest{Content: text}, nil
}

func resolveReactionInput(stdin io.Reader, payloadOptions rawPayloadOptions, args []string, explicitMessageID string, explicitIndex int, action string) (string, api.ToggleReactionRequest, error) {
	rawPayload, err := resolveRawPayload(stdin, payloadOptions)
	if err != nil {
		return "", api.ToggleReactionRequest{}, err
	}
	if rawPayload != nil {
		selector, resolveErr := resolveSelectorForRawPayload(action, args, explicitMessageID, explicitIndex)
		if resolveErr != nil {
			return "", api.ToggleReactionRequest{}, resolveErr
		}
		request, decodeErr := decodeStrictJSONObject[api.ToggleReactionRequest](rawPayload, action+" payload")
		if decodeErr != nil {
			return "", api.ToggleReactionRequest{}, decodeErr
		}
		return selector, request, nil
	}
	selector, emoji, err := resolveReactionArgs(args, explicitMessageID, explicitIndex)
	if err != nil {
		return "", api.ToggleReactionRequest{}, err
	}
	return selector, api.ToggleReactionRequest{Emoji: emoji}, nil
}

func resolveSelectorForRawPayload(action string, args []string, explicitMessageID string, explicitIndex int) (string, error) {
	if strings.TrimSpace(explicitMessageID) != "" || explicitIndex > 0 {
		if len(args) != 0 {
			return "", fmt.Errorf("%s accepts no positional args when raw payload flags are used with --message-id or --index", action)
		}
		return "", nil
	}
	if len(args) != 1 {
		return "", fmt.Errorf("%s requires <message-id-or-index> when raw payload flags are used", action)
	}
	return args[0], nil
}

func resolveEditArgs(args []string, explicitMessageID string, explicitIndex int) (string, string, error) {
	if strings.TrimSpace(explicitMessageID) != "" || explicitIndex > 0 {
		if len(args) != 1 {
			return "", "", errors.New("edit requires only <text> when --message-id or --index is set")
		}
		return "", args[0], nil
	}
	if len(args) != 2 {
		return "", "", errors.New("edit requires <message-id-or-index> and <text>")
	}
	return args[0], args[1], nil
}

func resolveDeleteArgs(args []string, explicitMessageID string, explicitIndex int) (string, error) {
	if strings.TrimSpace(explicitMessageID) != "" || explicitIndex > 0 {
		if len(args) != 0 {
			return "", errors.New("delete accepts no positional args when --message-id or --index is set")
		}
		return "", nil
	}
	if len(args) != 1 {
		return "", errors.New("delete requires <message-id-or-index>")
	}
	return args[0], nil
}

func resolveReactionArgs(args []string, explicitMessageID string, explicitIndex int) (string, string, error) {
	if strings.TrimSpace(explicitMessageID) != "" || explicitIndex > 0 {
		if len(args) != 1 {
			return "", "", errors.New("reaction commands require only <emoji> when --message-id or --index is set")
		}
		return "", args[0], nil
	}
	if len(args) == 1 {
		return "", args[0], nil
	}
	if len(args) != 2 {
		return "", "", errors.New("reaction commands require <message-id-or-index> and <emoji>")
	}
	return args[0], args[1], nil
}

func resolveMessageTarget(rt *Runtime, selectorArg string, explicitMessageID string, explicitIndex int) (string, string, error) {
	trimmedMessageID := strings.TrimSpace(explicitMessageID)
	if trimmedMessageID != "" && explicitIndex > 0 {
		return "", "", errors.New("choose only one of --message-id or --index")
	}
	if trimmedMessageID != "" {
		return trimmedMessageID, "message_id", nil
	}
	if explicitIndex > 0 {
		messageID, _, err := resolveMessageIDFromLastRead(rt, strconv.Itoa(explicitIndex))
		return messageID, "index", err
	}

	selector := strings.TrimSpace(selectorArg)
	if selector == "" {
		return "", "", errors.New("message ID or index is required")
	}
	if _, err := strconv.Atoi(selector); err == nil {
		messageID, _, resolveErr := resolveMessageIDFromLastRead(rt, selector)
		return messageID, "index", resolveErr
	}
	return selector, "message_id", nil
}

func resolveMessageIDFromLastRead(rt *Runtime, indexArg string) (messageID string, session config.ReadSession, err error) {
	index, err := parseMessageIndex(indexArg)
	if err != nil {
		return "", config.ReadSession{}, err
	}

	session, err = resolveLastReadSession(rt)
	if err != nil {
		return "", config.ReadSession{}, err
	}

	messageID = strings.TrimSpace(session.IndexToMessage[index])
	if messageID == "" {
		return "", config.ReadSession{}, fmt.Errorf("index %d not found in last read session; run `agent-message read <username>` first", index)
	}
	return messageID, session, nil
}

func parseMessageIndex(raw string) (int, error) {
	index, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || index < 1 {
		return 0, errors.New("index must be a positive integer")
	}
	return index, nil
}

func resolveLastReadSession(rt *Runtime) (config.ReadSession, error) {
	if rt.Config.ReadSessions == nil || len(rt.Config.ReadSessions) == 0 {
		return config.ReadSession{}, errors.New("no read session found; run `agent-message read <username>` first")
	}

	conversationID := strings.TrimSpace(rt.Config.LastReadConversationID)
	if conversationID != "" {
		session, ok := rt.Config.ReadSessions[conversationID]
		if !ok {
			return config.ReadSession{}, errors.New("last read session is unavailable; run `agent-message read <username>` again")
		}
		return session, nil
	}

	if len(rt.Config.ReadSessions) == 1 {
		for _, session := range rt.Config.ReadSessions {
			return session, nil
		}
	}
	return config.ReadSession{}, errors.New("multiple read sessions found; run `agent-message read <username>` to select one")
}
