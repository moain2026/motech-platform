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
