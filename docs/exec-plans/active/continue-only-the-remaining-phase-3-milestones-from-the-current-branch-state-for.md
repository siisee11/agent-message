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
- [x] M5. Implement WebSocket session runtime: register client with conversation subscriptions, pump hub events to socket, and parse client frames for `read` events with conversation subscription updates (status: completed)
- [x] M6. Emit `message.new`, `message.edited`, and `message.deleted` events from message mutation handlers to the conversation via hub broadcast (status: completed)
- [x] M7. Emit `reaction.added` and `reaction.removed` events from reaction handlers with payloads aligned to `SPEC.md` event contracts (status: completed)
- [x] M8. Add/expand tests for reaction endpoints, websocket read handling/subscription behavior, and message/reaction broadcast integration (status: completed)
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
- Completed M5 websocket session runtime:
  - Added websocket text-frame JSON I/O support in `server/ws/frames.go` (`ReadJSON`, `WriteJSON`) with masking validation for incoming client frames and ping/pong handling.
  - Updated `server/api/websocket.go` to bootstrap client subscriptions from `ListConversationsByUser` at connect-time, register those conversation IDs in the hub, and run an outbound write pump that forwards hub events to the socket.
  - Added client frame parsing for `{ "type": "read", "data": { "conversation_id": ... } }`; valid participant-scoped `read` events now subscribe the connection to that conversation through the hub.
  - Added focused websocket runtime tests in `server/api/websocket_test.go` for:
    - hub event delivery to a connected client over the websocket
    - post-connect `read` event subscription updates for conversations created after socket establishment
  - Verified with `cd server && go test ./api ./ws` (pass).
- Completed M6 message mutation websocket emission:
  - Wired a shared hub instance in router initialization so HTTP message handlers and websocket sessions publish/consume through the same `ws.Hub`.
  - Extended `messagesHandler` to hold a hub reference and emit:
    - `message.new` on successful `POST /api/conversations/:id/messages` with full `models.Message` payload
    - `message.edited` on successful `PATCH /api/messages/:id` with full `models.Message` payload
    - `message.deleted` on successful `DELETE /api/messages/:id` with `{ "id": "<message_id>" }` payload (aligned with `SPEC.md`)
  - Added websocket integration coverage in `server/api/websocket_test.go` validating that REST create/edit/delete mutations are broadcast to subscribed clients with expected event types and IDs.
  - Verified with `cd server && go test ./api` (pass).
- Completed M7 reaction mutation websocket emission:
  - Extended reaction handlers in `server/api/messages.go` to emit websocket events after successful store mutations:
    - `reaction.added` with full `models.Reaction` payload for toggle-add actions
    - `reaction.removed` with `{ "message_id", "emoji", "user_id" }` payload for toggle-remove and explicit delete actions, aligned to `SPEC.md`
  - Added a participant-scoped store lookup boundary (`GetMessageByIDForUser`) to resolve `conversation_id` from `message_id` safely before broadcasting reaction events.
  - Implemented the new store boundary across contracts and SQLite/noop implementations:
    - `server/models/store_params.go`
    - `server/store/store.go`
    - `server/store/sqlite.go`
  - Verified with `cd server && go test ./api ./store` (pass).
- Completed M8 Phase 3 test expansion:
  - Expanded websocket integration coverage in `server/api/websocket_test.go` for reaction mutation broadcast paths:
    - `reaction.added` payload assertions for toggle-add
    - `reaction.removed` payload assertions for toggle-remove
    - `reaction.removed` payload assertions for explicit `DELETE /api/messages/:id/reactions/:emoji`
  - Added websocket read/subscription negative-path coverage:
    - `read` event with a conversation the caller is not a participant of does not subscribe and does not deliver broadcast events.
  - Verified with `cd server && go test ./api` (pass).
- Remaining gap areas by inspection:
  - Full repository test sweep and final Phase 3 validation remain (tracked in M9)

## Key decisions
- Preserve existing Phase 3 commits and continue from their current behavior; do not refactor completed milestone surfaces unless required for compatibility.
- Treat this as a continuation plan that starts at remaining work (`M4`-`M9`) only.
- Keep event names/payload shapes aligned with `SPEC.md` to prevent contract drift.
- Keep each milestone scoped for one coding-loop iteration.
- `POST /api/messages/:id/reactions` returns `models.ToggleReactionResult` (includes `action` + `reaction`) to preserve store-level toggle semantics (`added` / `removed`) for upcoming websocket emission in M7.
- `DELETE /api/messages/:id/reactions/:emoji` returns the removed `models.Reaction` payload on success (`200 OK`) for deterministic client reconciliation.
- `read` websocket events are transport-level subscription updates only in Phase 3; no read-receipt persistence side effects are introduced.
- Message mutation broadcast failures are currently best-effort/non-blocking for REST success paths (mutation responses are not failed if websocket delivery cannot be enqueued).
- Reaction mutation broadcast failures are also best-effort/non-blocking for REST success paths, matching message mutation semantics.
- Integration test assertions for reaction websocket payloads now explicitly enforce `SPEC.md` contracts for both add and remove events.

## Remaining issues / open questions
- Bootstrap conversation subscription currently loads a single bounded page (`Limit=1000`) at connect-time; if higher conversation cardinality appears later, pagination strategy can be revisited without changing the runtime contract introduced in M5.

## Links to related documents
- `AGENTS.md`
- `PLAN.md`
- `SPEC.md`
- `docs/exec-plans/active/implement-only-phase-1-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-2-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-3-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/continue-only-the-remaining-phase-3-milestones-from-the-current-branch-state-for.md`
