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
- [x] M1. Upgrade `web/` to React 19 and align related dependencies/tooling until build and typecheck are green (status: completed)
- [x] M2. Introduce explicit message-kind protocol/types and parsing helpers for deterministic `text` vs `json_render` handling with backward-compatible text defaults (status: completed)
- [x] M3. Implement DM bubble render branching in `DmConversationPage.tsx` with deleted-message precedence, text bubble parity, and minimal read-only json-render registry rendering (status: completed)
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
- Completed M1 (React 19 upgrade in `web/`):
  - Updated `web/package.json` to React 19 dependency ranges:
    - `react` / `react-dom` from `^18.3.1` to `^19.0.0`
    - `@types/react` / `@types/react-dom` from `^18.x` to `^19.0.0`
  - Regenerated `web/package-lock.json` via `npm install` in `web/`.
  - Updated explicit `JSX.Element` return annotations to inferred return types in:
    - `web/src/App.tsx`
    - `web/src/auth/AuthProvider.tsx`
    - `web/src/realtime/RealtimeProvider.tsx`
    - `web/src/pages/ChatIndexPage.tsx`
    - `web/src/pages/ChatShellPage.tsx`
    - `web/src/pages/DmConversationPage.tsx`
    - `web/src/routes/ProtectedRoute.tsx`
  - Verified build + typecheck pass with `cd web && npm run build`.
- Completed M2 (explicit message protocol/types and parsing helpers):
  - Added message protocol types in `web/src/api/types.ts`:
    - `MessageKind = 'text' | 'json_render'`
    - `JsonRenderSpec = unknown`
    - optional protocol fields on `Message`: `kind`, `message_kind`, `json_render`, `json_render_spec`
  - Added `web/src/api/messageProtocol.ts` with deterministic helpers:
    - `resolveMessageKind` and `isJsonRenderMessage` (no JSON-text inference)
    - `resolveJsonRenderSpec`
    - `normalizeMessageProtocol`, `normalizeMessageDetailsProtocol`, `normalizeConversationSummaryProtocol`
    - `parseMessageContent` for future render branching
  - Integrated normalization in data ingestion paths:
    - API client normalizes `listConversations`, `listMessages`, `sendMessage`, `editMessage`, `deleteMessage`
    - SSE parsing normalizes `message.new` and `message.edited` payloads
  - Preserved backward compatibility by defaulting missing/unknown kinds to `text`.
  - Verified build + typecheck pass with `cd web && npm run build`.
- Completed M3 (DM render branching + minimal read-only json-render bubble support):
  - Added json-render runtime dependencies in `web/package.json`:
    - `@json-render/core`
    - `@json-render/react`
    - `zod` (required peer for json-render)
  - Added read-only json-render message renderer:
    - `web/src/components/MessageJsonRender.tsx`
    - `web/src/components/MessageJsonRender.module.css`
  - Implemented a small registry (`Stack`, `Text`, `Badge`) and fallback rendering for unknown/invalid specs, intentionally minimal for first-pass support.
  - Updated `web/src/pages/DmConversationPage.tsx` to branch on parsed message protocol:
    - `text` kind: render existing message text bubble behavior (`trim()` + same `<p className={styles.messageText}>`)
    - `json_render` kind: render via `MessageJsonRender` inside the bubble
    - deleted placeholder branch still precedes both and remains dominant
  - Kept attachment rendering and reaction rendering branches unchanged.
  - Verified build + typecheck pass with `cd web && npm run build`.

## Key decisions
- Use an explicit message-kind protocol field for deterministic rendering; do not parse arbitrary text as JSON specs.
- Preserve backward compatibility by treating legacy/unspecified messages as normal text.
- Keep deleted placeholder rendering as the highest-precedence branch.
- Keep json-render handling intentionally minimal and read-only for this iteration.
- Preserve existing attachment and reaction behavior without protocol drift.
- For React 19 type compatibility, prefer inferred component return types over global `JSX.Element` annotations.
- Normalize protocol from either `kind` or `message_kind` while keeping `text` as the fallback for absent or unexpected values.
- Use `@json-render/react` `Renderer` with a minimal component registry and no action handlers for read-only json-render message rendering.

## Remaining issues / open questions
- Confirm the compact preview label string for json-render messages (for example `[json-render]`) before final polish if product wording differs.
- React 19 install emitted a transient peer-resolution warning from stale `18.x` metadata during dependency resolution, but final tree is resolved to React/ReactDOM 19.x and build is green.
- Json-render payload shape is still accepted as `unknown` at the API edge; runtime rendering currently expects flat json-render spec shape (`root` + `elements`) and falls back to a compact placeholder when invalid.

## Links to related documents
- `AGENTS.md`
- `PLAN.md`
- `SPEC.md`
- `README.md`
- `docs/exec-plans/active/implement-only-phase-4-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-5-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-6-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
- `docs/exec-plans/active/implement-only-phase-7-from-plan-md-for-the-agent-messenger-project-in-this-repo.md`
