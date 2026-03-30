package ralphloop

import (
	"context"
	"fmt"
	"strings"
)

type landBaseResult struct {
	Head        string
	Method      string
	CommitCount int
}

func landWorkBranch(ctx context.Context, repoRoot string, baseBranch string, workBranch string) (landBaseResult, error) {
	baseBranch = strings.TrimSpace(baseBranch)
	workBranch = strings.TrimSpace(workBranch)
	if baseBranch == "" {
		return landBaseResult{}, fmt.Errorf("missing base branch for landing")
	}
	if workBranch == "" {
		return landBaseResult{}, fmt.Errorf("missing work branch for landing")
	}

	currentBranch, err := runCommand(ctx, repoRoot, "git", "-C", repoRoot, "branch", "--show-current")
	if err != nil {
		return landBaseResult{}, fmt.Errorf("failed to inspect current branch: %s", commandFailureMessage(currentBranch, err, "git branch --show-current"))
	}
	if strings.TrimSpace(currentBranch.Stdout) != baseBranch {
		return landBaseResult{}, fmt.Errorf("refusing to land onto %s: current branch in %s is %s", baseBranch, repoRoot, strings.TrimSpace(currentBranch.Stdout))
	}

	status, err := runCommand(ctx, repoRoot, "git", "-C", repoRoot, "status", "--porcelain")
	if err != nil {
		return landBaseResult{}, fmt.Errorf("git status failed before landing: %s", commandFailureMessage(status, err, "git status"))
	}
	if strings.TrimSpace(status.Stdout) != "" {
		return landBaseResult{}, fmt.Errorf("refusing to land onto %s: current worktree has uncommitted changes", baseBranch)
	}

	commits, err := runCommand(ctx, repoRoot, "git", "-C", repoRoot, "rev-list", "--reverse", baseBranch+".."+workBranch)
	if err != nil {
		return landBaseResult{}, fmt.Errorf("failed to enumerate commits to land: %s", commandFailureMessage(commits, err, "git rev-list"))
	}
	commitList := compactLines(commits.Stdout)
	if len(commitList) == 0 {
		head, headErr := runCommand(ctx, repoRoot, "git", "-C", repoRoot, "rev-parse", "HEAD")
		if headErr != nil {
			return landBaseResult{}, fmt.Errorf("failed to read base branch head: %s", commandFailureMessage(head, headErr, "git rev-parse"))
		}
		return landBaseResult{
			Head:        strings.TrimSpace(head.Stdout),
			Method:      "noop",
			CommitCount: 0,
		}, nil
	}

	mergeResult, mergeErr := runCommand(ctx, repoRoot, "git", "-C", repoRoot, "merge", "--ff-only", workBranch)
	if mergeErr == nil {
		head, headErr := runCommand(ctx, repoRoot, "git", "-C", repoRoot, "rev-parse", "HEAD")
		if headErr != nil {
			return landBaseResult{}, fmt.Errorf("fast-forward landing succeeded but HEAD lookup failed: %s", commandFailureMessage(head, headErr, "git rev-parse"))
		}
		return landBaseResult{
			Head:        strings.TrimSpace(head.Stdout),
			Method:      "fast-forward",
			CommitCount: len(commitList),
		}, nil
	}

	args := append([]string{"-C", repoRoot, "cherry-pick"}, commitList...)
	cherryResult, cherryErr := runCommand(ctx, repoRoot, "git", args...)
	if cherryErr != nil {
		abortResult, abortErr := runCommand(ctx, repoRoot, "git", "-C", repoRoot, "cherry-pick", "--abort")
		if abortErr != nil {
			return landBaseResult{}, fmt.Errorf("cherry-pick landing failed: %s; abort also failed: %s", commandFailureMessage(cherryResult, cherryErr, "git cherry-pick"), commandFailureMessage(abortResult, abortErr, "git cherry-pick --abort"))
		}
		return landBaseResult{}, fmt.Errorf("failed to land commits onto %s: ff-only merge failed (%s); cherry-pick failed (%s)", baseBranch, commandFailureMessage(mergeResult, mergeErr, "git merge --ff-only"), commandFailureMessage(cherryResult, cherryErr, "git cherry-pick"))
	}

	head, headErr := runCommand(ctx, repoRoot, "git", "-C", repoRoot, "rev-parse", "HEAD")
	if headErr != nil {
		return landBaseResult{}, fmt.Errorf("cherry-pick landing succeeded but HEAD lookup failed: %s", commandFailureMessage(head, headErr, "git rev-parse"))
	}
	return landBaseResult{
		Head:        strings.TrimSpace(head.Stdout),
		Method:      "cherry-pick",
		CommitCount: len(commitList),
	}, nil
}

func compactLines(value string) []string {
	raw := strings.Split(strings.TrimSpace(value), "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}
