package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"agent-message/cli/internal/api"
	"agent-message/cli/internal/config"

	"github.com/spf13/cobra"
)

const defaultReadLimit = 20

func newSendMessageCommand(rt *Runtime) *cobra.Command {
	var explicitText string
	var explicitJSONRender string
	var jsonRenderFile string
	var payload string
	var payloadFile string
	var payloadStdin bool
	var kind string
	var attachmentPath string
	var toUsername string
	var stdinInput bool
	cmd := &cobra.Command{
		Use:   "send [username] [text-or-inline-json]",
		Short: "Send a message to a user or your configured master",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 2 {
				return fmt.Errorf("accepts at most 2 arg(s), received %d", len(args))
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			options := sendMessageOptions{
				ToUsername:     strings.TrimSpace(toUsername),
				Kind:           strings.TrimSpace(kind),
				AttachmentPath: strings.TrimSpace(attachmentPath),
				Text:           explicitText,
				JSONRender:     explicitJSONRender,
				JSONRenderFile: jsonRenderFile,
				Payload:        payload,
				PayloadFile:    payloadFile,
				PayloadStdin:   payloadStdin,
				Stdin:          stdinInput,
			}
			resolved, err := resolveSendMessageInput(rt.Config, rt.Stdin, options, args)
			if err != nil {
				return err
			}
			if resolved.AttachmentPath != "" {
				return runSendAttachmentMessage(rt, resolved.Username, resolved.Text, resolved.AttachmentPath)
			}
			return runSendMessageWithRequest(rt, resolved.Username, resolved.Request)
		},
	}
	cmd.Flags().StringVar(&toUsername, "to", "", "Override recipient username")
	cmd.Flags().StringVar(&kind, "kind", "text", "Message kind: text or json_render")
	cmd.Flags().StringVar(&attachmentPath, "attach", "", "Path to a file or image to attach")
	cmd.Flags().StringVar(&explicitText, "text", "", "Explicit text message content")
	cmd.Flags().StringVar(&explicitJSONRender, "json-render", "", "Explicit inline json_render payload")
	cmd.Flags().StringVar(&jsonRenderFile, "json-render-file", "", "Read json_render payload from a file")
	cmd.Flags().StringVar(&payload, "payload", "", "Raw JSON payload matching the send request body")
	cmd.Flags().StringVar(&payloadFile, "payload-file", "", "Read the raw send JSON payload from a file")
	cmd.Flags().BoolVar(&payloadStdin, "payload-stdin", false, "Read the raw send JSON payload from stdin")
	cmd.Flags().BoolVar(&stdinInput, "stdin", false, "Read message content from stdin")
	return cmd
}

type sendMessageOptions struct {
	ToUsername     string
	Kind           string
	AttachmentPath string
	Text           string
	JSONRender     string
	JSONRenderFile string
	Payload        string
	PayloadFile    string
	PayloadStdin   bool
	Stdin          bool
}

type resolvedSendMessageInput struct {
	Username       string
	Text           string
	AttachmentPath string
	Request        api.SendMessageRequest
}

func resolveSendMessageInput(cfg config.Config, stdin io.Reader, options sendMessageOptions, args []string) (resolvedSendMessageInput, error) {
	trimmedTo := strings.TrimSpace(options.ToUsername)
	trimmedMaster := strings.TrimSpace(cfg.Master)
	trimmedAttachmentPath := strings.TrimSpace(options.AttachmentPath)

	rawRequest, err := resolveRawSendMessageRequest(stdin, options)
	if err != nil {
		return resolvedSendMessageInput{}, err
	}
	if rawRequest != nil {
		username, resolveErr := resolveExplicitRecipient(trimmedTo, trimmedMaster, args)
		if resolveErr != nil {
			return resolvedSendMessageInput{}, resolveErr
		}
		return resolvedSendMessageInput{
			Username: username,
			Request:  *rawRequest,
		}, nil
	}

	username, text, resolvedKind, err := resolveSendMessageInputs(cfg, stdin, options, args)
	if err != nil {
		return resolvedSendMessageInput{}, err
	}
	if trimmedAttachmentPath != "" {
		return resolvedSendMessageInput{
			Username:       username,
			Text:           text,
			AttachmentPath: trimmedAttachmentPath,
		}, nil
	}

	request, err := buildSendMessageRequest(text, resolvedKind)
	if err != nil {
		return resolvedSendMessageInput{}, err
	}
	return resolvedSendMessageInput{
		Username: username,
		Text:     text,
		Request:  request,
	}, nil
}

func resolveSendMessageInputs(cfg config.Config, stdin io.Reader, options sendMessageOptions, args []string) (string, string, string, error) {
	trimmedTo := strings.TrimSpace(options.ToUsername)
	trimmedMaster := strings.TrimSpace(cfg.Master)
	trimmedAttachmentPath := strings.TrimSpace(options.AttachmentPath)
	trimmedKind := strings.TrimSpace(options.Kind)
	if trimmedKind == "" {
		trimmedKind = "text"
	}

	explicitContent, resolvedKind, err := resolveExplicitSendContent(stdin, options, trimmedKind)
	if err != nil {
		return "", "", "", err
	}
	if explicitContent != nil {
		username, resolveErr := resolveExplicitRecipient(trimmedTo, trimmedMaster, args)
		if resolveErr != nil {
			return "", "", "", resolveErr
		}
		return username, *explicitContent, resolvedKind, nil
	}

	if trimmedTo != "" {
		switch len(args) {
		case 0:
			return trimmedTo, "", trimmedKind, nil
		case 1:
			return trimmedTo, args[0], trimmedKind, nil
		default:
			return "", "", "", errors.New("send accepts at most 1 text-or-inline-json arg when --to is set")
		}
	}

	if trimmedMaster != "" {
		switch len(args) {
		case 0:
			if trimmedAttachmentPath != "" {
				return trimmedMaster, "", trimmedKind, nil
			}
			if trimmedKind == "json_render" {
				return "", "", "", errors.New("json_render inline JSON object is required")
			}
			return "", "", "", errors.New("message text is required")
		case 1:
			return trimmedMaster, args[0], trimmedKind, nil
		case 2:
			return args[0], args[1], trimmedKind, nil
		default:
			return "", "", "", fmt.Errorf("accepts at most 2 arg(s), received %d", len(args))
		}
	}

	switch len(args) {
	case 0:
		return "", "", "", errors.New("username is required; set one with `agent-message config set master <username>` or pass --to <username>")
	case 1:
		return args[0], "", trimmedKind, nil
	case 2:
		return args[0], args[1], trimmedKind, nil
	default:
		return "", "", "", fmt.Errorf("accepts at most 2 arg(s), received %d", len(args))
	}
}

func resolveExplicitSendContent(stdin io.Reader, options sendMessageOptions, fallbackKind string) (*string, string, error) {
	modeCount := 0
	kind := fallbackKind
	var content string

	if strings.TrimSpace(options.Text) != "" {
		modeCount++
		kind = "text"
		content = options.Text
	}
	if strings.TrimSpace(options.JSONRender) != "" {
		modeCount++
		kind = "json_render"
		content = options.JSONRender
	}
	if strings.TrimSpace(options.JSONRenderFile) != "" {
		modeCount++
		fileBytes, err := os.ReadFile(strings.TrimSpace(options.JSONRenderFile))
		if err != nil {
			return nil, "", fmt.Errorf("read json-render file: %w", err)
		}
		kind = "json_render"
		content = string(fileBytes)
	}
	if options.Stdin {
		modeCount++
		if stdin == nil {
			return nil, "", errors.New("stdin reader is not initialized")
		}
		stdinBytes, err := io.ReadAll(stdin)
		if err != nil {
			return nil, "", fmt.Errorf("read stdin: %w", err)
		}
		content = string(stdinBytes)
	}
	if modeCount == 0 {
		return nil, kind, nil
	}
	if modeCount > 1 {
		return nil, "", errors.New("send accepts only one explicit content source among --text, --json-render, --json-render-file, and --stdin")
	}
	if strings.TrimSpace(options.Text) != "" && fallbackKind == "json_render" {
		return nil, "", errors.New("--text cannot be used with --kind json_render")
	}
	if (strings.TrimSpace(options.JSONRender) != "" || strings.TrimSpace(options.JSONRenderFile) != "") && fallbackKind == "text" {
		if strings.TrimSpace(options.Kind) != "" {
			return nil, "", errors.New("--json-render and --json-render-file require --kind json_render or no explicit --kind")
		}
	}

	return &content, kind, nil
}

func resolveExplicitRecipient(toUsername string, master string, args []string) (string, error) {
	if toUsername != "" {
		if len(args) > 0 {
			return "", errors.New("send accepts only a recipient via --to when explicit content flags are used")
		}
		return toUsername, nil
	}
	switch len(args) {
	case 0:
		if master == "" {
			return "", errors.New("username is required; set one with `agent-message config set master <username>` or pass --to <username>")
		}
		return master, nil
	case 1:
		return args[0], nil
	default:
		return "", errors.New("send accepts at most 1 username arg when explicit content flags are used")
	}
}

func resolveRawSendMessageRequest(stdin io.Reader, options sendMessageOptions) (*api.SendMessageRequest, error) {
	rawPayload, err := resolveRawPayload(stdin, rawPayloadOptions{
		Payload:      options.Payload,
		PayloadFile:  options.PayloadFile,
		PayloadStdin: options.PayloadStdin,
	})
	if err != nil {
		return nil, err
	}
	if rawPayload == nil {
		return nil, nil
	}
	if strings.TrimSpace(options.AttachmentPath) != "" {
		return nil, errors.New("raw send payload cannot be combined with --attach")
	}
	if strings.TrimSpace(options.Text) != "" || strings.TrimSpace(options.JSONRender) != "" || strings.TrimSpace(options.JSONRenderFile) != "" || options.Stdin {
		return nil, errors.New("raw send payload cannot be combined with --text, --json-render, --json-render-file, or --stdin")
	}
	request, err := decodeStrictJSONObject[api.SendMessageRequest](rawPayload, "send payload")
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func runSendMessage(rt *Runtime, username, text, kind, attachmentPath string) error {
	if strings.TrimSpace(attachmentPath) != "" {
		trimmedKind := strings.TrimSpace(kind)
		if trimmedKind == "" {
			trimmedKind = "text"
		}
		if trimmedKind != "text" {
			return errors.New("attachments are only supported with kind text")
		}
		return runSendAttachmentMessage(rt, username, text, attachmentPath)
	}
	request, err := buildSendMessageRequest(text, kind)
	if err != nil {
		return err
	}
	return runSendMessageWithRequest(rt, username, request)
}

func runSendAttachmentMessage(rt *Runtime, username, text, attachmentPath string) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}
	if err := ensureLoggedIn(rt); err != nil {
		return err
	}

	trimmedAttachmentPath := strings.TrimSpace(attachmentPath)
	if trimmedAttachmentPath == "" {
		return errors.New("attachment path is required")
	}

	conversationID, err := resolveConversationIDByUsername(context.Background(), rt, username)
	if err != nil {
		return err
	}

	var content *string
	trimmedText := strings.TrimSpace(text)
	if trimmedText != "" {
		content = &trimmedText
	}
	message, err := rt.Client.SendAttachmentMessage(context.Background(), conversationID, api.SendAttachmentMessageRequest{
		Content:        content,
		AttachmentPath: trimmedAttachmentPath,
	})
	if err != nil {
		return err
	}

	return writeTextOrJSON(rt, fmt.Sprintf("sent %s", message.ID), map[string]any{
		"message":         message,
		"recipient":       username,
		"conversation_id": conversationID,
	})
}

func runSendMessageWithRequest(rt *Runtime, username string, request api.SendMessageRequest) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}
	if err := ensureLoggedIn(rt); err != nil {
		return err
	}

	conversationID, err := resolveConversationIDByUsername(context.Background(), rt, username)
	if err != nil {
		return err
	}

	message, err := rt.Client.SendMessage(context.Background(), conversationID, request)
	if err != nil {
		return err
	}

	return writeTextOrJSON(rt, fmt.Sprintf("sent %s", message.ID), map[string]any{
		"message":         message,
		"recipient":       username,
		"conversation_id": conversationID,
	})
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
	if err := persistReadSession(rt, conversationID, username, messages); err != nil {
		return err
	}
	if rt.JSONOutput {
		return writeJSON(rt.Stdout, map[string]any{
			"conversation_id": conversationID,
			"username":        strings.TrimSpace(username),
			"messages":        messages,
		})
	}

	for idx, details := range messages {
		index := idx + 1
		_, _ = fmt.Fprintf(rt.Stdout, "[%d] %s %s: %s\n", index, details.Message.ID, messageSender(details), messageText(details))
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
