# Execution Plan: Implement Only Phase 4 from PLAN.md for Agent Messenger

## Goal / scope
Implement only Phase 4 from `PLAN.md` for the web client foundation under `web/`:
- Scaffold a React + TypeScript app with Vite.
- Install and wire minimal dependencies for routing, data fetching, and styling.
- Add a typed API client under `web/src/api/` for existing server REST endpoints.
- Implement auth state management and a `/login` page that logs in and auto-registers on first login.
- Add protected routing that redirects unauthenticated users to `/login`.
- Implement a WebSocket client hook with reconnect logic.

Out of scope: Phase 5+ chat UI work (conversation layout, message list UI, reactions UI, attachments UI beyond what is required to validate Phase 4 integration), CLI work, PostgreSQL/polish tasks from later phases.

## Background
`PLAN.md` defines Phase 4 as the web foundation milestone and explicitly calls out Vite scaffold, API client typing, auth flow, protected route behavior, and a reconnecting WebSocket hook.

`SPEC.md` defines the server contract this Phase 4 work must consume:
- Auth: `POST /api/auth/register`, `POST /api/auth/login`, `DELETE /api/auth/logout`
- Users/conversations/messages/reactions/upload REST surfaces
- WebSocket endpoint `GET /ws?token=<session_token>` and event types (`message.new`, `message.edited`, `message.deleted`, `reaction.added`, `reaction.removed`)

Existing phase execution docs show server-side foundations are already in place through Phase 3, so this phase should reuse the current server API contract without modifying server behavior.

Reviewed docs for this plan:
- `AGENTS.md`
- `PLAN.md`
- `SPEC.md`
- `docs/exec-plans/active/implement-only-phase-1-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-2-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-3-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/continue-only-the-remaining-phase-3-milestones-from-the-current-branch-state-for.md`

Referenced but missing (noted once):
- `ARCHITECTURE.md`
- `docs/PLANS.md`

## Milestones
- [x] M1. Scaffold `web/` with Vite React + TypeScript, configure baseline project structure, and install minimal dependencies for routing/data-fetching/styling (status: completed with install blocker documented)
- [x] M2. Implement typed API client in `web/src/api/` for auth/users/conversations/messages/reactions/upload endpoints using shared request/response types and centralized auth token handling (status: completed)
- [x] M3. Implement auth state management (provider + hooks + token persistence) and `/login` page with username/PIN form, login-first flow, and auto-register fallback on first login (status: completed)
- [x] M4. Implement protected route wrapper and minimal route wiring so unauthenticated users are redirected to `/login` and authenticated users can reach the protected app entry route (status: completed)
- [x] M5. Implement `web/src/hooks/useWebSocket.ts` with token-based connection, reconnect/backoff behavior, event parsing callbacks, and lifecycle cleanup; then run web build/tests and fix issues (status: completed)

## Current progress
- Verified worktree initialization with:
  - `./ralph-loop init --base-branch main --work-branch ralph-phase-4-web-client-foundation --output json`
- Verified returned values match prepared environment:
  - `worktree_path=/Users/dev/git/agent-messenger/.worktrees/phase-4-web-client-foundation`
  - `work_branch=ralph-phase-4-web-client-foundation`
  - `base_branch=main`
  - `worktree_id=phase-4-web-client-foundation-1c6918d3`
- Reviewed relevant repository documentation and phase execution context.
- Created this Phase 4 execution plan with milestones set to not started.
- Completed M1 scaffold by creating `web/` manually with Vite-equivalent React + TypeScript structure:
  - Added `package.json`, TypeScript configs, Vite config, `index.html`, and `src/main.tsx`.
  - Wired minimal dependencies for routing/data fetching in app bootstrap (`react-router-dom`, `@tanstack/react-query`).
  - Added baseline styling (`global.css` + CSS modules) and placeholder routed pages.
  - Created foundation directories for upcoming work: `src/api`, `src/auth`, `src/hooks`, `src/routes`.
- Attempted automated scaffold with `npm create vite@latest web -- --template react-ts`; failed due network resolution error to `registry.npmjs.org` in this environment.
- Completed M2 typed API client implementation under `web/src/api/`:
  - Added API contract types in `web/src/api/types.ts` aligned to current server JSON fields for auth, users, conversations, messages, reactions, and upload.
  - Added `ApiClient` in `web/src/api/client.ts` covering all Phase 4 REST endpoints:
    - Auth: register/login/logout
    - Users: search users, get current user
    - Conversations: list, start DM, fetch detail
    - Messages: list, create (JSON + multipart), edit, delete
    - Reactions: toggle and remove by emoji
    - Upload: multipart upload helper
  - Centralized HTTP behavior in one layer: base URL construction, query serialization, bearer token injection, typed JSON responses, and normalized API error handling via `ApiError`.
  - Added API exports in `web/src/api/index.ts` for client + types.
- Completed M3 auth foundation and `/login` flow:
  - Added `AuthProvider` + `useAuth` in `web/src/auth/AuthProvider.tsx` with auth states (`loading`, `authenticated`, `unauthenticated`), token persistence, and logout handling.
  - Added localStorage-backed token bootstrap that revalidates sessions via `GET /api/users/me` on app start.
  - Implemented login-first + auto-register fallback in `loginWithAutoRegister`:
    - First try `POST /api/auth/login`.
    - On `401`, attempt `POST /api/auth/register`.
    - If fallback register returns `409`, surface as invalid credentials.
  - Added `/login` page UI and form submission logic in `web/src/pages/LoginPage.tsx` with PIN validation and post-auth redirect behavior.
  - Wired `AuthProvider` at app bootstrap (`src/main.tsx`) and exposed `/login` route in `src/App.tsx`.
  - Added shared API runtime instance (`web/src/api/runtime.ts`) to connect auth state with centralized API token handling.
- Completed M4 protected routing:
  - Added `ProtectedRoute` wrapper in `web/src/routes/ProtectedRoute.tsx`.
  - Route guard behavior:
    - While auth bootstrap is in progress (`status=loading`), render a minimal loading state.
    - If unauthenticated, redirect to `/login` with `state.from` preserving the attempted location.
    - If authenticated, render protected route content through `<Outlet />`.
  - Updated `src/App.tsx` route wiring so app entry route (`/`) is nested under `ProtectedRoute`.
- Completed M5 WebSocket hook and final validation:
  - Added `web/src/hooks/useWebSocket.ts` with:
    - Token-based `/ws?token=...` connection URL handling.
    - Configurable reconnect backoff (`initialRetryDelayMs`, exponential growth, `maxRetryDelayMs`, optional `maxReconnectAttempts`).
    - Typed server event parsing and event-specific callbacks for:
      - `message.new`
      - `message.edited`
      - `message.deleted`
      - `reaction.added`
      - `reaction.removed`
    - Lifecycle-safe cleanup for socket close and reconnect timers on unmount/dependency change.
    - Outbound helpers: generic `sendEvent` and typed `sendReadEvent(conversationId)`.
  - Added hook exports in `web/src/hooks/index.ts`.
  - Installed declared web dependencies successfully (`npm install`, `npm install --include=dev`) and generated lockfile (`web/package-lock.json`).
  - Ran build:
    - Initial failure: `tsc: command not found` due missing dev dependencies.
    - Resolved by installing dev dependencies.
    - Second failure: TypeScript narrowing errors in hook callback dispatch.
    - Resolved by adding explicit event type guards.
    - Final `npm run build` succeeded.
  - Attempted test run: `npm run test` reports missing script (no web test script is currently defined).
  - Added ignore patterns in `web/.gitignore` for local TypeScript build artifacts (`*.tsbuildinfo`, `vite.config.js`, `vite.config.d.ts`).

## Key decisions
- Keep implementation strictly bounded to Phase 4 deliverables from `PLAN.md`.
- Reuse existing server contract from `SPEC.md` and current server implementation; avoid server-side contract changes.
- Choose the smallest reliable dependency set for Phase 4:
  - Routing: `react-router-dom`
  - Data fetching/cache: `@tanstack/react-query`
  - Styling: CSS modules + lightweight global CSS (no additional styling framework dependency)
- Define all API payloads as TypeScript types and centralize HTTP concerns (base URL, auth header injection, JSON handling, error normalization) in one API client layer.
- Keep WebSocket hook focused on connectivity contract (connect/reconnect/cleanup/event dispatch) and not on Phase 5 UI concerns.
- Initial scaffold iteration proceeded without npm registry access, so files were created manually to match Vite React+TS conventions; dependencies were later installed once network access succeeded.
- API base URL decision: default to same-origin paths with optional override via `VITE_API_BASE_URL`; token handling is centralized in `ApiClient` through `setAuthToken` and optional dynamic `getToken`.
- Auth flow decision: treat `/login` as login-first and attempt auto-register only on `401`; when register fallback conflicts (`409`), report invalid credentials rather than silently overriding an existing account.
- Protected-route decision: route guard lives at router layer (`src/routes/ProtectedRoute.tsx`) so auth checks are centralized and reusable for additional protected routes in later phases.
- WebSocket hook decision: expose a strongly-typed event callback surface while still forwarding unknown event types through a generic `onEvent` callback for forward compatibility.

## Remaining issues / open questions
- No remaining Phase 4 implementation milestones.
- Open follow-up (non-blocking for Phase 4): web package currently has no `test` script, so only build validation is available in this phase.

## Links to related documents
- `AGENTS.md`
- `PLAN.md`
- `SPEC.md`
- `docs/exec-plans/active/implement-only-phase-1-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-2-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-3-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/continue-only-the-remaining-phase-3-milestones-from-the-current-branch-state-for.md`
- `docs/exec-plans/active/implement-only-phase-4-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
