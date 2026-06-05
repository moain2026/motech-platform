# TODO.md — المهام المتبقية

## ✅ مكتمل (Phase 1 — Backend)
- [x] `go.mod` + هيكل كامل
- [x] Migrations SQL (6 جداول) + تطبيق تلقائي
- [x] Config loader (كل المتغيرات + fail-fast)
- [x] DB connection (sqlx)
- [x] JWT auth (admin+agent) + middleware + bcrypt + AES-GCM
- [x] Clients CRUD + Setup Token (one-time) + Activity Log + connection
- [x] Agent register + heartbeat
- [x] Dashboard أولي + Health + يعمل على :8080 (E2E مختبر)

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
