# Execution Plan: Implement Only Phase 2 from PLAN.md for Agent Messenger

## Goal / scope
Implement only Phase 2 from `PLAN.md` on top of the completed Phase 1 server:
- `GET /api/users` (username search)
- `GET /api/users/me` (current user profile)
- `GET /api/conversations` (current user's conversations)
- `POST /api/conversations` (start DM by username)
- `GET /api/conversations/:id` (conversation details)
- `GET /api/conversations/:id/messages` (cursor pagination with `before` and `limit`, default limit 20)
- `POST /api/conversations/:id/messages` (text or multipart attachment)
- `PATCH /api/messages/:id` (edit own message only)
- `DELETE /api/messages/:id` (soft-delete own message only)
- `POST /api/upload` (file/image upload up to 20 MB, returns `{ "url": "..." }`)
- Static serving for uploads at `/static/uploads/` from configurable `UPLOAD_DIR`

Out of scope: WebSocket/reactions (Phase 3), web client phases, CLI phase, PostgreSQL support, and other Phase 7 hardening beyond what Phase 2 explicitly requires.

## Background
`PLAN.md` defines Phase 2 as the full core REST layer for users, conversations, messages, and upload handling. `SPEC.md` defines endpoint intent, DM-only model assumptions, message editing/deletion behavior, and file handling constraints (20 MB max, `UPLOAD_DIR`, static serving under `/static/uploads/`).

Phase 1 has already established:
- Go server skeleton and env-based startup
- SQLite schema/migrations including users/conversations/messages/reactions/sessions
- Auth endpoints and bearer-token middleware
- CORS middleware and basic API routing

Required docs reviewed for this plan: `AGENTS.md`, `PLAN.md`, `SPEC.md`, `docs/exec-plans/active/implement-only-phase-1-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`.
Not found (noted once): `ARCHITECTURE.md`, `docs/PLANS.md`.

## Milestones
- [ ] M1. Expand domain/store contracts for Phase 2 reads/writes (status: not started)
  - Add model/request DTOs needed for user search, conversation summaries/details, message pagination, message create/update/soft-delete, and upload response shape.
  - Extend `store.Store` interface and `NoopStore` with Phase 2 operations and ownership/participant checks.

- [ ] M2. Implement SQLite data access for Phase 2 operations (status: not started)
  - Add SQL queries and methods for user search, get current user profile by auth context ID, get-or-create DM conversation, list user conversations, fetch conversation details with participant validation, list messages with cursor pagination, create message, edit own message, and soft-delete own message.
  - Add/adjust indexes or migration updates only if required for Phase 2 query correctness/performance.

- [ ] M3. Implement users and conversations REST handlers/routes (status: not started)
  - Add authenticated handlers for `/api/users`, `/api/users/me`, `/api/conversations` (GET/POST), and `/api/conversations/:id`.
  - Enforce validation, authorization, and consistent JSON error responses aligned with existing API behavior.

- [ ] M4. Implement messages REST handlers/routes with ownership rules (status: not started)
  - Add handlers for `GET /api/conversations/:id/messages`, `POST /api/conversations/:id/messages`, `PATCH /api/messages/:id`, and `DELETE /api/messages/:id`.
  - Support JSON text sends and multipart attachment sends; enforce own-message edit/delete rules and soft-delete semantics.

- [ ] M5. Implement upload endpoint and static file serving configuration (status: not started)
  - Add `POST /api/upload` for multipart uploads with 20 MB cap and safe file naming, returning `{ "url": "..." }`.
  - Add `UPLOAD_DIR` config (default `./uploads`) and serve `/static/uploads/` from that directory.

- [ ] M6. Add/extend tests and run validation for full Phase 2 completion (status: not started)
  - Add store and API tests covering happy paths and key authorization/validation failures for each Phase 2 endpoint.
  - Run `go test ./...` in `server/` and resolve failures until green.

## Current progress
- Phase 2 setup initialization verified with:
  - `./ralph-loop init --base-branch main --work-branch ralph-phase-2-core-rest-api --output json`
  - Verified JSON fields: `worktree_path=/Users/dev/git/agent-messenger/.worktrees/phase-2-core-rest-api`, `work_branch=ralph-phase-2-core-rest-api`, `base_branch=main`, `worktree_id=phase-2-core-rest-api-8a22ad8c`.
- Execution plan created for Phase 2 implementation.
- Milestones M1-M6 are all not started.

## Key decisions
- Keep scope strictly bounded to Phase 2 endpoints and upload/static serving requirements.
- Reuse the existing bearer-auth middleware and request auth context for all Phase 2 protected routes.
- Keep DM semantics constrained to two participants as defined by existing schema and SPEC.
- Keep message pagination cursor based on message ID (`before`) and enforce a bounded `limit` with default 20.
- Keep file serving rooted in configurable `UPLOAD_DIR` with path-safe handling to prevent traversal.
- Preserve existing project conventions for JSON responses and error handling unless Phase 2 requires a new shape.

## Remaining issues / open questions
- Confirm final response envelope shapes for conversation list/detail and message list where `SPEC.md` defines behavior but not strict JSON schema details.
- Confirm exact multipart field names for attachment upload in `POST /api/conversations/:id/messages` and `POST /api/upload` (to be standardized and tested during implementation).
- Decide whether upload MIME/type validation should be permissive (any file under 20 MB) or restricted to a known allowlist for clearer behavior.

## Links to related documents
- `AGENTS.md`
- `PLAN.md`
- `SPEC.md`
- `docs/exec-plans/active/implement-only-phase-1-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-2-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
