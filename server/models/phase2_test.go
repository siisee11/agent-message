package models

import (
	"encoding/json"
	"testing"
)

func stringPtr(value string) *string {
	return &value
}

func TestStartConversationRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		req     StartConversationRequest
		wantErr bool
	}{
		{
			name: "valid",
			req: StartConversationRequest{
				Username: "alice",
			},
		},
		{
			name: "invalid username format",
			req: StartConversationRequest{
				Username: "alice smith",
			},
			wantErr: true,
		},
		{
			name: "empty username",
			req: StartConversationRequest{
				Username: "   ",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr && err == nil {
				t.Fatalf("expected validation error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no validation error, got %v", err)
			}
		})
	}
}

func TestMessageRequestsValidate(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{name: "valid", content: "hello"},
		{name: "empty", content: "   ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run("send_"+tt.name, func(t *testing.T) {
			err := (SendMessageRequest{Content: stringPtr(tt.content)}).Validate()
			if tt.wantErr && err == nil {
				t.Fatalf("expected validation error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no validation error, got %v", err)
			}
		})

		t.Run("edit_"+tt.name, func(t *testing.T) {
			err := (EditMessageRequest{Content: tt.content}).Validate()
			if tt.wantErr && err == nil {
				t.Fatalf("expected validation error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no validation error, got %v", err)
			}
		})
	}
}

func TestSendMessageRequestValidateJSONRender(t *testing.T) {
	validSpec := json.RawMessage(`{"root":"stack-1","elements":{"stack-1":{"type":"Stack"}}}`)

	if err := (SendMessageRequest{
		Kind:           MessageKindJSONRender,
		JSONRenderSpec: validSpec,
	}).Validate(); err != nil {
		t.Fatalf("expected valid json_render request, got %v", err)
	}

	if err := (SendMessageRequest{
		Kind:           MessageKindJSONRender,
		JSONRenderSpec: json.RawMessage(`["not-an-object"]`),
	}).Validate(); err == nil {
		t.Fatalf("expected validation error for non-object json_render_spec")
	}
}

func TestListMessagesQueryNormalize(t *testing.T) {
	query := &ListMessagesQuery{}
	if err := query.Normalize(); err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if query.Limit != DefaultMessagePageLimit {
		t.Fatalf("expected default limit %d, got %d", DefaultMessagePageLimit, query.Limit)
	}

	query = &ListMessagesQuery{Before: " msg-1 ", Limit: 101}
	if err := query.Normalize(); err == nil {
		t.Fatalf("expected out-of-range limit validation error")
	}

	query = &ListMessagesQuery{Before: " msg-2 ", Limit: 10}
	if err := query.Normalize(); err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if query.Before != "msg-2" {
		t.Fatalf("expected trimmed before value, got %q", query.Before)
	}
}
