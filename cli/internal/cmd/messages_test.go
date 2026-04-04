package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"testing"

	"agent-message/cli/internal/api"
	"agent-message/cli/internal/config"
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

	if err := runSendMessage(rt, "bob", "hello world", "text", ""); err != nil {
		t.Fatalf("runSendMessage: %v", err)
	}
	if !seenOpen || !seenSend {
		t.Fatalf("expected both open and send calls, seenOpen=%v seenSend=%v", seenOpen, seenSend)
	}
	if got, want := strings.TrimSpace(stdout.String()), "sent m-send"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}

func TestSendCommandUsesConfiguredMasterWhenUsernameIsOmitted(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "tok-send", func(req *http.Request, body []byte) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/api/conversations":
			var payload map[string]string
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode open payload: %v", err)
			}
			if got, want := payload["username"], "jay"; got != want {
				t.Fatalf("open username mismatch: got %q want %q", got, want)
			}
			return jsonResponse(http.StatusOK, `{
				"conversation":{"id":"c-send","participant_a":"u1","participant_b":"u2","created_at":"2026-01-01T00:00:00Z"},
				"participant_a":{"id":"u1","username":"alice","created_at":"2026-01-01T00:00:00Z"},
				"participant_b":{"id":"u2","username":"jay","created_at":"2026-01-01T00:00:00Z"}
			}`), nil
		case req.Method == http.MethodPost && req.URL.Path == "/api/conversations/c-send/messages":
			var payload map[string]string
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode send payload: %v", err)
			}
			if got, want := payload["content"], "hello master"; got != want {
				t.Fatalf("send content mismatch: got %q want %q", got, want)
			}
			return jsonResponse(http.StatusCreated, `{
				"id":"m-master",
				"conversation_id":"c-send",
				"sender_id":"u1",
				"content":"hello master",
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
	rt.Config.Master = "jay"

	command := newSendMessageCommand(rt)
	command.SetArgs([]string{"hello master"})

	if err := command.Execute(); err != nil {
		t.Fatalf("execute send command: %v", err)
	}
	if got, want := strings.TrimSpace(stdout.String()), "sent m-master"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}

func TestSendCommandUsesToFlagToOverrideConfiguredMaster(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "tok-send", func(req *http.Request, body []byte) (*http.Response, error) {
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
				"conversation":{"id":"c-send","participant_a":"u1","participant_b":"u2","created_at":"2026-01-01T00:00:00Z"},
				"participant_a":{"id":"u1","username":"alice","created_at":"2026-01-01T00:00:00Z"},
				"participant_b":{"id":"u2","username":"bob","created_at":"2026-01-01T00:00:00Z"}
			}`), nil
		case req.Method == http.MethodPost && req.URL.Path == "/api/conversations/c-send/messages":
			var payload map[string]string
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode send payload: %v", err)
			}
			if got, want := payload["content"], "hello bob"; got != want {
				t.Fatalf("send content mismatch: got %q want %q", got, want)
			}
			return jsonResponse(http.StatusCreated, `{
				"id":"m-override",
				"conversation_id":"c-send",
				"sender_id":"u1",
				"content":"hello bob",
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
	rt.Config.Master = "jay"

	command := newSendMessageCommand(rt)
	command.SetArgs([]string{"--to", "bob", "hello bob"})

	if err := command.Execute(); err != nil {
		t.Fatalf("execute send command: %v", err)
	}
	if got, want := strings.TrimSpace(stdout.String()), "sent m-override"; got != want {
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

	if err := runSendMessage(rt, "bob", `{"root":"stack-1","elements":{"stack-1":{"type":"Stack"}}}`, "json_render", ""); err != nil {
		t.Fatalf("runSendMessage(json_render): %v", err)
	}
	if got, want := strings.TrimSpace(stdout.String()), "sent m-json"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}

func TestSendCommandSupportsRawPayload(t *testing.T) {
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
				t.Fatalf("kind mismatch: got %v want %q", got, want)
			}
			spec, ok := payload["json_render_spec"].(map[string]any)
			if !ok {
				t.Fatalf("expected json_render_spec object, got %T", payload["json_render_spec"])
			}
			if got, want := spec["root"], "stack-1"; got != want {
				t.Fatalf("spec root mismatch: got %v want %q", got, want)
			}
			return jsonResponse(http.StatusCreated, `{
				"id":"m-json-raw",
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

	command := newSendMessageCommand(rt)
	command.SetArgs([]string{"--to", "bob", "--payload", `{"kind":"json_render","json_render_spec":{"root":"stack-1","elements":{"stack-1":{"type":"Stack"}}}}`})

	if err := command.Execute(); err != nil {
		t.Fatalf("execute send command: %v", err)
	}
	if got, want := strings.TrimSpace(stdout.String()), "sent m-json-raw"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}

func TestRunSendMessageSupportsJSONOutput(t *testing.T) {
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
			var payload map[string]string
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode send payload: %v", err)
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
	rt.JSONOutput = true

	if err := runSendMessage(rt, "bob", "hello world", "text", ""); err != nil {
		t.Fatalf("runSendMessage: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode stdout json: %v", err)
	}
	if got, want := payload["recipient"], "bob"; got != want {
		t.Fatalf("recipient mismatch: got %v want %q", got, want)
	}
}

func TestRunSendMessageSupportsAttachment(t *testing.T) {
	t.Parallel()

	attachmentPath := createTestAttachmentFile(t, "diagram.png", []byte("png-bytes"))

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "tok-send", func(req *http.Request, body []byte) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/api/conversations":
			return jsonResponse(http.StatusOK, `{
				"conversation":{"id":"c-send","participant_a":"u1","participant_b":"u2","created_at":"2026-01-01T00:00:00Z"},
				"participant_a":{"id":"u1","username":"alice","created_at":"2026-01-01T00:00:00Z"},
				"participant_b":{"id":"u2","username":"bob","created_at":"2026-01-01T00:00:00Z"}
			}`), nil
		case req.Method == http.MethodPost && req.URL.Path == "/api/conversations/c-send/messages":
			mediaType, params, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
			if err != nil {
				t.Fatalf("parse content type: %v", err)
			}
			if got, want := mediaType, "multipart/form-data"; got != want {
				t.Fatalf("content type mismatch: got %q want %q", got, want)
			}

			reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
			contentFieldSeen := false
			attachmentSeen := false

			for {
				part, err := reader.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("read multipart part: %v", err)
				}

				partBody, err := io.ReadAll(part)
				if err != nil {
					t.Fatalf("read multipart part body: %v", err)
				}

				switch part.FormName() {
				case "content":
					contentFieldSeen = true
					if got, want := string(partBody), "hello with image"; got != want {
						t.Fatalf("content mismatch: got %q want %q", got, want)
					}
				case "attachment":
					attachmentSeen = true
					if got, want := part.FileName(), "diagram.png"; got != want {
						t.Fatalf("attachment filename mismatch: got %q want %q", got, want)
					}
					if got, want := part.Header.Get("Content-Type"), "image/png"; got != want {
						t.Fatalf("attachment content type mismatch: got %q want %q", got, want)
					}
					if got, want := string(partBody), "png-bytes"; got != want {
						t.Fatalf("attachment body mismatch: got %q want %q", got, want)
					}
				default:
					t.Fatalf("unexpected multipart field: %q", part.FormName())
				}
			}

			if !contentFieldSeen {
				t.Fatalf("expected content field")
			}
			if !attachmentSeen {
				t.Fatalf("expected attachment field")
			}

			return jsonResponse(http.StatusCreated, `{
				"id":"m-attachment",
				"conversation_id":"c-send",
				"sender_id":"u1",
				"content":"hello with image",
				"attachment_url":"/static/uploads/diagram.png",
				"attachment_type":"image",
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

	if err := runSendMessage(rt, "bob", "hello with image", "text", attachmentPath); err != nil {
		t.Fatalf("runSendMessage(attachment): %v", err)
	}
	if got, want := strings.TrimSpace(stdout.String()), "sent m-attachment"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}

func TestRunSendMessageRejectsAttachmentForJSONRender(t *testing.T) {
	t.Parallel()

	attachmentPath := createTestAttachmentFile(t, "diagram.png", []byte("png-bytes"))
	rt, _, _ := newTestRuntime(t, "http://example.test", "tok-send", func(req *http.Request, _ []byte) (*http.Response, error) {
		t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		return nil, nil
	})

	err := runSendMessage(rt, "bob", `{"root":"stack-1"}`, "json_render", attachmentPath)
	if err == nil {
		t.Fatalf("expected error")
	}
	if got := err.Error(); !strings.Contains(got, "attachments are only supported with kind text") {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestResolveSendMessageInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		cfg            config.Config
		to             string
		args           []string
		kind           string
		attachmentPath string
		wantUsername   string
		wantText       string
		wantErr        string
	}{
		{
			name:    "requires username without master",
			args:    nil,
			kind:    "text",
			wantErr: "config set master",
		},
		{
			name:         "uses master for single text arg",
			cfg:          config.Config{Master: "jay"},
			args:         []string{"hello"},
			kind:         "text",
			wantUsername: "jay",
			wantText:     "hello",
		},
		{
			name:         "allows explicit username with master",
			cfg:          config.Config{Master: "jay"},
			args:         []string{"bob", "hello"},
			kind:         "text",
			wantUsername: "bob",
			wantText:     "hello",
		},
		{
			name:           "uses master for attachment without text",
			cfg:            config.Config{Master: "jay"},
			args:           nil,
			kind:           "text",
			attachmentPath: "/tmp/file.png",
			wantUsername:   "jay",
		},
		{
			name:         "to flag overrides master",
			cfg:          config.Config{Master: "jay"},
			to:           "bob",
			args:         []string{"hello"},
			kind:         "text",
			wantUsername: "bob",
			wantText:     "hello",
		},
		{
			name:    "to flag rejects extra positional username",
			cfg:     config.Config{Master: "jay"},
			to:      "bob",
			args:    []string{"alice", "hello"},
			kind:    "text",
			wantErr: "when --to is set",
		},
		{
			name:    "master still requires json payload",
			cfg:     config.Config{Master: "jay"},
			args:    nil,
			kind:    "json_render",
			wantErr: "json_render inline JSON object is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotUsername, gotText, _, err := resolveSendMessageInputs(tc.cfg, nil, sendMessageOptions{
				ToUsername:     tc.to,
				Kind:           tc.kind,
				AttachmentPath: tc.attachmentPath,
			}, tc.args)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error")
				}
				if got := err.Error(); !strings.Contains(got, tc.wantErr) {
					t.Fatalf("unexpected error: %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveSendMessageInputs: %v", err)
			}
			if gotUsername != tc.wantUsername {
				t.Fatalf("username mismatch: got %q want %q", gotUsername, tc.wantUsername)
			}
			if gotText != tc.wantText {
				t.Fatalf("text mismatch: got %q want %q", gotText, tc.wantText)
			}
		})
	}
}

func TestResolveSendMessageInputsSupportsExplicitContentFlags(t *testing.T) {
	t.Parallel()

	gotUsername, gotText, gotKind, err := resolveSendMessageInputs(config.Config{Master: "jay"}, nil, sendMessageOptions{
		Text: "hello explicit",
	}, nil)
	if err != nil {
		t.Fatalf("resolveSendMessageInputs: %v", err)
	}
	if got, want := gotUsername, "jay"; got != want {
		t.Fatalf("username mismatch: got %q want %q", got, want)
	}
	if got, want := gotText, "hello explicit"; got != want {
		t.Fatalf("text mismatch: got %q want %q", got, want)
	}
	if got, want := gotKind, "text"; got != want {
		t.Fatalf("kind mismatch: got %q want %q", got, want)
	}
}

func TestResolveSendMessageInputsSupportsJSONRenderFile(t *testing.T) {
	t.Parallel()

	path := createTestAttachmentFile(t, "spec.json", []byte(`{"root":"stack-1"}`))
	gotUsername, gotText, gotKind, err := resolveSendMessageInputs(config.Config{}, nil, sendMessageOptions{
		JSONRenderFile: path,
	}, []string{"bob"})
	if err != nil {
		t.Fatalf("resolveSendMessageInputs: %v", err)
	}
	if got, want := gotUsername, "bob"; got != want {
		t.Fatalf("username mismatch: got %q want %q", got, want)
	}
	if got, want := gotText, `{"root":"stack-1"}`; got != want {
		t.Fatalf("text mismatch: got %q want %q", got, want)
	}
	if got, want := gotKind, "json_render"; got != want {
		t.Fatalf("kind mismatch: got %q want %q", got, want)
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

func createTestAttachmentFile(t *testing.T, name string, contents []byte) string {
	t.Helper()

	path := t.TempDir() + string(os.PathSeparator) + name
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write attachment file: %v", err)
	}
	return path
}
