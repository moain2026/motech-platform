# ACTION_PLAN — خطة العمل بناءً على البحث العميق (2026-06-09)

> مصدر هذه الخطة: `planning/DEEP_RESEARCH_REPORT.md` (مراجعة معمارية/أمنية معمقة، 31 مصدراً).
> **الخلاصة الكبرى:** الـ exe لم يكن يكرش — المشكلة كانت بيئة الاختبار (نفق Cloudflare + تداخل ssh→PowerShell→cmd). الكود سليم ومتين. النظام "فائق الجودة، يتفوق على المتوسط". الانتقال 3→1000 يحتاج **تحسين تشغيلي لا إعادة تصميم**.

---

## 🔴 P0 — هذا الأسبوع (يفتح كل شي بعده)

### P0.1 — بيئة اختبار ويندوز حقيقية على الـVM (وقف نفق Cloudflare)
- **الخيار المعتمد:** QEMU/KVM + Windows (مجاني، محلي، snapshots فورية) — أو Azure B2s (~$7/شهر) كبديل سريع.
- داخل ويندوز: فعّل OpenSSH + **اضبط DefaultShell = cmd.exe** (يحل لغز "لا مخرجات").
- النتيجة: `scp exe → ssh run → type out.txt` يلتقط كل شي بنظافة.

### P0.2 — بناء نسختين من الـ agent (يقتل وميض النوافذ نهائياً)
```bash
# نسخة الخدمة (بلا console إطلاقاً):
go build -trimpath -ldflags="-H windowsgui -s -w -buildid=" -o motech-connect.exe ./cmd/agent
# نسخة CLI (للديبَg):
go build -trimpath -ldflags="-s -w -buildid=" -o motech-connect-cli.exe ./cmd/agent
```
- + في `silentCmd`: عيّن `Stdin=nil, Stdout=io.Discard, Stderr=io.Discard` (حماية Session 0).

### P0.3 — إصلاح ACL بالـ SID بدل الأسماء الإنجليزية (⚠️ يكسر على ويندوز عربي)
```powershell
icacls $p /inheritance:r /grant *S-1-5-32-544:F /grant *S-1-5-18:F /setowner *S-1-5-18
```
- احذف شرط `if (Test-Path)` — `icacls` آمن لو الملف غير موجود.

---

## 🟠 P1 — الأسبوع القادم

- **P1.1** التحقق المسبق من NetBird setup key على الباكند قبل إرساله للـ agent (يحل تعليق `netbird up` جذرياً، لا مجرد timeout). + استطلاع `netbird status` بدل انتظار خروج CLI.
- **P1.2** `VerifySSHReady` 7 نقاط: sshd Running + المفتاح + **تجزئة ACL** + قاعدة Firewall + **TCP 127.0.0.1:22** + NetBird peer Connected + heartbeat acked.
- **P1.3** JWT → EdDSA (Ed25519) + `kid` header + جدول تدوير. تثبيت الخوارزمية في Parse (`WithValidMethods`). (W2,W3)
- **P1.4** تشفير: HKDF-SHA256 بـ salt لكل سجل + عمود version (يجعل تدوير المفتاح الرئيسي ممكناً بدل كارثة ADR-008). (W1)

---

## 🟡 P2 — خلال شهر

- **P2.1** توقيع Authenticode عبر **Azure Trusted Signing** (~$10/شهر، لا EV، يقتل SmartScreen). تحقق الهوية يأخذ 1-3 أسابيع → ابدأه مبكراً.
- **P2.2** مثبّت **WiX MSI** + بيان UAC `requireAdministrator` + دليل نشر GPO/Intune.
- **P2.3** Soft-delete للمفاتيح (`revoked_at`، لا حذف) + سلسلة تجزئة لسجل التدقيق (SOC2). (W8)
- **P2.4** **قناة تحديث ذاتي للـ agent** (`/api/agent/update` + sha256 + توقيع + استبدال ذرّي) — أكبر فجوة تشغيلية. (W20)

---

## 🟢 P3 — تحسينات عمق

- `x/sys/windows/svc` مباشر بدل kardianos · التحقق من توقيع مثبّت NetBird (W12) · rate-limit على login/register (W6) · سجلات منظمة slog/JSON (W7) · Entra/Google OIDC SSO للأدمن · device posture في heartbeat.

---

## ملاحظات نقاط القوة (لا تُغيّر)
AES-256-GCM envelope · TokenManager (TTL+ForceReload+retry) · single-instance mutex · scheduled task XML (Boot/Logon/RestartOnFailure) · logToFile io.Discard أولاً · نموذج المفتاح المملوك للخادم · rotation_test.go E2E.

## قائمة الثغرات الكاملة W1–W20
انظر `planning/DEEP_RESEARCH_REPORT.md` القسم (د) للتفاصيل بـ file:line.
