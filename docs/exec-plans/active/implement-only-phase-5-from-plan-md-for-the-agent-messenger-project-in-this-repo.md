# Execution Plan: Implement Only Phase 5 from PLAN.md for Agent Messenger

## Goal / scope
Implement only Phase 5 from `PLAN.md` in `web/`, building the full chat UI on top of the existing Phase 4 web foundation and existing server contract:
- Sidebar conversation list and active conversation navigation
- Start-new-DM flow from user search
- `/dm/:conversationId` chat route
- Cursor-based message list with load-older behavior
- Real-time updates via existing WebSocket hook
- Message bubbles including edited/deleted/attachment states
- Message input with file/image attach support
- Edit/delete interactions for own messages
- Emoji reaction toggle UI

Out of scope: Phase 6+ work except narrowly required plumbing to make this Phase 5 deliverable function.

## Background
`PLAN.md` defines Phase 5 as full web chat UI completion on top of the Phase 4 foundation. `SPEC.md` defines the API and WebSocket contracts this UI must consume (conversations/messages/reactions/upload and websocket mutation events).

Phase execution docs indicate:
- Phase 4 already delivered routing/auth/API client/websocket hook foundations in `web/`
- Server-side REST/WebSocket/reaction contracts are already implemented in Phases 1-3

Reviewed docs for this plan:
- `AGENTS.md`
- `PLAN.md`
- `SPEC.md`
- `docs/exec-plans/active/implement-only-phase-1-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-2-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-3-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/continue-only-the-remaining-phase-3-milestones-from-the-current-branch-state-for.md`
- `docs/exec-plans/active/implement-only-phase-4-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`

Referenced but missing (noted once):
- `ARCHITECTURE.md`
- `docs/PLANS.md`

## Milestones
- [x] M1. Implement chat shell routing and sidebar foundation (status: completed)
  - Wire protected app layout for sidebar + main pane.
  - Implement conversation list rendering with partner name and last-message preview.
  - Add start-new-DM entry UI and create/open flow using existing typed API client.

- [ ] M2. Implement `/dm/:conversationId` data loading and cursor-based message history (status: not started)
  - Add conversation detail fetch and message pagination state.
  - Implement load-older behavior using `before` cursor + limit.
  - Ensure route transitions and initial-scroll behavior are stable.

- [ ] M3. Implement message rendering for Phase 5 bubble requirements (status: not started)
  - Render sender, timestamp, edited badge (`[수정됨]`), and deleted placeholder (`"삭제된 메시지입니다"`).
  - Render inline image attachments and downloadable non-image file links.
  - Ensure own-vs-other bubble styling and deleted/edit state precedence are correct.

- [ ] M4. Implement composer and own-message actions (status: not started)
  - Build message input with text send + file/image attach flow (upload + send message payload integration).
  - Implement own-message edit mode (prefill, submit patch, cancel).
  - Implement own-message delete action with UI interaction (context menu/right-click equivalent on web).

- [ ] M5. Implement real-time synchronization and reaction toggle UX (status: not started)
  - Connect websocket event stream into conversation/message cache reconciliation.
  - Add grouped reaction bar with counts and own-user toggle affordance.
  - Implement add/remove reaction actions and optimistic or immediate server-synced updates.

- [ ] M6. Validate Phase 5 end-to-end and finalize (status: not started)
  - Run relevant `web` checks (tests if defined, plus build) and fix regressions.
  - Verify required Phase 5 paths manually in app behavior.
  - Keep scope limited to Phase 5 deliverable and record any explicit non-goal deferrals.

## Current progress
- Worktree/branch initialization verified via:
  - `./ralph-loop init --base-branch main --work-branch ralph-phase-5-web-client-chat-ui --output json`
- Verified fields match prepared environment:
  - `worktree_path=/Users/dev/git/agent-messenger/.worktrees/phase-5-web-client-chat-ui`
  - `work_branch=ralph-phase-5-web-client-chat-ui`
  - `base_branch=main`
  - `worktree_id=phase-5-web-client-chat-ui-a3cdc51d`
- Relevant docs have been reviewed for Phase 5 planning.
- Milestone M1 is complete:
  - Replaced the Phase 4 placeholder protected route with nested chat routes:
    - `/` renders chat shell + empty-state pane
    - `/dm/:conversationId` renders chat shell + active-conversation placeholder pane
  - Implemented sidebar conversation query using `apiClient.listConversations()` with:
    - participant username rendering (`other_user.username`)
    - last-message preview rendering including deleted and attachment markers
  - Implemented start-new-DM flow in sidebar:
    - username search via `apiClient.searchUsers()`
    - create/open via `apiClient.startConversation({ username })`
    - navigation to created/existing conversation route on success
  - Added sidebar account/logout action using existing auth context.
  - Verified web build passes: `npm run build` (after installing `web/` dependencies with `npm ci` in this worktree).
- Milestones M2-M6 remain not started.

## Key decisions
- Keep implementation strictly bounded to Phase 5 UI and required integration wiring.
- Reuse existing server/API/websocket contracts and existing Phase 4 web foundation instead of introducing new protocol changes.
- Sequence work so each milestone is implementable in one coding-loop iteration with verifiable outcomes.
- Prioritize stable message timeline behavior (pagination + realtime merges) before polishing interaction details.
- Keep `/dm/:conversationId` as a route-level placeholder in M1 so sidebar navigation works now while deferring message loading/pagination logic strictly to M2.

## Remaining issues / open questions
- M2 must implement actual conversation detail and cursor-based message timeline data loading for `/dm/:conversationId` (currently placeholder only).
- M2 must establish initial timeline UX rules (initial anchor and load-older trigger behavior) on top of the new shell routing.

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
