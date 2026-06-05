# SECURITY.md — الأمان والتشفير

## المصادقة (Authentication)
- **Admin:** email + bcrypt password → JWT (HS256, 12h). كلمات السر تُخزَّن bcrypt فقط.
- **Agent:** يُصدَر JWT (kind=`agent`, سنة) عند التسجيل بنجاح. middleware يفصل بين `admin` و`agent` فلا يستطيع أحدهما استخدام endpoints الآخر.
- `JWT_SECRET` من البيئة. **غيّره في الإنتاج** لقيمة عشوائية طويلة.

## رموز التفعيل (Setup Tokens) — استخدام واحد
- تُولَّد عشوائياً (18 bytes)، تُعرض للأدمن **مرة واحدة**.
- تُخزَّن **hash (SHA-256) فقط** — لا يمكن استرجاع النص من DB.
- `used_at` يُضبط عند أول استخدام → إعادة الاستخدام تُرفض (409).
- `expires_at` افتراضي 24 ساعة → بعدها 410.

## تشفير المفاتيح الخاصة (SSH التقليدي الثانوي)
- المفتاح الخاص يُولَّد **على جهاز العميل** (لا يغادر بنص).
- لو خُزِّن في DB: **AES-256-GCM** (envelope) عبر `auth.Encrypt/Decrypt`.
- المفتاح الرئيسي من `MASTER_KEY` (env فقط، خارج git/DB). يُشتق منه مفتاح AES بـ SHA-256.
- المسار الأساسي = **NetBird SSH/ACL** → لا حاجة لتخزين private keys أصلاً.

## النقل والشبكة
- **HTTPS إلزامي في الإنتاج** (الآن Caddy على الدومين العام، أو خلف reverse proxy).
- وصول الـ Agent ↔ Backend عبر شبكة NetBird mesh + JWT.
- القطع الفوري عبر NetBird API (حذف peer) — لا ينتظر heartbeat.

## سجل التدقيق
كل عملية حسّاسة (إنشاء/تدوير/تعطيل/حذف/نسخ اتصال/تسجيل agent) تُكتب في `activity_log` مع الفاعل والوقت.

## قائمة تصلّب الإنتاج (Hardening checklist)
- [ ] غيّر `JWT_SECRET` و`MASTER_KEY` لقيم عشوائية قوية.
- [ ] غيّر كلمة سر الأدمن الافتراضية فوراً.
- [ ] فعّل HTTPS فقط (HSTS).
- [ ] خزّن `NETBIRD_API_TOKEN` في secret manager.
- [ ] حدود معدّل (rate limiting) على `/api/auth/login` و`/api/agent/register`.
- [ ] توقيع ملف الـ .exe (code-signing) لتفادي تحذير SmartScreen.
