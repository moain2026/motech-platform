# 🔬 RESEARCH NEEDED — مسائل تحتاج بحث عميق

التاريخ: 2026-06-05 | المشروع: Motech Platform (Al-Abbasi Soft)
البيئة: عميل Windows 10 Pro، NetBird 0.71.4، الوكيل على Azure VM (Ubuntu) منضمّ لنفس شبكة NetBird.

---

## ✅ ما يعمل بشكل مؤكّد (مُختبَر فعلياً)
- اللوحة + Backend (Go) + PostgreSQL + توليد رموز التفعيل + NetBird Cloud API.
- التطبيق (motech-connect.exe): register → ينضم NetBird → يثبّت OpenSSH → يزامن مع اللوحة → يظهر online.
- **الاتصال SSH الفعلي نجح**: الوكيل اتصل بجهاز العميل وأنشأ مجلد/ملف على سطح المكتب.
- heartbeat loop يعمل عند تشغيل التطبيق **كمستخدم عادي** (log: `loop start: server=... hasToken=true`).

---

## ❌ المسألة 1: التشغيل الدائم عند الإقلاع (الأهم)
**العَرَض:**
- خدمة Windows (kardianos/service) تُثبَّت لكن تبقى `Stopped` + خطأ "did not respond to start in a timely fashion".
- بدّلنا إلى **Scheduled Task** (run as SYSTEM, ONSTART). الـ task يُسجَّل ويُشغّل العملية، لكن:
  - **as SYSTEM**: العملية تبدأ وتموت فوراً (لا تستمر، agent.log فارغ).
  - **as user (Moain)**: العملية تستمر وتعمل heartbeat بنجاح (agent.log يُكتب).

**المطلوب بحثه:**
1. ليش عملية Go (تستخدم netbird CLI + HTTP) تموت فوراً عند التشغيل as SYSTEM عبر Task Scheduler على Windows 10؟
2. هل المشكلة في وصول SYSTEM لـ netbird socket / الشبكة / ملفات ProgramData؟
3. أفضل ممارسة لتشغيل agent دائم على Windows: Service (NSSM؟ winsw؟) أم Scheduled Task (ONLOGON as user أم ONSTART as SYSTEM)؟
4. هل نشغّل الـ task كـ **المستخدم المسجّل** بدل SYSTEM (يحل المشكلة لكن يحتاج المستخدم يسجّل دخول)؟

**الحل المؤقت الممكن:** Scheduled Task بـ `/SC ONLOGON /RU <user>` (يشتغل عند دخول المستخدم).

---

## ❌ المسألة 2: NetBird SSH المدمج (اكتشاف مهم — قد يلغي OpenSSH كله)
**الاكتشاف:** NetBird v0.61+ (ونسختنا 0.71.4) فيه **خادم SSH مدمج** كامل:
```
netbird up --allow-server-ssh --enable-ssh-root
```
ومن لوحة NetBird (الصور): يحتاج **SSH Access Policy** صريحة + زر "Confirm & Enable".

**ما جرّبناه:**
- `netbird up --setup-key X --allow-server-ssh` → اتصل، لكن `netbird status` يعرض `sshServer.enabled = false`.
- في حساب NetBird فيه policy واحدة `Default` (protocol=all) — لكن SSH ظل معطّل.

**المطلوب بحثه:**
1. كيف نفعّل NetBird SSH server برمجياً عبر **NetBird REST API** (ليش `--allow-server-ssh` ما كفى)؟
2. ما هي بنية **SSH Access Policy** المطلوبة في NetBird API (v0.61+) للسماح بـ SSH لـ peer معيّن؟ (endpoint، payload).
3. هل `--allow-server-ssh` يحتاج إعداد إضافي في الـ management (flag على الـ account/network)؟
4. كيف يصادق الوكيل عند الاتصال بـ NetBird SSH (JWT? مفتاح؟) — أمر `netbird ssh <peer>`؟
5. مقارنة: NetBird SSH المدمج مقابل OpenSSH التقليدي — أيهما أنسب لسيناريو "إعطاء وصول لوكيل AI"؟

**لماذا مهم:** لو NetBird SSH يشتغل، نلغي: تثبيت OpenSSH + إدارة administrators_authorized_keys + تدوير المفاتيح يدوياً + مشاكل الصلاحيات. الوصول يصير سياسة لحظية في NetBird.

**المرجع:** https://docs.netbird.io/how-to/ssh

---

## ❌ المسألة 3 (ثانوية): تطبيق يستبدل مفاتيح SSH
- `Configuring SSH access` كان يكتب فوق `administrators_authorized_keys` كامل (يمسح مفاتيح أخرى). **أُصلِح** (الآن يدمج بـ tag `motech-agent`). لكن لو اعتمدنا NetBird SSH، تنتفي الحاجة.

---

## معطيات بيئة الاختبار
- جهاز العميل: MoTech, Windows 10 Pro, user `motech\moain` (admin), NetBird IP متغيّر (~100.95.x).
- الوكيل: Azure Ubuntu VM، netbird CLI 0.71.4، منضمّ لنفس الشبكة.
- NetBird Cloud (خطة مجانية)، PAT متاح.
- الاتصال البديل المؤقت المستخدم للاختبار: Cloudflare Tunnel (trycloudflare) + OpenSSH — يعمل لكن نفق مؤقت.
