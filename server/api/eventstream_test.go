package api

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-message/server/realtime"
	"agent-message/server/store"
)

func TestEventStreamAuthValidation(t *testing.T) {
	server, _ := newEventStreamTestServer(t)

	resp, err := http.Get(server.URL + "/api/events")
	if err != nil {
		t.Fatalf("get event stream: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, resp.StatusCode)
	}
	assertErrorBody(t, resp.Body, "missing or invalid bearer token")
}

func TestEventStreamBroadcastsMessageEvents(t *testing.T) {
	server, _ := newEventStreamTestServer(t)
	alice := registerAndLoginUser(t, server.Config.Handler, "alice", "1234")
	bob := registerAndLoginUser(t, server.Config.Handler, "bob", "1234")
	conversationID := mustStartConversation(t, server.Config.Handler, alice.Token, "bob")

	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/events?token="+bob.Token, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("connect event stream: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	created := mustSendJSONMessage(t, server.Config.Handler, alice.Token, conversationID, `{"content":"hello sse"}`)

	streamEvent, err := readSSEEventWithin(resp.Body, 2*time.Second)
	if err != nil {
		t.Fatalf("read sse event: %v", err)
	}
	if streamEvent.Type != realtime.EventTypeMessageNew {
		t.Fatalf("expected event type %q, got %q", realtime.EventTypeMessageNew, streamEvent.Type)
	}
	data, ok := streamEvent.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected event data map, got %T", streamEvent.Data)
	}
	if got, _ := data["id"].(string); got != created.ID {
		t.Fatalf("expected event data id %q, got %v", created.ID, data["id"])
	}
}

func TestEventStreamReceivesNewConversationEventsAfterBootstrap(t *testing.T) {
	server, _ := newEventStreamTestServer(t)
	alice := registerAndLoginUser(t, server.Config.Handler, "alice", "1234")
	bob := registerAndLoginUser(t, server.Config.Handler, "bob", "1234")

	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/events?token="+bob.Token, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("connect event stream: %v", err)
	}
	defer resp.Body.Close()

	conversationID := mustStartConversation(t, server.Config.Handler, alice.Token, "bob")
	created := mustSendJSONMessage(t, server.Config.Handler, alice.Token, conversationID, `{"content":"hello new conversation sse"}`)

	streamEvent, err := readSSEEventWithin(resp.Body, 2*time.Second)
	if err != nil {
		t.Fatalf("read sse event: %v", err)
	}
	if streamEvent.Type != realtime.EventTypeMessageNew {
		t.Fatalf("expected event type %q, got %q", realtime.EventTypeMessageNew, streamEvent.Type)
	}
	data, ok := streamEvent.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected event data map, got %T", streamEvent.Data)
	}
	if got, _ := data["id"].(string); got != created.ID {
		t.Fatalf("expected event data id %q, got %v", created.ID, data["id"])
	}
	if got, _ := data["conversation_id"].(string); got != conversationID {
		t.Fatalf("expected event conversation_id %q, got %v", conversationID, data["conversation_id"])
	}
}

func TestEventStreamBroadcastsWatcherPresenceTransitions(t *testing.T) {
	server, _ := newEventStreamTestServer(t)
	alice := registerAndLoginUser(t, server.Config.Handler, "alice", "1234")
	bob := registerAndLoginUser(t, server.Config.Handler, "bob", "1234")
	conversationID := mustStartConversation(t, server.Config.Handler, alice.Token, "bob")

	aliceResp := mustOpenEventStream(t, server.URL, alice.Token, "")
	defer aliceResp.Body.Close()

	watcherResp := mustOpenEventStream(t, server.URL, bob.Token, "watcher")

	onlineEvent, err := readSSEEventWithin(aliceResp.Body, 2*time.Second)
	if err != nil {
		t.Fatalf("read online presence event: %v", err)
	}
	if onlineEvent.Type != realtime.EventTypePresenceUpdated {
		t.Fatalf("expected event type %q, got %q", realtime.EventTypePresenceUpdated, onlineEvent.Type)
	}
	onlineData, ok := onlineEvent.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected online event data map, got %T", onlineEvent.Data)
	}
	if got, _ := onlineData["conversation_id"].(string); got != conversationID {
		t.Fatalf("conversation id mismatch: got %v want %q", onlineData["conversation_id"], conversationID)
	}
	if got, _ := onlineData["user_id"].(string); got != bob.User.ID {
		t.Fatalf("user id mismatch: got %v want %q", onlineData["user_id"], bob.User.ID)
	}
	if got, _ := onlineData["client_kind"].(string); got != realtime.ClientKindWatcher {
		t.Fatalf("client kind mismatch: got %v want %q", onlineData["client_kind"], realtime.ClientKindWatcher)
	}
	if got, _ := onlineData["online"].(bool); !got {
		t.Fatalf("expected online=true, got %v", onlineData["online"])
	}

	_ = watcherResp.Body.Close()

	offlineEvent, err := readSSEEventWithin(aliceResp.Body, 2*time.Second)
	if err != nil {
		t.Fatalf("read offline presence event: %v", err)
	}
	if offlineEvent.Type != realtime.EventTypePresenceUpdated {
		t.Fatalf("expected event type %q, got %q", realtime.EventTypePresenceUpdated, offlineEvent.Type)
	}
	offlineData, ok := offlineEvent.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected offline event data map, got %T", offlineEvent.Data)
	}
	if got, _ := offlineData["online"].(bool); got {
		t.Fatalf("expected online=false, got %v", offlineData["online"])
	}
}

func newEventStreamTestServer(t *testing.T) (*httptest.Server, *realtime.Hub) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "events_api.sqlite")
	sqliteStore, err := store.NewSQLiteStore(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	t.Cleanup(func() {
		_ = sqliteStore.Close()
	})

	hub := realtime.NewHub()
	router := NewRouter(Dependencies{
		Store:     sqliteStore,
		Hub:       hub,
		UploadDir: filepath.Join(t.TempDir(), "uploads"),
	})
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return server, hub
}

func mustOpenEventStream(t *testing.T, serverURL, token, clientKind string) *http.Response {
	t.Helper()

	requestURL := serverURL + "/api/events?token=" + token
	if strings.TrimSpace(clientKind) != "" {
		requestURL += "&client_kind=" + clientKind
	}
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		t.Fatalf("new event stream request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("connect event stream: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected event stream status %d, got %d", http.StatusOK, resp.StatusCode)
	}
	return resp
}

func readSSEEventWithin(bodyReader io.Reader, timeout time.Duration) (realtime.Event, error) {
	type result struct {
		event realtime.Event
		err   error
	}

	results := make(chan result, 1)
	go func() {
		event, err := readSSEEvent(bodyReader)
		results <- result{event: event, err: err}
	}()

	select {
	case outcome := <-results:
		return outcome.event, outcome.err
	case <-time.After(timeout):
		return realtime.Event{}, context.DeadlineExceeded
	}
}

func readSSEEvent(bodyReader io.Reader) (realtime.Event, error) {
	reader := bufio.NewReader(bodyReader)
	var eventType string
	var payloadLine string

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return realtime.Event{}, err
		}

		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "" {
			if eventType == "" || payloadLine == "" {
				continue
			}

			var envelope struct {
				Type string `json:"type"`
				Data any    `json:"data"`
			}
			if err := json.Unmarshal([]byte(payloadLine), &envelope); err != nil {
				return realtime.Event{}, err
			}
			return realtime.Event{Type: envelope.Type, Data: envelope.Data}, nil
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
