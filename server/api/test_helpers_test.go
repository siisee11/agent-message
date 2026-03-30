package api

import (
	"encoding/json"
	"io"
	"testing"
)

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
