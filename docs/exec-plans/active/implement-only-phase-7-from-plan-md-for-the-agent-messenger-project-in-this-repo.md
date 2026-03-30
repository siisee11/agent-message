# Execution Plan: Implement Only Phase 7 from PLAN.md for Agent Messenger

## Goal / scope
Implement only Phase 7 from `PLAN.md` in this worktree, reusing existing contracts unless Phase 7 explicitly requires changes.

In scope:
- Add PostgreSQL store support behind the existing store interface.
- Switch database backend via `DB_DRIVER` environment variable (SQLite/PostgreSQL).
- Make API error responses consistent across endpoints as JSON: `{ "error": "..." }`.
- Tighten input validation for usernames, PINs, and upload constraints.
- Add end-to-end integration tests using `httptest` and the server stack.
- Add Docker Compose for local production-like server + PostgreSQL testing.
- Update quickstart docs for server, web, and CLI.

Out of scope:
- Any non-Phase-7 feature expansion.
- Contract changes not required by the explicit Phase 7 tasks.

## Background
`PLAN.md` Phase 7 defines integration/polish hardening after Phases 1-6 are complete. `SPEC.md` already sets key behavioral constraints relevant to this phase:
- Database targets include SQLite (dev) and PostgreSQL (prod).
- Username + PIN auth contract (PIN is 4-6 numeric digits).
- Upload handling constraints and file serving behavior.

Docs reviewed for this plan:
- `AGENTS.md`
- `PLAN.md`
- `SPEC.md`
- `docs/exec-plans/active/implement-only-phase-1-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-2-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-3-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/continue-only-the-remaining-phase-3-milestones-from-the-current-branch-state-for.md`
- `docs/exec-plans/active/implement-only-phase-4-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-5-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-6-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`

Referenced but missing (noted once):
- `ARCHITECTURE.md`
- `docs/PLANS.md`

## Milestones
- [ ] M1. Add PostgreSQL store implementation and `DB_DRIVER` backend switching in server bootstrap (status: not started)
  - Implement Postgres-backed store with parity for existing store interface methods used by API/WebSocket flows.
  - Add config/bootstrap selection for SQLite vs PostgreSQL via `DB_DRIVER` and supporting DSN envs.
  - Preserve existing behavior/contracts for handlers and store callers.

- [ ] M2. Normalize API error response shape across endpoints to `{ "error": "..." }` (status: not started)
  - Audit shared HTTP helpers and all handlers to ensure uniform JSON error envelope and status mapping.
  - Remove remaining inconsistent bodies (plain text/non-standard payloads) while preserving status semantics.

- [ ] M3. Tighten validation for auth/user inputs and uploads (status: not started)
  - Enforce username validation rules consistently at API boundary.
  - Enforce PIN 4-6 numeric-digit validation consistently.
  - Tighten upload validation for size/type/field handling per Phase 7 constraints.

- [ ] M4. Add server-stack end-to-end integration tests with `httptest` (status: not started)
  - Add integration test coverage for critical auth/conversation/message/reaction/upload flows against the router/server stack.
  - Cover both success and key validation/error paths for newly tightened behavior.

- [ ] M5. Add Docker Compose for local production-like server + PostgreSQL runs (status: not started)
  - Provide `docker-compose` configuration with server and PostgreSQL services.
  - Wire env/config defaults for local compose startup and storage.

- [ ] M6. Update quickstart documentation and run final checks (status: not started)
  - Update `README.md` quickstart covering server, web, and CLI workflows (including DB driver selection).
  - Run relevant tests/build checks and fix regressions within Phase 7 scope.
  - Finalize with small logical milestone commits only.

## Current progress
- Verified environment initialization:
  - Ran `./ralph-loop init --base-branch main --work-branch ralph-phase-7-integration-polish`.
  - Confirmed JSON values:
    - `worktree_path=/Users/dev/git/agent-messenger/.worktrees/phase-7-integration-polish`
    - `work_branch=ralph-phase-7-integration-polish`
    - `base_branch=main`
    - `worktree_id=phase-7-integration-polish-8ab3ca8c`
- Reviewed the relevant existing repository docs listed above.
- No implementation milestones have started yet.

## Key decisions
- Keep scope strictly bounded to Phase 7 tasks from `PLAN.md`.
- Preserve existing API/store contracts unless a Phase 7 requirement explicitly forces a change.
- Keep milestones small enough for one coding-loop iteration and commit after each logical milestone.
- Validate each milestone with targeted tests/build checks before moving forward.

## Remaining issues / open questions
- Confirm exact repository location/creation path for the top-level `README.md` quickstart update if a root `README.md` is absent in current tree state.
- Confirm preferred PostgreSQL DSN/env naming conventions in existing server config before implementing `DB_DRIVER` switch wiring.

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
- `docs/exec-plans/active/implement-only-phase-6-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
