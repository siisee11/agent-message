package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"agent-messenger/cli/internal/api"
	"agent-messenger/cli/internal/config"

	"github.com/spf13/cobra"
)

const defaultReadLimit = 20

func newSendMessageCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "send <username> <text>",
		Short: "Send a message to a user",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return runSendMessage(rt, args[0], args[1])
		},
	}
}

func runSendMessage(rt *Runtime, username, text string) error {
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

	conversationID, err := resolveConversationIDByUsername(context.Background(), rt, username)
	if err != nil {
		return err
	}

	message, err := rt.Client.SendMessage(context.Background(), conversationID, trimmedText)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(rt.Stdout, "sent %s\n", message.ID)
	return nil
}

func newReadMessagesCommand(rt *Runtime) *cobra.Command {
	n := defaultReadLimit
	cmd := &cobra.Command{
		Use:   "read <username>",
		Short: "Read recent messages from a conversation",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runReadMessages(rt, args[0], n)
		},
	}
	cmd.Flags().IntP("n", "n", defaultReadLimit, "Number of most recent messages to fetch")
	return cmd
}

func runReadMessages(rt *Runtime, username string, limit int) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}
	if err := ensureLoggedIn(rt); err != nil {
		return err
	}
	if limit < 1 {
		return errors.New("n must be a positive integer")
	}

	conversationID, err := resolveConversationIDByUsername(context.Background(), rt, username)
	if err != nil {
		return err
	}

	messages, err := rt.Client.ListMessages(context.Background(), conversationID, "", limit)
	if err != nil {
		return err
	}

	for idx, details := range messages {
		index := idx + 1
		_, _ = fmt.Fprintf(rt.Stdout, "[%d] %s %s: %s\n", index, details.Message.ID, messageSender(details), messageText(details))
	}

	if err := persistReadSession(rt, conversationID, username, messages); err != nil {
		return err
	}
	return nil
}

func persistReadSession(rt *Runtime, conversationID string, username string, messages []api.MessageDetails) error {
	if rt.Config.ReadSessions == nil {
		rt.Config.ReadSessions = make(map[string]config.ReadSession)
	}

	indexToMessage := make(map[int]string, len(messages))
	for idx, details := range messages {
		index := idx + 1
		indexToMessage[index] = strings.TrimSpace(details.Message.ID)
	}

	session := config.ReadSession{
		ConversationID: strings.TrimSpace(conversationID),
		Username:       strings.TrimSpace(username),
		IndexToMessage: indexToMessage,
	}
	if len(messages) > 0 {
		session.LastReadMessage = strings.TrimSpace(messages[0].Message.ID)
	}

	rt.Config.ReadSessions[session.ConversationID] = session
	rt.Config.LastReadConversationID = session.ConversationID
	if err := rt.ConfigStore.Save(rt.Config); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}

func messageSender(details api.MessageDetails) string {
	username := strings.TrimSpace(details.Sender.Username)
	if username != "" {
		return username
	}
	return strings.TrimSpace(details.Message.SenderID)
}

func messageText(details api.MessageDetails) string {
	if details.Message.Deleted {
		return "deleted message"
	}
	if details.Message.Content != nil {
		content := strings.TrimSpace(*details.Message.Content)
		if content != "" {
			return content
		}
	}
	if details.Message.AttachmentURL != nil {
		attachmentURL := strings.TrimSpace(*details.Message.AttachmentURL)
		if attachmentURL != "" {
			return attachmentURL
		}
	}
	return ""
}
