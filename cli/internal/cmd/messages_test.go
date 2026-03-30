package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"agent-messenger/cli/internal/api"
)

func TestRunSendMessageResolvesConversationAndSends(t *testing.T) {
	t.Parallel()

	seenOpen := false
	seenSend := false

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "tok-send", func(req *http.Request, body []byte) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/api/conversations":
			seenOpen = true
			var payload map[string]string
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode open payload: %v", err)
			}
			if got, want := payload["username"], "bob"; got != want {
				t.Fatalf("open username mismatch: got %q want %q", got, want)
			}
			return jsonResponse(http.StatusOK, `{
				"conversation":{"id":"c-send","participant_a":"u1","participant_b":"u2","created_at":"2026-01-01T00:00:00Z"},
				"participant_a":{"id":"u1","username":"alice","created_at":"2026-01-01T00:00:00Z"},
				"participant_b":{"id":"u2","username":"bob","created_at":"2026-01-01T00:00:00Z"}
			}`), nil
		case req.Method == http.MethodPost && req.URL.Path == "/api/conversations/c-send/messages":
			seenSend = true
			var payload map[string]string
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode send payload: %v", err)
			}
			if got, want := payload["content"], "hello world"; got != want {
				t.Fatalf("send content mismatch: got %q want %q", got, want)
			}
			return jsonResponse(http.StatusCreated, `{
				"id":"m-send",
				"conversation_id":"c-send",
				"sender_id":"u1",
				"content":"hello world",
				"edited":false,
				"deleted":false,
				"created_at":"2026-01-01T00:00:00Z",
				"updated_at":"2026-01-01T00:00:00Z"
			}`), nil
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
			return nil, nil
		}
	})

	if err := runSendMessage(rt, "bob", "hello world", "text"); err != nil {
		t.Fatalf("runSendMessage: %v", err)
	}
	if !seenOpen || !seenSend {
		t.Fatalf("expected both open and send calls, seenOpen=%v seenSend=%v", seenOpen, seenSend)
	}
	if got, want := strings.TrimSpace(stdout.String()), "sent m-send"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}

func TestRunSendMessageSupportsJSONRenderKind(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "tok-send", func(req *http.Request, body []byte) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/api/conversations":
			return jsonResponse(http.StatusOK, `{
				"conversation":{"id":"c-send","participant_a":"u1","participant_b":"u2","created_at":"2026-01-01T00:00:00Z"},
				"participant_a":{"id":"u1","username":"alice","created_at":"2026-01-01T00:00:00Z"},
				"participant_b":{"id":"u2","username":"bob","created_at":"2026-01-01T00:00:00Z"}
			}`), nil
		case req.Method == http.MethodPost && req.URL.Path == "/api/conversations/c-send/messages":
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode send payload: %v", err)
			}
			if got, want := payload["kind"], "json_render"; got != want {
				t.Fatalf("send kind mismatch: got %v want %q", got, want)
			}
			spec, ok := payload["json_render_spec"].(map[string]any)
			if !ok {
				t.Fatalf("expected json_render_spec object in payload, got %T", payload["json_render_spec"])
			}
			if got, want := spec["root"], "stack-1"; got != want {
				t.Fatalf("spec root mismatch: got %v want %q", got, want)
			}
			return jsonResponse(http.StatusCreated, `{
				"id":"m-json",
				"conversation_id":"c-send",
				"sender_id":"u1",
				"kind":"json_render",
				"json_render_spec":{"root":"stack-1","elements":{"stack-1":{"type":"Stack"}}},
				"edited":false,
				"deleted":false,
				"created_at":"2026-01-01T00:00:00Z",
				"updated_at":"2026-01-01T00:00:00Z"
			}`), nil
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
			return nil, nil
		}
	})

	if err := runSendMessage(rt, "bob", `{"root":"stack-1","elements":{"stack-1":{"type":"Stack"}}}`, "json_render"); err != nil {
		t.Fatalf("runSendMessage(json_render): %v", err)
	}
	if got, want := strings.TrimSpace(stdout.String()), "sent m-json"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}

func TestRunReadMessagesPrintsAndPersistsIndexMapping(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "tok-read", func(req *http.Request, body []byte) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/api/conversations":
			var payload map[string]string
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode open payload: %v", err)
			}
			if got, want := payload["username"], "bob"; got != want {
				t.Fatalf("open username mismatch: got %q want %q", got, want)
			}
			return jsonResponse(http.StatusOK, `{
				"conversation":{"id":"c-read","participant_a":"u1","participant_b":"u2","created_at":"2026-01-01T00:00:00Z"},
				"participant_a":{"id":"u1","username":"alice","created_at":"2026-01-01T00:00:00Z"},
				"participant_b":{"id":"u2","username":"bob","created_at":"2026-01-01T00:00:00Z"}
			}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/api/conversations/c-read/messages":
			if got, want := req.URL.Query().Get("limit"), "2"; got != want {
				t.Fatalf("limit mismatch: got %q want %q", got, want)
			}
			return jsonResponse(http.StatusOK, `[
				{
					"message":{
						"id":"m2",
						"conversation_id":"c-read",
						"sender_id":"u2",
						"content":"second",
						"edited":false,
						"deleted":false,
						"created_at":"2026-01-01T00:01:00Z",
						"updated_at":"2026-01-01T00:01:00Z"
					},
					"sender":{"id":"u2","username":"bob","created_at":"2026-01-01T00:00:00Z"}
				},
				{
					"message":{
						"id":"m1",
						"conversation_id":"c-read",
						"sender_id":"u1",
						"content":"first",
						"edited":false,
						"deleted":false,
						"created_at":"2026-01-01T00:00:00Z",
						"updated_at":"2026-01-01T00:00:00Z"
					},
					"sender":{"id":"u1","username":"alice","created_at":"2026-01-01T00:00:00Z"}
				}
			]`), nil
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
			return nil, nil
		}
	})

	if err := runReadMessages(rt, "bob", 2); err != nil {
		t.Fatalf("runReadMessages: %v", err)
	}

	gotLines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(gotLines) != 2 {
		t.Fatalf("expected 2 output lines, got %d: %q", len(gotLines), stdout.String())
	}
	if got, want := gotLines[0], "[1] m2 bob: second"; got != want {
		t.Fatalf("line 1 mismatch: got %q want %q", got, want)
	}
	if got, want := gotLines[1], "[2] m1 alice: first"; got != want {
		t.Fatalf("line 2 mismatch: got %q want %q", got, want)
	}

	persisted, err := rt.ConfigStore.Load()
	if err != nil {
		t.Fatalf("load persisted config: %v", err)
	}
	session, ok := persisted.ReadSessions["c-read"]
	if !ok {
		t.Fatalf("expected read session for c-read")
	}
	if got, want := session.Username, "bob"; got != want {
		t.Fatalf("session username mismatch: got %q want %q", got, want)
	}
	if got, want := session.LastReadMessage, "m2"; got != want {
		t.Fatalf("last read message mismatch: got %q want %q", got, want)
	}
	if got, want := session.IndexToMessage[1], "m2"; got != want {
		t.Fatalf("index 1 mismatch: got %q want %q", got, want)
	}
	if got, want := session.IndexToMessage[2], "m1"; got != want {
		t.Fatalf("index 2 mismatch: got %q want %q", got, want)
	}
	if got, want := persisted.LastReadConversationID, "c-read"; got != want {
		t.Fatalf("last read conversation mismatch: got %q want %q", got, want)
	}
}

func TestRunReadMessagesRejectsInvalidLimit(t *testing.T) {
	t.Parallel()

	rt, _, _ := newTestRuntime(t, "http://example.test", "tok-read", func(req *http.Request, _ []byte) (*http.Response, error) {
		t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		return nil, nil
	})

	err := runReadMessages(rt, "bob", 0)
	if err == nil {
		t.Fatalf("expected error for invalid limit")
	}
	if got := err.Error(); !strings.Contains(got, "positive integer") {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestMessageTextUsesJSONRenderPlaceholder(t *testing.T) {
	t.Parallel()

	details := api.MessageDetails{
		Message: api.Message{
			ID:   "m-json",
			Kind: "json_render",
		},
	}

	if got, want := messageText(details), "[json-render]"; got != want {
		t.Fatalf("messageText mismatch: got %q want %q", got, want)
	}
}
