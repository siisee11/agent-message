# Execution Plan: Implement Only Phase 3 from PLAN.md for Agent Messenger

## Goal / scope
Implement only Phase 3 from `PLAN.md` on top of completed Phases 1 and 2:
- WebSocket hub in `server/ws/hub.go` to manage authenticated client connections and conversation-scoped broadcasts
- `GET /ws?token=<token>` endpoint that authenticates and upgrades to WebSocket
- Server-side real-time event emission on message send/edit/delete and reaction add/remove
- Client-to-server `read` event handling (`{ "type": "read", "data": { "conversation_id": "..." } }`)
- Reaction APIs:
  - `POST /api/messages/:id/reactions` (toggle/one-per-emoji-per-user)
  - `DELETE /api/messages/:id/reactions/:emoji` (remove caller's reaction)

Out of scope: Phase 4+ web/CLI deliverables, PostgreSQL support work, and non-Phase-3 hardening.

## Background
`PLAN.md` defines Phase 3 as WebSocket real-time delivery and reaction endpoint support. `SPEC.md` defines WebSocket transport/auth shape (`/ws?token=<token>`), required event names (`message.new`, `message.edited`, `message.deleted`, `reaction.added`, `reaction.removed`), and reaction semantics (one reaction per emoji per user per message).

Phase 1 and Phase 2 execution plans show the server already has auth middleware, SQLite persistence, messages API, and routing in place. Those are the integration points for Phase 3.

Reviewed docs: `AGENTS.md`, `PLAN.md`, `SPEC.md`, `docs/exec-plans/active/implement-only-phase-1-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`, `docs/exec-plans/active/implement-only-phase-2-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`.
Not found (noted once): `ARCHITECTURE.md`, `docs/PLANS.md`.

## Milestones
- [x] M1. Build conversation-aware WebSocket hub primitives in `server/ws/` (register/unregister clients, track user identity + subscribed conversation IDs, and broadcast typed events) (status: completed)
- [x] M2. Add authenticated WebSocket upgrade endpoint (`GET /ws?token=<token>`) and wire it into server routing, including token validation and connection lifecycle management (status: completed)
- [x] M3. Add reaction persistence contract + SQLite implementation for add/toggle/remove reaction behavior constrained to participants and caller ownership semantics (status: completed)
- [ ] M4. Implement reaction REST handlers/routes (`POST /api/messages/:id/reactions`, `DELETE /api/messages/:id/reactions/:emoji`) with validation/error mapping aligned to existing API conventions (status: not started)
- [ ] M5. Integrate real-time event emission from message/reaction mutations and implement client `read` event ingestion path in the WebSocket server loop (status: not started)
- [ ] M6. Add and run Phase 3 test coverage (hub behavior, WebSocket auth/upgrade flow, message/reaction broadcast events, reaction endpoint rules), then verify `cd server && go test ./...` (status: not started)

## Current progress
- Worktree re-init verified with:
  - `./ralph-loop init --base-branch main --work-branch ralph-phase-3-websocket-reactions --output json`
  - Verified JSON fields: `worktree_path=/Users/dev/git/agent-messenger/.worktrees/phase-3-websocket-reactions`, `work_branch=ralph-phase-3-websocket-reactions`, `base_branch=main`, `worktree_id=phase-3-websocket-reactions-b4cb9be5`.
- Required/relevant docs reviewed and scoped for Phase 3 implementation.
- Implemented `server/ws/hub.go` hub primitives for Phase 3:
  - Client registration/unregistration with user identity tracking
  - Conversation subscription management (`Subscribe`, `Unsubscribe`, `SetConversations`)
  - Typed event payload model and constants for `message.*` / `reaction.*` events
  - Conversation-targeted broadcasting with non-blocking fan-out and delivery/drop counters
  - Introspection helpers used by tests (`ConnectionCount`, `ConnectionsForUser`, `ConversationIDs`)
- Added `server/ws/hub_test.go` with coverage for:
  - Register/broadcast/unregister lifecycle
  - Conversation subscription mutation behavior
  - Validation errors and drop semantics for blocked client channels
- Verification run: `cd server && go test ./...` passed.
- Implemented authenticated WebSocket endpoint for Phase 3 in `server/api/websocket.go` and wired route `/ws` in `server/api/router.go`:
  - `GET /ws?token=<token>` token lookup via `GetUserBySessionToken`
  - Error mapping aligned with existing auth conventions (`missing or invalid bearer token`, `invalid session token`)
  - Upgrade + connection lifecycle: successful upgrade registers hub client and unregisters on disconnect
- Added lightweight HTTP upgrade support in `server/ws/upgrade.go` (RFC6455 handshake validation for upgrade headers, key, version; hijack + `101 Switching Protocols` response).
- Added endpoint tests in `server/api/websocket_test.go`:
  - Successful auth + upgrade handshake and hub connection lifecycle
  - Missing/invalid token auth failures
  - Missing upgrade header validation failure
- Verification run: `cd server && go test ./...` passed after M2 changes.
- Added reaction persistence contract in store/model boundaries:
  - New store methods in `server/store/store.go`:
    - `ToggleMessageReaction(ctx, params)` returning `models.ToggleReactionResult`
    - `RemoveMessageReaction(ctx, params)` returning the removed `models.Reaction`
  - New param shapes in `server/models/store_params.go`:
    - `ToggleMessageReactionParams`
    - `RemoveMessageReactionParams`
  - New toggle outcome model in `server/models/reaction.go`:
    - `ReactionMutationAction` (`added`, `removed`)
    - `ToggleReactionResult`
- Implemented SQLite reaction behavior in `server/store/sqlite.go`:
  - Toggle semantics for `message_id + user_id + emoji` uniqueness (add when absent, remove when present)
  - Explicit remove semantics for caller-owned reaction (`message_id + user_id + emoji`)
  - Participant gating enforced through message conversation membership checks
  - Correct domain errors:
    - `ErrNotFound` for missing message/reaction
    - `ErrForbidden` for non-participant actor
- Added store-level reaction tests in `server/store/sqlite_test.go`:
  - Add/toggle/remove flow and action assertions
  - One-per-emoji-per-user behavior verification
  - Participant/ownership and missing-resource error cases
- Verification run: `cd server && go test ./...` passed after M3 changes.

## Key decisions
- Keep implementation strictly bounded to Phase 3 deliverables from `PLAN.md`.
- Reuse existing auth/session store and router conventions for WebSocket token validation and reaction endpoint protection.
- Preserve event names/payload contracts from `SPEC.md` so downstream web/CLI phases can consume them without schema drift.
- Implement reaction toggling at API/store boundary in an idempotent way (existing same-emoji reaction by same user should be removable via add endpoint behavior).
- Prioritize deterministic, testable hub behavior over premature optimization.
- Hub broadcast is intentionally non-blocking per recipient (`Dropped` count recorded) to prevent a single slow client from stalling conversation fan-out.
- Hub does not close client channels on unregister; connection lifecycle ownership remains with the WebSocket endpoint loop planned in M2.
- M2 adopts query-token auth exactly as specified (`/ws?token=<token>`) and keeps WebSocket endpoint outside bearer middleware to avoid header-based/session-context coupling.
- Initial M2 connection registration does not pre-subscribe conversation IDs; subscriptions remain empty until event-path integration in M5.
- Reaction persistence contract now returns explicit mutation action (`added`/`removed`) from toggle to support M4 HTTP payload mapping and M5 websocket event emission without re-querying state.

## Remaining issues / open questions
- Confirm desired HTTP response payload shape for toggled-off `POST /api/messages/:id/reactions` in M4 (echoing `action=removed` vs reaction-only response contract).
- M5 implementation detail pending: parse websocket text frames for client `read` events and write outbound server events as framed websocket messages (currently M2 keeps transport connection/lifecycle only).

## Links to related documents
- `AGENTS.md`
- `PLAN.md`
- `SPEC.md`
- `docs/exec-plans/active/implement-only-phase-1-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-2-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-3-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
