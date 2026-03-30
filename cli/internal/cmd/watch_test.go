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
			events: []wsEvent{
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
	if got, want := capturedURL, "ws://example.test/ws?token=tok-watch"; got != want {
		t.Fatalf("websocket URL mismatch: got %q want %q", got, want)
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

type fakeWatchStream struct {
	events []wsEvent
	index  int
}

func (f *fakeWatchStream) ReadEvent() (wsEvent, error) {
	if f.index >= len(f.events) {
		return wsEvent{}, io.EOF
	}
	event := f.events[f.index]
	f.index++
	return event, nil
}

func (f *fakeWatchStream) Close() error {
	return nil
}
