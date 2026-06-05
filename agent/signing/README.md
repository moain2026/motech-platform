# Code Signing — Motech / Al-Abbasi Soft

## ما هذا
نوقّع الـ `.exe` بتوقيع Authenticode عشان ويندوز يعرض الناشر **Al-Abbasi Soft**
بدل **Unknown Publisher**، وعشان نمنع تعديل الملف بعد البناء.

## الملفات (بنية سلسلة من طبقتين — الصحيحة)
- `ca.crt` — **Root CA** (CA:TRUE). هذي اللي تُوزَّع على الأجهزة لتثق بالتوقيع. **في git**.
- `ca.key` — مفتاح الـ Root CA الخاص. **gitignored — لا يُرفع أبداً**.
- `motech-codesign.crt` — شهادة التوقيع (leaf, CA:FALSE, codeSigning) موقّعة من الـ CA. **في git**.
- `motech-codesign.key` — مفتاح التوقيع الخاص. **gitignored**.
- `motech-codesign-chain.crt` — leaf + ca مجمّعة (build.sh يضمّن السلسلة كاملة). **في git**.
- `cert.pfx` — (اختياري) شهادة CA حقيقية (OV/EV) تأخذ الأولوية مع `MOTECH_PFX_PASS`. **gitignored**.

> **مهم**: توقيع self-signed بطبقة واحدة يفشل برسالة "basic constraint extension has not been observed". الحل الصحيح (المطبّق) = Root CA منفصل يوقّع leaf. مُثبَت live: بعد استيراد ca.crt → Status=Valid "Signature verified".

## مستويات الثقة
| التوقيع | النتيجة على ويندوز |
|---------|---------------------|
| بدون توقيع | "Unknown Publisher" + تحذير SmartScreen أحمر |
| **self-signed (الحالي)** | يعرض اسم الناشر "Al-Abbasi Soft"؛ SmartScreen لا يزال يحذّر حتى تُبنى السمعة. لإزالة التحذير محلياً: ثبّت `motech-codesign.crt` في *Trusted Root* + *Trusted Publishers* عبر GPO على أجهزة الفروع. |
| **OV cert (CA)** | لا "Unknown Publisher"؛ SmartScreen يهدأ بعد بناء سمعة. ~200-400$/سنة. |
| **EV cert (CA)** | ثقة فورية بلا انتظار سمعة. أغلى + يحتاج HSM/USB token. |

## التوزيع على الفروع (لإزالة التحذير)
وزّع **ca.crt** (الـ Root CA) مرة واحدة (أو عبر Group Policy):
```powershell
Import-Certificate -FilePath ca.crt -CertStoreLocation Cert:\LocalMachine\Root
Import-Certificate -FilePath ca.crt -CertStoreLocation Cert:\LocalMachine\TrustedPublisher
```
بعدها كل الـ exe الموقّعة تُعتبر موثوقة بالكامل (Status=Valid). **مُختبَر live ✓**.

## الترقية لشهادة CA لاحقاً
1. اشترِ OV/EV code-signing cert (Sectigo / DigiCert / SSL.com…).
2. صدّرها كـ `cert.pfx` وضعها هنا.
3. `export MOTECH_PFX_PASS='...'` ثم `./build.sh` — يستخدمها تلقائياً مع timestamp.
