# Execution Plan: Implement Chat Message Rendering Support for json-render.dev in the Web Client

## Goal / scope
Implement deterministic text-vs-json-render chat message rendering in the web DM UI and upgrade the web app to React 19, while preserving current behavior for deleted messages, attachments, reactions, and existing text flows.

In scope:
- Upgrade `web/` to React 19 and make dependencies/build pass.
- Add explicit message kind/protocol handling for DM message rendering.
- Render normal text messages with existing bubble behavior.
- Render json-render messages using a minimal, read-only json-render registry inside the message bubble.
- Disable edit affordances and edit flows for json-render messages (delete remains allowed).
- Update conversation list preview so json-render messages do not show raw specs.
- Keep deleted-message precedence over all other rendering branches.
- Add/update tests covering parse/render branching and preview/editability rules.

Out of scope:
- PR flow changes.
- Expanding json-render registry beyond minimal read-only support needed for this task.

## Background
Task constraints target `web/src/pages/DmConversationPage.tsx` and `web/src/pages/ChatShellPage.tsx`, requiring deterministic protocol-level message kind handling (no inference from arbitrary JSON text).

Relevant docs reviewed:
- `AGENTS.md`
- `PLAN.md`
- `SPEC.md`
- `README.md`
- `docs/exec-plans/active/implement-only-phase-1-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-2-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-3-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/continue-only-the-remaining-phase-3-milestones-from-the-current-branch-state-for.md`
- `docs/exec-plans/active/implement-only-phase-4-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-5-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-6-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-7-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`

Referenced but missing (noted once):
- `ARCHITECTURE.md`
- `docs/PLANS.md`

## Milestones
- [ ] M1. Upgrade `web/` to React 19 and align related dependencies/tooling until build and typecheck are green (status: not started)
- [ ] M2. Introduce explicit message-kind protocol/types and parsing helpers for deterministic `text` vs `json_render` handling with backward-compatible text defaults (status: not started)
- [ ] M3. Implement DM bubble render branching in `DmConversationPage.tsx` with deleted-message precedence, text bubble parity, and minimal read-only json-render registry rendering (status: not started)
- [ ] M4. Update editability and preview behavior: disable edit affordances/flows for `json_render` messages, keep delete allowed, and add compact conversation preview fallback in `ChatShellPage.tsx` (status: not started)
- [ ] M5. Add/update tests for message-kind parsing, render branch selection, preview fallback, and editability constraints; run web verification commands and fix regressions (status: not started)

## Current progress
- Ran `./ralph-loop init --base-branch main --work-branch ralph-react19-json-render-chat --output json`.
- Verified output matches prepared environment:
  - `worktree_path=/Users/dev/git/agent-messenger/.worktrees/react19-json-render-chat`
  - `work_branch=ralph-react19-json-render-chat`
  - `base_branch=main`
  - `worktree_id=react19-json-render-chat-fd2a1bcf`
- Reviewed the repository docs listed above and scoped implementation milestones.
- No code changes started yet.

## Key decisions
- Use an explicit message-kind protocol field for deterministic rendering; do not parse arbitrary text as JSON specs.
- Preserve backward compatibility by treating legacy/unspecified messages as normal text.
- Keep deleted placeholder rendering as the highest-precedence branch.
- Keep json-render handling intentionally minimal and read-only for this iteration.
- Preserve existing attachment and reaction behavior without protocol drift.

## Remaining issues / open questions
- Confirm exact server payload/source field name for message kind if current API types differ from planned `text` / `json_render` naming.
- Confirm the compact preview label string for json-render messages (for example `[json-render]`) before final polish if product wording differs.

## Links to related documents
- `AGENTS.md`
- `PLAN.md`
- `SPEC.md`
- `README.md`
- `docs/exec-plans/active/implement-only-phase-4-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-5-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-6-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-7-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
