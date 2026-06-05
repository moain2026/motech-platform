# TROUBLESHOOTING.md — حل المشاكل الشائعة

> صيغة كل مدخل: **المشكلة** + **السبب** + **الحل** + **نصيحة للتجنّب**.

---

## DB-001: لا يمكن الاتصال بـ PostgreSQL محلياً
- **المشكلة:** `connection refused` على `127.0.0.1:5432`.
- **السبب:** الخدمة متوقفة، أو peer auth بدل password.
- **الحل:** `sudo systemctl start postgresql` + استخدم `DATABASE_URL=postgres://motech:...@127.0.0.1:5432/motech_platform?sslmode=disable`.
- **تجنّب:** اتصل دائماً عبر TCP (127.0.0.1) لا عبر socket، ليطابق سلوك الإنتاج.

---

## ENV-001: متغيرات البيئة الأساسية
- القيم المطلوبة: `DATABASE_URL`, `NETBIRD_API_URL`, `NETBIRD_API_TOKEN`, `JWT_SECRET`, `MASTER_KEY`, `PORT`.
- انسخ `backend/.env.example` إلى `backend/.env`.

---

> أضف كل مشكلة جديدة هنا فور حلّها.
