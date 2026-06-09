# 🚀 HANDOFF — Motech Platform

> **ملف التسليم الموحّد.** أي جلسة/حساب/بيئة جديدة تبدأ من هنا. اقرأ هذا الملف أولاً، ثم اتبع الروابط.
> آخر تحديث: 2026-06-09

---

## 1. ما هو المشروع (What)
**نظام إدارة الوصول الآمن عن بُعد** — Secure Remote Access Management System.
إدارة مركزية للوصول عبر SSH (فوق شبكة NetBird mesh VPN) إلى أجهزة عملاء/فروع Windows عن بُعد. مصمّم للتوسّع من 3 عملاء إلى 1000+.

**المكوّنات الثلاثة:**
1. **Dashboard** — واجهة أدمن ويب (Go + Tailwind + Alpine, RTL, باسم Al-Abbasi Soft): قائمة العملاء، حالة online/offline، إضافة عميل (يولّد setup token + installer)، إدارة مفاتيح SSH + تدويرها، تعطيل/حذف، نسخ معلومات الاتصال، سجل النشاط.
2. **Backend + DB** — Go + PostgreSQL: تخزين العملاء + مفاتيح SSH مشفّرة، تكامل NetBird، استقبال heartbeats، توزيع المفاتيح المُدوّرة، JWT auth.
3. **Agent** — ملف `motech-connect.exe` صغير لويندوز: إعداد لمرة واحدة عبر activation token → تثبيت NetBird → مفتاح SSH فريد → إعداد firewall → تسجيل كخدمة ويندوز → heartbeat → تطبيق تدوير/إلغاء المفاتيح تلقائياً.

---

## 2. الحالة الحالية (Status) — v1.0.0
- ✅ Backend MVP مكتمل ومُختبَر (JWT, AES-256-GCM, CRUD, setup tokens, activity log).
- ✅ NetBird LIVE متصل (Cloud free plan).
- ✅ Agent `.exe` مبني ومُختبَر E2E (register / heartbeat / disable / rotate).
- ✅ Dashboard معاد التصميم (M4).
- ✅ منشور **v1.0.0** على GitHub (private).
- 🚧 التالي: تنفيذ تدوير المفتاح كاملاً + GUI صغيرة للـ agent + توقيع exe + ربط peer_id الحقيقي للقطع.

تفاصيل دقيقة ومحدّثة → [`planning/PROGRESS.md`](planning/PROGRESS.md)

---

## 3. كيف تشغّله (Run)
```bash
# Backend (Go) — يستمع على :8080
cd backend && go run ./cmd/server
# أو عبر systemd (الإنتاج):
#   systemctl --user status motech-backend   (إن وُجد)

# Dashboard: يُخدّم من Go نفسه → https://qfetmfdn.gensparkclaw.com (عبر Caddy → :8080)

# Agent (cross-compile لويندوز):
cd agent && GOOS=windows GOARCH=amd64 go build -o motech-connect.exe ./cmd/agent
```
تفاصيل النشر → [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md) · الإعداد → [`docs/SETUP.md`](docs/SETUP.md)

---

## 4. الأسرار والمفاتيح (Secrets) — أين، لا تُكشف
- `backend/.env` (**gitignored**): `DATABASE_URL`, `JWT_SECRET`, `MOTECH_KEY_ENCRYPTION_KEY`, `NETBIRD_API_URL`, `NETBIRD_PAT`.
- ⚠️ **`MOTECH_KEY_ENCRYPTION_KEY` يُضبط مرة واحدة فقط** قبل إنشاء أي عميل — تغييره يكسر فك تشفير المفاتيح الموجودة (ADR-008).
- NetBird PAT: من حساب NetBird Cloud (free, 5 users).
- مخزن git credentials يحمل توكن GitHub (المالك: حساب `moain2026`).

---

## 5. الروابط (Links)
| العنصر | المكان |
|---|---|
| GitHub repo | `github.com/moain2026/motech-platform` (private) |
| Dashboard مباشر | https://qfetmfdn.gensparkclaw.com |
| منفذ Backend المحلي | `127.0.0.1:8080` (عبر Caddy) |
| مسار المشروع | `~/.openclaw/workspace/motech-platform/` |

---

## 6. خريطة التوثيق (Docs Map)
**`docs/` — التوثيق التقني المرجعي:**
- `ARCHITECTURE.md` · `API.md` · `DATABASE.md` · `NETBIRD.md` · `SECURITY.md` · `AGENT.md` · `SETUP.md` · `DEPLOYMENT.md` · `TROUBLESHOOTING.md` · `CHANGELOG.md`

**`planning/` — التخطيط والتتبّع:**
- `PROGRESS.md` (يومي) · `DECISIONS.md` (ADR log) · `ROADMAP.md` · `MILESTONES.md` · `TODO.md` · `RESEARCH_NEEDED.md`

**`LEARNINGS.md`** — الدروس المُستخلَصة ذاتياً (يحدّثه وكيل Hermes). انظر القسم 7.

---

## 7. التعلّم الذاتي (Hermes)
وكيل **Hermes** مسؤول عن استخلاص الدروس/القرارات من جلسات العمل وكتابتها في:
- `LEARNINGS.md` (هذا المشروع) — أخطاء وحلول وأنماط متكررة.
- تحديث `planning/PROGRESS.md` و `planning/DECISIONS.md` عند نقاط القرار.

نطاق Hermes: **توثيق وذاكرة فقط** — لا يغيّر كوداً ولا إعدادات نظام دون طلب صريح.

---

## 8. قواعد ذهبية (Gotchas)
- لا تغيّر `MOTECH_KEY_ENCRYPTION_KEY` بعد إنشاء عملاء.
- NetBird Cloud free لا يدعم `protocol:"ssh"` → نستخدم OpenSSH كأساس، NetBird SSH تحسين مستقبلي.
- جهاز العميل قد يحجب SYSTEM scheduled tasks → الـ agent يعمل كـ INTERACTIVE USER.
- مجلدات `C:\ProgramData` التي تظهر داخل الريبو = لوقات اختبار ويندوز كُتبت بالخطأ على لينكس — آمنة للحذف.

---

## 🔄 تحديث 2026-06-09 (مهم — اقرأه عند استئناف العمل)

### المعمارية الحالية: NetBird = مصدر الحقيقة (ADR-A5)
- **الوصول الأساسي = NetBird SSH المدمج** (مو OpenSSH+مفتاح). الأمر: `netbird ssh --strict-host-key-checking=false --user <user> <netbird-ip> '<powershell>'`.
- التثبيت يفعّله: `netbird up --allow-server-ssh --disable-ssh-auth`. الباكند يفعّل ssh_enabled تلقائياً + يرسل apply_ssh ليعيد الـ agent الـ up (bug 2816).
- **التعطيل/الحذف** = يحذف الـ peer من NetBird (يقطع الوصول فعلياً). تطبيق Motech = طبقة وصول/عرض/توزيع مفاتيح.

### الحالة الحالية (مُختبَر حقيقياً على ويندوز)
- ✅ التثبيت بنقرة (Alabbasi_soft.exe، GUI) → register → NetBird + SSH المدمج تلقائي → online بالداشبورد.
- ✅ AI agent يدخل أي جهاز عميل وينفّذ أوامر عبر `netbird ssh`. مُثبَت على جهازين مختلفين.
- ✅ بيئة اختبار ويندوز: GitHub Actions (`gh workflow run win-e2e.yml -f mode=full -f setup_token=...`).

### كيف تختبر (من بيئة لينكس بلا ويندوز)
1. أنشئ عميل عبر API → setup_token.
2. شغّل: `gh workflow run win-e2e.yml -R moain2026/motech-platform -f mode=full -f setup_token=<TOK>`.
3. `gh run download <id>` → اقرأ register-out.txt + netbird-diag.txt + agent.log.

### مفاتيح/أسرار
- NetBird PAT في backend/.env (NETBIRD_API_TOKEN). admin الداشبورد: moain2026@gmail.com.
- مفتاح SSH خاص بي (للوصول التقليدي الاحتياطي): ~/.ssh/motech_claw_key على VM.

### Migrations: 003 login_user, 004 ssh_enabled, 005 ssh_applied.
### تفاصيل كاملة: planning/PROGRESS.md (2026-06-09) + Memory_agent-one/ في الـ workspace.
