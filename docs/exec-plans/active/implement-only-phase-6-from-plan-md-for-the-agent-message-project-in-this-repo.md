# Execution Plan: Implement Only Phase 6 from PLAN.md for Agent Message

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
- `docs/exec-plans/active/implement-only-phase-1-from-plan-md-for-the-agent-message-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-2-from-plan-md-for-the-agent-message-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-3-from-plan-md-for-the-agent-message-project-in-this-repo.md`
- `docs/exec-plans/active/continue-only-the-remaining-phase-3-milestones-from-the-current-branch-state-for.md`
- `docs/exec-plans/active/implement-only-phase-4-from-plan-md-for-the-agent-message-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-5-from-plan-md-for-the-agent-message-project-in-this-repo.md`

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

- [x] M4. Implement `send` and `read`, including read-session index map persistence (status: completed)
  - `send <username> <text>` posts message to resolved conversation.
  - `read <username> [--n N]` fetches recent messages and prints indexed output.
  - Persist per-conversation read-session mapping from display index to message UUID.

- [x] M5. Implement index-based mutations and reactions (status: completed)
  - `edit <index> <text>` resolves index from last read session and patches message.
  - `delete <index>` resolves and soft-deletes message.
  - `react <index> <emoji>` and `unreact <index> <emoji>` resolve index and call reaction endpoints.

- [x] M6. Implement watch mode and Phase 6 validation/finalization (status: completed)
  - `watch <username>` opens websocket with token and streams `message.new` for that DM to stdout.
  - Add/update CLI tests for config, index resolution, command behavior, and websocket event handling where practical.
  - Run relevant `go test` and `go build` checks for `cli/`, keep scope strictly Phase 6, and finalize with small logical milestone commits.

## Current progress
- Verified prepared environment using:
  - `./ralph-loop init --base-branch main --work-branch ralph-phase-6-cli-client`
- Confirmed returned JSON matches:
  - `worktree_path=/Users/dev/git/agent-message/.worktrees/phase-6-cli-client`
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
- Completed M4 send/read and read-session persistence:
  - Implemented `send <username> <text>` and `read <username> [--n N]` handlers in `cli/internal/cmd/messages.go`.
  - `send` resolves DM conversation by username and posts to `POST /api/conversations/:id/messages`.
  - `read` resolves DM conversation by username, fetches latest messages with `limit`, prints indexed output in SPEC format (`[index] <uuid> <user>: <text>`), and persists per-conversation index→message mapping in config `read_sessions`.
  - Added helpers to persist session data including `conversation_id`, `username`, `index_to_message`, and `last_read_message`.
  - Added tests in `cli/internal/cmd/messages_test.go` for send flow, read output, read-session persistence, and invalid limit handling.
  - Fixed API client request-path parsing so query parameters are sent as URL query (not path), which was required for `read --n` to correctly call `GET /api/conversations/:id/messages?limit=...`.
- Additional validation run after M4:
  - `cd cli && GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/go-mod go test ./...`
  - `cd cli && GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/go-mod go build ./...`
- Completed M5 index-based mutations and reactions:
  - Implemented `edit`, `delete`, `react`, and `unreact` command handlers in `cli/internal/cmd/mutations.go`.
  - Added index-resolution helpers to map `<index>` from last `read` session to message ID.
  - Added deterministic “last read session” pointer by extending config with `last_read_conversation_id`.
  - Updated read-session persistence to set `last_read_conversation_id` on every successful `read`.
  - Added mutation tests in `cli/internal/cmd/mutations_test.go` covering edit/delete/react/unreact happy paths and index/session error cases.
  - Updated config tests and read tests to cover `last_read_conversation_id` behavior.
- Additional validation run after M5:
  - `cd cli && GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/go-mod go test ./...`
  - `cd cli && GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/go-mod go build ./...`
- Completed M6 watch mode and final validation:
  - Implemented `watch <username>` in `cli/internal/cmd/watch.go`.
  - Added minimal in-repo websocket client (HTTP upgrade handshake + frame reader) compatible with existing server websocket contract, without introducing external dependencies.
  - `watch` resolves/open DM by username, connects to `/ws?token=...`, filters incoming `message.new` events for the target conversation, and streams matching lines to stdout.
  - Added watch tests in `cli/internal/cmd/watch_test.go` (event filtering, URL/token usage, auth guard).
  - Removed final placeholder command file now that all Phase 6 commands are implemented.
- Final validation run after M6:
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
- Preserve server message ordering (newest-first) in `read` output and index mapping so local index resolution remains deterministic with server pagination order.
- Track a single explicit last-read conversation in config so index-only commands (`edit/delete/react/unreact`) resolve against the most recent `read` target.
- Use an internal websocket client implementation for Phase 6 watch mode due restricted dependency/network environment while keeping protocol compatibility with the existing server contract.

## Remaining issues / open questions
- Phase 6 CLI scope is fully implemented.
- Optional follow-up outside Phase 6 scope: replace local Cobra shim with upstream `github.com/spf13/cobra` once dependency fetch is available.

## Links to related documents
- `AGENTS.md`
- `PLAN.md`
- `SPEC.md`
- `docs/exec-plans/active/implement-only-phase-1-from-plan-md-for-the-agent-message-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-2-from-plan-md-for-the-agent-message-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-3-from-plan-md-for-the-agent-message-project-in-this-repo.md`
- `docs/exec-plans/active/continue-only-the-remaining-phase-3-milestones-from-the-current-branch-state-for.md`
- `docs/exec-plans/active/implement-only-phase-4-from-plan-md-for-the-agent-message-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-5-from-plan-md-for-the-agent-message-project-in-this-repo.md`
