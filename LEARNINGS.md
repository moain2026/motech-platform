# 🧠 LEARNINGS — Motech Platform

> دروس مُستخلَصة من جلسات العمل. يحدّثه وكيل **Hermes** تلقائياً.
> الصيغة: `[التاريخ] الدرس — السياق — الحل/القاعدة`

---

## مفاتيح التشفير
- `[2026-06-05]` تغيير `MOTECH_KEY_ENCRYPTION_KEY` بعد إنشاء عملاء يكسر فك التشفير → **اضبطه مرة واحدة فقط** قبل أول عميل. (ADR-008)

## NetBird
- `[2026-06-05]` NetBird Cloud free يرفض `protocol:"ssh"` (422) — أقدم من v0.61 → **OpenSSH أساسي، NetBird SSH لاحقاً**.
- `[2026-06-05]` NetBird المدمج يستخدم port 22022 (ليس 22)، JWT auth افتراضياً، يحتاج `--disable-ssh-auth` لمصادقة المفتاح فقط.

## أجهزة العملاء (Windows)
- `[2026-06-05]` بعض الأجهزة تحجب SYSTEM scheduled tasks (Last Result 267011) → **الـ agent يعمل كـ INTERACTIVE USER** عبر LogonTrigger+BootTrigger+InteractiveToken.
- `[2026-06-05]` تصلّب logToFile: io.Discard أولاً، لا تكتب لـ stderr أبداً.

## النشر / التشغيل
- `[2026-06-05]` Caddy يعمل كمستخدم `caddy` ولا يقرأ `/home/work/` → للملفات الثابتة انشر في `/var/www/`، وللخدمات استخدم `reverse_proxy`.

---
_أضف دروساً جديدة أعلى كل قسم. حافظ على الإيجاز._

## الحالة (online/offline) — أُصلح 2026-06-09
- `[2026-06-09]` الحالة كانت تُخزَّن كنص ثابت ولا تنتهي → عميل متوقّف يبقى "online" للأبد. **الإصلاح:** دالة `effectiveStatus(stored, last_seen)` تُحسب وقت العرض (ListClients/GetClient)، عتبة `offlineThreshold=60s` (3 heartbeats مفقودة). pending/disabled لا تتأثر.
- `[2026-06-09]` اختبار E2E على لينكس: ممكن بناء الـ agent لـ linux (`go build ./cmd/agent`) وتشغيله على هذا الـ VM (NetBird متصل أصلاً) — register + heartbeat + rotate حقيقية. تثبيت المفاتيح/الخدمة = no-op على لينكس (آمن). agent.json يُحفظ في مسار ويندوز ثابت حتى على لينكس (hardcoded).

## اختبار SSH حقيقي في بيئة الـVM — 2026-06-09
- `[2026-06-09]` تأكيد E2E حقيقي للـ SSH على الـVM: مفتاح ولّده backend (فُكّ تشفيره من DB) → ركّبته في `~/.ssh/authorized_keys` → SSH نجح فعلياً. التدوير: المفتاح القديم → `Permission denied`، الجديد → نجح. (أثبت توليد المفتاح + التشفير/الفك + التدوير end-to-end).
- `[2026-06-09]` تذبذب: SSH عبر NetBird self-IP (الـVM لنفسه) قد يعمل timeout أحياناً (self-routing). للاختبار المحلي استخدم 127.0.0.1 — نفس sshd ونفس منطق المفاتيح.
