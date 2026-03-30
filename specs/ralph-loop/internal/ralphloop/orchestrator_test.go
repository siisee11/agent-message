package ralphloop

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShouldSkipSetupPhaseWithExistingFile(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, "docs", "exec-plans", "active", "phase-6.md")
	if err := os.MkdirAll(filepath.Dir(planPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(planPath, []byte("# existing plan\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if !shouldSkipSetupPhase(planPath) {
		t.Fatal("shouldSkipSetupPhase() = false, want true for existing plan file")
	}
}

func TestShouldSkipSetupPhaseWithoutExistingFile(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, "docs", "exec-plans", "active", "phase-6.md")

	if shouldSkipSetupPhase(planPath) {
		t.Fatal("shouldSkipSetupPhase() = true, want false for missing plan file")
	}
}

func TestShouldSkipSetupPhaseRejectsDirectory(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, "docs", "exec-plans", "active")
	if err := os.MkdirAll(planPath, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	if shouldSkipSetupPhase(planPath) {
		t.Fatal("shouldSkipSetupPhase() = true, want false for directory path")
	}
}
