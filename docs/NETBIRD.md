# NETBIRD.md — تكامل NetBird

## المبدأ: URL قابل للتبديل
كل نداءات NetBird تمرّ عبر `internal/netbird` الذي يقرأ:
- `NETBIRD_API_URL` — افتراضي `https://api.netbird.io` (Cloud).
- `NETBIRD_API_TOKEN` — Personal Access Token (PAT).

**الانتقال إلى Self-Hosted = تغيير `NETBIRD_API_URL` فقط** (مثلاً `https://netbird.example.com`) + توكن جديد. لا تعديل كود.

## وضع Mock (للتطوير بلا توكن)
لو `NETBIRD_API_TOKEN` فارغ → الـ client يعمل في **MOCK mode**: يُرجع setup keys وهمية مُعلّمة بـ `MOCK-...` كي يعمل النظام كامل أثناء التطوير. الـ `/health` يُظهر `netbird_mode: mock`.

## دورة حياة العميل ↔ NetBird
| حدث في النظام | إجراء NetBird |
|----------------|----------------|
| إضافة عميل | `CreateSetupKey()` (one-off, usage_limit=1, 24h) → يُحفظ في `netbird_links` |
| تسجيل الـ Agent | يُسلَّم الـ setup key للـ agent → ينضم للشبكة |
| تعطيل/حذف عميل | `DeletePeer(peer_id)` → قطع فوري |
| تدوير المفتاح | حالياً على مستوى SSH؛ مستقبلاً سياسات ACL |

## كيف تحصل على PAT
لوحة NetBird → **Settings → Personal Access Tokens → Create**. الصقه في `backend/.env` كـ `NETBIRD_API_TOKEN=...` ثم أعد تشغيل الـ Backend. (الـ `.env` خارج git).

## ملاحظات API
- المصادقة: ترويسة `Authorization: Token <PAT>`.
- Endpoints مستخدمة: `POST /api/setup-keys`, `DELETE /api/peers/{id}`.
- التوسعة القادمة: إدارة Groups/Policies (ACL) لربط الوصول بالمجموعات بدل المفاتيح.

## Self-Hosted لاحقاً
عند نقل NetBird لسيرفرك:
1. انشر NetBird (Management + Signal + Coturn) على الـ VPS.
2. أنشئ PAT جديد من لوحتك المستضافة.
3. `NETBIRD_API_URL=https://<your-netbird>` + `NETBIRD_API_TOKEN=<new>` في `.env`.
4. أعد التشغيل. تمّ.
