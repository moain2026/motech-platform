# AGENT.md — وكيل العميل (motech-connect.exe)

ملف Go واحد، cross-compiled لويندوز (PE32+, ~5.7MB، بلا اعتماديات runtime).

## الأوامر
```
motech-connect register --token <TOKEN> [--server <URL>]   # التسجيل الأولي
motech-connect run                                         # حلقة الـ heartbeat (تستخدمها الخدمة)
motech-connect install / uninstall                         # إدارة خدمة ويندوز
```
المتغير `MOTECH_SERVER` يضبط الـ backend URL افتراضياً.

## التدفّق عند التثبيت (مرة واحدة)
1. العميل يشغّل `motech-connect register --token ABCD-1234-XYZ`.
2. الوكيل يستبدل الرمز (مرة واحدة) → يستلم: `agent_token` (JWT سنة) + `netbird_setupkey` + `netbird_api_url` + `heartbeat_secs`.
3. يحفظها في `C:\ProgramData\Motech\agent.json` (0600).
4. ينضمّ لشبكة NetBird: `netbird up --setup-key <key> --management-url <url>`.
5. يسجّل نفسه كخدمة ويندوز `MotechConnect` (تعمل دائماً، auto-start).

## الخلفية (الخدمة)
- ترسل `POST /api/agent/heartbeat` كل `heartbeat_secs` (افتراضي 20s).
- الرد `{disabled, rotate}`:
  - `disabled:true` → `netbird down` (قطع الوصول فوراً).
  - `rotate:true` → تطبيق تدوير المفتاح (Phase 3.1).

## الحالة المحفوظة (agent.json)
| الحقل | الوصف |
|------|-------|
| `agent_token` | JWT للمصادقة في الـ heartbeat |
| `netbird_setupkey` | مفتاح الانضمام للشبكة |
| `netbird_api_url` | عنوان NetBird (cloud/self-hosted) |
| `heartbeat_secs` | فترة الـ heartbeat |

## البناء
```bash
cd agent
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o motech-connect.exe ./cmd/agent
```

## ملاحظات اختبار
- اختُبر منطق التسجيل + heartbeat + أمر التعطيل **E2E ضد الـ backend الحقيقي + NetBird live** (من بيئة Linux؛ عمليات NetBird/الخدمة تُسجَّل بدل التنفيذ عند غياب الصلاحيات/الـ CLI).
- **الاختبار النهائي يتم على جهاز ويندوز حقيقي** (تثبيت netbird CLI + صلاحيات admin).

## TODO (Phase 3.1)
- [ ] تطبيق تدوير المفتاح فعلياً (تحديث `administrators_authorized_keys` أو الاعتماد على NetBird SSH).
- [ ] واجهة GUI صغيرة لإدخال الرمز (بدل CLI) لتجربة عميل أسهل.
- [ ] توقيع الـ .exe (code-signing) لتفادي SmartScreen.

---

## 🖥️ التطبيق الرسومي (GUI) — تجربة العميل النهائية

`motech-connect.exe` الآن **تطبيق نافذة** (Windows GUI, native Win32 عبر walk، بمانيفست يطلب صلاحيات admin + ثيم حديث). تجربة العميل:

1. يحمّل من: **https://qfetmfdn.gensparkclaw.com/download/motech-connect.exe**
2. يفتح الملف → نافذة "Al-Abbasi Soft — تفعيل الاتصال الآمن".
3. يلصق **مفتاح الترخيص** (رمز التفعيل) → يضغط **🔒 ابدأ التفعيل**.
4. التطبيق تلقائياً: يتحقق من الترخيص → ينضم NetBird → يجهّز SSH (يولّد مفتاح + authorized_keys) → يثبّت خدمة ويندوز → يزامن مع اللوحة → يظهر 🟢 متصل.
5. سجل العملية يظهر مباشرة في النافذة، وكل خطوة لها ✓.

نسختان منشورتان:
- `motech-connect.exe` — **GUI** (للعميل النهائي، نقرة واحدة).
- `motech-connect-cli.exe` — CLI (للاستخدام المتقدم / تشغيل الخدمة عبر `run`).

البناء (cross-compile من Linux):
```bash
# GUI (CGO + mingw)
CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 \
  go build -ldflags "-H windowsgui -s -w" -o motech-connect.exe ./cmd/gui
# CLI
GOOS=windows GOARCH=amd64 go build -o motech-connect-cli.exe ./cmd/agent
```

> ⚠️ لم يُوقَّع الملف بعد (code-signing) — لذلك ويندوز SmartScreen قد يحذّر أول مرة ("More info → Run anyway"). التوقيع يحتاج شهادة مدفوعة (M5).
