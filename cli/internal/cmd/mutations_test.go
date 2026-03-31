package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"agent-message/cli/internal/config"
)

func TestRunEditMessageByIndex(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "tok-edit", func(req *http.Request, body []byte) (*http.Response, error) {
		if req.Method != http.MethodPatch {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if req.URL.Path != "/api/messages/m2" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode edit payload: %v", err)
		}
		if got, want := payload["content"], "edited text"; got != want {
			t.Fatalf("edit content mismatch: got %q want %q", got, want)
		}

		return jsonResponse(http.StatusOK, `{
			"id":"m2",
			"conversation_id":"c-read",
			"sender_id":"u1",
			"content":"edited text",
			"edited":true,
			"deleted":false,
			"created_at":"2026-01-01T00:00:00Z",
			"updated_at":"2026-01-01T00:02:00Z"
		}`), nil
	})
	seedLastReadSession(t, rt, "c-read", "bob", map[int]string{1: "m2", 2: "m1"})

	if err := runEditMessage(rt, "1", "edited text"); err != nil {
		t.Fatalf("runEditMessage: %v", err)
	}
	if got, want := strings.TrimSpace(stdout.String()), "edited m2"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}

func TestRunDeleteMessageByIndex(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "tok-delete", func(req *http.Request, _ []byte) (*http.Response, error) {
		if req.Method != http.MethodDelete {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if req.URL.Path != "/api/messages/m1" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		return jsonResponse(http.StatusOK, `{
			"id":"m1",
			"conversation_id":"c-read",
			"sender_id":"u1",
			"edited":false,
			"deleted":true,
			"created_at":"2026-01-01T00:00:00Z",
			"updated_at":"2026-01-01T00:03:00Z"
		}`), nil
	})
	seedLastReadSession(t, rt, "c-read", "bob", map[int]string{1: "m2", 2: "m1"})

	if err := runDeleteMessage(rt, "2"); err != nil {
		t.Fatalf("runDeleteMessage: %v", err)
	}
	if got, want := strings.TrimSpace(stdout.String()), "deleted m1"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}

func TestRunReactByIndex(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "tok-react", func(req *http.Request, body []byte) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if req.URL.Path != "/api/messages/m1/reactions" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode react payload: %v", err)
		}
		if got, want := payload["emoji"], "👍"; got != want {
			t.Fatalf("emoji mismatch: got %q want %q", got, want)
		}
		return jsonResponse(http.StatusOK, `{
			"action":"added",
			"reaction":{"id":"r1","message_id":"m1","user_id":"u1","emoji":"👍","created_at":"2026-01-01T00:00:00Z"}
		}`), nil
	})
	seedLastReadSession(t, rt, "c-read", "bob", map[int]string{1: "m1"})

	if err := runReact(rt, "1", "👍"); err != nil {
		t.Fatalf("runReact: %v", err)
	}
	if got, want := strings.TrimSpace(stdout.String()), "reaction added m1 👍"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}

func TestRunUnreactByIndex(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "tok-unreact", func(req *http.Request, _ []byte) (*http.Response, error) {
		if req.Method != http.MethodDelete {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if got, want := req.URL.Path, "/api/messages/m1/reactions/👍"; got != want {
			t.Fatalf("unexpected path: got %q want %q", got, want)
		}
		return jsonResponse(http.StatusOK, `{
			"id":"r1",
			"message_id":"m1",
			"user_id":"u1",
			"emoji":"👍",
			"created_at":"2026-01-01T00:00:00Z"
		}`), nil
	})
	seedLastReadSession(t, rt, "c-read", "bob", map[int]string{1: "m1"})

	if err := runUnreact(rt, "1", "👍"); err != nil {
		t.Fatalf("runUnreact: %v", err)
	}
	if got, want := strings.TrimSpace(stdout.String()), "reaction removed m1 👍"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}

func TestResolveMessageIDFromLastReadRequiresSession(t *testing.T) {
	t.Parallel()

	rt, _, _ := newTestRuntime(t, "http://example.test", "tok", func(req *http.Request, _ []byte) (*http.Response, error) {
		t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		return nil, nil
	})

	_, _, err := resolveMessageIDFromLastRead(rt, "1")
	if err == nil {
		t.Fatalf("expected error")
	}
	if got := err.Error(); !strings.Contains(got, "no read session") {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestResolveMessageIDFromLastReadRequiresKnownIndex(t *testing.T) {
	t.Parallel()

	rt, _, _ := newTestRuntime(t, "http://example.test", "tok", func(req *http.Request, _ []byte) (*http.Response, error) {
		t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		return nil, nil
	})
	seedLastReadSession(t, rt, "c-read", "bob", map[int]string{1: "m1"})

	_, _, err := resolveMessageIDFromLastRead(rt, "2")
	if err == nil {
		t.Fatalf("expected error")
	}
	if got := err.Error(); !strings.Contains(got, "index 2 not found") {
		t.Fatalf("unexpected error: %q", got)
	}
}

func seedLastReadSession(t *testing.T, rt *Runtime, conversationID, username string, indexToMessage map[int]string) {
	t.Helper()

	if rt.Config.ReadSessions == nil {
		rt.Config.ReadSessions = make(map[string]config.ReadSession)
	}
	rt.Config.ReadSessions[conversationID] = config.ReadSession{
		ConversationID: conversationID,
		Username:       username,
		IndexToMessage: indexToMessage,
	}
	rt.Config.LastReadConversationID = conversationID

	if err := rt.ConfigStore.Save(rt.Config); err != nil {
		t.Fatalf("save seeded read session: %v", err)
	}
}
