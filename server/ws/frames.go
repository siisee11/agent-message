package ws

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"math"
)

const (
	opcodeText  = 0x1
	opcodeClose = 0x8
	opcodePing  = 0x9
	opcodePong  = 0xA

	maxIncomingFramePayloadBytes = 1 << 20
)

var (
	ErrWebSocketFrameMalformed       = errors.New("malformed websocket frame")
	ErrWebSocketClientFrameUnmasked  = errors.New("client websocket frame must be masked")
	ErrWebSocketFrameTooLarge        = errors.New("websocket frame payload too large")
	ErrWebSocketUnsupportedFrameType = errors.New("unsupported websocket frame opcode")
)

// ReadJSON reads one or more websocket frames until it receives a text frame,
// then unmarshals that frame payload as JSON.
func ReadJSON(conn *Conn, dst any) error {
	for {
		opcode, payload, err := readFrame(conn, true)
		if err != nil {
			return err
		}

		switch opcode {
		case opcodeText:
			if err := json.Unmarshal(payload, dst); err != nil {
				return err
			}
			return nil
		case opcodeClose:
			_ = writeFrame(conn, opcodeClose, nil)
			return io.EOF
		case opcodePing:
			if err := writeFrame(conn, opcodePong, payload); err != nil {
				return err
			}
		case opcodePong:
			continue
		default:
			return ErrWebSocketUnsupportedFrameType
		}
	}
}

// WriteJSON marshals the value and writes it in a websocket text frame.
func WriteJSON(conn *Conn, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return writeFrame(conn, opcodeText, payload)
}

func readFrame(conn *Conn, requireMasked bool) (byte, []byte, error) {
	var header [2]byte
	if _, err := io.ReadFull(conn, header[:]); err != nil {
		return 0, nil, err
	}

	fin := (header[0] & 0x80) != 0
	if !fin {
		return 0, nil, ErrWebSocketFrameMalformed
	}

	opcode := header[0] & 0x0F
	masked := (header[1] & 0x80) != 0
	if requireMasked && !masked {
		return 0, nil, ErrWebSocketClientFrameUnmasked
	}

	payloadLength, err := readPayloadLength(conn, header[1]&0x7F)
	if err != nil {
		return 0, nil, err
	}
	if payloadLength > maxIncomingFramePayloadBytes {
		return 0, nil, ErrWebSocketFrameTooLarge
	}

	var maskingKey [4]byte
	if masked {
		if _, err := io.ReadFull(conn, maskingKey[:]); err != nil {
			return 0, nil, err
		}
	}

	payload := make([]byte, payloadLength)
	if payloadLength > 0 {
		if _, err := io.ReadFull(conn, payload); err != nil {
			return 0, nil, err
		}
	}
	if masked {
		for i := range payload {
			payload[i] ^= maskingKey[i%4]
		}
	}
	return opcode, payload, nil
}

func readPayloadLength(conn *Conn, marker byte) (int, error) {
	switch marker {
	case 126:
		var ext [2]byte
		if _, err := io.ReadFull(conn, ext[:]); err != nil {
			return 0, err
		}
		return int(binary.BigEndian.Uint16(ext[:])), nil
	case 127:
		var ext [8]byte
		if _, err := io.ReadFull(conn, ext[:]); err != nil {
			return 0, err
		}
		value := binary.BigEndian.Uint64(ext[:])
		if value > math.MaxInt {
			return 0, ErrWebSocketFrameTooLarge
		}
		return int(value), nil
	default:
		return int(marker), nil
	}
}

func writeFrame(conn *Conn, opcode byte, payload []byte) error {
	length := len(payload)
	header := make([]byte, 0, 10)
	header = append(header, 0x80|(opcode&0x0F))

	switch {
	case length <= 125:
		header = append(header, byte(length))
	case length <= math.MaxUint16:
		header = append(header, 126, 0, 0)
		binary.BigEndian.PutUint16(header[len(header)-2:], uint16(length))
	default:
		header = append(header, 127, 0, 0, 0, 0, 0, 0, 0, 0)
		binary.BigEndian.PutUint64(header[len(header)-8:], uint64(length))
	}

	if _, err := conn.Write(header); err != nil {
		return err
	}
	if length == 0 {
		return nil
	}
	_, err := conn.Write(payload)
	return err
}
