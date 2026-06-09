

    
        
            
            
# مراجعة معمقة للهندسة المعمارية والأمن - منصة الوصول عن بعد من موتك

            
            
            
                2026-06-09
                إنجليزي
            
            

            
            
                
لقد تم بأفكار ورؤى من 31 مصدرا

# 

المستودع: github.com/moain2026/motech-platform · تم استنساخ الالتزام بتاريخ 2026-06-09 · حوالي 3150 سطرًا من التعليمات البرمجية بلغة Go (الخادم + الوكيل) + لوحة التحكم.
 النطاق: صحة التثبيت الصامت، وتدفق التثبيت الذي لا يتوقف أبدًا، والتحقق من SSH، ولغز "عدم وجود مخرجات"، وتحسين التحكم في الوصول، واختيار اللغة، والتشغيل بدون واجهة رسومية + الجلسة 0، ومختبر اختبار Linux→Windows، ومراجعة كاملة للتعليمات البرمجية، وتحليل فجوات المعايير الدولية.

## 0. ملخص تنفيذي - أهم 5 أولويات

بعد قراءة قاعدة الشفرة البرمجية كاملةً سطرًا بسطر، ومقارنتها بإرشادات مايكروسوفت الرسمية لـ OpenSSH، ووثائق واجهة سطر أوامر NetBird، وشفرة استدعاءات النظام بلغة Go، ووثائق SmartScreen / Trusted Signing، يتضح أن النظام مصمم هندسيًا بشكل ممتاز - ملف تنفيذي واحد بلغة Go، تشفير AES-256-GCM في حالة السكون، فصل رموز JWT بين المسؤول والوكيل، آلة حالة دوران متكررة مع اختبارات، ونموذج بقاء للمهام المجدولة مُحسَّن بشكل صحيح لمقاومة الأجهزة الميدانية المعادية. المشكلات التي واجهتها ليست معمارية، بل هي مشاكل تكاملية تتعلق بإنشاء العمليات الصامتة، والرجوع التفاعلي لواجهة سطر أوامر NetBird، وبروتوكول SSH، وبعض الثغرات في إجراءات التشفير والتشغيل.

أهم 5 أشياء يجب إصلاحها هذا الأسبوع:

- توقف عن تشغيل نظام ويندوز من خلال نفق سريع من Cloudflare مع اتصال متداخل ssh → PowerShell → cmd /c. هذا ما تسبب في ظهور مشكلة "يبدو أن ملف exe يتعطل" - وليس الملف التنفيذي نفسه. أنشئ بيئة اختبار حقيقية لتحويل نظام لينكس إلى ويندوز (باستخدام QEMU/KVM أو Azure B2s مقابل 7 دولارات شهريًا). ستجد الوصفة الدقيقة في القسم (ج) أدناه.

- اجعل العملية JoinNetbirdغير تفاعلية تمامًا ولا تعيق تسجيل الدخول الموحد عبر المتصفح. أضف --foreground-mode=falseدلالات: استخدم علامة الجمع netbird service install(+) netbird service startفقط، ومرّر مفتاح الإعداد عبر البرنامج الخفي. الوقت الحالي البالغ 45 ثانية context.DeadlineExceededهو حل مؤقت للمشكلة ، وليس حلاً جذريًا لها agent/internal/agent/agent.go:183-188.

- قم بتعزيز التوليد الصامت عن طريق تثبيت ملف GUI الثنائي دائمًا على-H windowsgui (أنت تفعل هذا لـ cmd/guiولكن ليس لـ cmd/agent)، و NULL بشكل صريح Stdin/Stdout/Stderrعلى كل silentCmdبحيث لا يمكن أن يؤدي وراثة Session-0 للمقابض المعطلة إلى إتلاف العناصر الفرعية ( cmd_windows.go:20-26).

- تم تشديدVerifySSHReady إجراءات التحقق من الجاهزية إلى خمس نقاط: تشغيل خدمة SSH + وجود المفتاح + التحقق من تجزئة قائمة التحكم بالوصول + وجود قاعدة جدار الحماية + 127.0.0.1:22فحص TCP حقيقي للحلقة الداخلية. حاليًا، يتم التحقق من نقطتين فقط من هذه النقاط الخمس sshd_windows.go:43-65.

- استبدل المفتاح الرئيسي المختصر باستخدام SHA-256 وسر JWT المفرد باستخدام HS256 بمفاتيح مخصصة مشتقة من HKDF بالإضافة إلى kidرأس وفاصل تدوير المفاتيح. خوارزمية AES-GCM نفسها جيدة؛ لكن اشتقاق المفتاح وسر JWT العالمي المفرد هما نقطة الضعف .backend/internal/auth/crypto.go:13-16auth.go:30,51

ما هو قوي: آلة حالة الدوران، والقفل المتبادل العالمي ذو النسخة الواحدة ( singleinstance_windows.go:17-46)، TokenManager.ForceReload( token.go:61-94)، ومهمة XML المجدولة مع BootTrigger+LogonTrigger+RestartOnFailure( task_windows.go:24-74)، setOutput(io.Discard)وإعادة توجيه السجل الدفاعي ( agent/cmd/agent/main.go:127-152)، ونموذج المفتاح المملوك للخادم الخلفي (لا توجد مفاتيح خاصة على القرص على العميل) - هذه قرارات نموذجية.

## 1. تحديد السبب الجذري لكل مشكلة + حل ملموس

### المشكلة الأولى - التثبيت الصامت (بدون وميض ويندوز)

ما تفعله اليوم (أساس صحيح):

silentCmdيُفعّل هذا النمط كلاً من ` HideWindow: true and` و` CreationFlags: 0x08000000 ) CREATE_NO_WINDOW` في كل عملية فرعية agent/internal/agent/cmd_windows.go:20-27. هذا هو نمط Go المتعارف عليه، والذي تم تأكيده من خلال مصدر استدعاءات النظام في Go والمشكلة المستمرة منذ فترة طويلة golang/go#13754(github.com 1 ). مع ذلك، فإن استخدام `and` HideWindowوحده لن يمنع ظهور وحدة التحكم في عملية فرعية لنظام وحدة التحكم (PowerShell، netsh، schtasks)، وهو CREATE_NO_WINDOWفي الواقع العلامة التي يعتمدها النواة (Stack Overflow 2 ). إضافة `and` -WindowStyle Hiddenإلى PowerShell، كما تفعل في ` sshd_windows.go:18-32<script>`، أمر صحيح ويزيل وميض WPF لجسم البرنامج النصي نفسه.

الفجوات الخفية التي وجدتها (ملف:سطر):

- 

cmd/agent/main.goهو ملف تنفيذي لنظام وحدة التحكم. واجهة المستخدم الرسومية الخاصة بك cmd/guiمبنية باستخدام -ldflags "-H windowsgui"( docs/AGENT.md)، ولكنها cmd/agent/main.goليست كذلك. إذا نقر المستخدم النهائي (وليس واجهة المستخدم الرسومية) نقرًا مزدوجًا motech-connect.exeأو تم تشغيله عبر مهمة مجدولة بتفسير خاطئ Hidden=true، فستظهر نافذة وحدة التحكم لفترة وجيزة قبل إنشاء أي عملية فرعية. هذه هي بالضبط أعراض "وميض وحدة التحكم" التي يراها المستخدمون حتى بعد إصلاحات Silent-Child. الحل: بناء واجهة سطر الأوامر للوكيل بطريقتين:

```
# silent service variant (no console at all, used by the scheduled task)
GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui -s -w" -o motech-connect.exe ./cmd/agent
# CLI variant for ops/debug
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o motech-connect-cli.exe ./cmd/agent

```

عندئذٍ، لا يمتلك البرنامج الرئيسي أي وحدة تحكم، CREATE_NO_WINDOWويزيل جميع الومضات من البرامج الفرعية. لاحظ التحذير الوارد في StackOverflow: عندما يُنشئ برنامج رئيسي في Go يعتمد على واجهة المستخدم الرسومية برنامجًا فرعيًا، فإن البرنامج الفرعي لا يحصل على أي وحدة تحكم افتراضيًا - CREATE_NO_WINDOWوهذا يُعدّ إجراءً احترازيًا (stackoverflow.com 2 ).

- 

silentCmdلا يُلغي هذا الإجراء معالجة مؤشرات Std. في الجلسة 0 (سياق الخدمة)، يمكن أن تكون المؤشرات الموروثة فارغة 0xFFFFFFFF، وأي عملية PowerShell فرعية تتعامل مع stderr قد تتسبب في حدوث خطأ فادح - وهي الآلية الموضحة في مدخلك LEARNINGS.md. logToFileلذا، عزز أمان النظام.

```
func silentCmd(name string, args ...string) *exec.Cmd {
    c := exec.Command(name, args...)
    c.SysProcAttr = &syscall.SysProcAttr{
        HideWindow:    true,
        CreationFlags: createNoWindow | 0x00000008, // DETACHED_PROCESS for true background
    }
    c.Stdin  = nil
    c.Stdout = io.Discard   // never inherit a broken Session-0 stdout
    c.Stderr = io.Discard   // ditto
    return c
}

```

بالنسبة للأوامر التي تتطلب إخراجًا ( run()في ملف sshd_windows.go:79-82)، استخدم مسارًا منفصلاً silentCmdCaptureيُوجّه إلى ملف الإخراج bytes.Buffer، ولا يرث أبدًا مؤشرات نظام التشغيل. لا تُعيّن كلا المسارين CREATE_NO_WINDOWللأوامر DETACHED_PROCESSالتي تتطلب انتظارًا .CombinedOutput()، لأن هذا المزيج يُعطّل المسارات. استخدم DETACHED_PROCESSالمسار فقط في حالات "التنفيذ والنسيان" (مثل إنهاء المهام اليتيمة، أو تشغيل المهام الفرعية).

- 

مُثبِّت NetBird. استخداماتك - هذا هو مفتاح NSIS الصامت وهو صحيح وفقًا لوثائق NetBird (docs.netbird.io 3 ). لكن المُثبِّت نفسه يُشغِّل واجهة مستخدم NetBird في شريط المهام افتراضيًا (يكتب إلى ). بالنسبة للخوادم/الأكشاك التي تعمل بدون واجهة رسومية netbird_install.go:56، فأنت تريد (إصدار MSI)، والذي تصفه نفس الوثائق صراحةً. انتقل من مُثبِّت EXE إلى MSI لعمليات نشر غير مراقبة بالكامل.silentCmd(tmp, "/S")HKLM\…\Runmsiexec /i netbird.msi /qn AUTOSTART=0

- 

installerScheduledTaskيتم استدعاء هذه الوظيفة schtasks /Run /TNبشكل متزامن بعد إنشائها. هذا Runالاستدعاء نفسه هو وظيفة فرعية جديدة - وهو يخفيها بالفعل، لكن هدفsilentCmd المهمة (وضع الوكيل ) يرث بيئة النظام المحددة في ملف XML . جيد. لكن هذا يخفيها فقط من واجهة مستخدم جدولة المهام - فهو لا يخفي نافذة وحدة التحكم الخاصة بالملف التنفيذي الذي تم تشغيله. هذا هو الغرض من هذه الوظيفة. (هذه هي بالضبط الثغرة رقم 1 المذكورة أعلاه).runHidden=truetask_windows.go:54<Hidden>true</Hidden>-H windowsgui

الخلاصة: استراتيجيتك الصامتة صحيحة بنسبة 80%. أما النسبة المتبقية البالغة 20% (النظام الفرعي للنظام الرئيسي، ومعالجة القيم الفارغة، وMSI لـ NetBird) فتغلق جميع مسارات التسريب المعروفة.

### المشكلة الثانية - التثبيت المتسلسل الذي لا يتوقف أبدًا ( netbird upيعلق)

أكد فريق تطوير NetBird السبب الجذري: netbird up يعود النظام إلى تسجيل الدخول الموحد التفاعلي عند رفض مسار مفتاح الإعداد (غير صالح، منتهي الصلاحية، عنوان URL إداري خاطئ)، ويفتح البرنامج متصفحًا لبدء عملية OAuth. لا يوجد --no-browserخيار لتفعيل هذهnetbird up الميزة - وهو طلب مفتوح منذ عام 2025 (مشكلة GitHub رقم 3608 4 ). تؤكد صفحة استكشاف الأخطاء وإصلاحها في NetBird هذا الخطأ: جهاز سبق تسجيل دخوله عبر تسجيل الدخول الموحد، ثم يحاول استخدام مفتاح الإعداد، أو العكس، ينتهي به الأمر "بمحاولة فتح نافذة متصفح تفاعلية للمصادقة عبر تسجيل الدخول الموحد، ويفشل" (NetBird 5 ).

إن مهلة السياق البالغة 45 ثانية تعالج agent/internal/agent/agent.go:183-189العرض: إذ يعيد التثبيت التحكم، لكن الجهاز لا يزال غير متصل بشكل كامل. هذا هو الإجراء الدفاعي الصحيح، لكنه ليس النهج الأساسي الأمثل .

النهج الصحيح - تم إثباته من خلال مراجعة وثائق NetBird نفسها:

- قم بتثبيت NetBird كخدمة أولاً (تم ذلك بالفعل): netbird service install+ netbird service start( agent.go:171-172). هذا هو التسلسل المتعارف عليه وفقًا لوثائق واجهة سطر الأوامر لـ NetBird 6 .

- تواصل مع البرنامج الخفي عبر مقبس الإدارة المحلي واستخدم وضع مفتاح الإعداد فقط :
```
netbird up --setup-key "<KEY>" --management-url https://api.netbird.io --hostname "<machine>" --log-file C:\ProgramData\Motech\netbird.log

```

- تحقق مسبقًا من مفتاح الإعداد باستخدام واجهة برمجة تطبيقات NetBird على الخادم قبل إرساله إلى الوكيل. يتم إنشاء المفتاح اليوم . إذا تم استخدام هذا المفتاح في تثبيت سابق، فسيتم الرجوع تلقائيًا إلى تسجيل الدخول الموحد (SSO) وسيتم تطبيق مهلة 45 ثانية. أضف شرطًا في ( ) مباشرةً قبل إعادة إصدار المفتاح - إذا لم يتم التحقق منه، CreateSetupKeyفأعد إنشاء المفتاح. أو ببساطة أعد إنشاء المفتاح تلقائيًا في كل سجل - فهي لا تكلف شيئًا.backend/internal/netbird/netbird.go:51-68type:"one-off", usage_limit:1, expires_in:86400netbird upGET /api/setup-keys/{id}AgentRegisterhandlers.go:used_count > 0

- استخدم فحص مراقبة النظام بدلاً من انتظار خروج واجهة سطر الأوامر. netbird up قد يُعيد الأمر نجاح العملية بينما لا يزال البرنامج الخفي قيد الاتصال. استبدل الأمر .Run()`-then-sleep` بما يلي:
```
if err := silentCmdCtx(upCtx, path, args...).Start(); err != nil { ... }
// Now poll netbird status --json for up to 30s
for i := 0; i < 30; i++ {
    if ip := netbirdPeerIP(); ip != "" { return nil }
    time.Sleep(time.Second)
}

```

هذا يفصل بين "هل تم تشغيل واجهة سطر الأوامر" و "هل نحن بالفعل على الشبكة".

- اجعل كل خطوة متكررة النتائج، وأصدر ملف تقدم بصيغة JSON يمكن لواجهة المستخدم الرسومية/لوحة التحكم استطلاعه، على سبيل المثال C:\ProgramData\Motech\install-progress.jsonباستخدام الأمر `.`. يكون الملف {"step":3,"status":"ok","detail":"netbird joined; peer=100.95.x.y"}الحالي مناسبًا لتشغيله في سطر الأوامر، ولكنه غير مرئي للعملية الفرعية لواجهة المستخدم الرسومية.fmt.Printf("[%d/5] ...")cmd/agent/main.go:68-98

ضمانة أقوى لإتمام التثبيت (شرط "الانتهاء دائمًا"): يتم تغليف كتلة الخطوات الخمس بالكامل في عنصر رئيسي context.WithTimeout(ctx, 5*time.Minute)وعنصر فرعي defer func() { writeFinalStatus(...) }(). ثم يقوم المثبت بكتابة إحدى حالات الإنهاء الثلاث على القرص: ok، warning(s)، failed-but-recoverable. لا توجد حالة رابعة. بالاقتران مع RestartOnFailure: 3×1minفي ملف XML للمهام المجدولة ( task_windows.go:59-62)، يتم إصلاح التثبيت الذي ينتهي warningتلقائيًا بعد إعادة التشغيل.

### المسألة 3 - التحقق النهائي: VerifySSHReadyالاكتمال

اليوم، تتحقق الدالة من أمرين ( sshd_windows.go:43-65):

- Get-Service sshd | .Status == "Running"

- يوجد نص base64 للمفتاح العام المتوقع فيadministrators_authorized_keys

هذا ضروري ولكنه غير كافٍ . قد ينجح الجهاز في كلا الأمرين ومع ذلك يرفض تسجيل دخول SSH الذي ينسخه المشغل من لوحة التحكم. أنماط الفشل الواقعية التي رأيتها والتي وثقتموها docs/TROUBLESHOOTING.mdبالفعل (SSH-001، NB-001، NB-002):

- قاعدة جدار الحماية المسماة OpenSSH-Server-In-TCPموجودة ولكنها معطلة / تشير إلى ملف تعريف قديم.

- المفتاح موجود في الملف ولكن قائمة التحكم بالوصول للملف خاطئة (المشكلة رقم 5) ← ينجح تبادل المفاتيح، ويفشل مصادقة المفتاح العام بصمت.

- خدمة sshd تعمل ولكنها مرتبطة ::1فقط بـ...

- تم الانضمام إلى NetBird ولكن النظير موجود في مجموعة مختلفة / سياسة ACL ترفض SSH.

- تم تسجيل المهمة المجدولة، لكن المسار الثنائي الذي تشير إليه قديم (إصدار سابق).

- token.go:61-94نبضة القلب 401 لأن TokenManager لم يتم إعادة تحميله بعد ( تم تصميم الحالة من أجل ذلك).

جودة إنتاجية VerifySSHReady— 5 + 2 فحص:

```
type ReadinessReport struct {
    SSHDRunning           bool
    PublicKeyInstalled    bool
    AuthKeysACLCorrect    bool   // owner+ACE list hash check
    FirewallRuleActive    bool   // Get-NetFirewallRule | Where Enabled
    Port22Listening       bool   // net.DialTimeout("tcp","127.0.0.1:22",2s)
    NetbirdPeerConnected  bool   // netbird status --json -> "ManagementState":"Connected"
    HeartbeatAcked        bool   // last server heartbeat <60s ago, no 401
}

```

يُسجّل كل فحص سطرًا واحدًا في سجل التدقيق الخاص agent.logبملف JSON الخاص بتقدم التثبيت. لا يُعلن المُثبِّت عن نجاح التثبيت إلا إذا كانت جميع النقاط السبع خضراء . إذا كانت خمس نقاط خضراء ونقطتان صفراء، يُعلن عن حالة "جاهز (متدهور)" ويُسجّل التحذير في لوحة التحكم - ولا يُعلن أبدًا عن "التثبيت بنجاح" في حالة الجاهزية الجزئية.

لإجراء فحص ACL تحديدًا، قم بتجزئة قائمة ACE ومقارنتها:

```
(Get-Acl C:\ProgramData\ssh\administrators_authorized_keys).Access |
  Select IdentityReference,FileSystemRights | ConvertTo-Json

```

المتوقع: مدخلان بالضبط — NT AUTHORITY\SYSTEM:FullControlو BUILTIN\Administrators:FullControl، بالإضافة إلى AreAccessRulesProtected=$true(مع تعطيل التوريث). أي شيء آخر يفشل. هذا يطابق عقد مايكروسوفت الموثق (Microsoft Learn 7 ، وPowerShell/Win32-OpenSSH wiki 8 ).

### المشكلة الرابعة - لغز "يبدو أن ملف exe يتعطل، ولا يوجد مخرج" - السبب الجذري

الخلاصة: لم يتعطل البرنامج. لقد خدعك نظام الحماية.

شهادة:

- تم تشغيل نفس البرنامج بشكل جيد على نفس الجهاز في 2026-06-05 (جهازك HANDOFF.md).

- يقوم الكود الجديد cmd/agent/main.go:37-50بإعادة توجيه التسجيل بشكل وقائي إلى ملف قبل أن يقوم الوكيل بأي شيء يمكن أن يمس stderr، مع log.SetOutput(io.Discard)وجوده كحد أدنى للسلامة - وهو النمط المطلوب بالضبط في الجلسة 0 (Microsoft TechCommunity 9 ).

- أثبتت عملية go vetالبناء النظيفة عدم وجود أي تلف في وقت الترجمة.

ما حدث خطأً في جهاز الاختبار الخاص بك: كنت تقوم بتوصيل سلسلة من الروابط:

```
local-bash → ssh → cloudflare quick-tunnel → sshd-on-Windows
   → PowerShell (the Windows OpenSSH default shell since 2019)
   → cmd /c "motech-connect.exe ... > out.txt 2>&1"

```

تتفاقم أربع مشاكل هنا:

- الصدفة الافتراضية هي PowerShell. يُعيّن Windows OpenSSH صدفة PowerShell كنظام فرعي افتراضي منذ إصدارات Win32-OpenSSH 2019 وما بعدها (GitHub Wiki — DefaultShell 10 ). تُحلل PowerShell سطر الأوامر باستخدام قواعد الاقتباس الخاصة بها ، ثم تُعيد إرساله cmd /cباستخدام قواعد CMD . يتم التعامل مع السلاسل النصية التي تحتوي على &علامات اقتباس مفردة |أو ^مزدوجة أو أقواس غير متوازنة بشكل غير مُعلن.

- تضيف الأنفاق السريعة من Cloudflare انقطاعًا مؤقتًا لمدة 100 ثانية وتعيد كتابة بعض أكواد التحكم - قد يتم تخزين الإخراج مؤقتًا حتى ينقطع الاتصال، وعند هذه النقطة ترى "لا يوجد إخراج" ولكن تم تشغيل الملف التنفيذي بشكل جيد.

- stdoutيتم تخزين البيانات عبر SSH على مستوى الأسطر بواسطة Go ولكن يتم تخزينها على مستوى الكتل بواسطة PowerShell عند إعادة إصدارها ، لذلك يمكن لعملية الخروج السريع كتابة 50 سطرًا لا تراها أبدًا لأن عملية ssh الأصلية تفصل أولاً.

- تم تعطيل نشر رمز الخروج في PowerShell → cmd /c: يتم تعيين رمز الخروج الخاص بـ PowerShell $LASTEXITCODE، لكن عميل ssh لا يرى سوى رمز الخروج الخاص بـ PowerShell، والذي يكون 0بغض النظر عن cmd /cرمز الخروج الخاص بـ إلا إذا قمت بكتابة رمز الخروج بشكل صريح exit $LASTEXITCODE.

حزام أمان موثوق - ثلاثة خيارات مرتبة حسب الأفضلية:

أ. قم بفرض استخدام الصدفة الافتراضية لـ OpenSSH cmd.exeمرة واحدة على جهاز الاختبار، ثم قم بتشغيل الملفات الثنائية مباشرة:

```
# on the Windows test machine (once, elevated)
New-ItemProperty -Path "HKLM:\SOFTWARE\OpenSSH" -Name DefaultShell `
  -Value "C:\Windows\System32\cmd.exe" -PropertyType String -Force
Restart-Service sshd

```

عندها يصبح حزامك بسيطًا:

```
ssh test@win 'C:\Users\test\motech-connect.exe register --token ABCD > C:\tmp\out.txt 2>&1 & echo ERR=%ERRORLEVEL%'
ssh test@win 'type C:\tmp\out.txt'

```

هذا موثق ومدعوم على موقع Win32-OpenSSH wiki (github.com 10 ).

ب. قم بتغليفها باستخدام PowerShell واحد Start-Process -Wait -PassThruوالتقط مخرجات + تدفقات البيانات:

```
ssh test@win 'powershell -NoProfile -Command "$p = Start-Process -FilePath \"C:\Users\test\motech-connect.exe\" -ArgumentList \"register\",\"--token\",\"ABCD\" -RedirectStandardOutput \"C:\tmp\so.txt\" -RedirectStandardError \"C:\tmp\se.txt\" -PassThru -Wait; exit $p.ExitCode"; echo "exit=$?"; ssh test@win type C:\tmp\so.txt'

```

يستمر ذلك في الاقتباس في عالم أداة واحدة (StackOverflow التقاط stdout 11 ).

ج. استخدم WinRM ( pwshعبر winrsأو عن بُعد باستخدام pwsh) - لكنه يتطلب HTTPS أو نطاقًا. تخطَّ هذه الخطوة للاختبارات الفردية.

استخدم الخيار أ للتطوير اليومي، والخيار ب للتكامل المستمر. وقم بإزالة نفق Cloudflare السريع لجهاز الاختبار الفعلي - استخدم منفذًا حقيقيًا (Tailscale، أو شبكة NetBird الخاصة، أو قم ببساطة بتوجيه الجهاز الظاهري للاختبار إلى نفس مستأجر NetBird).

### المشكلة 5 - administrators_authorized_keysACL: هل تم icaclsإصلاح المشكلة بالكامل؟

ملفك (sshd_windows.go:31-32):

```
$p='C:\ProgramData\ssh\administrators_authorized_keys'
if (Test-Path $p) { icacls $p /inheritance:r /grant 'Administrators:F' /grant 'SYSTEM:F' | Out-Null }

```

هذا يطابق بايتًا ببايت أمر Microsoft Learn المتعارف عليه (Microsoft Learn — إدارة مفاتيح OpenSSH 12 ) وويكي Win32-OpenSSH 8 :

```
icacls administrators_authorized_keys /inheritance:r
icacls administrators_authorized_keys /grant SYSTEM:(F)
icacls administrators_authorized_keys /grant BUILTIN\Administrators:(F)

```

وتؤكد صفحة تكوين خادم OpenSSH من مايكروسوفت رقم 7 ذلك صراحةً: "يجب أن يحتوي ملف administrators_authorized_keys فقط على إدخالات أذونات لحساب NT Authority\SYSTEM ومجموعة الأمان BUILTIN\Administrators".

ثلاثة تحسينات طفيفة على اكتمال البيانات (جميعها صالحة لأنظمة التشغيل Windows 10/11/Server 2016–2025):

- 

أمان الترجمة. 'Administrators' هذه السلسلة باللغة الإنجليزية. في أنظمة ويندوز العربية والفرنسية والألمانية وغيرها، يتم ترجمتها ( المسؤولون،، Administratoren...) icaclsوستفشل إما بصمت أو برسالة خطأ مبهمة. استخدم معرّفات الأمان (SIDs).

```
icacls $p /inheritance:r /grant *S-1-5-32-544:F /grant *S-1-5-18:F

```

S-1-5-32-544= BUILTIN\Administrators، S-1-5-18= NT AUTHORITY\SYSTEM. هذه توصية مايكروسوفت نفسها في الصفحة نفسها عند وجود مخاوف بشأن التوطين (Microsoft Learn 12 ). ونظرًا لقاعدة مستخدميكم الناطقين باللغة العربية، فهذا أمر لا غنى عنه.

- 

حدد المالك صراحةً. يفرض sshd أيضًا الملكية، وليس فقط قائمة التحكم بالوصول (ACE). أضف:

```
icacls $p /setowner *S-1-5-18

```

هذا ما Set-Acl … | icacls /setowner systemورد في دليل Win32-OpenSSH wiki.

- 

قم بتشغيله بعد كتابة الملف. ترتيبك ensureSSHServerالحالي sshd_windows.go:16-35يضع إصلاح قائمة التحكم بالوصول في الخطوة 4، ولكن فقط إذا كان الملف موجودًا بالفعلif (Test-Path $p) . عند التثبيت الأول، عندما يقوم installAuthorizedKeyالبرنامج keys.go:17-48بإنشاء الملف، تتم هذه الخطوة في installServerPublicKeyالبرنامج agent.go:148-156الذي يستدعي installAuthorizedKey أولًا ، ثم ensureSSHServer. يبدو الترتيب صحيحًا agent.go، ولكن ensureSSHServerيتم تشغيل الخطوة 4 بغض النظر عن ذلك، حتى في الاستدعاءات اللاحقة عندما يكون الملف موجودًا بالفعل مع قوائم تحكم بالوصول خاطئة من تثبيت فاشل سابق. ✅ احتفظ بالترتيب؛ فقط تأكد من تشغيله دون شروط (احذف if (Test-Path)- إنه آمن؛ icaclsفي حالة عدم وجود ملف، لن يحدث أي تغيير).

الخلاصة: تعديل قائمة التحكم بالوصول (ACL) صحيح ، لكن مشكلة التوطين ستؤثر على أكثر من 100 جهاز مضيف باللغة العربية. استخدم صيغة SID. ( learn.microsoft.com).

## أ. لغة/بيئة تشغيل الوكيل — احتفظ بلغة Go، ولكن اذهب إلى أبعد من ذلك

النظام الذي لديك هو بالضبط ما يجب أن يكون عليه وكيل ويندوز بحجم 1000 مستخدم: ملف تنفيذي واحد مرتبط بشكل ثابت، بحجم 10 ميجابايت تقريبًا، بدون بيئة تشغيل، يتم نشره من ملف واحد، مُصمم للعمل مع أنظمة لينكس، وقابل للتوقيع باستخدام Authenticode . دعني أؤكد ذلك بمقارنته بكل بديل بأدلة ملموسة:

لغة
الحجم الثنائي
التبعية في وقت التشغيل
بيئة عمل مريحة لخدمة العملاء
سهولة استخدام AV/EDR
البناء المتقاطع من لينكس
الحكم

اذهب (لك)
حوالي 10 ميجابايت
لا أحد
kardianos/service+golang.org/x/sys/windows/svc
جيد (موقع من قبل PE، بدون كتل نصية)
نعم ، أصلي
يحفظ

الصدأ
حوالي 3-6 ميجابايت
لا أحد
windows-serviceصندوق (ممتاز)
الأفضل (بدون جمع البيانات المهملة، بدون وقت تشغيل)
نعم عبر cross+xwin
فوز هامشي؛ تكلفة إعادة الكتابة غير مبررة

سي شارب / دوت نت 9
حجمها يتراوح بين 60 و140 ميجابايت كوحدة مستقلة، أو حوالي 50 كيلوبايت تحتاج إلى بيئة تشغيل .NET
بيئة تشغيل .NET (حجمها الأساسي 140 ميجابايت)
الأفضل (نظام إدارة الإصدارات الأصلي، خدمة الخلفية)
الأفضل (مناسب للمدافعين لأنه موقع من قبل CLR)
نعم، لكنها ثقيلة
والأسوأ من ذلك ، أن حجم وقت التشغيل يصل إلى 1000 (متوسط ​​13 )

لغة سي++
صغيرة الحجم لكنك تكتب كل شيء
لا أحد
Win32 SCM مباشرة
ممتاز
مؤلم
ألم لا داعي له

باور شيل
غير متوفر
PSCore ~70 ميجابايت
يتطلب برنامجًا تنفيذيًا وسيطًا
سيء — برنامج الحماية Defender / EDR يحب حظر ملفات .ps1 + وضع اللغة المقيد يعطلك
غير متوفر
لا ، قواعد التعرف التلقائي على الكلام الصارمة تمنع ذلك في بيئات مزودي خدمات الإدارة.

الترقية المادية الوحيدة مقارنةً ببرنامج Go الحالي لديك هي استبداله kardianos/serviceبالحزمة الرسميةgolang.org/x/sys/windows/svc ونظام إدارة الإصدارات (SCM) مباشرةً. kardianosهو غلاف قابلية نقل؛ أنت تستهدف نظام Windows فقط للوكيل. يوفر لك نظام إدارة الإصدارات المباشر تقارير أفضل عن حالات فشل بدء التشغيل، ويتيح لك التعيين SERVICE_AUTO_STARTباستخدام dwDelayedAutoStart=1(تبعية برنامج NetBird daemon!)، والتحكم RecoveryActions(إعادة التشغيل عند الفشل الأول/الثاني/الثالث) من التعليمات البرمجية، وتجنب تسريبات التجريد التي تم الإبلاغ عنها في المنتديات ("يعمل تطبيق Go الخاص بي بشكل جيد في الوضع التفاعلي، ولكنه يفشل في البدء بشكل صحيح عند تشغيله كخدمة Windows" 14 ).

الشبكة: Go صحيح. حسّن من خلال (أ) بناء نظامين فرعيين لنفس المصدر، (ب) الاستبدال kardianosبـ x/sys/windows/svc، (ج) إضافة -trimpath -ldflags="-s -w -buildid="من أجل عمليات بناء قابلة للتكرار (مطلوب لسلسلة الحفظ Authenticode على نطاق 1000).

## ب. الدليل الشامل لإنشاء عمليات فرعية صامتة/بدون واجهة رسومية + الجلسة 0

ثلاث طبقات، بهذا الترتيب تحديداً. أي خطأ في إحداها سيؤدي إلى وميض نافذة أو إغلاق الطبقة الرئيسية.

### الطبقة 1 - النظام الفرعي للنظام الرئيسي

ملف Go الثنائي الذي تم إنشاؤه بدون وحدة تحكم -H windowsguiهو ملف PE لنظام وحدة التحكم . عند تشغيله بواسطة SCM/Task Scheduler، يقوم Windows بإنشاء وحدة تحكم؛ كل شيء يعمل بشكل صحيح، ولكن قد يظهر مربع أسود وامض. قم بإنشاء نسخة الخدمة طويلة الأمد من برنامجك -H windowsguiبدون تخصيص وحدة تحكم على الإطلاق، واحتفظ بنسخة CLI منفصلة للعمليات.

```
# service variant (run/install/uninstall paths still work — no console needed)
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-H windowsgui -s -w -buildid=" \
    -o motech-connect.exe ./cmd/agent
# ops/CLI variant (you want to see output)
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w -buildid=" \
    -o motech-connect-cli.exe ./cmd/agent

```

عندما يقوم نظام فرعي لواجهة المستخدم الرسومية بإنشاء نظام فرعي لوحدة التحكم (powershell، schtasks، netbird)، فإن النظام الفرعي لا يرث وحدة تحكم افتراضيًا - ولكن نظام التشغيل Windows سيقوم بتخصيص وحدة تحكم جديدة ما لم تقم بتمرير CREATE_NO_WINDOW(StackOverflow 2 ).

### الطبقة 2 — SysProcAttrالأعلام

```
c.SysProcAttr = &syscall.SysProcAttr{
    HideWindow:    true,                   // SW_HIDE for window-class children
    CreationFlags: 0x08000000,             // CREATE_NO_WINDOW for console children
    // do NOT add DETACHED_PROCESS unless you also discard the pipes — incompatible with .CombinedOutput()
}

```

HideWindowيُترجم هذا إلى STARTUPINFO.wShowWindow = SW_HIDE(مصدر استدعاء النظام Go 15 ). CREATE_NO_WINDOWوهو الخيار الوحيد الذي يمنع kernel32!AllocConsoleتشغيل العمليات الفرعية في وحدة التحكم. ويشمل هذا الخيار العمليات الفرعية التي تملك نافذة (مثل شريط مهام NetBird) والعمليات الفرعية التي تعمل في وحدة التحكم فقط (مثل PowerShell).

### الطبقة 3 - جلسة 0: انضباط الإخراج القياسي/الخطأ القياسي

تعمل خدمة ويندوز في الجلسة 0 ، بمعزل عن المستخدمين التفاعليين منذ نظام التشغيل فيستا (Microsoft TechCommunity 9 ، Core Technologies PDF 16 ). لا تحتوي العملية افتراضيًا على مدخلات/مخرجات/أخطاء قياسية (stdin/stdout/stderr) - os.Stderrإما أن يكون مؤشرًا إلى \\Device\\Nullأو إلى مؤشر غير صالح. قد يؤدي خطأ واحد fmt.Fprintln(os.Stderr, ...)من دالة مؤجلة في حالة ذعر إلى إيقاف العملية.

إن ما تقوم به أنت cmd/agent/main.go:37-50و logToFile( :127-152) بالفعل هو التصرف الصحيح - log.SetOutput(io.Discard)يتم تعيينه أولاً ، ثم يتم ترقيته إلى ملف إذا كان بالإمكان فتحه. طبق نفس المبدأ على الأطفال:

```
func silentCmd(name string, args ...string) *exec.Cmd {
    c := exec.Command(name, args...)
    c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
    // CRITICAL: never inherit a service-context broken handle to a child.
    c.Stdin, c.Stdout, c.Stderr = nil, io.Discard, io.Discard
    return c
}
func silentCmdCapture(name string, args ...string) (string, error) {
    c := exec.Command(name, args...)
    c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
    var buf bytes.Buffer
    c.Stdout, c.Stderr = &buf, &buf
    err := c.Run()
    return buf.String(), err
}

```

بالنسبة للأوامر التي يجب ألا يتم حظر الوكيل نفسه بواسطة (على سبيل المثال، taskkill)، قم بتغليفها بكائن مهمة بحيث يمكن إنهاء العملية الفرعية المعلقة عند إيقاف تشغيل الوكيل:

```
// pseudo — use github.com/kolesnikovae/go-paddle-sdk/jobobject or roll your own
job := createJobObject()
assignProcessToJob(job, cmd.Process.Pid)
// closing job kills all children atomically

```

ميزة إضافية: يتيح لك كائن المهمة أيضًا فرض JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSEعدم ترك أي عنصر أب معطل أي عناصر يتيمة - وهو نوع من الأخطاء التي رأيتها بالفعل ("تم تسجيل بدء الحلقة مرتين"، وهذا هو سبب singleinstance_windows.goوجودها).

## ج. ★ بيئة اختبار ويندوز على جهازك الظاهري الذي يعمل بنظام لينكس — وصفة دقيقة، انسخها والصقها

لديك ثلاثة خيارات جيدة. استخدم الخيار الثاني للعمل اليومي، والخيار الأول للتكامل المستمر المتكرر، والخيار الثالث للفحوصات العشوائية. تجنب استخدام Wine ، فهو لا يدعم برنامج تشغيل TUN الخاص بـ NetBird، أو خادم OpenSSH، أو نظام إدارة الإصدارات في Windows بشكل صحيح.

### الخيار 1 — QEMU/KVM مع نظام التشغيل Windows 11 (جودة كاملة، مجاني، سريع)

الأفضل لـ: إجراء اختبارات عالية الدقة على جهازك الظاهري الذي يعمل بنظام لينكس (بافتراض تفعيل خاصية المحاكاة الافتراضية المتداخلة). يستغرق الإعداد حوالي 30 دقيقة لمرة واحدة.

```
# 0) verify nested virt is on (lscpu | grep -i virt; output should mention VT-x or AMD-V)
sudo apt update && sudo apt install -y qemu-kvm libvirt-daemon-system virt-manager bridge-utils ovmf swtpm

# 1) download Windows 11 ISO + virtio drivers
mkdir -p ~/winlab && cd ~/winlab
# Windows 11 ISO via Microsoft (use your account or fido scripts):
#   https://www.microsoft.com/software-download/windows11
# Or use the official Microsoft Evaluation Center for Windows Server 2022 (free 180-day):
wget 'https://software-static.download.prss.microsoft.com/sg/download/888969d5-f34g-4e03-ac9d-1f9786c66749/SERVER_EVAL_x64FRE_en-us.iso' -O winsrv.iso
wget https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/stable-virtio/virtio-win.iso

# 2) create a 60 GB qcow2 disk
qemu-img create -f qcow2 win11.qcow2 60G

# 3) launch the install (8 GB RAM, 4 CPU, virtio NIC, both ISOs attached)
qemu-system-x86_64 \
  -enable-kvm -cpu host,hv_relaxed,hv_vapic,hv_spinlocks=0x1fff,hv_time \
  -smp 4 -m 8192 \
  -machine q35,smm=on \
  -global driver=cfi.pflash01,property=secure,value=on \
  -drive if=pflash,format=raw,unit=0,file=/usr/share/OVMF/OVMF_CODE_4M.secboot.fd,readonly=on \
  -drive if=pflash,format=raw,unit=1,file=OVMF_VARS_4M.ms.fd \
  -tpmdev emulator,id=tpm0,chardev=chrtpm -chardev socket,id=chrtpm,path=/tmp/swtpm.sock \
  -device tpm-tis,tpmdev=tpm0 \
  -drive file=win11.qcow2,if=virtio,cache=none,format=qcow2 \
  -drive file=winsrv.iso,media=cdrom,index=2 \
  -drive file=virtio-win.iso,media=cdrom,index=3 \
  -netdev user,id=n0,hostfwd=tcp::2222-:22,hostfwd=tcp::3389-:3389 \
  -device virtio-net,netdev=n0 \
  -vga virtio -display gtk

# (start swtpm in another shell, see https://wiki.gbe0.com/en/linux/guides/windows-11-kvm)
swtpm socket --tpm2 --tpmstate dir=/tmp/swtpm --ctrl type=unixio,path=/tmp/swtpm.sock -d

```

اتبع دليل RedHat رقم 17 عندما يطلب منك مُثبِّت Windows برامج تشغيل الأقراص (قم بتحميلها viostorمن ملف virtio ISO). إجمالي وقت التثبيت: حوالي 15 دقيقة على معالج حديث.

بعد تشغيل نظام ويندوز، داخل الجهاز الظاهري (باستخدام PowerShell، بصلاحيات المسؤول):

```
# enable OpenSSH Server and set cmd as default shell — fixes Problem 4
Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0
Start-Service sshd
Set-Service sshd -StartupType Automatic
New-NetFirewallRule -Name OpenSSH-Server-In-TCP -DisplayName 'OpenSSH Server' -Enabled True -Direction Inbound -Protocol TCP -Action Allow -LocalPort 22
New-ItemProperty -Path "HKLM:\SOFTWARE\OpenSSH" -Name DefaultShell -Value "C:\Windows\System32\cmd.exe" -PropertyType String -Force
Restart-Service sshd
# create test user with admin
net user motechtest 'P@ssw0rd!' /add
net localgroup administrators motechtest /add

```

من جهازك الظاهري الذي يعمل بنظام Linux (خرائط إعادة توجيه المنفذ في QEMU localhost:2222→ guest:22):

```
# transfer the freshly built exe
scp -P 2222 motech-connect.exe motechtest@127.0.0.1:'C:\Users\motechtest\'

# run it and capture EVERYTHING:
ssh -p 2222 motechtest@127.0.0.1 'C:\Users\motechtest\motech-connect.exe register --token ABCD-1234-XYZ --server https://qfetmfdn.gensparkclaw.com > C:\tmp\out.txt 2>&1 & echo ERR=%ERRORLEVEL%'
ssh -p 2222 motechtest@127.0.0.1 'type C:\tmp\out.txt'
ssh -p 2222 motechtest@127.0.0.1 'type C:\ProgramData\Motech\agent.log'

# install NetBird inside, then watch heartbeats live
ssh -p 2222 motechtest@127.0.0.1 'powershell -c "Get-Content C:\ProgramData\Motech\agent.log -Wait -Tail 10"'

```

نصيحة احترافية: التقط صورة للشاشة قبل كل اختبار:

```
qemu-img snapshot -c clean-state win11.qcow2
# ... break things ...
qemu-img snapshot -a clean-state win11.qcow2   # one-second revert

```

هذه هي الميزة الأساسية التي لا يمكنك الحصول عليها في جهاز حقيقي.

### الخيار الثاني - جهاز Azure الظاهري الرخيص (أرخص مسار، لا يتطلب إعدادًا محليًا)

في حال عدم رغبتك بتشغيل QEMU، تبلغ التكلفة حوالي 0.10 دولار أمريكي/ساعة عند التوقف، وحوالي 0.07 دولار أمريكي/ساعة عند B2sالتشغيل (أسعار azure.microsoft.com 18 ). تكلفة الإيقاف بعد كل اختبار = 0 دولار أمريكي.

```
# install Azure CLI on your Linux VM
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
az login

# create a Windows VM in 30 seconds (uses your default region)
az group create -n motech-test -l eastus
az vm create \
  --resource-group motech-test \
  --name winlab \
  --image MicrosoftWindowsServer:WindowsServer:2022-datacenter-azure-edition:latest \
  --size Standard_B2s \
  --admin-username motech --admin-password 'YourStr0ngP@ss!' \
  --public-ip-sku Standard

# open RDP + SSH ports
az vm open-port --resource-group motech-test --name winlab --port 22 --priority 1010
az vm open-port --resource-group motech-test --name winlab --port 3389 --priority 1020

# get the IP
IP=$(az vm show -d -g motech-test -n winlab --query publicIps -o tsv)
echo "Windows VM at $IP"

# transfer the agent + run it
scp motech-connect.exe motech@$IP:'C:\Users\motech\'
ssh motech@$IP 'C:\Users\motech\motech-connect.exe register --token ABCD ...'

# DELETE when done — important for cost
az group delete -n motech-test --yes --no-wait

```

يُعد دليل البدء السريع من مايكروسوفت مع Azure CLI 19 المرجع الأساسي.

### الخيار 3 - بيئة معزولة لنظام التشغيل ويندوز (نظام تشغيل ويندوز 11، خفيف الوزن، محدود النطاق مع NetBird)

إذا كان لديك جهاز يعمل بنظام Windows 10/11 Pro/Enterprise في أي مكان (جهازك الشخصي، جهاز المستخدم، إلخ)، فإن Windows Sandbox يُنشئ جهازًا افتراضيًا مؤقتًا بحجم 4 جيجابايت في 5 ثوانٍ. يمكنك ضبط الإعدادات باستخدام .wsbملف (Microsoft Learn 20 ).

```
<Configuration>
  <Networking>Enable</Networking>
  <MappedFolders>
    <MappedFolder>
      <HostFolder>C:\motech-build</HostFolder>
      <SandboxFolder>C:\motech</SandboxFolder>
      <ReadOnly>true</ReadOnly>
    </MappedFolder>
  </MappedFolders>
  <LogonCommand>
    <Command>powershell -ExecutionPolicy Bypass -Command "Start-Process C:\motech\motech-connect.exe -ArgumentList 'register','--token','ABCD' -Wait -RedirectStandardOutput C:\Users\WDAGUtilityAccount\out.txt -RedirectStandardError C:\Users\WDAGUtilityAccount\err.txt"</Command>
  </LogonCommand>
</Configuration>

```

ملاحظات: لا يستمر Windows Sandbox (كل تشغيل جديد - مثالي لـ "هل يعمل التثبيت من الصفر؟")، ولا يمكنه تشغيل الأجهزة الافتراضية المتداخلة ، ويتم تحميل برنامج تشغيل NetBird TUN ولكن التوجيه يتم NAT بشكل كبير، ويعمل SCM لتثبيت الخدمة ولكن ليس دائمًا لاختبارات التشغيل التلقائي عند التمهيد (Sandbox = لا يوجد تمهيد).

### نبيذ؟ — لا، لا داعي لذلك.

يُغطي برنامج Wine جزءًا من واجهة برمجة تطبيقات Windows. فهو لا يدعم: نموذج برنامج تشغيل TUN الخاص بـ WireGuard/NetBird، أو خادم OpenSSH الخاص بنظام Windows (لا يدعمه Wine sshd.exe)، أو التحقق من رمز المصادقة، أو عزل الجلسة 0، أو مخطط XML 1.3 الخاص بجدولة المهام الذي تستخدمه في Windows task_windows.go:24. ستكون بذلك تختبر نظام Windows افتراضيًا. تخطَّ هذه الخطوة.

## د. نتائج مستوى الكود - نقاط القوة، نقاط الضعف، البدائل الأقوى (ملف:سطر)

### نقاط القوة (حافظ عليها، لا تغيرها)

- auth/crypto.goAES-256-GCM مع قيمة عشوائية 12 بايت + نص مشفر = nonce || ct— بناء مغلف نموذجي ( crypto.go:18-34).

- token.goTokenManager — ذاكرة تخزين مؤقتة TTL + إعادة تحميل إجبارية عند 401 + 3 × 100 مللي ثانية لإعادة المحاولة ضد قفل الملف المؤقت — يحل مشكلة حقيقية في بيئة الإنتاج مع الاختبارات في token_test.go.

- singleinstance_windows.go— يمنع القفل المتبادل المسمى عالميًا خطأ الحلقة المزدوجة الذي لاحظته في الميدان.

- task_windows.goتُعد مهمة XML المجدولة — LogonTrigger+BootTrigger+RestartOnFailure+StartWhenAvailable+ <Hidden>true</Hidden>أفضل ممارسة للبقاء على المضيفين المحصنين (وثائق Microsoft Win32 21 ).

- cmd/agent/main.go:37-50إعادة توجيه سجلات الحماية io.Discardقبل أي شيء آخر. هذا هو أهم إجراء أمني لجلسة 0.

- نموذج المفتاح المملوك للخادم (ADR-008). لا يغادر المفتاح الخاص الخادم مطلقًا؛ إذ يقوم البرنامج الوسيط بتثبيت الجزء العام فقط. هذا يقضي على فئة كاملة من هجمات تسريب البيانات ويجعل عملية التناوب سهلة للغاية.

- rotation_test.go— في الواقع، يختبر آلة الحالة من البداية إلى النهاية بما في ذلك خاصية التكرار في ظل عمليات الدفع المتكررة وحارس "rotate=true بدون install_pubkey".

### نقاط الضعف + بدائل أقوى ملموسة

8
ملف: سطر
مشكلة
بديل أقوى

W1
backend/internal/auth/crypto.go:13-16
deriveKey= خوارزمية SHA-256 واحدة للسلسلة الرئيسية. لا يوجد ملح، ولا فصل بين النطاقات. تغيير بايت واحد في السلسلة الرئيسية يُغير جميع النصوص المشفرة - وهذا في الواقع هو الخطأ الفادح في معيار ADR-008 الموثق "لا تُغير بعد إنشاء العملاء".
استخدم خوارزمية HKDF-SHA-256 مع قيمة ملح لكل سجل وسلسلة معلومات لكل غرض. خزّن قيمة الملح مع النص المشفر: `version`

W2
backend/internal/auth/auth.go:30,44-53
JWT_SECRETهو سلسلة نصية عالمية واحدة، HS256، بدون kidرأس، بدون فاصل دوران.
الانتقال إلى التوقيع غير المتماثل EdDSA (Ed25519) . يحتفظ النظام الخلفي بالمفتاح الخاص؛ ويمكن للوكلاء التحقق محليًا باستخدام المفتاح العام دون الحاجة إلى تبادل البيانات. إضافة kidرأسية ← دعم عدد N من المفاتيح قيد الاستخدام، والتدوير بإضافة مفاتيح جديدة، وإزالة القديمة بعد انتهاء صلاحيتها. تشير إرشادات Curity (curity.io 22 ) و n0rdy (n0rdy.foo 23 ) إلى أن هذا هو الحد الأدنى المطلوب للتدوير.

W3
backend/internal/auth/auth.go:51
jwt.SigningMethodHS256يتم تضمينها بشكل ثابت في Issue؛ لا يفرض Parse سوى "طريقة HMAC"، لذلك سيتم قبول رمز HS256 مزور يحتوي على سر قابل للتخمين.
قم بتثبيت طريقة واحدة فقط في كل من Issue و Parse: if t.Method.Alg() != "HS256"(أو EdDSA بعد W2). استخدم jwt.WithValidMethods([]string{"EdDSA"})خيار المحلل.

W4
backend/internal/config/config.go:34
JWT_SECRETالقيمة الافتراضية هي السلسلة النصية الحرفية "insecure-dev-secret". إذا قام شخص ما بنشر المنتج دون تحديد متغيرات البيئة، فسيستخدمها بيئة الإنتاج.
ارفض بدء التشغيل إلا إذا JWT_SECRETتم تعيين قيمة وكان طول الملف ≥ 32 بايت. MOTECH_KEY_ENCRYPTION_KEYوينطبق الأمر نفسه على . قم بتعطيل البرنامج بسرعة في config.Load.

W5
backend/internal/handlers/handlers.go: AgentRegister
عمر وكيل JWT = 365 يومًا ( 365*24*time.Hour).
يمكن تقليل مدة الصلاحية إلى 24 ساعة مع تدوير رمز التحديث، أو استخدام رمز JWT قصير مع مفتاح متماثل طويل الأمد لكل وكيل يُستخدم لإنشاء إشارات نبضات القلب. يُعدّ بروتوكول mTLS مع شهادة عميل لكل وكيل (صادرة من خادمك) الترقية القياسية.

W6
backend/internal/handlers/handlers.go: Login
لا يوجد تحديد لمعدل الوصول ( قائمة التحقق /api/auth/loginالخاصة بك للتحصين مدرجة على هذا النحو ).docs/SECURITY.md[ ]
أضف httprate.LimitByIP(5, time.Minute)من go-chi/httprate. وينطبق الأمر نفسه على /api/agent/register.

W7
backend/cmd/server/main.go:48-49
middleware.Recovererهذا جيد، لكنك تسجل البيانات في الإخراج القياسي فقط؛ لا توجد سجلات منظمة.
التحويل إلى log/slogJSON؛ إضافة وسيط معرف الطلب؛ الربط بنظام إدارة معلومات الأمان (SIEM) (مثل Elastic أو Loki). يتطلب التدقيق الأمني ​​عالي المستوى هذا الإجراء.

W8
backend/internal/handlers/handlers.go: RotateKey
DELETE FROM ssh_keys WHERE client_id=$1— يحذف السجل بشكل دائمrotated_at ، بما في ذلك الطابع الزمني السابق اللازم للتحليل الجنائي.
الحذف الناعم: أضف revoked_at، احتفظ بالسجل، لا تحذف أبدًا. ينص معيار NIST SP 800-57 الجزء 1 الإصدار 5 على الاحتفاظ بالمفاتيح الملغاة لأغراض التدقيق (NIST 24 ).

W9
agent/internal/agent/keys.go:27-44
يتطابق الاستبدال مع السلسلة التي تحتوي على علامة "motech-agent" - وهو أمر هش إذا قام أي شخص بتحرير الملف يدويًا.
قم بالمطابقة باستخدام نص المفتاح العام ( keyBody()الموجود لديك بالفعل sshd_windows.go:69-75). استبدل النص بنص المفتاح المحدد، وليس باستخدام وسم نصي حر.

W10
agent/internal/agent/agent.go:300-303
ترسل حمولة نبضات القلب peer_id(عنوان IP الخاص بجهاز NetBird) كمعرّف. ويمكن إعادة تعيين عناوين IP الخاصة بجهاز NetBird .
أرسل معرّف نظير NetBird (معرّف الكائن) ، الذي يتم جلبه مرة واحدة عند الانضمام، ويُخزّن مؤقتًا في ملف agent.json. قم بحلّ عنوان IP ← معرّف النظير على جانب الخادم (أنت تفعل ذلك بالفعل netbird.go:99-114) — فقط قم بتأجيل هذا الحلّ خطوة واحدة.

W11
agent/internal/agent/sshd_windows.go:18-32
كل استدعاء لـ PowerShell يدفع تكلفة بدء تشغيل PS التي تبلغ حوالي 300 مللي ثانية؛ على الأقراص البطيئة، يمكن أن يستغرق تسلسل الاستدعاءات الأربعة من 3 إلى 5 ثوانٍ.
قم بدمجها في -Commandكتلة نصية واحدة؛ أو الأفضل من ذلك، قم بإسقاط PowerShell واستدعاء واجهات برمجة التطبيقات الأصلية من Go لقاعدة جدار الحماية (golang.org/x/sys/windows + INetFwPolicy2COM) Add-WindowsCapabilityوعبر واجهة برمجة تطبيقات DISM.

W12
agent/internal/agent/netbird_install.go:55-58
/Sتثبيت NSIS الصامت بدون التحقق من توقيع برنامج التثبيت الذي تم تنزيله . في حالة تفعيل هجوم الوسيط (MITM)، https://pkgs.netbird.ioسيتمكن المهاجم من زرع برنامج تثبيت مُخترق يقوم البرنامج بتشغيله كمسؤول.
ثبّت قيمة SHA-256 الخاصة بالمثبّت في نظامك الخلفي (وفقًا لإصدار NetBird). مشاكل النظام الخلفي {url, sha256, signing_thumbprint}. يتحقق الوكيل من قيمة SHA-256 ويتحقق من بصمة توقيع Authenticode مقابل اسم ناشر NetBird المثبّت ( WireTrustee S.r.l./ NetBird GmbH). استخدم WinVerifyTrustعبر golang.org/x/sys/windows...

W13
backend/internal/handlers/handlers.go: Connection
يُعيد المفتاح الخاص المُفكّك إلى المسؤول /api/clients/:id/connection. يتم تسجيل الدخول إلى سجل النشاط، ولكن المفتاح نفسه ينتقل بتنسيق JSON عبر HTTPS.
كحد أدنى: قم بتصفير مخزن JSON المؤقت بعد الكتابة (لن يقوم جامع البيانات المهملة في Go بذلك، ولكن هذا crypto/subtleأسلوب مناسب). الأفضل: إرجاع رابط تنزيل لمرة واحدة ينتهي صلاحيته خلال 60 ثانية ويُستخدم لمرة واحدة فقط؛ يُقدّم الرابط مرفقًا application/x-pem-file، وليس JSON أبدًا.

W14
agent/internal/agent/service.go:123-156
InstallServiceيحاول النظام أولاً تنفيذ المهمة المجدولة؛ إذا نجح ذلك، يعود النظام؛ وفي حالة الفشل فقط، يلجأ إلى خدمة كارديانوس. لا تتم إعادة محاولة تنفيذ المهمة المجدولة.
أضف محاولة إعادة متكررة مع التراجع؛ سجل الاستراتيجية التي نجحت state.ServiceStrategyحتى يعرف برنامج إلغاء التثبيت ما يجب إزالته.

W15
جميع استدعاءات HTTP (agent.go، netbird.go، netbird_install.go)
لا يوجد إعادة محاولة HTTP/تراجع عند حدوث خطأ 5xx مؤقت؛ مهلة الوكيل الافتراضية البالغة 15 ثانية تعني أن الخادم المشغول خلال يوم الثلاثاء التصحيحي يسقط نصف نبضات القلب.
قم بتغليف http.Client بـ hashicorp/go-retryablehttpor cenkalti/backoff; أعد محاولة عمليات GET المتكررة بشكل غير مشروط؛ أعد محاولة عمليات POST فقط مع رأس idempotency-key.

W16
agent/internal/agent/agent.go:299-303
بيانات نبضات القلب فقط {peer_id, rotated_ok}، لا شيء آخر. لا يمكن للوحة التحكم عرض إصدار نظام التشغيل، أو إصدار الوكيل، أو حالة برنامج NetBird.
إضافة {agent_version, os_version, build_id, netbird_version, install_strategy, last_rotation_at}. من السهل إضافتها الآن، لكن من الصعب تعديلها عند وجود 1000 عميل.

W17
agent/internal/agent/sshd_windows.go:79-82
run()الاستخدامات silentCmd(...).CombinedOutput()جيدة، ولكن لا يوجد مهلة زمنية. سيؤدي تعليق PowerShell إلى تعليق التثبيت.
استخدم دائمًا silentCmdCtxمهلة زمنية قدرها 15 ثانية لكل أمر. قم بتطبيق ذلك على كل run()مكالمة.

W18
backend/internal/handlers/handlers.go: AgentRegister
يقوم التحقق من صحة رمز الإعداد بذلك WHERE token_hash=$1- وهو عرضة لثغرات التوقيت في مقارنة SHA-256 في PostgreSQL.
طول إدخال SHA-256 ثابت، لذا فإن التوقيت ثابت عمليًا، ولكن للاستخدام الصحي الكامل crypto/subtle.ConstantTimeCompareعلى جانب التطبيق بعد الجلب بواسطة البادئة المفهرسة.

W19
agent/internal/agent/agent.go:305-308
http.NewRequestيتجاهل الخطأ ( req, _ := ...). إذا كان غير صحيح (لا ينبغي أن يحدث ذلك، ولكن) ← محاولة الوصول إلى مؤشر فارغ.
تعامل مع كل خطأ بشكل صريح. errcheckيجب أن يكون برنامج التحقق من الأخطاء (linter) موجودًا في نظام التكامل المستمر (CI).

W20
لا توجد قناة تحديث
لا يملك البرنامج الوكيل أي وسيلة لتحديث نفسه. عند وجود 1000 مضيف، يصبح التحديث الذاتي ضروريًا.
أضف /api/agent/updateعملية إعادة التنزيل {version, url, sha256, signature}؛ يقوم الوكيل بتنزيل الملفات إلى مجلد مؤقت، والتحقق منها، واستبدالها بشكل ذري، وإعادة تشغيلها عبر الخدمة. هذا هو أكبر خطر تشغيلي تواجهه حاليًا.

## هـ. المعايير الدولية وأفضل الممارسات - تحليل الفجوات + خارطة طريق ذات أولوية

### E.1 - مقارنة شبكات VPN الشبكية: NetBird مقابل WireGuard مقابل Tailscale

اختيارك (الخطة المجانية لخدمة NetBird Cloud) له ما يبرره. إليك المقارنة الموضوعية:

- WireGuard وحده : تشفير من فئة النواة، لكن بدون أي نظام إدارة. ستحتاج إلى بناء NetBird بنفسك.

- Tailscale : يتميز بأفضل تجربة مستخدم، لكن الخطة المجانية محدودة بثلاثة مستخدمين لكل 100 جهاز، كما أن منسق النظام مغلق المصدر. تبلغ التكلفة عند استخدام 1000 جهاز حوالي 5000 دولار أمريكي سنويًا.

- NetBird : مفتوح المصدر، قابل للاستضافة الذاتية، طبقة مجانية (5 مستخدمين)، واجهة برمجة تطبيقات كاملة لأتمتة قوائم التحكم بالوصول (ACL). نقطة ضعف جوهرية واحدة: الطبقة السحابية المجانية أقدم من الإصدار 0.61 وتفتقر إلى protocol:"ssh"قوائم التحكم بالوصول الخاصة بمفاتيح الإعداد ([دروسك 2026-06-05]) - وهذا تحديدًا سبب اختيارك الصحيح لـ OpenSSH عبر شبكة NetBird كمسار أساسي.

- مصادر مقارنة الشبكات: defguard.net 25 ، wz-it.com 26 ، netbird.io/knowledge-hub 27 .

التوافق مع المعايير : تتوافق أنفاق NetBird WireGuard مع متطلبات التشفير الخاصة بمعيار NIST SP 800-77 Rev 1 (إرشادات IPsec/VPN) (X25519، ChaCha20-Poly1305). ✅

ثغرة : لا يوجد تكامل بين تسجيل الدخول الموحد (SSO) وموفر الهوية (IdP) في تصميمك. تتطلب المعايير (NIST 800-207 Zero Trust، SOC 2 CC6.1) هوية موحدة للوصول الإداري. اقتراح للتطوير: استخدام بروتوكول OIDC في Microsoft Entra / Google Workspace للمسؤولين ، مع الإبقاء على رمز JWT الخاص بالوكيل للأجهزة.

### E.2 — دورة حياة مفتاح SSH على نطاق واسع (NIST SP 800-57 + NIST IR 7966)

يحدد معيار NIST IR 7966 ("أمن إدارة الوصول التفاعلية والآلية باستخدام SSH") المعيار المطلوب. مقارنة الوضع الحالي بالوضع المستهدف:

يتحكم
وضعك الحالي
هدف

جرد
✅ قاعدة البيانات تحتوي على جميع المفاتيح
✅

تناوب
✅ تعمل آلة الحالة
✅

إبطال
⚠️ حذف الصف (W8)
حذف مؤقت + تدقيق revoked_at

مراجعة
⚠️ جدول سجل النشاط؛ لا يوجد ثبات في الجدول
تخزين WORM / سلسلة الإضافة فقط

الخوارزمية
✅ ed25519 ( keygen.go:21)
✅ — يفي بمعيار NIST 800-186

تخزين
✅ AES-256-GCM
ترقية KDF إلى HKDF (W1)

تاريخ الانتهاء
❌ لا تنتهي صلاحية المفاتيح أبدًا
إضافة expires_atعمود، تطبيق

اكتشاف
✅ مركزي
✅

منع التوسع العمراني العشوائي
✅ تم وضع علامة + تم استبدالها بالعلامة
المطابقة حسب الجسم (W9)

### E.3 — تعزيز أمان برنامج تثبيت ويندوز

معيار
أنت
فجوة

موقع من قبل Authenticode PE
تم التوقيع ذاتياً اليوم (وفقاً لملف HANDOFF.md)
اشترِ شهادة OV أو (الأفضل) خدمة التوقيع الموثوق من Azure، بتكلفة تقريبية 10 دولارات شهريًا (learn.microsoft.com 28 )

EV مقابل OV
غير متوفر
لم يعد نظام EV يتجاوز ميزة SmartScreen منذ عام 2023 (Microsoft Learn 28 ) — لا تدفع مقابل نظام EV

مقارنة بين برنامج MSI وبرنامج EXE لتثبيت الملفات
إملف تنفيذى
إنشاء ملف WiX MSI للنشر الجماعي لـ GPO/SCCM/Intune - مطلوب على نطاق مزود خدمات الإدارة

بيان UAC
ضمنيًا عبر المشي
أضف requireAdministratorبيانًا صريحًا

مناسب لأنظمة التعرف التلقائي على الكلام
مجهول
مسار استثناء برنامج Document Defender:C:\Program Files\Motech\motech-connect.exe

التوقيع الموثوق من Azure (المعروف سابقًا باسم التوقيع الموثوق، ويُسمى الآن "توقيع القطع الأثرية"): حوالي 10 دولارات شهريًا، لا يتطلب رمزًا ماديًا، ويتكامل مع GitHub Actions، ويتم التحقق من الهوية بواسطة Microsoft. يدعم النظام التشغيلي من Linux عبر TrustedSigningAction الرسمي 29 وأيضًا عبر jsign (Stack Overflow 30 ). التحقق من الهوية: بطاقة هوية حكومية + فاتورة خدمات (hendrik-erz.de 31 ). هذه هي الخطوة ذات أعلى عائد استثمار يمكنك القيام بها لـ SmartScreen - ولكن السمعة لا تزال تتراكم مع التنزيلات، ولا يتم تجاوزها على الفور.

### هـ.4 - انعدام الثقة / NIST SP 800-207

- ✅ هوية الجهاز (agent_token، peer_id)

- ✅ تقسيم الشبكة إلى أجزاء صغيرة (شبكة NetBird الشبكية، وليست شبكة محلية مسطحة)

- ⚠️ التحقق المستمر (يتم ذلك عن طريق نبضات القلب؛ لا يوجد فحص لوضعية الجهاز - لا يوجد برنامج مكافحة فيروسات، أو BitLocker، أو مستوى تصحيح)

- ❌ الوصول في الوقت المناسب (يحصل المسؤول على رمز JWT دائم؛ لا يوجد كسر للزجاج مع الموافقة)

- ❌ إدارة الوصول المميز (لا يتم تسجيل جلسات SSH الخاصة بالمشغل)

### E.5 — تسجيل عمليات التدقيق (توقعات SOC 2 / ISO 27001)

لديك الآن activity_logأساس متين يتضمن: الممثل، والإجراء، ومعرّف العميل، والبيانات الوصفية، وتاريخ الإنشاء handlers.go:84-90. لاجتياز اختبار SOC 2 من النوع الثاني:

- عدم قابلية التغيير : سجل إلى وحدة تخزين WORM للإضافة فقط (قفل كائن S3، أو INSERT-ONLYدور PostgreSQL).

- مدة الاحتفاظ : 12 شهرًا كحد أدنى (قابلة للتكوين لكل مستأجر).

- السلامة : يتم استخدام سلسلة التجزئة لكل صف ( prev_hash || row_hash) بحيث يمكن اكتشاف التلاعب.

- التصدير : /api/audit/export?from=&to=إرجاع JSON أو CEF موقعة لنظام SIEM.

- يُسجّل سطح التغطية كل عملية نسخ لمعلومات الاتصال، وكل عملية وصول إلى المفتاح الخاص، وكل عملية تعطيل/حذف، وكل عملية تسجيل دخول وخروج. يتم تسجيل معظم عمليات تسجيل الخروج، باستثناء عمليات تسجيل الدخول الفاشلة.

### هـ.6 — خارطة طريق ذات أولوية لمدة 90 يومًا (نفذ بهذا الترتيب)

أولوية
غرض
جهد
انخفاض المخاطر

P0
قم بإنشاء بيئة اختبار من لينكس إلى ويندوز (الخيار 1 أو 2 في القسم ج). توقف عن استخدام نفق Cloudflare السريع.
يوم واحد
يقضي على فئة الإنذارات الكاذبة "تعطل ملف exe" بالكامل؛ ويفتح جميع الأعمال المستقبلية.

P0
قم ببناء كلا windowsguiالإصدارين consoleمن الوكيل؛ معالجات قياسية فارغة في silentCmd؛ التبديل إلى MSI لمثبت NetBird.
يوم واحد
يسد آخر فجوات "وميض النافذة".

P0
إصلاح ACL ← قائم على SID (W5 #1). يضيف ميزة أمان اللغة العربية.
ساعة واحدة
لا يعمل حالياً على نظام ويندوز المحلي.

P1
التحقق المسبق من مفتاح إعداد NetBird على الواجهة الخلفية قبل إرساله إلى الوكيل؛ إزالة الاعتماد على مهلة 45 ثانية كوضع فشل أساسي (المشكلة 2).
يومين
يزيل السبب الجذري للتثبيت والتوقف.

P1
7 نقاط VerifySSHReady(المسألة 3).
يوم واحد
يقضي على حالات الفشل الصامتة "المثبتة ولكن غير القابلة للوصول".

P1
قم بتحويل JWT إلى EdDSA + kidرأس + جدول تدوير المفاتيح (W2، W3).
3 أيام
لم يعد الكشف عن سر واحد يعني الكشف عن كل شيء.

P1
مفتاح AES مشتق من HKDF مع ملح لكل سجل + عمود الإصدار (W1).
يومين
يصبح نظام تدوير المفاتيح الرئيسية فعالاً، وليس كارثياً.

P2
التوقيع باستخدام Authenticode عبر Azure Trusted Signing في كل إصدار (E.3).
إعداد لمدة يوم واحد + التحقق من هوية مايكروسوفت من أسبوع إلى ثلاثة أسابيع
التعرض لتقنية SmartScreen في اليوم الأول.

P2
برنامج تثبيت WiX MSI مع بيان UAC + وثيقة نشر GPO/Intune (E.3).
أسبوع واحد
مطلوب لتنفيذ المشروع على نطاق واسع.

P2
الحذف الناعم + سجل المفاتيح (W8)؛ سلسلة تجزئة سجل التدقيق (E.5).
يومين
جاهزية مركز عمليات الأمن 2.

P2
قناة التحديث الذاتي للوكيل (W20).
أسبوع واحد
أكبر فجوة تشغيلية اليوم.

P3
مباشر golang.org/x/sys/windows/svcبدلاً من kardianos؛ التحقق الثنائي الموقّع من برنامج تثبيت NetBird (W12)؛ نقاط نهاية المصادقة ذات الحد الأقصى للمعدل (W6)؛ السجلات المنظمة (W7).
أسبوع واحد مجتمع
الدفاع المتعمق.

P3
Microsoft Entra / Google OIDC SSO لتسجيل دخول المسؤول (E.1).
أسبوعين
مطلوب لعملاء المؤسسات.

P3
وضع الجهاز في نبضات القلب (حالة AV، BitLocker، مستوى تصحيح نظام التشغيل) (E.4).
أسبوع واحد
التوافق مع مبدأ انعدام الثقة.

## التقييم الختامي

هذه منصة وصول عن بُعد فائقة الجودة، تتفوق على المتوسط، مُصممة خصيصًا لمزودي خدمات الإدارة (MSP): بنيتها سليمة، وقواعد بيانات الوصول المُدارة (ADRs) دقيقة، وآلية تدوير الحالة مُختبرة، وتحصين الجلسة 0 مُحكم، ونمط التوليد الصامت مُعتمد. كانت معظم حالات الفشل التي واجهتها ناتجة عن خلل في بيئة الاختبار ، وليست عيوبًا برمجية - فبرنامجك أكثر متانة من بيئة الاختبار. أصلح بيئة الاختبار (القسم ج)، وحسّن من إجراءات التشفير الأساسية (W1-W3، W12)، واشترِ خدمة التوقيع الموثوق، وستحصل على منتج MSP موثوق يدعم 1000 مضيف. الانتقال من 3 إلى 1000 عميل يتطلب تحسينًا تشغيليًا ، وليس إعادة تصميم البنية.

مراجعة من قبل كبير مهندسي الأنظمة/الأمن، 9 يونيو 2026

            

            
            

            
            
        

        
    

    
    
    النص الأصليتقييم هذه الترجمةسيتم استخدام ملاحظاتك وآرائك للمساعدة في تحسين "ترجمة Google".


