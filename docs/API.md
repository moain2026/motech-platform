# API.md — توثيق واجهة الـ Backend

Base URL (dev): `http://127.0.0.1:8080`
كل الردود JSON. المصادقة عبر `Authorization: Bearer <JWT>`.

نوعان من التوكنات:
- **admin** — للـ Dashboard (صلاحية 12 ساعة).
- **agent** — لأجهزة العملاء (صلاحية سنة، يُصدر عند التسجيل).

---

## Health
```
GET /health
```
```json
{"status":"ok","db":"ok","netbird_mode":"mock"}
```

## تسجيل دخول الأدمن
```
POST /api/auth/login
{"email":"admin@motech.local","password":"admin123"}
→ {"token":"<jwt>","email":"admin@motech.local"}
```
```bash
curl -s -X POST $URL/api/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"admin@motech.local","password":"admin123"}'
```

## العملاء (admin)
| Method | Path | الوصف |
|--------|------|-------|
| GET | `/api/clients` | قائمة كل العملاء (الأحدث أولاً) |
| POST | `/api/clients` | إنشاء عميل + setup token + netbird key |
| GET | `/api/clients/{id}` | تفاصيل عميل |
| GET | `/api/clients/{id}/connection` | معلومات الاتصال (نسخ للوكلاء) |
| POST | `/api/clients/{id}/rotate-key` | طلب تدوير مفتاح SSH |
| POST | `/api/clients/{id}/disable` | تعطيل + قطع NetBird |
| DELETE | `/api/clients/{id}` | حذف نهائي |
| GET | `/api/activity` | آخر 100 عملية في السجل |

### إنشاء عميل — مثال
```bash
curl -s -X POST $URL/api/clients -H "Authorization: Bearer $T" \
  -H 'Content-Type: application/json' \
  -d '{"name":"فرع الرياض","branch":"الرياض","contact_name":"معين","contact_phone":"0500000000"}'
```
```json
{
  "id":"7d643348-...",
  "name":"فرع الرياض",
  "setup_token":"3e12-0b4a-5e2d",          // يظهر مرة واحدة فقط
  "netbird_setup":"MOCK-SETUP-KEY-...",
  "netbird_mode":"mock",
  "installer":"motech-connect.exe"
}
```

## الـ Agent
```
POST /api/agent/register      (لا يحتاج توكن — يستهلك setup token)
{"token":"3e12-0b4a-5e2d"}
→ {"agent_token":"<jwt>","netbird_setupkey":"...","netbird_api_url":"...","heartbeat_secs":20}
```
- الرمز **يُستخدم مرة واحدة** → إعادة الاستخدام تُرجع `409`.
- منتهي الصلاحية → `410`.

```
POST /api/agent/heartbeat     (Bearer agent token)
→ {"disabled":false,"rotate":true}
```
- يحدّث `last_seen` + `status=online`، ويُرجع الأوامر المعلّقة (تدوير/تعطيل).

## رموز الحالة
`200` نجاح · `201` أُنشئ · `400` طلب خاطئ · `401` غير مصرّح · `404` غير موجود · `409` رمز مستخدم · `410` رمز منتهٍ · `502` خطأ NetBird.
