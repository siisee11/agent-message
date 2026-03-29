package ws

import (
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"strings"
)

const (
	webSocketGUID      = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	webSocketVersion13 = "13"
)

var (
	ErrWebSocketUpgradeRequired   = errors.New("websocket upgrade required")
	ErrWebSocketVersionRequired   = errors.New("unsupported websocket version")
	ErrWebSocketKeyInvalid        = errors.New("invalid websocket key")
	ErrWebSocketHijackUnsupported = errors.New("http hijacking is not supported")
)

type Conn struct {
	net.Conn
}

func Upgrade(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	if !isWebSocketUpgradeRequest(r) {
		return nil, ErrWebSocketUpgradeRequired
	}

	if strings.TrimSpace(r.Header.Get("Sec-WebSocket-Version")) != webSocketVersion13 {
		return nil, ErrWebSocketVersionRequired
	}

	rawKey := strings.TrimSpace(r.Header.Get("Sec-WebSocket-Key"))
	if !isValidWebSocketKey(rawKey) {
		return nil, ErrWebSocketKeyInvalid
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, ErrWebSocketHijackUnsupported
	}

	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return nil, err
	}

	response := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + computeAcceptKey(rawKey) + "\r\n" +
		"\r\n"
	if _, err := rw.WriteString(response); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := rw.Flush(); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return &Conn{Conn: conn}, nil
}

func isWebSocketUpgradeRequest(r *http.Request) bool {
	if !headerContainsToken(r.Header, "Connection", "upgrade") {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("Upgrade")), "websocket")
}

func headerContainsToken(header http.Header, key, token string) bool {
	values := header.Values(key)
	for _, value := range values {
		parts := strings.Split(value, ",")
		for _, part := range parts {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}

func isValidWebSocketKey(rawKey string) bool {
	decoded, err := base64.StdEncoding.DecodeString(rawKey)
	if err != nil {
		return false
	}
	return len(decoded) == 16
}

func computeAcceptKey(rawKey string) string {
	sum := sha1.Sum([]byte(rawKey + webSocketGUID))
	return base64.StdEncoding.EncodeToString(sum[:])
}
