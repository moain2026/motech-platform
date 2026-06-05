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
