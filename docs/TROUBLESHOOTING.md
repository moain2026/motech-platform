# TROUBLESHOOTING.md — حل المشاكل الشائعة

> صيغة كل مدخل: **المشكلة** + **السبب** + **الحل** + **نصيحة للتجنّب**.

---

## DB-001: لا يمكن الاتصال بـ PostgreSQL محلياً
- **المشكلة:** `connection refused` على `127.0.0.1:5432`.
- **السبب:** الخدمة متوقفة، أو peer auth بدل password.
- **الحل:** `sudo systemctl start postgresql` + استخدم `DATABASE_URL=postgres://motech:...@127.0.0.1:5432/motech_platform?sslmode=disable`.
- **تجنّب:** اتصل دائماً عبر TCP (127.0.0.1) لا عبر socket، ليطابق سلوك الإنتاج.

---

## ENV-001: متغيرات البيئة الأساسية
- القيم المطلوبة: `DATABASE_URL`, `NETBIRD_API_URL`, `NETBIRD_API_TOKEN`, `JWT_SECRET`, `MASTER_KEY`, `PORT`.
- انسخ `backend/.env.example` إلى `backend/.env`.

---

> أضف كل مشكلة جديدة هنا فور حلّها.

---

## EXE-001: SmartScreen warning on first run (متوقّع)
- **المشكلة:** ويندوز يعرض "تمكّن Windows من حماية جهازك / SmartScreen".
- **السبب:** الملف غير موقّع رقمياً (code-signing cert غير مُشترى بعد).
- **الحل:** "مزيد من المعلومات" → "تشغيل على أي حال" (Run anyway).
- **تجنّب:** شراء code-signing certificate (مرحلة M5/الإنتاج).

## EXE-002: GUI لا تظهر بعد التشغيل
- **المشكلة:** بعد فتح motech-connect.exe (GUI) لا تظهر نافذة إدخال الترخيص.
- **الأسباب المحتملة:** رفض UAC (يحتاج admin)، أو فشل walk runtime على بعض إصدارات ويندوز.
- **الحل (موثوق):** استخدم النسخة البديلة **motech-setup.exe** (نافذة console تفاعلية، تسأل الرمز نصياً وتثبّت كل شي). دائماً تعمل.
- **رابط:** https://qfetmfdn.gensparkclaw.com/download/motech-setup.exe

## NB-001: "invalid setup-key ... invalid UUID length: 20" عند netbird up
- **المشكلة:** الـ Agent يفشل في الانضمام لـ NetBird بهذا الخطأ.
- **السبب:** الـ backend كان يرسل للـ Agent معرّف المفتاح المختصر (id, 20 char) بدل المفتاح الكامل (key, 36-char UUID).
- **الحل:** خزّن `setup_key_full` (المفتاح الكامل) في netbird_links وأرسله في /agent/register. (migration 002).
- **تم التحقق:** اختبار حقيقي على VM → netbird up: Connected، ظهر peer بـ IP حقيقي.

## NB-002: التعطيل لا يحذف الـ peer (peer_id = IP)
- **المشكلة:** DeletePeer يحتاج معرّف الـ peer الكائني، لكن الـ Agent يبلّغ الـ IP.
- **الحل:** الـ backend يحلّ IP→peer-id عبر GET /api/peers قبل الحذف. (DeletePeer يقبل IP أو id).
- **تم التحقق:** تعطيل عميل → peer انحذف فعلياً من NetBird ✅.

## SSH-001: الوكيل لا يستطيع SSH رغم أن الجهاز متصل بـ NetBird
- **المشكلة:** ping يعمل لكن port 22 مقفل / "connection refused".
- **السبب:** ويندوز لا يفعّل OpenSSH Server افتراضياً؛ التطبيق كان يضع المفتاح فقط.
- **الحل:** أضيف ensureSSHServer() — يثبّت OpenSSH.Server capability، يشغّل sshd (auto-start)، يفتح Firewall port 22، ويصلح ACL على administrators_authorized_keys. يعمل ضمن SetupAccess.
- **ملاحظة:** الوكيل المتصل يجب أن يكون على جهاز منضمّ لنفس شبكة NetBird + يملك المفتاح الخاص.
