# Execution Plan: Implement Only Phase 6 from PLAN.md for Agent Messenger

## Goal / scope
Implement only Phase 6 from `PLAN.md`: build the full `msgr` CLI client in Go under `cli/` against the existing server REST/WebSocket contract.

In scope:
- Go CLI module and command wiring
- Config management at `~/.msgr/config` (`server_url`, `token`, plus local read-session index state required by `edit/delete/react/unreact`)
- Auth commands: `register`, `login`, `logout`
- Conversation commands: `ls`, `open`
- Message commands: `send`, `read`, `edit`, `delete` with index resolution from last `read`
- Reaction commands: `react`, `unreact`
- Watch mode: `watch <username>` streaming `message.new` websocket events to stdout

Out of scope: Phase 7+ items except narrowly required plumbing to make the Phase 6 deliverable functional.

## Background
`PLAN.md` Phase 6 and `SPEC.md` CLI sections define the required `msgr` command surface and output behaviors. Server-side auth, conversations, messages, reactions, and websocket events were implemented in prior phases, so Phase 6 should consume existing APIs/contracts without changing them.

Docs reviewed for this planning step:
- `AGENTS.md`
- `PLAN.md`
- `SPEC.md`
- `docs/exec-plans/active/implement-only-phase-1-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-2-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-3-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/continue-only-the-remaining-phase-3-milestones-from-the-current-branch-state-for.md`
- `docs/exec-plans/active/implement-only-phase-4-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-5-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`

Referenced but missing (noted once):
- `ARCHITECTURE.md`
- `docs/PLANS.md`

## Milestones
- [x] M1. Scaffold `cli/` module, root command tree, HTTP client layer, and config store primitives (status: completed)
  - Initialize `cli/go.mod` and command entrypoint.
  - Add shared REST client wrappers for existing server endpoints used by Phase 6.
  - Implement config load/save for `~/.msgr/config` and safe defaults.

- [x] M2. Implement auth commands and token lifecycle (status: completed)
  - `register <username> <pin>` and `login <username> <pin>` against `/api/auth/*`.
  - Persist returned token and server URL in config.
  - `logout` clears token and calls server logout endpoint when possible.

- [x] M3. Implement conversation commands and username-to-conversation resolution (status: completed)
  - `ls` lists user conversations.
  - `open <username>` get-or-create DM via conversations API.
  - Add shared helper used by `send/read/watch` to resolve DM conversation by username.

- [ ] M4. Implement `send` and `read`, including read-session index map persistence (status: not started)
  - `send <username> <text>` posts message to resolved conversation.
  - `read <username> [--n N]` fetches recent messages and prints indexed output.
  - Persist per-conversation read-session mapping from display index to message UUID.

- [ ] M5. Implement index-based mutations and reactions (status: not started)
  - `edit <index> <text>` resolves index from last read session and patches message.
  - `delete <index>` resolves and soft-deletes message.
  - `react <index> <emoji>` and `unreact <index> <emoji>` resolve index and call reaction endpoints.

- [ ] M6. Implement watch mode and Phase 6 validation/finalization (status: not started)
  - `watch <username>` opens websocket with token and streams `message.new` for that DM to stdout.
  - Add/update CLI tests for config, index resolution, command behavior, and websocket event handling where practical.
  - Run relevant `go test` and `go build` checks for `cli/`, keep scope strictly Phase 6, and finalize with small logical milestone commits.

## Current progress
- Verified prepared environment using:
  - `./ralph-loop init --base-branch main --work-branch ralph-phase-6-cli-client`
- Confirmed returned JSON matches:
  - `worktree_path=/Users/dev/git/agent-messenger/.worktrees/phase-6-cli-client`
  - `work_branch=ralph-phase-6-cli-client`
  - `base_branch=main`
  - `worktree_id=phase-6-cli-client-121b0b8b`
- Reviewed available planning/spec docs listed above.
- Completed M1 scaffold under `cli/`:
  - Added `cli/go.mod`, `main.go`, and root command wiring for all Phase 6 commands.
  - Added `internal/api` REST client wrappers for auth, users, conversations, messages, reactions, and websocket URL generation.
  - Added `internal/config` store with default path `~/.msgr/config`, default `server_url`, token/read-session state, normalize/load/save behavior, and unit tests.
- Added a local `github.com/spf13/cobra` compatibility module via `replace` at `cli/third_party/cobra` because the sandbox cannot reach `proxy.golang.org`; this keeps command wiring API-compatible for Phase 6 while allowing offline build/test.
- Validation run for this milestone:
  - `cd cli && GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/go-mod go test ./...`
  - `cd cli && GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/go-mod go build ./...`
- Completed M2 auth/token lifecycle:
  - Implemented `register`, `login`, and `logout` command handlers in `cli/internal/cmd/auth.go`.
  - `register`/`login` call existing `/api/auth/*` endpoints, update runtime token, and persist updated config (`server_url`, `token`) to `~/.msgr/config`.
  - `logout` attempts server-side logout when a local token exists, always clears local token, persists config, and prints a warning if remote logout fails.
  - Added auth command tests in `cli/internal/cmd/auth_test.go` for register/login persistence and logout local-clear fallback behavior.
- Additional validation run after M2:
  - `cd cli && GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/go-mod go test ./...`
  - `cd cli && GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/go-mod go build ./...`
- Completed M3 conversation commands/resolver:
  - Implemented `ls` and `open` command handlers in `cli/internal/cmd/conversations.go`.
  - `ls` now calls `GET /api/conversations` and prints one line per conversation as `<conversation_id> <other_username>`.
  - `open <username>` now resolves (get-or-create) DM via `POST /api/conversations` and prints `<conversation_id> <username>`.
  - Added shared helpers `resolveConversationByUsername` and `resolveConversationIDByUsername` for reuse by upcoming `send/read/watch` milestones.
  - Added auth gating helper `ensureLoggedIn` used by conversation operations.
  - Added `cli/internal/cmd/conversations_test.go` covering list output, open behavior, conversation-id resolution, and not-logged-in guard.
- Additional validation run after M3:
  - `cd cli && GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/go-mod go test ./...`
  - `cd cli && GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/go-mod go build ./...`

## Key decisions
- Keep implementation bounded to Phase 6 CLI deliverable and existing server/API/WS contracts.
- Use milestone-sized commits (one or a few tightly related changes per milestone) to preserve clean history and rollback safety.
- Persist read-session index mapping locally to satisfy index-based commands without introducing server changes.
- Prefer deterministic, parseable CLI output where required by SPEC while keeping human-readable defaults for core commands.
- Use a local Cobra-compatible shim (`replace github.com/spf13/cobra => ./third_party/cobra`) due offline dependency constraints in this environment.
- For `logout`, prioritize local token invalidation durability: clear and persist local token even when remote `/api/auth/logout` returns an error.
- Keep conversation command output stable and simple (`<conversation_id> <username>`) to support scripting and easier manual inspection.

## Remaining issues / open questions
- M4+ command behavior is still stubbed and must be implemented sequentially.
- Decide whether to keep the local Cobra shim or switch to upstream `github.com/spf13/cobra` when networked module fetch is available.

## Links to related documents
- `AGENTS.md`
- `PLAN.md`
- `SPEC.md`
- `docs/exec-plans/active/implement-only-phase-1-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-2-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-3-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/continue-only-the-remaining-phase-3-milestones-from-the-current-branch-state-for.md`
- `docs/exec-plans/active/implement-only-phase-4-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-5-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
