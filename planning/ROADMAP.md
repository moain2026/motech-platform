# ROADMAP.md — خارطة الطريق

## 🎯 الرؤية
نظام مركزي لإدارة الوصول الآمن لأجهزة العملاء عن بُعد، يتوسّع من 3 إلى 1000+ عميل، قابل للنقل بالكامل (DB + NetBird Cloud→Self-hosted).

---

## المراحل (Phases)

### Phase 0 — التهيئة (Environment) ✅
- [x] تثبيت Go 1.23 + PostgreSQL 16
- [x] إنشاء DB `motech_platform` + user `motech`
- [x] هيكل المشروع + التوثيق الأساسي

### Phase 1 — MVP Backend Core 🚧
- [ ] هيكل مشروع Go (cmd/server, internal/...)
- [ ] Migrations + DB schema (clients, setup_tokens, ssh_keys, netbird, activity_log, admins)
- [ ] Config عبر env (`DATABASE_URL`, `NETBIRD_API_URL`, `JWT_SECRET`, ...)
- [ ] JWT auth (login admin) + middleware
- [ ] CRUD العملاء + Setup Token (one-time) + Activity Log
- [ ] Health endpoint

### Phase 2 — NetBird Integration
- [ ] NetBird client (env-switchable URL/token)
- [ ] إضافة عميل → توليد Setup Key + Group/Policy
- [ ] تعطيل/حذف عميل → إبطال peer + سياسة

### Phase 3 — Agent (Windows)
- [ ] Agent CLI: إدخال token → register → heartbeat
- [ ] تثبيت/ربط NetBird تلقائياً
- [ ] تسجيل Windows Service
- [ ] استقبال أوامر التدوير/القطع عبر poll
- [ ] بناء .exe (cross-compile)

### Phase 4 — Dashboard
- [ ] صفحة دخول (JWT)
- [ ] قائمة عملاء + الحالة (online/offline/last seen)
- [ ] إضافة عميل (يولّد token + رابط تنزيل)
- [ ] نسخ معلومات الاتصال
- [ ] تدوير المفتاح / تعطيل
- [ ] سجل العمليات

### Phase 5 — Hardening & Prod
- [ ] تشفير المفاتيح (AES-GCM envelope)
- [ ] HTTPS + توقيع الـ .exe
- [ ] Redis (status/cache) عند الحاجة
- [ ] دليل الانتقال لـ NetBird Self-Hosted + VPS

---

## 🎬 هدف الجلسة الأولى (MVP Demo)
Backend شغّال + DB schema + إضافة عميل (token حقيقي) + Dashboard أساسي يعرض القائمة.
