package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestRunListConversationsPrintsConversationRows(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "tok-1", func(req *http.Request, _ []byte) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if req.URL.Path != "/api/conversations" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		if got, want := req.Header.Get("Authorization"), "Bearer tok-1"; got != want {
			t.Fatalf("authorization mismatch: got %q want %q", got, want)
		}

		return jsonResponse(http.StatusOK, `[
			{"conversation":{"id":"c1","participant_a":"u1","participant_b":"u2","created_at":"2026-01-01T00:00:00Z"},"other_user":{"id":"u2","username":"bob","created_at":"2026-01-01T00:00:00Z"}},
			{"conversation":{"id":"c2","participant_a":"u1","participant_b":"u3","created_at":"2026-01-01T00:00:00Z"},"other_user":{"id":"u3","username":"carol","created_at":"2026-01-01T00:00:00Z"}}
		]`), nil
	})

	if err := runListConversations(rt); err != nil {
		t.Fatalf("runListConversations: %v", err)
	}

	gotLines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(gotLines) != 2 {
		t.Fatalf("expected 2 output lines, got %d: %q", len(gotLines), stdout.String())
	}
	if gotLines[0] != "c1 bob" {
		t.Fatalf("unexpected first line: %q", gotLines[0])
	}
	if gotLines[1] != "c2 carol" {
		t.Fatalf("unexpected second line: %q", gotLines[1])
	}
}

func TestRunOpenConversationUsesResolverAndPrintsConversation(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "tok-2", func(req *http.Request, body []byte) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if req.URL.Path != "/api/conversations" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}

		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode request payload: %v", err)
		}
		if got, want := payload["username"], "boB"; got != want {
			t.Fatalf("username mismatch: got %q want %q", got, want)
		}

		return jsonResponse(http.StatusCreated, `{
			"conversation":{"id":"c-open","participant_a":"u1","participant_b":"u2","created_at":"2026-01-01T00:00:00Z"},
			"participant_a":{"id":"u1","username":"alice","created_at":"2026-01-01T00:00:00Z"},
			"participant_b":{"id":"u2","username":"bob","created_at":"2026-01-01T00:00:00Z"}
		}`), nil
	})

	if err := runOpenConversation(rt, "boB"); err != nil {
		t.Fatalf("runOpenConversation: %v", err)
	}

	if got, want := strings.TrimSpace(stdout.String()), "c-open bob"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}

func TestResolveConversationIDByUsernameReturnsConversationID(t *testing.T) {
	t.Parallel()

	rt, _, _ := newTestRuntime(t, "http://example.test", "tok-3", func(req *http.Request, body []byte) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/api/conversations" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["username"] != "dave" {
			t.Fatalf("unexpected payload: %+v", payload)
		}

		return jsonResponse(http.StatusOK, `{
			"conversation":{"id":"c-123","participant_a":"u1","participant_b":"u2","created_at":"2026-01-01T00:00:00Z"},
			"participant_a":{"id":"u1","username":"alice","created_at":"2026-01-01T00:00:00Z"},
			"participant_b":{"id":"u2","username":"dave","created_at":"2026-01-01T00:00:00Z"}
		}`), nil
	})

	conversationID, err := resolveConversationIDByUsername(context.Background(), rt, "dave")
	if err != nil {
		t.Fatalf("resolveConversationIDByUsername: %v", err)
	}
	if got, want := conversationID, "c-123"; got != want {
		t.Fatalf("conversation ID mismatch: got %q want %q", got, want)
	}
}

func TestResolveConversationByUsernameRequiresLogin(t *testing.T) {
	t.Parallel()

	rt, _, _ := newTestRuntime(t, "http://example.test", "", func(req *http.Request, _ []byte) (*http.Response, error) {
		t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		return nil, nil
	})

	_, err := resolveConversationByUsername(context.Background(), rt, "bob")
	if err == nil {
		t.Fatalf("expected an error")
	}
	if got := err.Error(); !strings.Contains(got, "not logged in") {
		t.Fatalf("expected not logged in error, got %q", got)
	}
}
