# web-otp

> **Stateless email OTP service** — issue, verify, and audit one-time passwords via a clean REST API.  
> Built with Go · TiDB Cloud · Gmail API · Deployable to Vercel in one command.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go&logoColor=white)
![TiDB](https://img.shields.io/badge/TiDB-Cloud-E20B34?style=flat&logo=pingcap&logoColor=white)
![Vercel](https://img.shields.io/badge/Vercel-Serverless-000000?style=flat&logo=vercel&logoColor=white)
![Gmail API](https://img.shields.io/badge/Gmail-API-EA4335?style=flat&logo=gmail&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-blue?style=flat)

---

## What it does

A backend service that handles the full OTP lifecycle for email-based authentication:

1. **Issue** — generates a cryptographically random 6-digit code, persists it with a 1-hour TTL, and delivers it to the user's inbox via the Gmail API.
2. **Verify** — validates the code against the stored record, enforcing session binding, expiry, and single-use semantics in a single atomic update.
3. **Audit** — exposes a filterable, paginated stats endpoint with summary counters (issued / verified / expired / pending) for operational visibility.

---

## Architecture

```
Client
  │
  ▼
POST /api/otp          →  handlers.IssueOTP
  │  validate email + session_token
  │  otp.Issue()  →  crypto/rand 6-digit code  →  TiDB (otps table)
  └─ email.SendOTP()  →  Gmail API (OAuth2 refresh flow)

POST /api/otp/verify   →  handlers.VerifyOTP
  │  otp.Verify()  →  SELECT WHERE email + code + session_token + !used + !expired
  └─ UPDATE used=true, used_at=NOW()  →  returns { valid: bool }

GET  /api/stats        →  handlers.Stats
     filter by from/to/email  →  summary counts + paginated records
```

**Session token binding** prevents OTP replay across devices: the same token generated at issue time must be presented at verify time. A mismatch returns `valid: false` even if the code is correct.

---

## Tech stack

| Layer | Choice | Why |
|---|---|---|
| Web framework | [Fiber v2](https://gofiber.io) | FastHTTP core, zero-alloc routing, Vercel-compatible |
| ORM | [GORM](https://gorm.io) | Auto-migration, clean query DSL, TiDB-compatible |
| Database | [TiDB Cloud](https://tidbcloud.com) | MySQL-compatible, serverless, globally distributed |
| Email | [Gmail API](https://developers.google.com/gmail/api) via OAuth2 | No SMTP credentials, scoped send-only access |
| Deploy | [Vercel](https://vercel.com) | Zero-infra, instant global CDN, env var injection |

---

## Project structure

```
.
├── api/
│   └── handler.go          # Vercel serverless entrypoint
├── cmd/
│   ├── server/main.go      # Local HTTP server
│   └── gettoken/main.go    # One-time Gmail OAuth2 token helper
├── internal/
│   ├── app/app.go          # Fiber app + route registration (singleton)
│   ├── handlers/
│   │   ├── issue.go        # POST /api/otp
│   │   ├── verify.go       # POST /api/otp/verify
│   │   ├── stats.go        # GET  /api/stats
│   │   └── helpers.go      # Shared validation (email regex)
│   ├── otp/otp.go          # Issue + Verify business logic
│   ├── email/gmail.go      # Gmail API client (dual credential modes)
│   ├── db/tidb.go          # TiDB connection (singleton, TLS)
│   └── models/otp.go       # GORM model + table mapping
├── public/
│   ├── openapi.json        # OpenAPI 3.0 spec
│   ├── docs.html           # Swagger UI
│   └── view.html           # Stats dashboard
├── schema.sql              # Table DDL (TiDB / MySQL)
├── vercel.json             # Rewrite rules
└── .env.example            # All env vars with documentation
```

---

## API

Full interactive docs available at `/docs` (Swagger UI).

### `POST /api/otp` — Issue

```bash
curl -X POST https://your-app.vercel.app/api/otp \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","session_token":"550e8400-e29b-41d4-a716-446655440000"}'
# → {"message":"OTP sent to your email"}
```

### `POST /api/otp/verify` — Verify

```bash
curl -X POST https://your-app.vercel.app/api/otp/verify \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","otp":"482910","session_token":"550e8400-e29b-41d4-a716-446655440000"}'
# → {"valid":true}
```

### `GET /api/stats` — Audit

```bash
curl "https://your-app.vercel.app/api/stats?from=2026-01-01T00:00:00Z&email=@example.com&limit=50"
```

```json
{
  "summary": { "total_issued": 142, "total_verified": 98, "total_expired": 31, "total_pending": 13 },
  "records": [{ "id": 142, "email": "user@example.com", "created_at": "...", "used": true }],
  "total": 142
}
```

---

## Quick start

### 1. Clone and configure

```bash
git clone https://github.com/krispaisarn/web-otp.git
cd web-otp
cp .env.example .env
# fill in TIDB_* and GMAIL_* values — see usage.md for step-by-step credential guides
```

### 2. Run the database schema

```bash
mysql -h <TIDB_HOST> -P 4000 -u <TIDB_USER> -p <DB> < schema.sql
```

### 3. Authorize Gmail (one-time)

```bash
go run ./cmd/gettoken -credentials credentials.json -out token.json
```

### 4. Start the server

```bash
go run ./cmd/server
# API available at http://localhost:8080/api/
```

### Deploy to Vercel

```bash
vercel deploy
```

Set all env vars in the Vercel dashboard. Use raw JSON content for `GMAIL_CREDENTIALS` and `GMAIL_TOKEN` (no filesystem on serverless).

---

## Security design

- **Cryptographic OTP generation** — `crypto/rand` with modular reduction over 10⁶, no `math/rand`.
- **Session token binding** — OTP is tied to the token used at issue; stolen codes can't be replayed from a different session.
- **Single-use enforcement** — `used` flag is set atomically on first successful verify; re-submission returns `valid: false`.
- **Hard expiry** — `expires_at` checked in SQL (`expires_at > NOW()`), not application code, eliminating clock drift risk.
- **Secret never returned** — the `otp` and `session_token` columns are excluded from all JSON output via `json:"-"` tags.
- **Scoped OAuth2** — Gmail client requests only `gmail.send` scope; no read or full-account access.

---

## Environment variables

See [`.env.example`](.env.example) and [`usage.md`](usage.md) for the complete reference, including step-by-step instructions for obtaining TiDB and Google Cloud credentials.

---

## License

MIT
