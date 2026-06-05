# ARCHITECTURE.md — البنية المعمارية

## المكوّنات

```mermaid
flowchart LR
  subgraph Admin
    D[Dashboard<br/>HTML+Tailwind+Alpine]
  end
  subgraph Server[Go Backend :8080]
    API[chi Router + JWT]
    NBC[NetBird Client<br/>switchable URL]
  end
  DB[(PostgreSQL)]
  NB[NetBird<br/>Cloud / Self-hosted]
  subgraph Client[جهاز العميل - Windows]
    AG[Agent .exe<br/>Windows Service]
  end

  D -- HTTPS + JWT(admin) --> API
  API <--> DB
  API -- REST + Token --> NBC --> NB
  AG -- register/heartbeat (JWT agent) --> API
  AG -- joins mesh (setup key) --> NB
```

## التدفقات الرئيسية

### إضافة عميل
```mermaid
sequenceDiagram
  Admin->>API: POST /api/clients
  API->>DB: INSERT client + setup_token(hash) + ssh_key
  API->>NetBird: CreateSetupKey()
  API->>DB: INSERT netbird_link
  API-->>Admin: setup_token (مرة واحدة) + netbird key
```

### تسجيل الـ Agent
```mermaid
sequenceDiagram
  Agent->>API: POST /api/agent/register {token}
  API->>DB: تحقق hash + لم يُستخدم + لم ينتهِ
  API->>DB: mark used + status=online
  API-->>Agent: agent_token (JWT) + netbird setup key
  Agent->>NetBird: ينضم للشبكة بالـ setup key
  loop كل 20s
    Agent->>API: POST /api/agent/heartbeat
    API-->>Agent: {disabled, rotate}
  end
```

## مبادئ التصميم
- **Portability:** DB عبر `DATABASE_URL`، NetBird عبر `NETBIRD_API_URL` — تبديل بمتغيّر واحد.
- **Stateless API:** JWT، بلا session store → يتوسّع أفقياً.
- **Poll بدل Push:** الـ Agent يسحب الأوامر كل 20s (بسيط ومتين)؛ القطع الفوري يتم عبر NetBird API مباشرة.
- **Layered access:** NetBird ACLs أساسي + SSH تقليدي ثانوي (مفتاح مشفّر في DB).

## الحزم (Go packages)
| package | المسؤولية |
|---------|-----------|
| `config` | تحميل البيئة (fail-fast على DATABASE_URL) |
| `db` | اتصال sqlx + تشغيل migrations |
| `models` | كيانات قاعدة البيانات |
| `auth` | JWT + bcrypt + middleware + AES-GCM |
| `netbird` | عميل NetBird (mock/live) |
| `handlers` | كل الـ endpoints |
