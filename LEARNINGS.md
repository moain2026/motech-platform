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
