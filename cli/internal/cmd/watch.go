package cmd

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
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

var watcherHeartbeatInterval = 10 * time.Second

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
	session, err := openWatchSession(rt)
	if err != nil {
		return err
	}
	defer func() {
		closeWatchSession(rt, session, false)
	}()

	for {
		event, err := session.stream.ReadEvent()
		if err != nil {
			reconnect := session.reconnectRequested()
			closeWatchSession(rt, session, reconnect)
			if reconnect {
				session, err = openWatchSession(rt)
				if err != nil {
					return fmt.Errorf("reconnect watch stream: %w", err)
				}
				continue
			}
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

func newWatcherSessionID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generate watcher session id: %w", err)
	}
	return hex.EncodeToString(bytes[:]), nil
}

type watchSession struct {
	watcherSessionID string
	stream           watchStream
	cancelHeartbeat  context.CancelFunc
	reconnectCh      <-chan struct{}
}

func openWatchSession(rt *Runtime) (*watchSession, error) {
	watcherSessionID, err := newWatcherSessionID()
	if err != nil {
		return nil, err
	}

	streamURL, err := rt.Client.EventStreamURLWithWatcherSession("watcher", watcherSessionID)
	if err != nil {
		return nil, err
	}

	stream, err := connectWatchStream(streamURL)
	if err != nil {
		return nil, err
	}

	heartbeatCtx, cancelHeartbeat := context.WithCancel(context.Background())
	reconnectCh := make(chan struct{})
	go runWatcherHeartbeats(heartbeatCtx, rt, watcherSessionID, stream, reconnectCh)

	return &watchSession{
		watcherSessionID: watcherSessionID,
		stream:           stream,
		cancelHeartbeat:  cancelHeartbeat,
		reconnectCh:      reconnectCh,
	}, nil
}

func closeWatchSession(rt *Runtime, session *watchSession, skipUnregister bool) {
	if session == nil {
		return
	}
	if session.cancelHeartbeat != nil {
		session.cancelHeartbeat()
	}
	if session.stream != nil {
		_ = session.stream.Close()
	}
	if !skipUnregister {
		unregisterWatcherSession(rt, session.watcherSessionID)
	}
}

func (s *watchSession) reconnectRequested() bool {
	if s == nil || s.reconnectCh == nil {
		return false
	}
	select {
	case <-s.reconnectCh:
		return true
	default:
		return false
	}
}

func runWatcherHeartbeats(
	ctx context.Context,
	rt *Runtime,
	watcherSessionID string,
	stream watchStream,
	reconnectCh chan<- struct{},
) {
	ticker := time.NewTicker(watcherHeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			heartbeatCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := rt.Client.WatcherHeartbeat(heartbeatCtx, watcherSessionID)
			cancel()
			if err == nil || errors.Is(err, context.Canceled) {
				continue
			}
			if isWatcherSessionNotFound(err) {
				close(reconnectCh)
				_ = stream.Close()
				return
			}
			_, _ = fmt.Fprintf(rt.Stderr, "warning: watcher heartbeat failed: %v\n", err)
		}
	}
}

func isWatcherSessionNotFound(err error) bool {
	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == http.StatusNotFound &&
		strings.EqualFold(strings.TrimSpace(apiErr.Message), "watcher session not found")
}

func unregisterWatcherSession(rt *Runtime, watcherSessionID string) {
	unregisterCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rt.Client.UnregisterWatcherSession(unregisterCtx, watcherSessionID); err != nil {
		_, _ = fmt.Fprintf(rt.Stderr, "warning: watcher session cleanup failed: %v\n", err)
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
