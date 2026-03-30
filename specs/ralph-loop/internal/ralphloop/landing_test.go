package ralphloop

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLandWorkBranchFastForward(t *testing.T) {
	repoRoot := initGitRepoForLandingTest(t)
	writeFileForLandingTest(t, repoRoot, "README.md", "base\n")
	gitInLandingTest(t, repoRoot, "add", "README.md")
	gitInLandingTest(t, repoRoot, "commit", "-m", "base")

	gitInLandingTest(t, repoRoot, "checkout", "-b", "ralph-phase")
	writeFileForLandingTest(t, repoRoot, "feature.txt", "phase\n")
	gitInLandingTest(t, repoRoot, "add", "feature.txt")
	gitInLandingTest(t, repoRoot, "commit", "-m", "phase")
	gitInLandingTest(t, repoRoot, "checkout", "main")

	result, err := landWorkBranch(context.Background(), repoRoot, "main", "ralph-phase")
	if err != nil {
		t.Fatalf("landWorkBranch() error = %v", err)
	}
	if result.Method != "fast-forward" {
		t.Fatalf("method = %q, want fast-forward", result.Method)
	}
	if result.CommitCount != 1 {
		t.Fatalf("commit count = %d, want 1", result.CommitCount)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "feature.txt")); err != nil {
		t.Fatalf("expected landed file to exist: %v", err)
	}
}

func TestLandWorkBranchRejectsDirtyBaseWorktree(t *testing.T) {
	repoRoot := initGitRepoForLandingTest(t)
	writeFileForLandingTest(t, repoRoot, "README.md", "base\n")
	gitInLandingTest(t, repoRoot, "add", "README.md")
	gitInLandingTest(t, repoRoot, "commit", "-m", "base")

	gitInLandingTest(t, repoRoot, "checkout", "-b", "ralph-phase")
	writeFileForLandingTest(t, repoRoot, "feature.txt", "phase\n")
	gitInLandingTest(t, repoRoot, "add", "feature.txt")
	gitInLandingTest(t, repoRoot, "commit", "-m", "phase")
	gitInLandingTest(t, repoRoot, "checkout", "main")

	writeFileForLandingTest(t, repoRoot, "dirty.txt", "dirty\n")

	if _, err := landWorkBranch(context.Background(), repoRoot, "main", "ralph-phase"); err == nil {
		t.Fatal("expected dirty base worktree to be rejected")
	}
}

func initGitRepoForLandingTest(t *testing.T) string {
	t.Helper()
	repoRoot := t.TempDir()
	gitInLandingTest(t, repoRoot, "init", "-b", "main")
	gitInLandingTest(t, repoRoot, "config", "user.name", "Ralph Loop Test")
	gitInLandingTest(t, repoRoot, "config", "user.email", "ralph-loop-test@example.com")
	return repoRoot
}

func writeFileForLandingTest(t *testing.T, repoRoot string, relativePath string, content string) {
	t.Helper()
	fullPath := filepath.Join(repoRoot, relativePath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", fullPath, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", fullPath, err)
	}
}

func gitInLandingTest(t *testing.T, repoRoot string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}
	return strings.TrimSpace(string(output))
}
