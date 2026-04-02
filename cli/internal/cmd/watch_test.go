package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestRunWatchStreamsOnlyTargetConversationMessageNewEvents(t *testing.T) {
	t.Parallel()

	rt, stdout, stderr := newTestRuntime(t, "http://example.test", "tok-watch", func(req *http.Request, body []byte) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/api/conversations" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode open payload: %v", err)
		}
		if got, want := payload["username"], "bob"; got != want {
			t.Fatalf("open username mismatch: got %q want %q", got, want)
		}

		return jsonResponse(http.StatusOK, `{
			"conversation":{"id":"c-target","participant_a":"u1","participant_b":"u2","created_at":"2026-01-01T00:00:00Z"},
			"participant_a":{"id":"u1","username":"alice","created_at":"2026-01-01T00:00:00Z"},
			"participant_b":{"id":"u2","username":"bob","created_at":"2026-01-01T00:00:00Z"}
		}`), nil
	})

	var capturedURL string
	originalConnect := connectWatchStream
	connectWatchStream = func(rawURL string) (watchStream, error) {
		capturedURL = rawURL
		return &fakeWatchStream{
			events: []streamEvent{
				{Type: "reaction.added", Data: json.RawMessage(`{"message_id":"m-0"}`)},
				{Type: "message.new", Data: json.RawMessage(`{"id":"m-other","conversation_id":"c-other","sender_id":"u9","content":"nope","edited":false,"deleted":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`)},
				{Type: "message.new", Data: json.RawMessage(`{"id":"m-target","conversation_id":"c-target","sender_id":"u2","content":"hello","edited":false,"deleted":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`)},
			},
		}, nil
	}
	t.Cleanup(func() {
		connectWatchStream = originalConnect
	})

	if err := runWatch(rt, "bob"); err != nil {
		t.Fatalf("runWatch: %v", err)
	}

	if got := strings.TrimSpace(stderr.String()); got != "" {
		t.Fatalf("expected empty stderr, got %q", got)
	}
	if got, want := strings.TrimSpace(stdout.String()), "m-target u2: hello"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
	if got, want := capturedURL, "http://example.test/api/events?token=tok-watch"; got != want {
		t.Fatalf("event stream URL mismatch: got %q want %q", got, want)
	}
}

func TestRunWatchRequiresLogin(t *testing.T) {
	t.Parallel()

	rt, _, _ := newTestRuntime(t, "http://example.test", "", func(req *http.Request, _ []byte) (*http.Response, error) {
		t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		return nil, nil
	})

	err := runWatch(rt, "bob")
	if err == nil {
		t.Fatalf("expected error")
	}
	if got := err.Error(); !strings.Contains(got, "not logged in") {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestRunWatchJSONWritesStructuredNDJSON(t *testing.T) {
	t.Parallel()

	rt, stdout, stderr := newTestRuntime(t, "http://example.test", "tok-watch", func(req *http.Request, body []byte) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/api/conversations" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode open payload: %v", err)
		}
		if got, want := payload["username"], "bob"; got != want {
			t.Fatalf("open username mismatch: got %q want %q", got, want)
		}

		return jsonResponse(http.StatusOK, `{
			"conversation":{"id":"c-target","participant_a":"u1","participant_b":"u2","created_at":"2026-01-01T00:00:00Z"},
			"participant_a":{"id":"u1","username":"alice","created_at":"2026-01-01T00:00:00Z"},
			"participant_b":{"id":"u2","username":"bob","created_at":"2026-01-01T00:00:00Z"}
		}`), nil
	})

	originalConnect := connectWatchStream
	connectWatchStream = func(rawURL string) (watchStream, error) {
		return &fakeWatchStream{
			events: []streamEvent{
				{Type: "message.new", Data: json.RawMessage(`{"id":"m-target","conversation_id":"c-target","sender_id":"u2","content":"hello","edited":false,"deleted":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`)},
			},
		}, nil
	}
	t.Cleanup(func() {
		connectWatchStream = originalConnect
	})

	if err := runWatchWithOptions(rt, "bob", watchOptions{jsonOutput: true}); err != nil {
		t.Fatalf("runWatchWithOptions(json): %v", err)
	}

	if got := strings.TrimSpace(stderr.String()); got != "" {
		t.Fatalf("expected empty stderr, got %q", got)
	}

	var event watchJSONEvent
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &event); err != nil {
		t.Fatalf("decode stdout NDJSON: %v", err)
	}
	if got, want := event.Type, "message.new"; got != want {
		t.Fatalf("event type mismatch: got %q want %q", got, want)
	}
	if got, want := event.ConversationID, "c-target"; got != want {
		t.Fatalf("conversation id mismatch: got %q want %q", got, want)
	}
	if got, want := event.Message.Sender.ID, "u2"; got != want {
		t.Fatalf("sender id mismatch: got %q want %q", got, want)
	}
	if got, want := event.Message.Sender.Username, "bob"; got != want {
		t.Fatalf("sender username mismatch: got %q want %q", got, want)
	}
	if got, want := event.Message.Kind, "text"; got != want {
		t.Fatalf("message kind mismatch: got %q want %q", got, want)
	}
	if event.Message.Content == nil || *event.Message.Content != "hello" {
		t.Fatalf("expected content hello, got %+v", event.Message.Content)
	}
}

func TestRunWaitReturnsAfterFirstMatchingEvent(t *testing.T) {
	t.Parallel()

	rt, stdout, stderr := newTestRuntime(t, "http://example.test", "tok-wait", func(req *http.Request, body []byte) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/api/conversations" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode open payload: %v", err)
		}
		if got, want := payload["username"], "bob"; got != want {
			t.Fatalf("open username mismatch: got %q want %q", got, want)
		}

		return jsonResponse(http.StatusOK, `{
			"conversation":{"id":"c-target","participant_a":"u1","participant_b":"u2","created_at":"2026-01-01T00:00:00Z"},
			"participant_a":{"id":"u1","username":"alice","created_at":"2026-01-01T00:00:00Z"},
			"participant_b":{"id":"u2","username":"bob","created_at":"2026-01-01T00:00:00Z"}
		}`), nil
	})

	originalConnect := connectWatchStream
	connectWatchStream = func(rawURL string) (watchStream, error) {
		return &fakeWatchStream{
			events: []streamEvent{
				{Type: "reaction.added", Data: json.RawMessage(`{"message_id":"m-0"}`)},
				{Type: "message.new", Data: json.RawMessage(`{"id":"m-1","conversation_id":"c-target","sender_id":"u2","content":"first","edited":false,"deleted":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`)},
				{Type: "message.new", Data: json.RawMessage(`{"id":"m-2","conversation_id":"c-target","sender_id":"u2","content":"second","edited":false,"deleted":false,"created_at":"2026-01-01T00:00:01Z","updated_at":"2026-01-01T00:00:01Z"}`)},
			},
		}, nil
	}
	t.Cleanup(func() {
		connectWatchStream = originalConnect
	})

	if err := runWait(rt, "bob"); err != nil {
		t.Fatalf("runWait: %v", err)
	}

	if got := strings.TrimSpace(stderr.String()); got != "" {
		t.Fatalf("expected empty stderr, got %q", got)
	}
	if got, want := strings.TrimSpace(stdout.String()), "m-1 u2: first"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}

type fakeWatchStream struct {
	events []streamEvent
	index  int
}

func (f *fakeWatchStream) ReadEvent() (streamEvent, error) {
	if f.index >= len(f.events) {
		return streamEvent{}, io.EOF
	}
	event := f.events[f.index]
	f.index++
	return event, nil
}

func (f *fakeWatchStream) Close() error {
	return nil
}
