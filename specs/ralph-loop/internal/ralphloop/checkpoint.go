package ralphloop

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const interruptedCheckpointCommitMessage = "checkpoint: preserve interrupted ralph loop state"

type checkpointResult struct {
	Dirty     bool
	Committed bool
	Head      string
}

func checkpointDirtyWorktree(worktreePath string) (checkpointResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	status, err := runCommand(ctx, worktreePath, "git", "-C", worktreePath, "status", "--porcelain")
	if err != nil {
		return checkpointResult{}, fmt.Errorf("git status failed during checkpoint: %s", commandFailureMessage(status, err, "git status"))
	}
	if strings.TrimSpace(status.Stdout) == "" {
		return checkpointResult{}, nil
	}

	addResult, err := runCommand(ctx, worktreePath, "git", "-C", worktreePath, "add", "-A")
	if err != nil {
		return checkpointResult{}, fmt.Errorf("git add failed during checkpoint: %s", commandFailureMessage(addResult, err, "git add"))
	}

	commitResult, err := runCommand(ctx, worktreePath, "git", "-C", worktreePath, "commit", "-m", interruptedCheckpointCommitMessage)
	if err != nil {
		message := strings.ToLower(strings.TrimSpace(commitResult.Stdout + "\n" + commitResult.Stderr))
		if strings.Contains(message, "nothing to commit") {
			return checkpointResult{Dirty: true}, nil
		}
		return checkpointResult{}, fmt.Errorf("git commit failed during checkpoint: %s", commandFailureMessage(commitResult, err, "git commit"))
	}

	return checkpointResult{
		Dirty:     true,
		Committed: true,
		Head:      currentHead(worktreePath),
	}, nil
}
