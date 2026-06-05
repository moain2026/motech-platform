# CHANGELOG.md

تنسيق [Keep a Changelog](https://keepachangelog.com/) + إصدارات دلالية.

## [Unreleased]
### Added
- 2026-06-05: تهيئة المشروع — Go 1.23, PostgreSQL 16, هيكل المجلدات، التوثيق الأساسي، قاعدة البيانات `motech_platform`.
- 2026-06-05: قرارات معمارية (Go + Postgres + Tailwind/Alpine + NetBird switchable).
- 2026-06-05: **Backend MVP (M1)** — schema (6 tables), JWT auth (admin+agent), clients CRUD, one-time setup tokens, activity log, NetBird mock/live client, agent register+heartbeat, AES-256-GCM key crypto. Built clean, E2E verified via curl.
- 2026-06-05: Dashboard أولي (Tailwind+Alpine, RTL, Al-Abbasi Soft).
- 2026-06-05: توثيق كامل: API, ARCHITECTURE, DATABASE, SECURITY, NETBIRD, SETUP, DEPLOYMENT (+ Mermaid diagrams).
- 2026-06-05: **NetBird LIVE** — PAT مدمج (في .env)، الـ backend ينشئ setup keys حقيقية في NetBird Cloud (متحقق E2E). أضيف auto_groups للـ payload.
- 2026-06-05: **Agent built** — motech-connect.exe (Go, Windows PE32+), register/heartbeat/service, E2E tested vs live backend+NetBird. + docs/AGENT.md.
- 2026-06-05: **(ج) complete** — heartbeat ingests peer_id+public_key; real SSH key rotation on agent (ed25519 → administrators_authorized_keys); connection endpoint shows peer IP + ssh cmd + pubkey; disable/delete revoke real NetBird peer. E2E verified.
- 2026-06-05: **Dashboard redesign (فاخر)** — sidebar+logo, stats cards, status bars, activity feed, search+filter, desktop table + mobile cards, dark/light mode, toasts, loading states, crisp SVG icon set (no emoji-font dependency), Tajawal font, brand palette. Responsive Desktop+Mobile.
- 2026-06-05: **GUI installer + download link** — motech-connect.exe is now a native Windows GUI (walk, admin manifest): paste license code → one-click auto-install (NetBird+SSH+service+sync). Published at /download/motech-connect.exe via Caddy; dashboard has download buttons.
- 2026-06-05: **End-to-end verified on real VM** — full setup-key UUID join (fixed UUID-length bug), agent reports clean NetBird IP, disable → real peer deletion from NetBird. NetBird CLI auto-install on Windows. All flows tested live, not just locally.
