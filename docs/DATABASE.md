# DATABASE.md — قاعدة البيانات

PostgreSQL 16 قياسي. الوصول حصراً عبر `DATABASE_URL` (قابل للنقل: محلي → Supabase → VPS).
UUIDs عبر `gen_random_uuid()` (إضافة `pgcrypto`). Migrations في `backend/migrations/` تُطبَّق تلقائياً عند الإقلاع (idempotent بـ IF NOT EXISTS).

## ER Diagram
```mermaid
erDiagram
  admins {
    uuid id PK
    text email UK
    text password_hash
    text role
  }
  clients {
    uuid id PK
    text name
    text branch
    text status
    timestamptz last_seen
  }
  setup_tokens {
    uuid id PK
    uuid client_id FK
    text token_hash UK
    timestamptz used_at
    timestamptz expires_at
  }
  ssh_keys {
    uuid id PK
    uuid client_id FK
    text public_key
    bytea private_key_enc
    bool active
  }
  netbird_links {
    uuid client_id PK_FK
    text peer_id
    text setup_key_ref
    text group_id
  }
  activity_log {
    uuid id PK
    text actor
    uuid client_id FK
    text action
    jsonb metadata
  }
  clients ||--o{ setup_tokens : has
  clients ||--o{ ssh_keys : has
  clients ||--|| netbird_links : has
  clients ||--o{ activity_log : logs
```

## الجداول
| جدول | الغرض | ملاحظات |
|------|-------|---------|
| `admins` | مستخدمو اللوحة | كلمة السر bcrypt، لا تُخزّن أبداً كنص |
| `clients` | الأجهزة المُدارة | `status`: pending/online/offline/disabled |
| `setup_tokens` | رموز التفعيل (مرة واحدة) | تُخزَّن **hash فقط** (SHA-256)، `used_at` + `expires_at` |
| `ssh_keys` | مفاتيح SSH لكل عميل | `private_key_enc` = AES-256-GCM (اختياري، للـ SSH التقليدي) |
| `netbird_links` | ربط العميل بـ NetBird | peer/group/setup-key |
| `activity_log` | سجل التدقيق | `metadata` JSONB مرن |

## فهارس
- `idx_activity_created` على `activity_log(created_at DESC)`
- `idx_clients_status` على `clients(status)`

## النقل (Migration to Supabase/VPS)
1. صدّر: `pg_dump $OLD_DATABASE_URL > dump.sql`
2. استورد إلى الوجهة.
3. غيّر `DATABASE_URL` في `.env` فقط. لا تغيير في الكود.
