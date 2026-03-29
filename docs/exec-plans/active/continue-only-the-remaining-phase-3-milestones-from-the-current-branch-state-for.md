# Execution Plan: Continue Only the Remaining Phase 3 Milestones from the Current Branch State

## Goal / scope
Continue Phase 3 implementation from the current `ralph-phase-3-websocket-reactions` branch state without redoing completed work.

In scope for this continuation:
- Remaining WebSocket runtime behavior for authenticated clients and conversation-scoped event flow
- Server event emission for message and reaction mutations
- Reaction API endpoint wiring and handler behavior
- Phase 3 test completion and validation

Out of scope:
- Reworking already completed Phase 3 commits (`m1`-`m3`)
- Any Phase 4+ web/CLI/PostgreSQL/polish work

## Background
`PLAN.md` Phase 3 requires: WebSocket hub + authenticated `/ws`, mutation event emission (`message.new`, `message.edited`, `message.deleted`, `reaction.added`, `reaction.removed`), client `read` handling, and reaction endpoints.

From current branch history, the following Phase 3 milestones are already committed and must be preserved:
- `phase3 m1: implement websocket hub primitives` (`3d8f0ac`)
- `phase3 m2: add authenticated websocket upgrade endpoint` (`9e4d025`)
- `phase3 m3: add reaction persistence contract and sqlite impl` (`18bda69`)

Docs reviewed for this continuation plan:
- `AGENTS.md`
- `PLAN.md`
- `SPEC.md`
- `docs/exec-plans/active/implement-only-phase-1-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-2-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-3-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`

Referenced but missing (noted once):
- `ARCHITECTURE.md`
- `docs/PLANS.md`

## Milestones
- [x] M4. Implement reaction API surface (`POST /api/messages/:id/reactions`, `DELETE /api/messages/:id/reactions/:emoji`) with validation and route wiring, reusing existing store reaction methods (status: completed)
- [ ] M5. Implement WebSocket session runtime: register client with conversation subscriptions, pump hub events to socket, and parse client frames for `read` events with conversation subscription updates (status: not started)
- [ ] M6. Emit `message.new`, `message.edited`, and `message.deleted` events from message mutation handlers to the conversation via hub broadcast (status: not started)
- [ ] M7. Emit `reaction.added` and `reaction.removed` events from reaction handlers with payloads aligned to `SPEC.md` event contracts (status: not started)
- [ ] M8. Add/expand tests for reaction endpoints, websocket read handling/subscription behavior, and message/reaction broadcast integration (status: not started)
- [ ] M9. Run `cd server && go test ./...`, resolve regressions, and verify Phase 3 deliverable completeness from current state only (status: not started)

## Current progress
- Verified worktree setup using:
  - `./ralph-loop init --base-branch main --work-branch ralph-phase-3-websocket-reactions --output json`
- Verified returned values match prepared environment:
  - `worktree_path=/Users/dev/git/agent-messenger/.worktrees/phase-3-websocket-reactions`
  - `work_branch=ralph-phase-3-websocket-reactions`
  - `base_branch=main`
  - `worktree_id=phase-3-websocket-reactions-b4cb9be5`
- Confirmed current Phase 3 implementation baseline from branch state:
  - WebSocket hub primitives are present (`server/ws/hub.go`, tests)
  - Authenticated websocket upgrade endpoint exists (`server/api/websocket.go`, route wiring)
  - Reaction persistence/store contracts and SQLite implementation exist (`server/models/reaction.go`, `server/models/store_params.go`, `server/store/store.go`, `server/store/sqlite.go`)
- Completed M4 reaction API surface:
  - Added `POST /api/messages/:id/reactions` handler using `ToggleMessageReaction`, with JSON body validation via `models.ToggleReactionRequest` (`emoji` required).
  - Added `DELETE /api/messages/:id/reactions/:emoji` handler using `RemoveMessageReaction`, including URL-path emoji decoding and ownership-scoped removal behavior via store boundary.
  - Extended `/api/messages/...` dispatch to route reaction paths and preserve existing `PATCH/DELETE` message mutation behavior.
  - Added API coverage in `server/api/messages_test.go` for reaction toggle add/remove, explicit delete by emoji path, validation, forbidden actor, and not-found removal.
  - Verified with `cd server && go test ./api` (pass).
- Remaining gap areas by inspection:
  - Message handlers do not yet broadcast websocket mutation events
  - WebSocket handler currently upgrades/authenticates but does not process JSON events or flush hub events to clients

## Key decisions
- Preserve existing Phase 3 commits and continue from their current behavior; do not refactor completed milestone surfaces unless required for compatibility.
- Treat this as a continuation plan that starts at remaining work (`M4`-`M9`) only.
- Keep event names/payload shapes aligned with `SPEC.md` to prevent contract drift.
- Keep each milestone scoped for one coding-loop iteration.
- `POST /api/messages/:id/reactions` returns `models.ToggleReactionResult` (includes `action` + `reaction`) to preserve store-level toggle semantics (`added` / `removed`) for upcoming websocket emission in M7.
- `DELETE /api/messages/:id/reactions/:emoji` returns the removed `models.Reaction` payload on success (`200 OK`) for deterministic client reconciliation.

## Remaining issues / open questions
- Decide exact behavior for `read` events in Phase 3: subscription-only transport behavior vs additional persistence side effects (no persistence requirement currently defined in `PLAN.md`).

## Links to related documents
- `AGENTS.md`
- `PLAN.md`
- `SPEC.md`
- `docs/exec-plans/active/implement-only-phase-1-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-2-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-3-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/continue-only-the-remaining-phase-3-milestones-from-the-current-branch-state-for.md`
