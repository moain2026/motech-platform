# 🔍 تحليل NetBird — طرق الاتصال والتحكم المتاحة (2026-06-09)

> استكشاف فعلي عبر API (توكن صالح) + الوثائق الرسمية. الحساب: NetBird Cloud, الأجهزة v0.71.4.

## ما هو متاح في API (مُختبَر، كله HTTP 200)
| Endpoint | الاستخدام |
|---|---|
| `/peers` | الأجهزة (+ ssh_enabled, local_flags, dns_label, geo, version) |
| `/groups` | تجميع الأجهزة (للـ ACLs) |
| `/setup-keys` | مفاتيح الانضمام (reusable/one-off/ephemeral) — 13 موجود |
| `/policies` | قواعد الوصول (ACL): بروتوكول/منافذ/مصدر/وجهة |
| `/routes` | توجيه subnet (Site-to-Site, Exit Nodes) |
| `/networks` | ⭐ شبكات VPN-to-Site (الجديد) |
| `/dns/*` | DNS داخلي + nameservers |
| `/posture-checks` | فحص وضعية الجهاز (OS version, AV, location) |
| `/events/audit` | سجل تدقيق كامل (219 حدث) — للأمان/SOC |
| `/users`, `/accounts` | إدارة المستخدمين والحساب |

## طرق الاتصال/الوصول (مرتّبة)

### 1. ⭐ NetBird SSH المدمج (نستخدمه الآن — الأفضل)
- `netbird up --allow-server-ssh [--disable-ssh-auth]` + ssh_enabled في dashboard.
- SSH على port 22 → يُعاد توجيهه داخلياً لـ 22022. بلا OpenSSH/ACL/firewall/بورت مكشوف.
- مصادقة: JWT(OIDC) أو machine-identity (ACLs). يرفض المستخدمين المميّزين افتراضياً.

### 2. ⭐⭐ Networks (VPN-to-Site) — اكتشاف قوي لم نستغله
- **توصل شبكة فرع كاملة بجهاز routing واحد فقط** — بدون تثبيت agent على كل جهاز!
- مثالي لمنتجك: بدل تثبيت Alabbasi على كل كمبيوتر في الفرع، ثبّته على جهاز واحد (routing peer) → توصل كل أجهزة الفرع (LAN).
- routing peer + network resource policy. عبر `/networks` API.

### 3. Network Routes (Site-to-Site + Exit Nodes)
- ربط شبكتين كاملتين، أو توجيه كل الإنترنت عبر جهاز معيّن (exit node).

### 4. Reverse Proxy (ميزة في القائمة)
- كشف خدمة داخلية (HTTP/TCP) عبر المش بدون فتح بورت. (يحتاج بحث أعمق).

### 5. الوصول التقليدي (احتياطي)
- OpenSSH + المفتاح الخاص (يعمل لكن فيه مشكلة ACL على ويندوز — نتجنّبه).

## قدرات تحكم/أمان إضافية (للمستقبل)
- **Posture Checks**: امنع وصول جهاز لو OS قديم / بلا AV / من بلد غير مصرّح (zero-trust). `/posture-checks`.
- **Audit Events**: سجل كامل لكل دخول/تغيير (`/events/audit`) — جاهز لـ SOC2/تدقيق.
- **Groups + Policies**: تحكّم دقيق (أي مستخدم/جهاز يوصل أي جهاز، أي بروتوكول/منفذ).
- **DNS داخلي**: كل جهاز له `dns_label` (مثل `motech.netbird.cloud`) → اتصل بالاسم بدل الـ IP.
- **Ephemeral peers**: أجهزة مؤقتة تُحذف تلقائياً بعد فترة (للـ CI/الاختبار).

## التوصية لمنتج Motech
1. **أبقِ NetBird SSH المدمج** للأجهزة الفردية (يعمل).
2. **استغل Networks** للفروع: جهاز routing واحد يكشف LAN كامل → توسّع أسرع وأرخص (agent واحد بدل 50).
3. **أضف Posture Checks** لاحقاً (zero-trust: امنع الأجهزة غير الآمنة).
4. **اربط Audit Events** بسجل النشاط في الداشبورد (تدقيق موحّد).
5. **استخدم dns_label** في معلومات الاتصال (اسم ثابت بدل IP متغيّر).
