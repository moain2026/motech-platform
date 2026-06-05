# DEPLOYMENT.md — دليل النشر

## التطوير/العرض الحالي (LIVE)
- Backend خدمة systemd دائمة على :8080، PostgreSQL محلي.
- NetBird في **live mode** (PAT في .env).
- **اللوحة منشورة على:** https://qfetmfdn.gensparkclaw.com (عبر Caddy reverse_proxy + HTTPS تلقائي).
- ملف Caddy: `/etc/caddy/conf.d/custom.caddy`.
- كلمة سر الأدمن الافتراضية **تم تغييرها** (محفوظة في SEED_ADMIN_PASSWORD داخل .env).
- ⚠️ للإنتاج: أضف rate-limiting على login/register، وراجع SECURITY.md.

## الإنتاج (مستقبلاً — VPS خاص)
1. **VPS**: ثبّت Go + PostgreSQL (أو DB مُدارة عبر `DATABASE_URL`).
2. **Backend**: ابنِ ثنائي `go build -o server ./cmd/server`، شغّله كخدمة systemd.
3. **HTTPS**: Caddy reverse proxy على دومينك → `127.0.0.1:8080`.
4. **NetBird**: انشر Self-Hosted، ثم `NETBIRD_API_URL` + `NETBIRD_API_TOKEN`.
5. **الأسرار**: secret manager لـ JWT_SECRET / MASTER_KEY / NETBIRD token.

### نموذج خدمة systemd
```ini
[Unit]
Description=Motech Backend
After=network.target postgresql.service

[Service]
WorkingDirectory=/opt/motech/backend
EnvironmentFile=/opt/motech/backend/.env
ExecStart=/opt/motech/backend/server
Restart=always

[Install]
WantedBy=multi-user.target
```

### Caddy
```
your-domain.com {
    reverse_proxy 127.0.0.1:8080
}
```

## توزيع الـ Agent
- ابنِ `motech-connect.exe` (cross-compile من Linux).
- **وقّعه** (code-signing cert) لتفادي تحذير SmartScreen.
- وزّعه للعملاء مع رمز التفعيل لكل جهاز.
