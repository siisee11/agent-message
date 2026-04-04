package cmd

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func TestRootHelpRendersUsage(t *testing.T) {
	command := NewRootCommand()
	command.SetArgs([]string{"--help"})

	output := captureStdout(t, func() {
		if err := command.Execute(); err != nil {
			t.Fatalf("execute help: %v", err)
		}
	})

	if !strings.Contains(output, "Usage:") {
		t.Fatalf("expected usage in help output, got %q", output)
	}
	if !strings.Contains(output, "send") {
		t.Fatalf("expected send command in help output, got %q", output)
	}
	if !strings.Contains(output, "--json") {
		t.Fatalf("expected --json flag in help output, got %q", output)
	}
}

func TestRootHelpSupportsJSON(t *testing.T) {
	command := NewRootCommand()
	command.SetArgs([]string{"help", "--json", "send"})

	output := captureStdout(t, func() {
		if err := command.Execute(); err != nil {
			t.Fatalf("execute help json: %v", err)
		}
	})

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode help json: %v", err)
	}
	if got, want := payload["command"], "send"; got != want {
		t.Fatalf("command mismatch: got %v want %q", got, want)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	defer reader.Close()

	os.Stdout = writer
	defer func() {
		os.Stdout = original
	}()

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	return string(data)
}
