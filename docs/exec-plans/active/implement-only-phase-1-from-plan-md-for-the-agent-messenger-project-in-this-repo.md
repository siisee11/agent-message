# Execution Plan: Implement Only Phase 1 from PLAN.md for Agent Messenger

## Goal / scope
Implement only Phase 1 from `PLAN.md` for the server foundation:
- Initialize Go server module and directory layout
- Define `User`, `Conversation`, `Message`, `Reaction` models
- Implement SQLite schema/migrations and data access
- Add auth endpoints (`register`, `login`, `logout`) with bcrypt PIN hashing and opaque session tokens
- Add bearer-token auth middleware and CORS middleware
- Provide env-configured `main.go` entrypoint

Out of scope: all Phase 2+ API, WebSocket behaviors, web client work, CLI work, PostgreSQL implementation.

## Background
`PLAN.md` defines Phase 1 as the foundational server deliverable with working auth and DB-backed persistence.
`SPEC.md` defines authentication behavior (username + 4-6 digit PIN, opaque tokens, bearer auth) and core data-model fields for users, conversations, messages, and reactions.
`AGENTS.md` provides operational guardrails for the local `ralph-loop` workflow.

## Milestones
- [x] M1. Scaffold `server/` Go module, package directories (`api/`, `ws/`, `store/`, `models/`), and baseline wiring (status: completed)
- [x] M2. Implement core model structs and validation-friendly request/response shapes needed by Phase 1 auth and store boundaries (status: completed)
- [x] M3. Build SQLite store layer with schema migrations for users, conversations, messages, reactions, and sessions; add repository/data-access methods needed by auth flow (status: completed)
- [ ] M4. Implement auth application flow and HTTP handlers for `POST /api/auth/register`, `POST /api/auth/login`, and `DELETE /api/auth/logout` with bcrypt PIN hashing and opaque token issuance/revocation (status: not started)
- [ ] M5. Add bearer auth middleware, CORS middleware, env-driven config, and `main.go` startup path; run Phase 1 smoke checks (status: not started)

## Current progress
- Worktree and branch verified via `./ralph-loop init --base-branch main --work-branch ralph-phase1-go-server-auth-data --output json`.
- Relevant docs reviewed: `AGENTS.md`, `PLAN.md`, `SPEC.md`.
- Not found: `ARCHITECTURE.md`, `docs/PLANS.md`.
- M1 completed:
  - Added `server/go.mod` with module `agent-messenger/server`.
  - Added baseline package structure: `server/api/`, `server/ws/`, `server/store/`, `server/models/`.
  - Added compile-safe wiring:
    - `server/main.go` bootstraps HTTP server on `:8080` with router dependencies.
    - `server/api/router.go` defines dependency container and `/healthz` handler.
    - `server/store/store.go` defines initial store interface and noop store implementation.
    - `server/ws/hub.go` defines initial hub placeholder and constructor.
    - `server/models/doc.go` initializes models package.
  - Validation: `cd server && go test ./...` passes.
- M2 completed:
  - Added Phase 1 domain models aligned with `SPEC.md`:
    - `server/models/user.go` (`User`, `UserProfile`)
    - `server/models/conversation.go` (`Conversation`)
    - `server/models/message.go` (`Message`, `AttachmentType`)
    - `server/models/reaction.go` (`Reaction`)
    - `server/models/session.go` (`Session`)
  - Added auth request/response DTOs with validation for auth boundary:
    - `server/models/auth.go` (`RegisterRequest`, `LoginRequest`, `AuthResponse`)
    - Validation enforces non-empty username and 4-6 digit numeric PIN.
  - Added store-boundary parameter DTOs for upcoming auth persistence:
    - `server/models/store_params.go` (`CreateUserParams`, `CreateSessionParams`)
  - Added validation tests:
    - `server/models/auth_test.go`
  - Validation: `cd server && go test ./...` passes.
- M3 completed:
  - Replaced placeholder store contract with auth-ready store interface in `server/store/store.go`:
    - `CreateUser`, `GetUserByUsername`, `GetUserByID`
    - `CreateSession`, `GetSessionByToken`, `DeleteSessionByToken`, `GetUserBySessionToken`
    - Introduced shared store errors: `ErrNotFound`, `ErrNotImplemented`.
  - Implemented SQLite-backed store in `server/store/sqlite.go`:
    - `NewSQLiteStore(ctx, dsn)` opens DB, enables FK enforcement, and runs migrations.
    - Implemented auth data access methods for users/sessions required by upcoming auth handlers and bearer middleware.
    - Added explicit RFC3339Nano timestamp serialization/parsing for deterministic SQLite storage.
  - Implemented migration system in `server/store/migrations.go`:
    - Added `schema_migrations` tracking table.
    - Added ordered migrations for `users`, `conversations`, `messages`, `reactions`, `sessions`, and supporting indexes.
  - Added store tests in `server/store/sqlite_test.go`:
    - Migration application verification.
    - Auth persistence flow verification (create user/session, resolve by token, delete session, not-found behavior).
  - Added SQLite dependency in `server/go.mod`/`server/go.sum` (`modernc.org/sqlite`).
  - Validation: `cd server && go test ./...` passes.

## Key decisions
- Enforce strict phase boundary: only Phase 1 deliverables are implemented.
- Use SQLite as the Phase 1 persistence backend with explicit migration steps.
- Use bcrypt for PIN storage and cryptographically secure opaque tokens for sessions.
- Keep middleware minimal and production-safe: bearer auth extraction/validation and configurable CORS policy.
- Keep startup configuration environment-driven with sane defaults for local execution.
- For M1 scaffolding, keep runtime wiring intentionally minimal (health route + placeholder dependencies) so later milestones can layer auth/store logic without restructuring.
- Keep model tags explicit for both JSON and DB mapping to reduce translation glue in SQLite repository methods.
- Keep auth payload validation centralized in model DTOs so handlers can reuse shared rules when implemented in M4.
- Use pure-Go SQLite driver `modernc.org/sqlite` to avoid CGO/toolchain coupling for local and CI execution.
- Apply migrations during SQLite store initialization so Phase 1 server startup can guarantee schema readiness.

## Remaining issues / open questions
- Confirm final env var names and defaults during implementation (aligning with repo conventions if discovered).
- Determine the minimal store interface surface needed now vs deferred for Phase 2.
- Next milestone is M4: auth service flow + `/api/auth/register|login|logout` handlers using bcrypt + opaque session tokens.

## Links to related documents
- `AGENTS.md`
- `PLAN.md`
- `SPEC.md`
- Target plan file: `docs/exec-plans/active/implement-only-phase-1-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
