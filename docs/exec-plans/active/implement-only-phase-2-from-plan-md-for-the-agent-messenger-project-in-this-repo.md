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
- [x] M1. Expand domain/store contracts for Phase 2 reads/writes (status: completed)
  - Add model/request DTOs needed for user search, conversation summaries/details, message pagination, message create/update/soft-delete, and upload response shape.
  - Extend `store.Store` interface and `NoopStore` with Phase 2 operations and ownership/participant checks.

- [ ] M2. Implement SQLite data access for Phase 2 operations (status: not started)
  - Add SQL queries and methods for user search, get current user profile by auth context ID, get-or-create DM conversation, list user conversations, fetch conversation details with participant validation, list messages with cursor pagination, create message, edit own message, and soft-delete own message.
  - Add/adjust indexes or migration updates only if required for Phase 2 query correctness/performance.

- [x] M3. Implement users and conversations REST handlers/routes (status: completed)
  - Add authenticated handlers for `/api/users`, `/api/users/me`, `/api/conversations` (GET/POST), and `/api/conversations/:id`.
  - Enforce validation, authorization, and consistent JSON error responses aligned with existing API behavior.

- [x] M4. Implement messages REST handlers/routes with ownership rules (status: completed)
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
- M1 completed:
  - Added Phase 2 API/domain DTOs in `server/models/phase2.go`:
    - Request DTOs: `StartConversationRequest`, `SendMessageRequest`, `EditMessageRequest`.
    - Pagination DTO: `ListMessagesQuery` with default limit `20` and max limit `100`.
    - Response/projection DTOs: `ConversationSummary`, `ConversationDetails`, `MessageDetails`, `UploadResponse`.
  - Added Phase 2 store-boundary parameter DTOs in `server/models/store_params.go`:
    - `SearchUsersParams`, `ListUserConversationsParams`, `GetOrCreateDirectConversationParams`,
      `GetConversationForUserParams`, `ListConversationMessagesParams`,
      `CreateMessageParams`, `UpdateMessageParams`, `SoftDeleteMessageParams`.
  - Extended store contracts in `server/store/store.go`:
    - Added Phase 2 methods for user search, conversation list/detail/get-or-create DM, message list/create/update/delete.
    - Added `ErrForbidden` to represent ownership/participant authorization failures at store boundary.
    - Expanded `NoopStore` with all new method stubs.
  - Added compile-safe placeholder methods on `SQLiteStore` in `server/store/sqlite.go` that return `ErrNotImplemented` for new Phase 2 operations; concrete SQL behavior is deferred to M2.
  - Added model validation tests in `server/models/phase2_test.go`.
  - Validation: `cd server && go test ./...` passes.
- M2 completed:
  - Implemented concrete SQLite methods in `server/store/sqlite.go` for all Phase 2 store operations:
    - `SearchUsersByUsername`
    - `ListConversationsByUser`
    - `GetOrCreateDirectConversation`
    - `GetConversationByIDForUser`
    - `ListMessagesByConversation` with ID-based cursor semantics via `before` message lookup and `(created_at, id)` ordering
    - `CreateMessage`
    - `UpdateMessage` with own-message enforcement
    - `SoftDeleteMessage` with own-message enforcement
  - Added participant/ownership enforcement in store layer:
    - conversation participant checks return `ErrForbidden`
    - message edit/delete ownership violations return `ErrForbidden`
  - Added SQLite helper routines for scanning and conversion:
    - message and conversation lookup helpers
    - nullable string/message decoding helpers
  - Added migration `version 7` in `server/store/migrations.go` for Phase 2 query correctness/performance:
    - unique participant-pair index for DM conversation deduplication
    - participant lookup indexes on conversations
    - additional message lookup indexes for pagination/sender filtering
  - Expanded store tests in `server/store/sqlite_test.go`:
    - user search
    - DM get-or-create idempotency
    - conversation list/detail participant behavior
    - message list pagination with `before`
    - create/edit/soft-delete message ownership rules
  - Validation: `cd server && go test ./...` passes.
- M3 completed:
  - Added authenticated users + conversations API handlers in `server/api/users_conversations.go`:
    - `GET /api/users` with `username` query search and optional positive `limit`
    - `GET /api/users/me` from bearer-auth context
    - `GET /api/conversations` with optional positive `limit`
    - `POST /api/conversations` using `{ "username": "..." }` to get-or-create DM
    - `GET /api/conversations/:id` for participant-scoped details
  - Added request validation and error mapping aligned with existing API style:
    - `400` for invalid JSON/validation/query values
    - `401` for missing/invalid bearer context
    - `403` for participant-forbidden conversation access
    - `404` for unknown usernames/conversations
    - `500` for unexpected storage failures
  - Router wiring updates in `server/api/router.go`:
    - registered new routes under bearer middleware.
  - Added API tests in `server/api/users_conversations_test.go`:
    - users profile and search endpoint behavior
    - conversation create/list/detail behavior
    - idempotent DM get-or-create behavior
    - forbidden and validation error paths
  - Validation: `cd server && go test ./...` passes.
- M4 completed:
  - Added message REST handlers in `server/api/messages.go`:
    - `GET /api/conversations/:id/messages`
      - supports cursor pagination with `before` and `limit` (default 20, max enforced by model validation).
    - `POST /api/conversations/:id/messages`
      - supports JSON text payloads (`application/json`, body `{ "content": "..." }`)
      - supports multipart attachment payloads (`multipart/form-data`, fields: `content`, file field `attachment`)
      - also supports multipart `attachment_url` + `attachment_type` fallback fields
      - enforces 20 MB max attachment size for multipart file uploads.
    - `PATCH /api/messages/:id`
      - edits only caller-owned messages.
    - `DELETE /api/messages/:id`
      - soft-deletes only caller-owned messages and returns updated message payload.
  - Added route wiring in `server/api/router.go`:
    - dispatch for `/api/conversations/:id/messages` under existing bearer auth middleware
    - `PATCH/DELETE` routing for `/api/messages/:id`.
  - Updated CORS allow-methods in `server/api/middleware.go` to include `PATCH`.
  - Added API tests in `server/api/messages_test.go`:
    - message create/list/paginate flows
    - multipart attachment message send
    - own-message edit/delete authorization constraints
    - forbidden access for non-participants.
  - Validation: `cd server && go test ./...` passes.

## Key decisions
- Keep scope strictly bounded to Phase 2 endpoints and upload/static serving requirements.
- Reuse the existing bearer-auth middleware and request auth context for all Phase 2 protected routes.
- Keep DM semantics constrained to two participants as defined by existing schema and SPEC.
- Keep message pagination cursor based on message ID (`before`) and enforce a bounded `limit` with default 20.
- Keep file serving rooted in configurable `UPLOAD_DIR` with path-safe handling to prevent traversal.
- Preserve existing project conventions for JSON responses and error handling unless Phase 2 requires a new shape.
- Establish explicit Phase 2 request/response DTOs up front so API handlers and store methods can share one canonical contract.
- Encode ownership/participant authorization at the store boundary using actor-scoped method signatures and `store.ErrForbidden`.
- Canonicalize DM participant ordering in persistence (`participant_a`, `participant_b`) to enforce stable get-or-create behavior backed by a unique index.
- Implement message pagination using `before=<message_id>` translated into an internal `(created_at, id)` seek condition for deterministic ordering.
- Keep users and conversations routes mounted under the existing bearer middleware and resolve current user identity directly from auth context for `/api/users/me` and actor-scoped conversation operations.
- Standardized message multipart field naming for Phase 2 implementation:
  - message text: `content`
  - attachment file: `attachment`
  - optional fallback fields: `attachment_url`, `attachment_type`.

## Remaining issues / open questions
- Decide whether upload MIME/type validation should be permissive (any file under 20 MB) or restricted to a known allowlist for clearer behavior.

## Links to related documents
- `AGENTS.md`
- `PLAN.md`
- `SPEC.md`
- `docs/exec-plans/active/implement-only-phase-1-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-2-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
