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
	return &cobra.Command{
		Use:   "edit <index> <text>",
		Short: "Edit a message by index from last read",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return runEditMessage(rt, args[0], args[1])
		},
	}
}

func runEditMessage(rt *Runtime, indexArg string, text string) error {
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

	messageID, _, err := resolveMessageIDFromLastRead(rt, indexArg)
	if err != nil {
		return err
	}

	message, err := rt.Client.EditMessage(context.Background(), messageID, trimmedText)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(rt.Stdout, "edited %s\n", message.ID)
	return nil
}

func newDeleteMessageCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <index>",
		Short: "Delete a message by index from last read",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runDeleteMessage(rt, args[0])
		},
	}
}

func runDeleteMessage(rt *Runtime, indexArg string) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}
	if err := ensureLoggedIn(rt); err != nil {
		return err
	}

	messageID, _, err := resolveMessageIDFromLastRead(rt, indexArg)
	if err != nil {
		return err
	}

	message, err := rt.Client.DeleteMessage(context.Background(), messageID)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(rt.Stdout, "deleted %s\n", message.ID)
	return nil
}

func newReactCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "react <message-id> <emoji>",
		Short: "React to a message by message ID",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return runReact(rt, args[0], args[1])
		},
	}
}

func runReact(rt *Runtime, messageIDArg string, emoji string) error {
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

	messageID := strings.TrimSpace(messageIDArg)
	if messageID == "" {
		return errors.New("message ID is required")
	}

	result, err := rt.Client.AddReaction(context.Background(), messageID, trimmedEmoji)
	if err != nil {
		return err
	}

	action := strings.TrimSpace(result.Action)
	if action == "" {
		action = "added"
	}
	_, _ = fmt.Fprintf(rt.Stdout, "reaction %s %s %s\n", action, result.Reaction.MessageID, result.Reaction.Emoji)
	return nil
}

func newUnreactCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "unreact <message-id> <emoji>",
		Short: "Remove a reaction from a message by message ID",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return runUnreact(rt, args[0], args[1])
		},
	}
}

func runUnreact(rt *Runtime, messageIDArg string, emoji string) error {
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

	messageID := strings.TrimSpace(messageIDArg)
	if messageID == "" {
		return errors.New("message ID is required")
	}

	reaction, err := rt.Client.RemoveReaction(context.Background(), messageID, trimmedEmoji)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(rt.Stdout, "reaction removed %s %s\n", reaction.MessageID, reaction.Emoji)
	return nil
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
