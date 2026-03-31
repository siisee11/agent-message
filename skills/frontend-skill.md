---
name: frontend-skill
version: 1
description: Work safely on the Vite React frontend by tracing routes, auth, realtime state, page-local CSS modules, and validating with a production build.
---

# Frontend Skill

1. Start by reading `web/package.json`, `web/src/App.tsx`, and the target page or provider before making changes.
2. Route-level entry points live in `web/src/pages/`. Current primary screens are `LoginPage`, `ChatShellPage`, and `DmConversationPage`.
3. Auth state lives in `web/src/auth/AuthProvider.tsx`. Realtime message updates and reaction state live in `web/src/realtime/RealtimeProvider.tsx` plus `web/src/realtime/state.ts`.
4. Keep data fetching aligned with the existing React Query patterns. Reuse the current query keys such as `['conversations']`, `['conversation', conversationId]`, and `['messages', conversationId]` instead of inventing parallel cache shapes.
5. Prefer page-local CSS modules for screen styling and use `web/src/styles/global.css` only for app-wide tokens and document-level defaults.
6. Preserve the existing phone-first messaging UX. Check mobile layout, loading states, empty states, optimistic-feeling interactions, and realtime updates when changing message or conversation flows.
7. Validate frontend changes with `cd web && npm run build`. If the change depends on the integrated stack, use `./dev-up` from the repo root for a production-like local run.
