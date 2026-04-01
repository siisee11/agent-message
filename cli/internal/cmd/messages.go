package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"agent-message/cli/internal/api"
	"agent-message/cli/internal/config"

	"github.com/spf13/cobra"
)

const defaultReadLimit = 20

func newSendMessageCommand(rt *Runtime) *cobra.Command {
	var kind string
	var attachmentPath string
	cmd := &cobra.Command{
		Use:   "send <username> [text-or-inline-json]",
		Short: "Send a message to a user",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) < 1 || len(args) > 2 {
				return fmt.Errorf("accepts 1 to 2 arg(s), received %d", len(args))
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			text := ""
			if len(args) == 2 {
				text = args[1]
			}
			return runSendMessage(rt, args[0], text, kind, attachmentPath)
		},
	}
	cmd.Flags().StringVar(&kind, "kind", "text", "Message kind: text or json_render")
	cmd.Flags().StringVar(&attachmentPath, "attach", "", "Path to a file or image to attach")
	return cmd
}

func runSendMessage(rt *Runtime, username, text, kind, attachmentPath string) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}
	if err := ensureLoggedIn(rt); err != nil {
		return err
	}

	trimmedKind := strings.TrimSpace(kind)
	if trimmedKind == "" {
		trimmedKind = "text"
	}

	trimmedAttachmentPath := strings.TrimSpace(attachmentPath)
	if trimmedAttachmentPath != "" && trimmedKind != "text" {
		return errors.New("attachments are only supported with kind text")
	}

	conversationID, err := resolveConversationIDByUsername(context.Background(), rt, username)
	if err != nil {
		return err
	}

	var message api.Message
	if trimmedAttachmentPath != "" {
		var content *string
		trimmedText := strings.TrimSpace(text)
		if trimmedText != "" {
			content = &trimmedText
		}
		message, err = rt.Client.SendAttachmentMessage(context.Background(), conversationID, api.SendAttachmentMessageRequest{
			Content:        content,
			AttachmentPath: trimmedAttachmentPath,
		})
	} else {
		request, buildErr := buildSendMessageRequest(text, trimmedKind)
		if buildErr != nil {
			return buildErr
		}
		message, err = rt.Client.SendMessage(context.Background(), conversationID, request)
	}
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(rt.Stdout, "sent %s\n", message.ID)
	return nil
}

func buildSendMessageRequest(rawText, kind string) (api.SendMessageRequest, error) {
	switch kind {
	case "text":
		trimmedText := strings.TrimSpace(rawText)
		if trimmedText == "" {
			return api.SendMessageRequest{}, errors.New("message text is required")
		}
		return api.SendMessageRequest{Content: &trimmedText}, nil
	case "json_render":
		spec, err := parseInlineJSONObject(rawText)
		if err != nil {
			return api.SendMessageRequest{}, err
		}
		return api.SendMessageRequest{
			Kind:           "json_render",
			JSONRenderSpec: spec,
		}, nil
	default:
		return api.SendMessageRequest{}, errors.New("kind must be text or json_render")
	}
}

func parseInlineJSONObject(raw string) (json.RawMessage, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("json_render inline JSON object is required")
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return nil, fmt.Errorf("invalid json_render inline JSON object: %w", err)
	}

	normalized, err := json.Marshal(decoded)
	if err != nil {
		return nil, fmt.Errorf("normalize json_render inline JSON object: %w", err)
	}
	return json.RawMessage(normalized), nil
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
	if strings.TrimSpace(details.Message.Kind) == "json_render" {
		return "[json-render]"
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
