# Code Signing — Motech / Al-Abbasi Soft

## ما هذا
نوقّع الـ `.exe` بتوقيع Authenticode عشان ويندوز يعرض الناشر **Al-Abbasi Soft**
بدل **Unknown Publisher**، وعشان نمنع تعديل الملف بعد البناء.

## الملفات
- `motech-codesign.crt` — الشهادة العامة (self-signed، صالحة 10 سنوات). **تُحفظ في git**.
- `motech-codesign.key` — المفتاح الخاص. **gitignored — لا يُرفع أبداً**.
- `cert.pfx` — (اختياري) شهادة CA حقيقية (OV/EV). لو وُجدت يستخدمها `build.sh` تلقائياً
  مع `MOTECH_PFX_PASS`. **gitignored**.

## مستويات الثقة
| التوقيع | النتيجة على ويندوز |
|---------|---------------------|
| بدون توقيع | "Unknown Publisher" + تحذير SmartScreen أحمر |
| **self-signed (الحالي)** | يعرض اسم الناشر "Al-Abbasi Soft"؛ SmartScreen لا يزال يحذّر حتى تُبنى السمعة. لإزالة التحذير محلياً: ثبّت `motech-codesign.crt` في *Trusted Root* + *Trusted Publishers* عبر GPO على أجهزة الفروع. |
| **OV cert (CA)** | لا "Unknown Publisher"؛ SmartScreen يهدأ بعد بناء سمعة. ~200-400$/سنة. |
| **EV cert (CA)** | ثقة فورية بلا انتظار سمعة. أغلى + يحتاج HSM/USB token. |

## التوزيع على الفروع (لإزالة التحذير مع self-signed)
على أجهزة العملاء (أو عبر Group Policy):
```powershell
Import-Certificate -FilePath motech-codesign.crt -CertStoreLocation Cert:\LocalMachine\Root
Import-Certificate -FilePath motech-codesign.crt -CertStoreLocation Cert:\LocalMachine\TrustedPublisher
```
بعدها التطبيق الموقّع يُعتبر موثوقاً بالكامل على تلك الأجهزة.

## الترقية لشهادة CA لاحقاً
1. اشترِ OV/EV code-signing cert (Sectigo / DigiCert / SSL.com…).
2. صدّرها كـ `cert.pfx` وضعها هنا.
3. `export MOTECH_PFX_PASS='...'` ثم `./build.sh` — يستخدمها تلقائياً مع timestamp.
