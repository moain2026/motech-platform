# TODO.md — المهام المتبقية

## 🔴 الآن (Phase 1 — Backend)
- [ ] `go.mod` + هيكل (`cmd/server`, `internal/{config,db,auth,handlers,models,netbird}`)
- [ ] Migrations SQL (6 جداول)
- [ ] Config loader (env: DATABASE_URL, NETBIRD_API_URL, NETBIRD_API_TOKEN, JWT_SECRET, MASTER_KEY, PORT)
- [ ] DB connection (sqlx)
- [ ] JWT auth + admin login + middleware
- [ ] Clients CRUD + Setup Token (one-time) + Activity Log
- [ ] Health endpoint + run on :8080

## 🟡 التالي (Phase 2 — NetBird)
- [ ] NetBird client (switchable URL)
- [ ] add client → setup key + group/policy
- [ ] disable/delete → revoke

## 🟢 لاحقاً
- [ ] Agent (Go, Windows service, heartbeat, NetBird auto-join)
- [ ] Dashboard (login, list, add, copy, rotate, disable, log)
- [ ] AES-GCM key encryption
- [ ] .exe build + code signing
- [ ] NetBird self-hosted migration guide

## ❓ بانتظار المستخدم
- [ ] NetBird Personal Access Token (للتكامل الحقيقي Phase 2)
- [ ] هوية بصرية للوحة (شعار/ألوان Motech) — اختياري
