package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHowtoListPrintsMarkdownFilesSorted(t *testing.T) {
	dir := seedHowtoDir(t)
	t.Setenv(howtoDirEnv, dir)

	stdout := &bytes.Buffer{}
	rt := &Runtime{Stdout: stdout}

	if err := runHowtoList(rt); err != nil {
		t.Fatalf("runHowtoList: %v", err)
	}

	if got, want := stdout.String(), "alpha.md\nzeta.md\n"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}

func TestRunHowtoListPrintsJSON(t *testing.T) {
	dir := seedHowtoDir(t)
	t.Setenv(howtoDirEnv, dir)

	stdout := &bytes.Buffer{}
	rt := &Runtime{Stdout: stdout, JSONOutput: true}

	if err := runHowtoList(rt); err != nil {
		t.Fatalf("runHowtoList: %v", err)
	}

	var payload struct {
		Files []string `json:"files"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if got, want := strings.Join(payload.Files, ","), "alpha.md,zeta.md"; got != want {
		t.Fatalf("files mismatch: got %q want %q", got, want)
	}
}

func TestRunHowtoReadPrintsContent(t *testing.T) {
	dir := seedHowtoDir(t)
	t.Setenv(howtoDirEnv, dir)

	stdout := &bytes.Buffer{}
	rt := &Runtime{Stdout: stdout}

	if err := runHowtoRead(rt, "alpha"); err != nil {
		t.Fatalf("runHowtoRead: %v", err)
	}

	if got, want := stdout.String(), "# Alpha\n\nHello.\n"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}

func TestRunHowtoReadRejectsPaths(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	rt := &Runtime{Stdout: stdout}

	err := runHowtoRead(rt, "../secret.md")
	if err == nil {
		t.Fatalf("expected path rejection")
	}
	if got := err.Error(); !strings.Contains(got, "not a path") {
		t.Fatalf("unexpected error: %q", got)
	}
}

func seedHowtoDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "zeta.md"), []byte("# Zeta\n"), 0o600); err != nil {
		t.Fatalf("write zeta: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "alpha.md"), []byte("# Alpha\n\nHello.\n"), 0o600); err != nil {
		t.Fatalf("write alpha: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ignored.txt"), []byte("ignore\n"), 0o600); err != nil {
		t.Fatalf("write ignored: %v", err)
	}
	return dir
}
