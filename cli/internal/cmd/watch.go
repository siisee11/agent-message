package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"agent-message/cli/internal/api"

	"github.com/spf13/cobra"
)

type streamEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type watchStream interface {
	ReadEvent() (streamEvent, error)
	Close() error
}

var connectWatchStream = connectSSEWatchStream

type watchOptions struct {
	jsonOutput bool
	once       bool
}

type watchConversation struct {
	conversationID string
	senderNames    map[string]string
}

type watchJSONEvent struct {
	Type           string           `json:"type"`
	ConversationID string           `json:"conversation_id"`
	Message        watchJSONMessage `json:"message"`
}

type watchJSONMessage struct {
	ID             string          `json:"id"`
	ConversationID string          `json:"conversation_id"`
	Sender         watchJSONSender `json:"sender"`
	Content        *string         `json:"content,omitempty"`
	Kind           string          `json:"kind"`
	JSONRenderSpec json.RawMessage `json:"json_render_spec,omitempty"`
	AttachmentURL  *string         `json:"attachment_url,omitempty"`
	AttachmentType *string         `json:"attachment_type,omitempty"`
	Edited         bool            `json:"edited"`
	Deleted        bool            `json:"deleted"`
	CreatedAt      string          `json:"created_at"`
	UpdatedAt      string          `json:"updated_at"`
}

type watchJSONSender struct {
	ID       string `json:"id"`
	Username string `json:"username,omitempty"`
}

func newWatchCommand(rt *Runtime) *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "watch <username>",
		Short: "Watch incoming messages in real time",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runWatchWithOptions(rt, args[0], watchOptions{jsonOutput: jsonOutput})
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output matching events as NDJSON")
	return cmd
}

func newWaitCommand(rt *Runtime) *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "wait <username>",
		Short: "Wait for the next message in a conversation",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runWatchWithOptions(rt, args[0], watchOptions{jsonOutput: jsonOutput, once: true})
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output the next matching event as NDJSON")
	return cmd
}

func runWatch(rt *Runtime, username string) error {
	return runWatchWithOptions(rt, username, watchOptions{})
}

func runWait(rt *Runtime, username string) error {
	return runWatchWithOptions(rt, username, watchOptions{once: true})
}

func runWatchWithOptions(rt *Runtime, username string, options watchOptions) error {
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
	conversation := newWatchConversation(details)

	streamURL, err := rt.Client.EventStreamURL("watcher")
	if err != nil {
		return err
	}

	stream, err := connectWatchStream(streamURL)
	if err != nil {
		return err
	}
	defer stream.Close()

	for {
		event, err := stream.ReadEvent()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		if strings.TrimSpace(event.Type) != "message.new" {
			continue
		}

		var message api.Message
		if err := json.Unmarshal(event.Data, &message); err != nil {
			_, _ = fmt.Fprintf(rt.Stderr, "warning: failed to decode stream message.new payload: %v\n", err)
			continue
		}
		if !conversation.matches(message) {
			continue
		}

		if options.jsonOutput {
			if err := writeWatchJSONEvent(rt.Stdout, conversation, event.Type, message); err != nil {
				return fmt.Errorf("write watch JSON event: %w", err)
			}
		} else {
			_, _ = fmt.Fprintf(rt.Stdout, "%s %s: %s\n", message.ID, strings.TrimSpace(message.SenderID), watchMessageText(message))
		}
		if options.once {
			return nil
		}
	}
}

func watchMessageText(message api.Message) string {
	if message.Deleted {
		return "deleted message"
	}
	if strings.TrimSpace(message.Kind) == "json_render" {
		return "[json-render]"
	}
	if message.Content != nil {
		content := strings.TrimSpace(*message.Content)
		if content != "" {
			return content
		}
	}
	if message.AttachmentURL != nil {
		attachmentURL := strings.TrimSpace(*message.AttachmentURL)
		if attachmentURL != "" {
			return attachmentURL
		}
	}
	return ""
}

func newWatchConversation(details api.ConversationDetails) watchConversation {
	senderNames := map[string]string{
		strings.TrimSpace(details.ParticipantA.ID): strings.TrimSpace(details.ParticipantA.Username),
		strings.TrimSpace(details.ParticipantB.ID): strings.TrimSpace(details.ParticipantB.Username),
	}
	delete(senderNames, "")
	return watchConversation{
		conversationID: strings.TrimSpace(details.Conversation.ID),
		senderNames:    senderNames,
	}
}

func (c watchConversation) matches(message api.Message) bool {
	return strings.TrimSpace(message.ConversationID) == c.conversationID
}

func (c watchConversation) senderUsername(senderID string) string {
	return c.senderNames[strings.TrimSpace(senderID)]
}

func writeWatchJSONEvent(w io.Writer, conversation watchConversation, eventType string, message api.Message) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(watchJSONEvent{
		Type:           strings.TrimSpace(eventType),
		ConversationID: strings.TrimSpace(message.ConversationID),
		Message: watchJSONMessage{
			ID:             strings.TrimSpace(message.ID),
			ConversationID: strings.TrimSpace(message.ConversationID),
			Sender: watchJSONSender{
				ID:       strings.TrimSpace(message.SenderID),
				Username: conversation.senderUsername(message.SenderID),
			},
			Content:        message.Content,
			Kind:           watchMessageKind(message),
			JSONRenderSpec: message.JSONRenderSpec,
			AttachmentURL:  message.AttachmentURL,
			AttachmentType: message.AttachmentType,
			Edited:         message.Edited,
			Deleted:        message.Deleted,
			CreatedAt:      message.CreatedAt.UTC().Format(time.RFC3339Nano),
			UpdatedAt:      message.UpdatedAt.UTC().Format(time.RFC3339Nano),
		},
	})
}

func watchMessageKind(message api.Message) string {
	kind := strings.TrimSpace(message.Kind)
	if kind == "" {
		return "text"
	}
	return kind
}

type sseStream struct {
	body   io.ReadCloser
	reader *bufio.Reader
}

func connectSSEWatchStream(rawURL string) (watchStream, error) {
	req, err := http.NewRequest(http.MethodGet, strings.TrimSpace(rawURL), nil)
	if err != nil {
		return nil, fmt.Errorf("build event stream request: %w", err)
	}

	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connect event stream: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("event stream failed: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}

	return &sseStream{
		body:   resp.Body,
		reader: bufio.NewReader(resp.Body),
	}, nil
}

func (s *sseStream) ReadEvent() (streamEvent, error) {
	var eventType string
	var payloadLine string

	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			return streamEvent{}, err
		}

		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "" {
			if eventType == "" || payloadLine == "" {
				continue
			}

			var event streamEvent
			if err := json.Unmarshal([]byte(payloadLine), &event); err != nil {
				return streamEvent{}, err
			}
			return event, nil
		}

		if value, ok := strings.CutPrefix(trimmed, "event: "); ok {
			eventType = value
			continue
		}
		if value, ok := strings.CutPrefix(trimmed, "data: "); ok {
			payloadLine = value
		}
	}
}

func (s *sseStream) Close() error {
	return s.body.Close()
}
