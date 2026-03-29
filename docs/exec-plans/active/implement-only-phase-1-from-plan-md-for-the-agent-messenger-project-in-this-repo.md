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
- [ ] M1. Scaffold `server/` Go module, package directories (`api/`, `ws/`, `store/`, `models/`), and baseline wiring (status: not started)
- [ ] M2. Implement core model structs and validation-friendly request/response shapes needed by Phase 1 auth and store boundaries (status: not started)
- [ ] M3. Build SQLite store layer with schema migrations for users, conversations, messages, reactions, and sessions; add repository/data-access methods needed by auth flow (status: not started)
- [ ] M4. Implement auth application flow and HTTP handlers for `POST /api/auth/register`, `POST /api/auth/login`, and `DELETE /api/auth/logout` with bcrypt PIN hashing and opaque token issuance/revocation (status: not started)
- [ ] M5. Add bearer auth middleware, CORS middleware, env-driven config, and `main.go` startup path; run Phase 1 smoke checks (status: not started)

## Current progress
- Worktree and branch verified via `./ralph-loop init --base-branch main --work-branch ralph-phase1-go-server-auth-data --output json`.
- Relevant docs reviewed: `AGENTS.md`, `PLAN.md`, `SPEC.md`.
- Not found: `ARCHITECTURE.md`, `docs/PLANS.md`.
- Phase 1 implementation work has not started yet.

## Key decisions
- Enforce strict phase boundary: only Phase 1 deliverables are implemented.
- Use SQLite as the Phase 1 persistence backend with explicit migration steps.
- Use bcrypt for PIN storage and cryptographically secure opaque tokens for sessions.
- Keep middleware minimal and production-safe: bearer auth extraction/validation and configurable CORS policy.
- Keep startup configuration environment-driven with sane defaults for local execution.

## Remaining issues / open questions
- Confirm final env var names and defaults during implementation (aligning with repo conventions if discovered).
- Decide migration application strategy (startup auto-migrate vs explicit migration call) while staying inside Phase 1 scope.
- Determine the minimal store interface surface needed now vs deferred for Phase 2.

## Links to related documents
- `AGENTS.md`
- `PLAN.md`
- `SPEC.md`
- Target plan file: `docs/exec-plans/active/implement-only-phase-1-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
