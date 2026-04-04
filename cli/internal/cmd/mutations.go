package cmd

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"agent-message/cli/internal/config"

	"github.com/spf13/cobra"
)

func newEditMessageCommand(rt *Runtime) *cobra.Command {
	var explicitMessageID string
	cmd := &cobra.Command{
		Use:   "edit [message-id-or-index] <text>",
		Short: "Edit a message by explicit message ID or by index from the last read",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) < 1 || len(args) > 2 {
				return fmt.Errorf("accepts 1 or 2 arg(s), received %d", len(args))
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			return nil
		},
	}
	cmd.Flags().StringVar(&explicitMessageID, "message-id", "", "Edit by explicit message ID")
	indexPtr := cmd.Flags().IntP("index", "", 0, "Edit by index from the last read")
	cmd.RunE = func(_ *cobra.Command, args []string) error {
		selector, text, err := resolveEditArgs(args, explicitMessageID, *indexPtr)
		if err != nil {
			return err
		}
		return runEditMessage(rt, selector, text, explicitMessageID, *indexPtr)
	}
	return cmd
}

func runEditMessage(rt *Runtime, selectorArg string, text string, explicitMessageID string, explicitIndex int) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}
	if err := ensureLoggedIn(rt); err != nil {
		return err
	}

	trimmedText := strings.TrimSpace(text)
	if trimmedText == "" {
		return errors.New("message text is required")
	}

	messageID, source, err := resolveMessageTarget(rt, selectorArg, explicitMessageID, explicitIndex)
	if err != nil {
		return err
	}

	message, err := rt.Client.EditMessage(context.Background(), messageID, trimmedText)
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
	cmd := &cobra.Command{
		Use:   "react [message-id-or-index] <emoji>",
		Short: "React to a message by explicit message ID or by index from the last read",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) < 1 || len(args) > 2 {
				return fmt.Errorf("accepts 1 or 2 arg(s), received %d", len(args))
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
	indexPtr := cmd.Flags().IntP("index", "", 0, "React by index from the last read")
	cmd.RunE = func(_ *cobra.Command, args []string) error {
		selector, emoji, err := resolveReactionArgs(args, explicitMessageID, *indexPtr)
		if err != nil {
			return err
		}
		return runReact(rt, selector, emoji, explicitMessageID, *indexPtr)
	}
	return cmd
}

func runReact(rt *Runtime, selectorArg string, emoji string, explicitMessageID string, explicitIndex int) error {
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
	result, err := rt.Client.AddReaction(context.Background(), messageID, trimmedEmoji)
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
			if len(args) < 1 || len(args) > 2 {
				return fmt.Errorf("accepts 1 or 2 arg(s), received %d", len(args))
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
