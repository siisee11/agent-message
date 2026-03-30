package ralphloop

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckpointDirtyWorktreeCommitsChanges(t *testing.T) {
	repoRoot := initGitRepoForLandingTest(t)
	writeFileForLandingTest(t, repoRoot, "README.md", "base\n")
	gitInLandingTest(t, repoRoot, "add", "README.md")
	gitInLandingTest(t, repoRoot, "commit", "-m", "base")

	writeFileForLandingTest(t, repoRoot, "feature.txt", "phase\n")

	result, err := checkpointDirtyWorktree(repoRoot)
	if err != nil {
		t.Fatalf("checkpointDirtyWorktree() error = %v", err)
	}
	if !result.Dirty {
		t.Fatal("expected dirty worktree to be detected")
	}
	if !result.Committed {
		t.Fatal("expected dirty worktree to be checkpoint committed")
	}
	if got := gitInLandingTest(t, repoRoot, "log", "-1", "--pretty=%s"); got != interruptedCheckpointCommitMessage {
		t.Fatalf("checkpoint commit subject = %q, want %q", got, interruptedCheckpointCommitMessage)
	}
	if result.Head == "" {
		t.Fatal("expected checkpoint head to be recorded")
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "feature.txt")); err != nil {
		t.Fatalf("expected feature file to remain after checkpoint: %v", err)
	}
}

func TestCheckpointDirtyWorktreeSkipsCleanWorktree(t *testing.T) {
	repoRoot := initGitRepoForLandingTest(t)
	writeFileForLandingTest(t, repoRoot, "README.md", "base\n")
	gitInLandingTest(t, repoRoot, "add", "README.md")
	gitInLandingTest(t, repoRoot, "commit", "-m", "base")

	result, err := checkpointDirtyWorktree(repoRoot)
	if err != nil {
		t.Fatalf("checkpointDirtyWorktree() error = %v", err)
	}
	if result.Dirty {
		t.Fatal("expected clean worktree to stay clean")
	}
	if result.Committed {
		t.Fatal("did not expect checkpoint commit for a clean worktree")
	}
	if result.Head != "" {
		t.Fatalf("head = %q, want empty", result.Head)
	}
}
