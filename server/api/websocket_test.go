package api

import (
	"bufio"
	"context"
	"encoding/json"
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
