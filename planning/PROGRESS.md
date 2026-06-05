# PROGRESS.md — تقدّم العمل اليومي

## 2026-06-05 (Session 1)

### ✅ تم إنجازه
- تحليل معماري كامل + اختيار الـ stack (Go + Postgres + Tailwind/Alpine + NetBird).
- **البيئة:** تثبيت Go 1.23.4 + PostgreSQL 16.14.
- إنشاء قاعدة البيانات `motech_platform` (user: `motech`).
- هيكل المشروع: `backend/ dashboard/ agent/ docs/ planning/`.
- التوثيق الأساسي: README, DECISIONS, ROADMAP, PROGRESS, TODO, MILESTONES.

### ✅ Backend MVP (Phase 1) — مكتمل ومُختبَر
- هيكل Go: `cmd/server` + `internal/{config,db,models,auth,handlers,netbird}`.
- Migrations (6 جداول) تُطبَّق تلقائياً + seed admin.
- JWT auth (admin + agent، مفصولان) + bcrypt + AES-256-GCM للمفاتيح.
- CRUD العملاء + setup token (مرة واحدة) + activity log + connection info.
- NetBird client قابل للتبديل (mock/live).
- Agent endpoints: register (one-time) + heartbeat (أوامر معلّقة).
- Dashboard أولي (Tailwind+Alpine, RTL، باسم Al-Abbasi Soft): دخول/قائمة/إضافة/نسخ/تدوير/تعطيل/حذف.
- **`go build` + `go vet` نظيفان.** تحقق E2E بـ curl ناجح بالكامل:
  - login → JWT ✓
  - create client → setup token + netbird key ✓
  - list / activity ✓
  - agent register (one-time) ✓ ، reuse → 409 ✓ ، expired logic ✓
  - heartbeat → status=online + pending commands ✓
  - فصل الصلاحيات admin/agent → 401 عند التجاوز ✓

### ✅ NetBird LIVE (Phase 2 — جزئي) — متصل ومُختبَر
- استلمت PAT وخزّنته في `.env` (gitignored).
- الـ health يعرض `netbird_mode: live`.
- تحقق مباشر مع NetBird Cloud API: peers (MoTech 100.95.163.196 ، ai-sandbox 100.95.128.10) ✓، groups ✓.
- **إنشاء عميل من الـ API → يولّد setup key حقيقي في NetBird** (تأكّد بالاستعلام المباشر) ✓.
- تنظيف بيانات الاختبار (DB + NetBird key).
- أضفت `auto_groups` للـ payload (جاهز لربط ACLs لاحقاً).

### 🚧 التالي
- ربط DeletePeer بالـ peer_id الحقيقي بعد انضمام العميل (إكمال Phase 2).
- بناء الـ Agent (.exe) الفعلي (Phase 3). ✅ (انظر أدناه)
### ✅ Agent (.exe) — Phase 3 (جوهر) — مبني ومُختبَر
- Go module في `agent/`، أوامر: register / run / install / uninstall.
- يسجّل بالرمز → يستلم agent JWT + netbird key + يحفظها في ProgramData\Motech\agent.json.
- ينضم لـ NetBird (`netbird up`) + يسجّل خدمة ويندوز (kardianos/service).
- heartbeat كل 20s يطبّق أوامر disable/rotate.
- **`motech-connect.exe` (PE32+ x86-64, 5.7MB) مبني بنجاح** عبر cross-compile.
- اختبار E2E ضد backend حقيقي + NetBird live: register ✓، online ✓، disable→heartbeat reports disabled ✓.

### 🚧 التالي
- Phase 3.1: تنفيذ تدوير المفتاح فعلياً + GUI صغيرة + توقيع exe.
- إكمال Phase 2: ربط peer_id الحقيقي للقطع عبر DeletePeer.


### 📌 قرارات اليوم
- sqlx بدل GORM (شفافية SQL). [ADR-002]
- Dashboard خفيف (Tailwind+Alpine) يُخدَّم من Go. [ADR-005]
- كل تكامل NetBird عبر `NETBIRD_API_URL` قابل للتبديل. [ADR-003]

### ⏭️ التالي
- إكمال Backend Phase 1 ثم تكامل NetBird (Phase 2).

### ✅ Dashboard redesign (M4) — فاخر ومتجاوب
- Sidebar + شعار SVG، stats cards، توزيع الحالة (bars)، activity feed.
- صفحة العملاء: جدول desktop احترافي + **cards على الموبايل** بأزرار أيقونية.
- بحث + فلترة، dark/light mode، toasts، loading states.
- أيقونات SVG (بلا اعتماد على خط emoji)، خط Tajawal، لوحة ألوان brand.
- منشور ومتحقق بصرياً على الدومين.

### ✅ اختبار حقيقي كامل (Self E2E test on VM)
- ثبّت netbird CLI على الـ VM، شغّل الـ agent الحقيقي ضد الـ backend الحقيقي.
- النتيجة: register ✓، **netbird up: Connected** (peer حقيقي 100.95.255.69) ✓، heartbeat يبلّغ peer+key ✓، اللوحة تعرض IP حقيقي ✓، **التعطيل حذف الـ peer فعلياً من NetBird** ✓.
- إصلاحات: المفتاح الكامل UUID (migration 002)، IP نظيف بلا /16، DeletePeer يحلّ IP→peer-id.

## 2026-06-05 (Session 2)

### ✅ تم
- استئناف من الذاكرة بالضبط (memory/2026-06-05.md). لا فقدان تقدّم.
- **حفظ الإصلاح الجوهري في git** (كان غير محفوظ): بقاء الـ agent حياً تحت Windows SYSTEM/Session 0
  — logging لملف فقط (`C:\ProgramData\Motech\agent.log`) + recover() لأمر `run`، وإيقاف أي مهمة/عملية قديمة قبل النسخ للمسار الثابت. (commit `2c7ab14`)
- تحقق: `go build` + cross-compile لـ Windows .exe نظيفان.
- Backend حيّ (systemd, :8080) واللوحة حيّة (200) على https://qfetmfdn.gensparkclaw.com.

### 🔎 تشخيص القضية المفتوحة (هل عملية SYSTEM المجدوَلة تستمر؟)
- نفق Cloudflare من الجلسة السابقة كان لا يزال شغّالاً للحظة (نجح اتصالان SSH: أنا `motech\moain`، ساعة الجهاز 12:10).
- لكن **استعلام عمليات motech-connect رجع فارغاً** → لا توجد عملية agent حيّة الآن ⇒ `last_seen` عالق عند 12:04:29.
- ثم سقط النفق: `websocket: bad handshake` على `ruby-owner-tend-ads.trycloudflare.com`
  ⇒ نفق المستخدم على جهاز MoTech (cloudflared tunnel --url tcp://localhost:22) **توقّف**.

### ⛔ البلوكر الحالي (يحتاج المستخدم)
- لإكمال التحقق طويل الأمد لازم نفق حيّ. المطلوب من المستخدم: إعادة تشغيل على جهاز MoTech:
  `cloudflared tunnel --url tcp://localhost:22` ثم يعطيني الـ URL الجديد.

### ⏭️ عند عودة النفق (خطة التحقق الدقيقة)
1. تنظيف: قتل كل عمليات motech القديمة + حذف العملاء التجريبيين (إبقاء عميل واحد نظيف).
2. تثبيت نظيف واحد عبر motech-setup.exe بـ exe الجديد (commit 2c7ab14).
3. تأكيد: schtasks Last Result=0، الـ PID يبقى حياً 2+ دقيقة، agent.log يُظهر loop مستمر، last_seen يتقدّم تلقائياً تحت SYSTEM.

## 2026-06-05 (Session 2 — continued)

### ✅ Persistence/SYSTEM SOLVED + 401 mystery cracked
- النسخ المنشورة كانت قديمة (قبل إصلاح stdout) → أعدت بناء ونشر الـ3 exe.
- نفّذت 3 تحسينات: (1) Scheduled Task XML مرن (RestartOnFailure PT1M×3, StartWhenAvailable, BootTrigger Delay PT1M, ExecutionTimeLimit PT0S, Hidden, IgnoreNew). (2) Single-Instance Mutex Global\MotechConnectAgent. (3) TokenManager thread-safe (TTL 30s) + reload-on-401 (retry مرة واحدة، skip لو التوكن ما تغيّر، 3×100ms file-lock retry) + 4 unit tests pass.
- **السبب الجذري للـ401**: نسخة قديمة `C:\Users\moain\Downloads\mc.exe` كانت لا تزال شغّالة، تضرب نفس agent.log بتوكن عميل محذوف كل 20s. الوكيل الحقيقي كان سليماً طوال الوقت. قتلت+حذفت mc.exe → الـlog نظيف تماماً (401_count=0).
- **تثبيت نظيف + مراقبة 5 دقائق نجحت**: PID واحد ثابت تحت SYSTEM، last_seen يتقدّم كل دقيقة، صفر أخطاء.

### ✅ Private Key في Dashboard (مكتمل ومُختبَر live)
- Backend Connection endpoint: يفك تشفير المفتاح الخاص (AES-256-GCM) + يرجّع key_file + أمر `ssh -i` كامل + بلوك "ready" جاهز للصق (حفظ المفتاح 0600 → ssh عبر NetBird).
- Dashboard: زر "نسخ الاتصال" يفتح modal: NetBird IP + User + أمر SSH + Private Key + بلوك AI جاهز، كل واحد بزر نسخ + زر "نسخ الكل". كل نسخة تُسجَّل في Activity Log.
- تحقق live بالمتصفح (Alpine via CDP + screenshot): يعمل بالكامل لـ MoTech-v2 (IP 100.95.193.14).

### Commits
2c7ab14, 98bde57, 7d29789, 0402a06, 86ed5d7

## 2026-06-05 (Session 2 — تطوير ذاتي بدون توقف)
قاعدة جديدة من المستخدم: طوّر+اختبر عندي بدقة، وبس لما ينجح تماماً نختبره عنده.

### ✅ (أ) توقيع الـ exe (Authenticode)
- شهادة code-signing ذاتية التوقيع لـ Al-Abbasi Soft (10 سنوات، CodeSigning EKU). المفتاح الخاص gitignored، الشهادة العامة في git للتوزيع عبر GPO.
- `agent/build.sh`: cross-compile الـ3 exe → توقيع (osslsigncode) → نشر. يدعم شهادة CA حقيقية (cert.pfx + MOTECH_PFX_PASS + timestamp) تلقائياً.
- ويندوز يعرض الناشر "Al-Abbasi Soft" بدل "Unknown Publisher". verify=ok عند الثقة بالشهادة. README فيه مستويات الثقة + GPO import + ترقية لـ CA.

### ✅ (ب) تدوير المفاتيح E2E (state machine سليم)
- استبدلت العلَمين القديمين (كانا يُرسلان rotated_ok للأبد) بـ RotateConfirmPending واحد: apply→confirm→ack→clear. idempotent، ما يعيد توليد المفتاح أثناء انتظار الـack، ولا يؤكّد تدوير جديد قبل أوانه.
- **مُختبَر live ضد backend حقيقي**: تدوير أول (عند register) + تدوير ثانٍ (من اللوحة) → دورتي confirm→ack نظيفتين، DB: مفتاح قديم active=f، جديد active=t، المفتاح العام تغيّر فعلاً. + TestRotationStateMachine.

### ✅ (ج) صفحة تثبيت ذكية /setup/{token}
- صفحة عامة self-contained (RTL عربي): اسم الجهاز + زر تحميل + الرمز (مدمج) + خطوات + ملاحظة تحذير ويندوز. تتحقق من الرمز بلا استهلاكه، وتعرض خطأ للرموز المجهولة/المستخدمة/المنتهية.
- CreateClient يرجّع setup_url. اللوحة: modal الإنشاء يعرض الرابط + زر "فتح صفحة التثبيت".
- مُختبَر live (screenshot للصفحة + CDP لتدفق اللوحة).

### Commits: 0c90bdb, 239e6a9, df19e3a (+ docs)
### ⏳ جاهز للاختبار على جهاز المستخدم: الـ3 كلها (exe موقّع منشور، تدوير، صفحة تثبيت)

## 2026-06-05 (Session 2 — اختبار الثلاثة على جهاز المستخدم)

### اختبار 1: توقيع الـ exe ✅ (وأصلحت bug حقيقي)
- أول محاولة: ويندوز عرض Publisher=Al-Abbasi Soft لكن Status=UnknownError (شهادة self-signed بطبقة واحدة → "basic constraint extension has not been observed").
- الإصلاح: بنية سلسلة صحيحة Root CA (CA:TRUE) → leaf code-signing (CA:FALSE). build.sh يضمّن السلسلة كاملة.
- النتيجة على جهاز المستخدم بعد استيراد ca.crt لـ Trusted Root: **Status=Valid، "Signature verified"، Publisher=Al-Abbasi Soft** ✓. توزّع ca.crt عبر GPO على الفروع.

### اختبار 2: تدوير المفاتيح ✅ (واكتشفت أن جهازه كان يشغّل نسخة قديمة)
- أول تدوير فشل: جهازه كان على نسخة 13:42 (قبل إصلاح rotation) → النسخة القديمة ترسل rotated_ok=true دائماً → الـbackend أكّد فوراً وخزّن المفتاح القديم كـ"جديد"، المفتاح ما تغيّر فعلياً.
- نشرت النسخة الجديدة (الموقّعة + rotation الصحيح) على جهازه، أعدت التدوير: **المفتاح تغيّر فعلاً** (AQ7QVTTiDwyl → inXe5UgVX534) في authorized_keys + agent.json، rotate_confirm_pending=False ✓.

### اختبار 3: صفحة التثبيت /setup/{token} ✅
- على جهازه: الصفحة تفتح (200، تعرض اسم الجهاز + الرمز + زر التحميل)، تحميل الـexe الموقّع (sig=Valid)، التسجيل بالرمز → registered → NetBird → task → online.
- الدورة الكاملة من رابط واحد → online نجحت. نظّفت العميل القديم (عميل واحد نظيف: MoTech-setup-e2e online).

### Commits: c6a5cf5 (signing chain fix) + سابقاتها

## 2026-06-05 (Session 2 — Private Key المرحلة 4 على جهاز المستخدم)
- نشرت النموذج backend-owned على جهاز المستخدم عبر MoTech-prod (عميل نظيف، الـbackend ولّد المفتاح).
- **bug حقيقي مكتشف**: جهاز المستخدم **يحجب SYSTEM scheduled tasks** (probe بسيط echo>file كـSYSTEM ما اشتغل، Last Result 267011). مو bug بكودنا.
  - الإصلاح: المهمة المجدولة تشتغل الآن كـ**المستخدم التفاعلي (Moain)** عبر LogonTrigger+BootTrigger + InteractiveToken. العملية تبقى حية (PID ثابت، Session 5). + صلّبت logToFile (io.Discard fallback، ما تلمس stderr أبداً).
- E2E نجح على جهازه: SSH من Azure→MoTech بمفتاح Dashboard (motech\administrator @ MoTech) ✓، تدوير من Dashboard→المفتاح القديم رُفض، الجديد اشتغل ✓، عميل واحد + مفتاح واحد + log نظيف (0 errors).

## 2026-06-05 (Session 2 — نشر على GitHub)
- repo خاص نهائي: github.com/moain2026/motech-platform (origin). كان منشور مؤقتاً على MoTechSys ثم نُقل لحساب moain2026 بتوكنه. توكن moain2026 مكشوف بالشات — يُفضّل إلغاؤه.
- أمان: فحص history كامل — لا PAT/passwords/private keys/tokens. .gitignore صلّب (*.env, *.key, ca.key, *.pem, *.pfx, *.exe, logs, .vscode/.idea). المفاتيح العامة فقط (ca.crt/leaf) في git.
- أضفت: LICENSE (proprietary — Al-Abbasi Soft)، root SECURITY.md، README + Mermaid architecture + Features.
- أزلت ملف bogus متعقّب "C:\ProgramData\..\agent.log" (artifact من اختبار linux).
- push: main + tag v1.0.0 + GitHub Release "v1.0.0 — MVP Complete". 36 commits، 63 files، private.

## 2026-06-05 (Session 3 — استئناف + إصلاح قفل الدخول)
- استأنفت من الذاكرة. تحقق حيّ: backend (db:ok, netbird live), dashboard 200, git نظيف على main، v1.0.0 منشور.
- **شكوى المستخدم "مافي ردة فعل بالأزرار"**: السبب الجذري = قفل دخول، مو bug بالواجهة.
  - فحصت اللوحة عبر CDP: Alpine 3.15.12 شغّال، صفر أخطاء JS، الأزرار كلها تشتغل (clients/add/إلخ مختبرة برمجياً). backend يرد 200 على كل شيء.
  - المستخدم حاول تغيير الإيميل/كلمة المرور من Settings لـ moain2026@gmail.com لكن **التغيير ما حُفظ** (DB ظل admin@motech.local). الأرجح: أدخل "كلمة المرور الحالية" خطأ → backend رفض (401 "كلمة المرور الحالية غير صحيحة") والـ toast ما انتبه له. ثم حاول يدخل بإيميل غير محفوظ → 401 → "مافي ردة فعل".
  - **الإصلاح**: ضبطت admin مباشرة في DB: email=moain2026@gmail.com، password=moain2026@gmail.com (bcrypt cost10 عبر bcrypt الخاص بالمشروع). تحقق: login بالجديد → 200 ✓، بالقديم → 401 ✓.
  - كود UpdateMe + saveProfile سليم (يطلب current_password ويعرض الخطأ كـ toast). لا تغيير كود مطلوب.

### تكملة Session 3 — السبب الجذري لـ"الأزرار ميتة": كاش قديم
- بعد الدخول، شكا المستخدم إن نسخ/إضافة/تعديل ما تستجيب. فحصت عبر CDP على نفس اللوحة:
  - كل الدوال تشتغل (openConn يملأ conn بـ ip/key/ssh، page switch، add modal). صفر أخطاء JS. لا overlay حاجب.
  - الاكتشاف: اللوحة تُخدَّم عبر Cloudflare (`cf-cache-status: DYNAMIC`) وHTML بلا أي Cache-Control → متصفح المستخدم يخدم نسخة index.html قديمة (قبل إصلاح handler) ⇒ الأزرار تبدو ميتة عنده بينما تشتغل على VM (نسخة طازجة).
- **الإصلاح الدائم**: لفّيت FileServer بـ handler يضيف `Cache-Control: no-store/no-cache/must-revalidate` + Pragma + Expires لكل ملف HTML (الـ shell اللي يحمل Alpine + كل الـ handlers). go build OK، أعدت بناء+تشغيل الخدمة، تحققت الهيدر ظاهر.
- يتبقى: المستخدم يعمل hard-refresh مرة واحدة (Ctrl+Shift+R) لجلب النسخة الجديدة.

### تكملة Session 3 — السبب الجذري الحقيقي: Tailwind CDN + modals غير مرئية
- دليل حاسم من المستخدم: **الحذف والتعطيل يشتغلان، الإضافة/التعديل/النسخ لا**.
  - الفرق في الكود: del/disable = `@click` يستدعي دالة فورًا (confirm+fetch، صفر CSS). إضافة/تعديل/نسخ = تضبط متغير حالة لإظهار **modal** عبر `x-show` بكلاسات Tailwind (`fixed inset-0 z-50 flex bg-black/50`).
  - السبب: اللوحة تعتمد `cdn.tailwindcss.com` (يولّد CSS وقت التشغيل). على شبكة/جهاز المستخدم الـ CDN بطيء أو محجوب → كلاسات الـ modal ما تُولَّد → الحالة تتغير (showAdd=true) لكن الـ modal بلا position/size = غير مرئي. del/disable يشتغلان لأنهما لا يحتاجان CSS إطلاقًا. = "الأزرار ميتة" عند المستخدم بينما تشتغل على VM.
- **الإصلاح**: أضفت CSS صريح `.mt-modal` (position:fixed;inset:0;z-index:50;display:flex;background rgba) داخل `<style>` المحلي + علّمت الـ3 modals بالكلاس. الآن تُعرض بغض النظر عن Tailwind CDN. تحقق عبر CDP بعد reload: modal display:flex, fixed, 937x947, visible ✓. (سابقًا أضفت no-cache headers — مفيد لجلب النسخة الجديدة.)
- ملف اللوحة ثابت (FileServer) — التغيير حيّ فورًا، بدون build.
