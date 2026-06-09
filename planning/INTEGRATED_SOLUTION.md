# 🎯 الحل المتكامل — تثبيت مثالي + NetBird SSH المدمج

> بناءً على طلب المستخدم (2026-06-09) + استكشاف NetBird API الفعلي.
> الهدف: تثبيت تلقائي مثالي على أي ويندوز (حتى بنت ضعيف)، متسلسل لا يتوقف، يتعافى من الفشل، يرفع المفتاح للواجهة، ويستغل كامل إمكانيات NetBird.

---

## 🔑 الاكتشاف الكبير: NetBird SSH المدمج (يحل أصعب مشاكلنا)

أجهزتك على NetBird **v0.71.4** (> 0.61 المطلوبة) → **NetBird SSH المدمج مدعوم بالكامل**.

### ليش هذا يغيّر كل شي:
| الطريقة الحالية (OpenSSH ويندوز) | NetBird SSH المدمج |
|---|---|
| تثبيت OpenSSH.Server capability | ❌ غير مطلوب |
| ضبط `administrators_authorized_keys` + ACL | ❌ غير مطلوب (هذا كان سبب فشل SSH على جهاز مهند!) |
| فتح firewall port 22 | ❌ غير مطلوب |
| إدارة المفاتيح يدوياً | ✅ NetBird ACLs أو machine-identity |
| تعرّض port 22 | ✅ صفر تعرّض — كله عبر mesh |

**التفعيل:** `netbird up --allow-server-ssh` (+ تفعيل SSH للـ peer في الـ dashboard/API). يصل على port 22 ظاهرياً، يُعاد توجيهه داخلياً لـ 22022.

**المصادقة:** إما JWT (عبر OIDC) أو `--disable-ssh-auth` (machine-identity عبر NetBird ACLs — الأنسب لنا، بلا تفاعل).

---

## خطة التثبيت المثالي (طلب المستخدم)

### المتطلبات:
1. **تلقائي وسريع** — يثبّت كل شي بنفسه.
2. **متسلسل، لا يتوقف** — كل مرحلة تكمل قبل التالية؛ ما ينتقل حتى يخلّص.
3. **يتعافى من الفشل** — لو فشلت مرحلة: يعيد المحاولة (retry+backoff)، وإذا فشل نهائياً يقول للمستخدم "أعد المحاولة" بوضوح.
4. **يرفع المفتاح/الحالة للواجهة تلقائياً** بعد التثبيت.
5. يعمل حتى مع نت ضعيف (timeouts + retries على كل خطوة شبكية).

### المراحل (state machine، كل مرحلة idempotent + retry):
```
[1/6] التسجيل بالرمز (retry 3x) → احفظ agent.json
[2/6] الانضمام لـ NetBird (--allow-server-ssh, setup-key مُتحقَّق مسبقاً, timeout+retry)
[3/6] تفعيل NetBird SSH المدمج (بدل OpenSSH+ACL+firewall)
[4/6] أول heartbeat → يرفع الحالة+المفتاح+معلومات الجهاز للواجهة
[5/6] تثبيت الخدمة/المهمة المجدولة (auto-start, RestartOnFailure)
[6/6] التحقق النهائي (NetBird peer connected + SSH reachable + heartbeat acked)
→ "✅ تم التثبيت بنجاح" أو "⚠️ فشل عند [المرحلة] — أعد المحاولة"
```

### تحسينات على الموجود:
- **نت ضعيف**: غلّف كل HTTP بـ retry+backoff (الموجود: timeout فقط). كل خطوة شبكية idempotent.
- **التحقق المسبق من setup key** على الباكند قبل إرساله (يمنع تعليق netbird up — من بحث #1).
- **install-progress.json** على القرص → الواجهة تستطلعه وتعرض التقدّم الحي.
- **يرفع للواجهة**: agent_version, os_version, netbird_version, peer_id, ssh_method, last_rotation.

---

## NetBird — إمكانيات مكتشفة (API يعمل، توكن صالح)
- **Peers**: جهازي (Ubuntu, 100.95.205.190), MoTech (Win10, 100.95.51.27), ai-sandbox. كلها v0.71.4.
- **Policies**: Default (all) + Motech SSH (tcp/22). 
- **Setup-keys**: متاح إنشاء reusable/ephemeral عبر API (مفيد لـ CI/الاختبار).
- **Posture-checks**: متاح (فحص وضعية الجهاز — AV/OS version — لـ zero-trust لاحقاً).
- **Routes/DNS**: متاح (شبكات فرعية + DNS داخلي).

## القرار
- **اعتماد NetBird SSH المدمج كأساس** (بدل OpenSSH ويندوز) — أبسط، أأمن، يحل مشكلة ACL نهائياً.
- إبقاء OpenSSH كخيار احتياطي (fallback) للأجهزة < v0.61.
- الاختبار عبر GitHub Actions Windows runner (مجاني).
