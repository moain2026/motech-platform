# SETUP.md — دليل التثبيت والتشغيل

## المتطلبات
- Go 1.23+
- PostgreSQL 16+

## 1) قاعدة البيانات
```bash
sudo -u postgres psql -c "CREATE USER motech WITH PASSWORD 'motech_dev_2026';"
sudo -u postgres psql -c "CREATE DATABASE motech_platform OWNER motech;"
```

## 2) إعداد البيئة
```bash
cd backend
cp .env.example .env
# عدّل القيم — على الأقل DATABASE_URL و JWT_SECRET
```
المتغيرات:
| المتغير | الوصف |
|---------|-------|
| `DATABASE_URL` | سلسلة اتصال PostgreSQL قياسية |
| `NETBIRD_API_URL` | `https://api.netbird.io` (cloud) أو self-hosted |
| `NETBIRD_API_TOKEN` | PAT (فارغ = mock mode) |
| `JWT_SECRET` | سر توقيع JWT (طويل وعشوائي) |
| `MASTER_KEY` | مفتاح AES-256-GCM للمفاتيح المخزّنة |
| `PORT` | منفذ HTTP (افتراضي 8080) |
| `SEED_ADMIN_EMAIL` / `SEED_ADMIN_PASSWORD` | أول أدمن يُنشأ تلقائياً |

## 3) التشغيل
```bash
export PATH=$PATH:/usr/local/go/bin
cd backend
go run ./cmd/server      # أو: go build -o server ./cmd/server && ./server
```
عند الإقلاع: يطبّق migrations + ينشئ أدمن افتراضي (لو لا يوجد).

## 4) الوصول
- Dashboard: http://localhost:8080
- Health: http://localhost:8080/health
- دخول افتراضي: `admin@motech.local` / `admin123` → **غيّرها فوراً**.

## 5) بناء الـ Agent لويندوز (لاحقاً)
```bash
cd agent
GOOS=windows GOARCH=amd64 go build -o motech-connect.exe ./cmd/agent
```

## النشر خلف Caddy (دومين عام)
```
qfetmfdn.gensparkclaw.com {
    reverse_proxy 127.0.0.1:8080
}
```
ضعه في `/etc/caddy/conf.d/custom.caddy`.
