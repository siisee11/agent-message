package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestRunSetConversationTitle(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "tok-title", func(req *http.Request, body []byte) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/api/conversations":
			return jsonResponse(http.StatusOK, `{
				"conversation":{"id":"c-title","participant_a":"u1","participant_b":"u2","created_at":"2026-01-01T00:00:00Z"},
				"participant_a":{"id":"u1","account_id":"alice","username":"alice","created_at":"2026-01-01T00:00:00Z"},
				"participant_b":{"id":"u2","account_id":"bob","username":"bob","created_at":"2026-01-01T00:00:00Z"}
			}`), nil
		case req.Method == http.MethodPatch && req.URL.Path == "/api/conversations/c-title":
			var payload map[string]string
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode title payload: %v", err)
			}
			if got, want := payload["title"], "Frontend polish"; got != want {
				t.Fatalf("title mismatch: got %q want %q", got, want)
			}
			return jsonResponse(http.StatusOK, `{
				"conversation":{"id":"c-title","participant_a":"u1","participant_b":"u2","title":"Frontend polish","created_at":"2026-01-01T00:00:00Z"},
				"participant_a":{"id":"u1","account_id":"alice","username":"alice","created_at":"2026-01-01T00:00:00Z"},
				"participant_b":{"id":"u2","account_id":"bob","username":"bob","created_at":"2026-01-01T00:00:00Z"}
			}`), nil
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
			return nil, nil
		}
	})

	if err := runSetConversationTitle(rt, "bob", "Frontend polish"); err != nil {
		t.Fatalf("runSetConversationTitle: %v", err)
	}
	if got, want := strings.TrimSpace(stdout.String()), "title set for bob"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}

func TestRunClearConversationTitle(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "tok-title", func(req *http.Request, body []byte) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/api/conversations":
			return jsonResponse(http.StatusOK, `{
				"conversation":{"id":"c-title","participant_a":"u1","participant_b":"u2","title":"Frontend polish","created_at":"2026-01-01T00:00:00Z"},
				"participant_a":{"id":"u1","account_id":"alice","username":"alice","created_at":"2026-01-01T00:00:00Z"},
				"participant_b":{"id":"u2","account_id":"bob","username":"bob","created_at":"2026-01-01T00:00:00Z"}
			}`), nil
		case req.Method == http.MethodPatch && req.URL.Path == "/api/conversations/c-title":
			var payload map[string]string
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode clear title payload: %v", err)
			}
			if got := payload["title"]; got != "" {
				t.Fatalf("expected empty title payload, got %q", got)
			}
			return jsonResponse(http.StatusOK, `{
				"conversation":{"id":"c-title","participant_a":"u1","participant_b":"u2","created_at":"2026-01-01T00:00:00Z"},
				"participant_a":{"id":"u1","account_id":"alice","username":"alice","created_at":"2026-01-01T00:00:00Z"},
				"participant_b":{"id":"u2","account_id":"bob","username":"bob","created_at":"2026-01-01T00:00:00Z"}
			}`), nil
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
			return nil, nil
		}
	})

	if err := runSetConversationTitle(rt, "bob", ""); err != nil {
		t.Fatalf("runSetConversationTitle(clear): %v", err)
	}
	if got, want := strings.TrimSpace(stdout.String()), "title cleared for bob"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}
