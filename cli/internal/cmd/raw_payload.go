package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

type rawPayloadOptions struct {
	Payload      string
	PayloadFile  string
	PayloadStdin bool
}

func resolveRawPayload(stdin io.Reader, options rawPayloadOptions) ([]byte, error) {
	modeCount := 0
	var raw []byte

	if strings.TrimSpace(options.Payload) != "" {
		modeCount++
		raw = []byte(strings.TrimSpace(options.Payload))
	}
	if strings.TrimSpace(options.PayloadFile) != "" {
		modeCount++
		fileBytes, err := os.ReadFile(strings.TrimSpace(options.PayloadFile))
		if err != nil {
			return nil, fmt.Errorf("read payload file: %w", err)
		}
		raw = fileBytes
	}
	if options.PayloadStdin {
		modeCount++
		if stdin == nil {
			return nil, errors.New("stdin reader is not initialized")
		}
		stdinBytes, err := io.ReadAll(stdin)
		if err != nil {
			return nil, fmt.Errorf("read payload stdin: %w", err)
		}
		raw = stdinBytes
	}

	if modeCount == 0 {
		return nil, nil
	}
	if modeCount > 1 {
		return nil, errors.New("choose only one raw payload source among --payload, --payload-file, and --payload-stdin")
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, errors.New("raw payload must not be empty")
	}
	return raw, nil
}

func decodeStrictJSONObject[T any](raw []byte, label string) (T, error) {
	var out T
	if len(bytes.TrimSpace(raw)) == 0 {
		return out, fmt.Errorf("%s must not be empty", label)
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&out); err != nil {
		return out, fmt.Errorf("decode %s: %w", label, err)
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return out, fmt.Errorf("%s must contain exactly one JSON object", label)
		}
		return out, fmt.Errorf("decode %s trailing data: %w", label, err)
	}
	return out, nil
}
