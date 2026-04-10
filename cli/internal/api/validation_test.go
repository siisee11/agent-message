package api

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestSetServerURLRejectsQueryAndFragment(t *testing.T) {
	t.Parallel()

	client, err := NewClient("http://example.test", "")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if err := client.SetServerURL("http://example.test/api?token=oops"); err == nil || !strings.Contains(err.Error(), "query string") {
		t.Fatalf("expected query string error, got %v", err)
	}
	if err := client.SetServerURL("http://example.test/api#frag"); err == nil || !strings.Contains(err.Error(), "fragment") {
		t.Fatalf("expected fragment error, got %v", err)
	}
}

func TestRegisterRejectsInvalidAccountID(t *testing.T) {
	t.Parallel()

	client := newValidationTestClient(t)
	_, err := client.Register(context.Background(), "bob?admin=true", "1234")
	if err == nil || !strings.Contains(err.Error(), "account_id") {
		t.Fatalf("expected account_id validation error, got %v", err)
	}
}

func TestOpenConversationRejectsInvalidUsername(t *testing.T) {
	t.Parallel()

	client := newValidationTestClient(t)
	_, err := client.OpenConversation(context.Background(), "jay\nops")
	if err == nil || !strings.Contains(err.Error(), "username") {
		t.Fatalf("expected username validation error, got %v", err)
	}
}

func TestGetConversationRejectsPathTraversalLikeID(t *testing.T) {
	t.Parallel()

	client := newValidationTestClient(t)
	_, err := client.GetConversation(context.Background(), "../secret")
	if err == nil || !strings.Contains(err.Error(), "dot segments") {
		t.Fatalf("expected dot segment validation error, got %v", err)
	}
}

func TestListMessagesRejectsPercentEncodedBeforeCursor(t *testing.T) {
	t.Parallel()

	client := newValidationTestClient(t)
	_, err := client.ListMessages(context.Background(), "conv-123", "msg%2e%2e", 20)
	if err == nil || !strings.Contains(err.Error(), "percent-encoded segments") {
		t.Fatalf("expected percent-encoding validation error, got %v", err)
	}
}

func TestEditMessageRejectsQuerySyntaxInMessageID(t *testing.T) {
	t.Parallel()

	client := newValidationTestClient(t)
	_, err := client.EditMessage(context.Background(), "msg-123?admin=true", "hello")
	if err == nil || !strings.Contains(err.Error(), "path or query syntax") {
		t.Fatalf("expected path/query validation error, got %v", err)
	}
}

func TestSendAttachmentMessageRejectsControlCharsInAttachmentPath(t *testing.T) {
	t.Parallel()

	client := newValidationTestClient(t)
	_, err := client.SendAttachmentMessage(context.Background(), "conv-123", SendAttachmentMessageRequest{
		AttachmentPath: "note.txt\nshadow",
	})
	if err == nil || !strings.Contains(err.Error(), "control characters") {
		t.Fatalf("expected attachment path validation error, got %v", err)
	}
}

func TestDoJSONRejectsFragmentInRequestPath(t *testing.T) {
	t.Parallel()

	client := newValidationTestClient(t)
	err := client.doJSON(context.Background(), http.MethodGet, "/api/users/me#fragment", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "fragment") {
		t.Fatalf("expected request path fragment validation error, got %v", err)
	}
}

func newValidationTestClient(t *testing.T) *Client {
	t.Helper()

	client, err := NewClient("http://example.test", "test-token")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	client.SetHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected HTTP request: %s %s", req.Method, req.URL.String())
		return nil, nil
	})})
	return client
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
