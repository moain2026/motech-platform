# PROGRESS.md — تقدّم العمل اليومي

## 2026-06-05 (Session 1)

### ✅ تم إنجازه
- تحليل معماري كامل + اختيار الـ stack (Go + Postgres + Tailwind/Alpine + NetBird).
- **البيئة:** تثبيت Go 1.23.4 + PostgreSQL 16.14.
- إنشاء قاعدة البيانات `motech_platform` (user: `motech`).
- هيكل المشروع: `backend/ dashboard/ agent/ docs/ planning/`.
- التوثيق الأساسي: README, DECISIONS, ROADMAP, PROGRESS, TODO, MILESTONES.

### ✅ Backend MVP (Phase 1) — مكتمل ومُختبَر
- هيكل Go: `cmd/server` + `internal/{config,db,models,auth,handlers,netbird}`.
- Migrations (6 جداول) تُطبَّق تلقائياً + seed admin.
- JWT auth (admin + agent، مفصولان) + bcrypt + AES-256-GCM للمفاتيح.
- CRUD العملاء + setup token (مرة واحدة) + activity log + connection info.
- NetBird client قابل للتبديل (mock/live).
- Agent endpoints: register (one-time) + heartbeat (أوامر معلّقة).
- Dashboard أولي (Tailwind+Alpine, RTL، باسم Al-Abbasi Soft): دخول/قائمة/إضافة/نسخ/تدوير/تعطيل/حذف.
- **`go build` + `go vet` نظيفان.** تحقق E2E بـ curl ناجح بالكامل:
  - login → JWT ✓
  - create client → setup token + netbird key ✓
  - list / activity ✓
  - agent register (one-time) ✓ ، reuse → 409 ✓ ، expired logic ✓
  - heartbeat → status=online + pending commands ✓
  - فصل الصلاحيات admin/agent → 401 عند التجاوز ✓

### 🚧 التالي
- تكامل NetBird الحقيقي (Phase 2) عند استلام PAT.
- بناء الـ Agent (.exe) الفعلي (Phase 3).

### 📌 قرارات اليوم
- sqlx بدل GORM (شفافية SQL). [ADR-002]
- Dashboard خفيف (Tailwind+Alpine) يُخدَّم من Go. [ADR-005]
- كل تكامل NetBird عبر `NETBIRD_API_URL` قابل للتبديل. [ADR-003]

### ⏭️ التالي
- إكمال Backend Phase 1 ثم تكامل NetBird (Phase 2).
