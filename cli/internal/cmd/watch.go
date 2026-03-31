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

func newWatchCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "watch <username>",
		Short: "Watch incoming messages in real time",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runWatch(rt, args[0])
		},
	}
}

func runWatch(rt *Runtime, username string) error {
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

	streamURL, err := rt.Client.EventStreamURL()
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
		if strings.TrimSpace(message.ConversationID) != conversationID {
			continue
		}

		_, _ = fmt.Fprintf(rt.Stdout, "%s %s: %s\n", message.ID, strings.TrimSpace(message.SenderID), watchMessageText(message))
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
