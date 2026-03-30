package cmd

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"agent-messenger/cli/internal/api"

	"github.com/spf13/cobra"
)

const (
	wsOpcodeText  = 0x1
	wsOpcodeClose = 0x8
	wsOpcodePing  = 0x9
	wsOpcodePong  = 0xA

	wsGUID              = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	wsVersion13         = "13"
	wsDialTimeout       = 10 * time.Second
	wsMaxFramePayload   = 1 << 20
	wsClientMaskKeySize = 4
)

type wsEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type watchStream interface {
	ReadEvent() (wsEvent, error)
	Close() error
}

var connectWatchStream = connectWebSocketWatchStream

func newWatchCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "watch <username>",
		Short: "Watch incoming messages in real time",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runWatch(rt, args[0])
		},
	}
}

func runWatch(rt *Runtime, username string) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}
	if err := ensureLoggedIn(rt); err != nil {
		return err
	}

	conversationID, err := resolveConversationIDByUsername(context.Background(), rt, username)
	if err != nil {
		return err
	}

	wsURL, err := rt.Client.WebSocketURL()
	if err != nil {
		return err
	}

	stream, err := connectWatchStream(wsURL)
	if err != nil {
		return err
	}
	defer stream.Close()

	for {
		event, err := stream.ReadEvent()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		if strings.TrimSpace(event.Type) != "message.new" {
			continue
		}

		var message api.Message
		if err := json.Unmarshal(event.Data, &message); err != nil {
			_, _ = fmt.Fprintf(rt.Stderr, "warning: failed to decode websocket message.new payload: %v\n", err)
			continue
		}
		if strings.TrimSpace(message.ConversationID) != conversationID {
			continue
		}

		_, _ = fmt.Fprintf(rt.Stdout, "%s %s: %s\n", message.ID, strings.TrimSpace(message.SenderID), watchMessageText(message))
	}
}

func watchMessageText(message api.Message) string {
	if message.Deleted {
		return "deleted message"
	}
	if message.Content != nil {
		content := strings.TrimSpace(*message.Content)
		if content != "" {
			return content
		}
	}
	if message.AttachmentURL != nil {
		attachmentURL := strings.TrimSpace(*message.AttachmentURL)
		if attachmentURL != "" {
			return attachmentURL
		}
	}
	return ""
}

type websocketStream struct {
	conn   net.Conn
	reader *bufio.Reader
}

func connectWebSocketWatchStream(rawURL string) (watchStream, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("invalid websocket URL: %w", err)
	}
	if strings.TrimSpace(u.Host) == "" {
		return nil, errors.New("websocket URL host is required")
	}

	address, tlsConfig, err := dialAddressAndTLSConfig(u)
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{Timeout: wsDialTimeout}
	var conn net.Conn
	if tlsConfig != nil {
		conn, err = tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
	} else {
		conn, err = dialer.Dial("tcp", address)
	}
	if err != nil {
		return nil, fmt.Errorf("dial websocket server: %w", err)
	}

	reader := bufio.NewReader(conn)
	if err := performClientHandshake(conn, reader, u); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return &websocketStream{
		conn:   conn,
		reader: reader,
	}, nil
}

func dialAddressAndTLSConfig(u *url.URL) (address string, tlsConfig *tls.Config, err error) {
	host := strings.TrimSpace(u.Host)
	switch strings.ToLower(strings.TrimSpace(u.Scheme)) {
	case "ws":
		if strings.Contains(host, ":") {
			return host, nil, nil
		}
		return net.JoinHostPort(host, "80"), nil, nil
	case "wss":
		serverName := host
		if strings.Contains(serverName, ":") {
			var splitErr error
			serverName, _, splitErr = net.SplitHostPort(serverName)
			if splitErr != nil {
				return "", nil, fmt.Errorf("invalid websocket host: %w", splitErr)
			}
		}
		if !strings.Contains(host, ":") {
			host = net.JoinHostPort(host, "443")
		}
		return host, &tls.Config{MinVersion: tls.VersionTLS12, ServerName: serverName}, nil
	default:
		return "", nil, errors.New("websocket URL must use ws:// or wss://")
	}
}

func performClientHandshake(conn net.Conn, reader *bufio.Reader, u *url.URL) error {
	requestPath := u.RequestURI()
	if strings.TrimSpace(requestPath) == "" {
		requestPath = "/"
	}

	key, err := generateWebSocketKey()
	if err != nil {
		return fmt.Errorf("generate websocket key: %w", err)
	}

	if _, err := fmt.Fprintf(conn,
		"GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: %s\r\n\r\n",
		requestPath,
		u.Host,
		key,
		wsVersion13,
	); err != nil {
		return fmt.Errorf("write websocket handshake: %w", err)
	}

	req := &http.Request{Method: http.MethodGet}
	resp, err := http.ReadResponse(reader, req)
	if err != nil {
		return fmt.Errorf("read websocket handshake response: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusSwitchingProtocols {
		return fmt.Errorf("websocket handshake failed: %s", resp.Status)
	}
	if !strings.EqualFold(strings.TrimSpace(resp.Header.Get("Upgrade")), "websocket") {
		return errors.New("websocket handshake failed: missing Upgrade: websocket header")
	}
	if !headerContainsToken(resp.Header, "Connection", "upgrade") {
		return errors.New("websocket handshake failed: missing Connection: Upgrade header")
	}

	expectedAccept := computeAcceptKey(key)
	gotAccept := strings.TrimSpace(resp.Header.Get("Sec-WebSocket-Accept"))
	if gotAccept != expectedAccept {
		return errors.New("websocket handshake failed: invalid Sec-WebSocket-Accept value")
	}

	return nil
}

func generateWebSocketKey() (string, error) {
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

func computeAcceptKey(clientKey string) string {
	hash := sha1.Sum([]byte(clientKey + wsGUID))
	return base64.StdEncoding.EncodeToString(hash[:])
}

func headerContainsToken(header http.Header, key string, token string) bool {
	values := header.Values(key)
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}

func (s *websocketStream) ReadEvent() (wsEvent, error) {
	for {
		opcode, payload, err := readWebSocketFrame(s.reader)
		if err != nil {
			return wsEvent{}, err
		}

		switch opcode {
		case wsOpcodeText:
			var event wsEvent
			if err := json.Unmarshal(payload, &event); err != nil {
				return wsEvent{}, err
			}
			return event, nil
		case wsOpcodeClose:
			return wsEvent{}, io.EOF
		case wsOpcodePing:
			if err := writeMaskedWebSocketFrame(s.conn, wsOpcodePong, payload); err != nil {
				return wsEvent{}, err
			}
		case wsOpcodePong:
			continue
		default:
			continue
		}
	}
}

func (s *websocketStream) Close() error {
	return s.conn.Close()
}

func readWebSocketFrame(reader *bufio.Reader) (opcode byte, payload []byte, err error) {
	var header [2]byte
	if _, err := io.ReadFull(reader, header[:]); err != nil {
		return 0, nil, err
	}

	fin := (header[0] & 0x80) != 0
	if !fin {
		return 0, nil, errors.New("unsupported fragmented websocket frame")
	}

	opcode = header[0] & 0x0F
	masked := (header[1] & 0x80) != 0

	payloadLength, err := readWebSocketPayloadLength(reader, header[1]&0x7F)
	if err != nil {
		return 0, nil, err
	}
	if payloadLength > wsMaxFramePayload {
		return 0, nil, errors.New("websocket frame payload too large")
	}

	var maskingKey [wsClientMaskKeySize]byte
	if masked {
		if _, err := io.ReadFull(reader, maskingKey[:]); err != nil {
			return 0, nil, err
		}
	}

	payload = make([]byte, payloadLength)
	if payloadLength > 0 {
		if _, err := io.ReadFull(reader, payload); err != nil {
			return 0, nil, err
		}
	}

	if masked {
		for i := range payload {
			payload[i] ^= maskingKey[i%wsClientMaskKeySize]
		}
	}

	return opcode, payload, nil
}

func readWebSocketPayloadLength(reader *bufio.Reader, marker byte) (int, error) {
	switch marker {
	case 126:
		var ext [2]byte
		if _, err := io.ReadFull(reader, ext[:]); err != nil {
			return 0, err
		}
		return int(binary.BigEndian.Uint16(ext[:])), nil
	case 127:
		var ext [8]byte
		if _, err := io.ReadFull(reader, ext[:]); err != nil {
			return 0, err
		}
		value := binary.BigEndian.Uint64(ext[:])
		if value > math.MaxInt {
			return 0, errors.New("websocket frame payload too large")
		}
		return int(value), nil
	default:
		return int(marker), nil
	}
}

func writeMaskedWebSocketFrame(conn net.Conn, opcode byte, payload []byte) error {
	length := len(payload)
	header := make([]byte, 0, 14)
	header = append(header, 0x80|(opcode&0x0F))

	switch {
	case length <= 125:
		header = append(header, 0x80|byte(length))
	case length <= math.MaxUint16:
		header = append(header, 0x80|126, 0, 0)
		binary.BigEndian.PutUint16(header[len(header)-2:], uint16(length))
	default:
		header = append(header, 0x80|127, 0, 0, 0, 0, 0, 0, 0, 0)
		binary.BigEndian.PutUint64(header[len(header)-8:], uint64(length))
	}

	maskKey := make([]byte, wsClientMaskKeySize)
	if _, err := rand.Read(maskKey); err != nil {
		return err
	}
	header = append(header, maskKey...)

	maskedPayload := make([]byte, length)
	copy(maskedPayload, payload)
	for i := range maskedPayload {
		maskedPayload[i] ^= maskKey[i%wsClientMaskKeySize]
	}

	if _, err := conn.Write(header); err != nil {
		return err
	}
	if length == 0 {
		return nil
	}
	_, err := conn.Write(maskedPayload)
	return err
}
