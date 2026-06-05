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

### ✅ NetBird LIVE (Phase 2 — جزئي) — متصل ومُختبَر
- استلمت PAT وخزّنته في `.env` (gitignored).
- الـ health يعرض `netbird_mode: live`.
- تحقق مباشر مع NetBird Cloud API: peers (MoTech 100.95.163.196 ، ai-sandbox 100.95.128.10) ✓، groups ✓.
- **إنشاء عميل من الـ API → يولّد setup key حقيقي في NetBird** (تأكّد بالاستعلام المباشر) ✓.
- تنظيف بيانات الاختبار (DB + NetBird key).
- أضفت `auto_groups` للـ payload (جاهز لربط ACLs لاحقاً).

### 🚧 التالي
- ربط DeletePeer بالـ peer_id الحقيقي بعد انضمام العميل (إكمال Phase 2).
- بناء الـ Agent (.exe) الفعلي (Phase 3).

### 📌 قرارات اليوم
- sqlx بدل GORM (شفافية SQL). [ADR-002]
- Dashboard خفيف (Tailwind+Alpine) يُخدَّم من Go. [ADR-005]
- كل تكامل NetBird عبر `NETBIRD_API_URL` قابل للتبديل. [ADR-003]

### ⏭️ التالي
- إكمال Backend Phase 1 ثم تكامل NetBird (Phase 2).
