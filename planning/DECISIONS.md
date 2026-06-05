# DECISIONS.md — قرارات معمارية (Architecture Decision Records)

كل قرار تقني مهم مع سببه والبدائل المرفوضة.

---

## ADR-001: لغة الـ Backend = Go
- **القرار:** Go (مع `chi` router + `sqlx`).
- **السبب:** ثنائي واحد بلا اعتماديات، أداء عالٍ، تكامل سهل مع NetBird API (HTTP)، ونفس لغة الـ Agent = انسجام وإعادة استخدام للكود.
- **البدائل المرفوضة:** Node/NestJS (runtime ثقيل)، Python/FastAPI (نشر أعقد على ويندوز للـ agent).

## ADR-002: قاعدة البيانات = PostgreSQL قياسي عبر DATABASE_URL
- **القرار:** PostgreSQL 16، الوصول حصراً عبر `DATABASE_URL`، باستخدام `sqlx` + migrations يدوية (golang-migrate).
- **السبب:** قابلية النقل — الانتقال إلى Supabase/VPS = تغيير `DATABASE_URL` فقط.
- **قيود مفروضة:** ممنوع استخدام أي ميزة خاصة بمنصّة (Supabase Auth/Realtime/SDK). الـ Auth يُكتب يدوياً (JWT في Go).
- **ORM:** اخترنا `sqlx` بدل GORM لشفافية SQL والتحكم الكامل (المستخدم ذكر GORM/sqlx كخيارين مقبولين). [قابل للمراجعة]

## ADR-003: تكامل NetBird عبر NETBIRD_API_URL قابل للتبديل
- **القرار:** كل نداءات NetBird تمرّ عبر client يقرأ `NETBIRD_API_URL` + `NETBIRD_API_TOKEN`.
- **السبب:** الانتقال من Cloud (`https://api.netbird.io`) إلى Self-Hosted = تغيير متغيّر واحد، بدون تعديل كود.
- **استغلال NetBird ACLs/SSH** بدل توزيع مفاتيح SSH يدوياً وتعديل `administrators_authorized_keys` (يقلّل التعقيد والمخاطر على ويندوز).

## ADR-004: الـ Agent = Go، ثنائي ويندوز
- **القرار:** Go مع cross-compile `GOOS=windows GOARCH=amd64`، تسجيل Windows Service عبر `kardianos/service`.
- **السبب:** ثنائي واحد بلا .NET runtime، يُبنى من بيئة Linux مباشرة.
- **البدائل المرفوضة:** C# (.NET runtime)، Rust (منحنى تعلّم أطول، لا حاجة).

## ADR-005: Dashboard = HTML + Tailwind + Alpine.js (يُخدَّم من Backend)
- **القرار:** بناءً على اقتراح المستخدم — لوحة خفيفة تُخدَّم من نفس Go server.
- **السبب:** لا CORS، تطوير أسرع، تنقل مع الـ Backend، كافية للوحة إدارية.
- **البديل المؤجّل:** Next.js لو احتجنا SPA ضخم لاحقاً.

## ADR-006: مصادقة الـ Agent
- **القرار (مبدئي):** Setup Token (استخدام واحد) للتسجيل الأولي → يستبدله الـ Agent بـ JWT طويل العمر / agent token مربوط بـ client_id لكل heartbeat.
- **يُحسم نهائياً في** `docs/SECURITY.md` (mTLS مقابل signed token).

## ADR-007: تخزين المفاتيح الخاصة (SSH التقليدي الثانوي)
- **القرار:** المفتاح الخاص يُولَّد على جهاز العميل، يُرفَع مشفّراً (AES-256-GCM, envelope encryption) ويُخزَّن في DB. master key في متغيّر بيئة فقط.
- **ملاحظة:** المسار الأساسي = NetBird SSH (بلا private keys مخزّنة). هذا للحالات الخاصة فقط.

---
> سجّل أي قرار جديد هنا فور اتخاذه.

## ADR-008: Backend-owned SSH keys + encryption key management (2026-06-05)
- **القرار**: الـbackend يولّد ويملك مفاتيح SSH (مو الوكيل). المفتاح الخاص يُشفّر AES-256-GCM في DB، يُسلَّم للـadmin فقط عبر HTTPS من `/api/clients/:id/private-key`. الوكيل يثبّت المفتاح العام المدفوع فقط.
- **مفتاح التشفير**: `MOTECH_KEY_ENCRYPTION_KEY` (fallback: `MASTER_KEY`). 
  - ⚠️ **مهم**: يجب ضبطه **مرة واحدة قبل إنشاء أي عميل** وعدم تغييره — تغييره يكسر فك تشفير المفاتيح الموجودة. لتدويره مستقبلاً نحتاج migration يفك بالقديم ويعيد التشفير بالجديد (TODO مستقبلي).
- **autostart**: المهمة المجدولة تشتغل كـ**المستخدم التفاعلي** (مو SYSTEM) — بعض الأجهزة تحجب SYSTEM tasks (شوهد على جهاز المستخدم: Last Result 267011، probe بسيط فشل).
