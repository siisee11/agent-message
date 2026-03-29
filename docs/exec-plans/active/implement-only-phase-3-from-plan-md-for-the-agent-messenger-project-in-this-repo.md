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
- [ ] M1. Build conversation-aware WebSocket hub primitives in `server/ws/` (register/unregister clients, track user identity + subscribed conversation IDs, and broadcast typed events) (status: not started)
- [ ] M2. Add authenticated WebSocket upgrade endpoint (`GET /ws?token=<token>`) and wire it into server routing, including token validation and connection lifecycle management (status: not started)
- [ ] M3. Add reaction persistence contract + SQLite implementation for add/toggle/remove reaction behavior constrained to participants and caller ownership semantics (status: not started)
- [ ] M4. Implement reaction REST handlers/routes (`POST /api/messages/:id/reactions`, `DELETE /api/messages/:id/reactions/:emoji`) with validation/error mapping aligned to existing API conventions (status: not started)
- [ ] M5. Integrate real-time event emission from message/reaction mutations and implement client `read` event ingestion path in the WebSocket server loop (status: not started)
- [ ] M6. Add and run Phase 3 test coverage (hub behavior, WebSocket auth/upgrade flow, message/reaction broadcast events, reaction endpoint rules), then verify `cd server && go test ./...` (status: not started)

## Current progress
- Worktree re-init verified with:
  - `./ralph-loop init --base-branch main --work-branch ralph-phase-3-websocket-reactions --output json`
  - Verified JSON fields: `worktree_path=/Users/dev/git/agent-messenger/.worktrees/phase-3-websocket-reactions`, `work_branch=ralph-phase-3-websocket-reactions`, `base_branch=main`, `worktree_id=phase-3-websocket-reactions-b4cb9be5`.
- Required/relevant docs reviewed and scoped for Phase 3 implementation.
- Milestones M1-M6 are defined and all currently not started.

## Key decisions
- Keep implementation strictly bounded to Phase 3 deliverables from `PLAN.md`.
- Reuse existing auth/session store and router conventions for WebSocket token validation and reaction endpoint protection.
- Preserve event names/payload contracts from `SPEC.md` so downstream web/CLI phases can consume them without schema drift.
- Implement reaction toggling at API/store boundary in an idempotent way (existing same-emoji reaction by same user should be removable via add endpoint behavior).
- Prioritize deterministic, testable hub behavior over premature optimization.

## Remaining issues / open questions
- Clarify whether WebSocket clients should auto-subscribe to all user conversations on connect or subscribe dynamically via traffic-triggered association; default plan is conversation-targeted delivery based on known conversation IDs from mutation events.
- Confirm desired response payload for toggled-off `POST /api/messages/:id/reactions` (explicit removed indicator vs returning current aggregate state).

## Links to related documents
- `AGENTS.md`
- `PLAN.md`
- `SPEC.md`
- `docs/exec-plans/active/implement-only-phase-1-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-2-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-3-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
