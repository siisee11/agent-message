package api

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"agent-messenger/server/store"
	"agent-messenger/server/ws"
)

const sampleWebSocketKey = "dGhlIHNhbXBsZSBub25jZQ=="

func TestWebSocketEndpointUpgradeAndLifecycle(t *testing.T) {
	server, hub := newWebSocketTestServer(t)
	token := registerAndLoginUser(t, server.Config.Handler, "alice", "1234").Token

	conn, resp := performWebSocketRequest(t, server.URL, http.MethodGet, token, true)
	defer conn.Close()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("expected status %d, got %d", http.StatusSwitchingProtocols, resp.StatusCode)
	}
	if got := resp.Header.Get("Upgrade"); got != "websocket" {
		t.Fatalf("expected upgrade header websocket, got %q", got)
	}
	if got := resp.Header.Get("Connection"); got != "Upgrade" {
		t.Fatalf("expected connection header Upgrade, got %q", got)
	}
	if got := resp.Header.Get("Sec-WebSocket-Accept"); got != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
		t.Fatalf("unexpected sec-websocket-accept: %q", got)
	}

	waitForHubConnections(t, hub, 1)

	if err := conn.Close(); err != nil {
		t.Fatalf("close websocket conn: %v", err)
	}
	waitForHubConnections(t, hub, 0)
}

func TestWebSocketEndpointAuthValidation(t *testing.T) {
	server, _ := newWebSocketTestServer(t)

	t.Run("missing token", func(t *testing.T) {
		conn, resp := performWebSocketRequest(t, server.URL, http.MethodGet, "", true)
		defer conn.Close()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, resp.StatusCode)
		}
		assertErrorBody(t, resp.Body, "missing or invalid bearer token")
	})

	t.Run("invalid token", func(t *testing.T) {
		conn, resp := performWebSocketRequest(t, server.URL, http.MethodGet, "not-a-session", true)
		defer conn.Close()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, resp.StatusCode)
		}
		assertErrorBody(t, resp.Body, "invalid session token")
	})
}

func TestWebSocketEndpointUpgradeValidation(t *testing.T) {
	server, _ := newWebSocketTestServer(t)
	token := registerAndLoginUser(t, server.Config.Handler, "alice", "1234").Token

	conn, resp := performWebSocketRequest(t, server.URL, http.MethodGet, token, false)
	defer conn.Close()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
	assertErrorBody(t, resp.Body, ws.ErrWebSocketUpgradeRequired.Error())
}

func TestWebSocketRuntimePumpsHubEventsToClient(t *testing.T) {
	server, hub := newWebSocketTestServer(t)
	alice := registerAndLoginUser(t, server.Config.Handler, "alice", "1234")
	_ = registerAndLoginUser(t, server.Config.Handler, "bob", "1234")
	conversationID := mustStartConversation(t, server.Config.Handler, alice.Token, "bob")

	conn, resp := performWebSocketRequest(t, server.URL, http.MethodGet, alice.Token, true)
	defer conn.Close()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("expected status %d, got %d", http.StatusSwitchingProtocols, resp.StatusCode)
	}

	waitForHubConnections(t, hub, 1)

	_, err := hub.BroadcastToConversation(conversationID, ws.Event{
		Type: ws.EventTypeMessageNew,
		Data: map[string]any{
			"id":              "msg-bootstrap-subscription",
			"conversation_id": conversationID,
		},
	})
	if err != nil {
		t.Fatalf("broadcast event: %v", err)
	}

	event, err := readServerEventWithin(conn, 2*time.Second)
	if err != nil {
		t.Fatalf("read server event: %v", err)
	}
	if event.Type != ws.EventTypeMessageNew {
		t.Fatalf("expected event type %q, got %q", ws.EventTypeMessageNew, event.Type)
	}
	data, ok := event.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected event data map, got %T", event.Data)
	}
	if got, _ := data["id"].(string); got != "msg-bootstrap-subscription" {
		t.Fatalf("expected event data id msg-bootstrap-subscription, got %v", data["id"])
	}
}

func TestWebSocketRuntimeReadEventSubscribesConversation(t *testing.T) {
	server, hub := newWebSocketTestServer(t)
	alice := registerAndLoginUser(t, server.Config.Handler, "alice", "1234")
	_ = registerAndLoginUser(t, server.Config.Handler, "bob", "1234")

	conn, resp := performWebSocketRequest(t, server.URL, http.MethodGet, alice.Token, true)
	defer conn.Close()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("expected status %d, got %d", http.StatusSwitchingProtocols, resp.StatusCode)
	}
	waitForHubConnections(t, hub, 1)

	conversationID := mustStartConversation(t, server.Config.Handler, alice.Token, "bob")

	_, err := hub.BroadcastToConversation(conversationID, ws.Event{
		Type: ws.EventTypeMessageNew,
		Data: map[string]any{"id": "before-read"},
	})
	if err != nil {
		t.Fatalf("broadcast before read event: %v", err)
	}
	assertNoServerEventWithin(t, conn, 150*time.Millisecond)

	writeClientJSONFrame(t, conn, map[string]any{
		"type": "read",
		"data": map[string]any{
			"conversation_id": conversationID,
		},
	})

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_, err := hub.BroadcastToConversation(conversationID, ws.Event{
			Type: ws.EventTypeMessageNew,
			Data: map[string]any{"id": "after-read"},
		})
		if err != nil {
			t.Fatalf("broadcast after read event: %v", err)
		}

		event, readErr := readServerEventWithin(conn, 120*time.Millisecond)
		if readErr == nil {
			if event.Type != ws.EventTypeMessageNew {
				t.Fatalf("expected event type %q, got %q", ws.EventTypeMessageNew, event.Type)
			}
			data, ok := event.Data.(map[string]any)
			if !ok {
				t.Fatalf("expected event data map, got %T", event.Data)
			}
			if got, _ := data["id"].(string); got != "after-read" {
				t.Fatalf("expected event data id after-read, got %v", data["id"])
			}
			return
		}
		if !isTimeoutError(readErr) {
			t.Fatalf("read event after read subscription: %v", readErr)
		}
	}

	t.Fatalf("expected websocket event after read subscription update")
}

func newWebSocketTestServer(t *testing.T) (*httptest.Server, *ws.Hub) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "ws_api.sqlite")
	sqliteStore, err := store.NewSQLiteStore(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	t.Cleanup(func() {
		_ = sqliteStore.Close()
	})

	hub := ws.NewHub()
	router := NewRouter(Dependencies{
		Store:     sqliteStore,
		Hub:       hub,
		UploadDir: filepath.Join(t.TempDir(), "uploads"),
	})
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return server, hub
}

func performWebSocketRequest(t *testing.T, serverURL, method, token string, includeUpgradeHeaders bool) (net.Conn, *http.Response) {
	t.Helper()

	parsed, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}

	path := "/ws"
	if token != "" {
		path += "?token=" + url.QueryEscape(token)
	}

	req, err := http.NewRequest(method, "http://"+parsed.Host+path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Host = parsed.Host
	if includeUpgradeHeaders {
		req.Header.Set("Connection", "Upgrade")
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Sec-WebSocket-Version", "13")
		req.Header.Set("Sec-WebSocket-Key", sampleWebSocketKey)
	}

	conn, err := net.Dial("tcp", parsed.Host)
	if err != nil {
		t.Fatalf("dial server: %v", err)
	}

	if err := req.Write(conn); err != nil {
		_ = conn.Close()
		t.Fatalf("write request: %v", err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		_ = conn.Close()
		t.Fatalf("read response: %v", err)
	}

	return conn, resp
}

func assertErrorBody(t *testing.T, body io.Reader, expected string) {
	t.Helper()

	var payload map[string]string
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if payload["error"] != expected {
		t.Fatalf("expected error %q, got %q", expected, payload["error"])
	}
}

func waitForHubConnections(t *testing.T, hub *ws.Hub, expected int) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if hub.ConnectionCount() == expected {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("expected hub connections=%d, got %d", expected, hub.ConnectionCount())
}

func writeClientJSONFrame(t *testing.T, conn net.Conn, value any) {
	t.Helper()

	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal frame payload: %v", err)
	}
	writeMaskedClientTextFrame(t, conn, payload)
}

func writeMaskedClientTextFrame(t *testing.T, conn net.Conn, payload []byte) {
	t.Helper()

	header := make([]byte, 0, 14)
	header = append(header, 0x81)

	payloadLen := len(payload)
	switch {
	case payloadLen <= 125:
		header = append(header, 0x80|byte(payloadLen))
	case payloadLen <= 0xFFFF:
		header = append(header, 0x80|126, 0, 0)
		binary.BigEndian.PutUint16(header[len(header)-2:], uint16(payloadLen))
	default:
		header = append(header, 0x80|127, 0, 0, 0, 0, 0, 0, 0, 0)
		binary.BigEndian.PutUint64(header[len(header)-8:], uint64(payloadLen))
	}

	maskingKey := [4]byte{0x11, 0x22, 0x33, 0x44}
	frame := make([]byte, 0, len(header)+len(maskingKey)+payloadLen)
	frame = append(frame, header...)
	frame = append(frame, maskingKey[:]...)

	for i := range payload {
		frame = append(frame, payload[i]^maskingKey[i%4])
	}

	if _, err := conn.Write(frame); err != nil {
		t.Fatalf("write masked client frame: %v", err)
	}
}

func readServerEventWithin(conn net.Conn, timeout time.Duration) (ws.Event, error) {
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	defer conn.SetReadDeadline(time.Time{})

	opcode, payload, err := readServerFrame(conn)
	if err != nil {
		return ws.Event{}, err
	}
	if opcode != 0x1 {
		return ws.Event{}, fmt.Errorf("unexpected server frame opcode: %d", opcode)
	}

	var event ws.Event
	if err := json.Unmarshal(payload, &event); err != nil {
		return ws.Event{}, err
	}
	return event, nil
}

func assertNoServerEventWithin(t *testing.T, conn net.Conn, timeout time.Duration) {
	t.Helper()

	if _, err := readServerEventWithin(conn, timeout); err == nil {
		t.Fatalf("expected no event, but received one")
	} else if !isTimeoutError(err) {
		t.Fatalf("expected timeout while asserting no event, got %v", err)
	}
}

func readServerFrame(conn net.Conn) (byte, []byte, error) {
	var header [2]byte
	if _, err := io.ReadFull(conn, header[:]); err != nil {
		return 0, nil, err
	}

	payloadLenMarker := header[1] & 0x7F
	payloadLen := 0
	switch payloadLenMarker {
	case 126:
		var ext [2]byte
		if _, err := io.ReadFull(conn, ext[:]); err != nil {
			return 0, nil, err
		}
		payloadLen = int(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err := io.ReadFull(conn, ext[:]); err != nil {
			return 0, nil, err
		}
		payloadLen = int(binary.BigEndian.Uint64(ext[:]))
	default:
		payloadLen = int(payloadLenMarker)
	}

	if (header[1] & 0x80) != 0 {
		return 0, nil, errors.New("unexpected masked server frame")
	}

	payload := make([]byte, payloadLen)
	if payloadLen > 0 {
		if _, err := io.ReadFull(conn, payload); err != nil {
			return 0, nil, err
		}
	}
	return header[0] & 0x0F, payload, nil
}

func isTimeoutError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
