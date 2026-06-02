# Web OTP — Usage Guide

A Go API service that issues and verifies 6-digit email one-time passwords (OTPs) via Gmail, backed by TiDB. Deployable to Vercel or run locally.

---

## Table of Contents

- [Getting Credentials](#getting-credentials)
  - [TiDB Cloud connection string](#tidb-cloud-connection-string)
  - [Google Cloud — credentials.json](#google-cloud--credentialsjson)
- [Prerequisites](#prerequisites)
- [Environment Configuration](#environment-configuration)
- [Database Setup](#database-setup)
- [Gmail OAuth Setup](#gmail-oauth-setup)
- [Running Locally](#running-locally)
- [Deploying to Vercel](#deploying-to-vercel)
- [API Reference](#api-reference)
  - [Issue OTP](#1-issue-otp)
  - [Verify OTP](#2-verify-otp)
  - [Usage Statistics](#3-usage-statistics)
- [Frontend Pages](#frontend-pages)
- [Error Handling](#error-handling)

---

## Getting Credentials

### TiDB Cloud connection string

1. Go to [tidbcloud.com](https://tidbcloud.com) and sign in.
2. Open your cluster, then click **Connect**.
3. Choose **General** connection type.
4. Copy the connection parameters — you'll need **Host**, **Port**, **User**, **Password**, and **Database**.  
   Alternatively, copy the full **DSN** string if shown.
5. Download **`isrgrootx1.pem`** from the same dialog (or use the one in the repo root).

Set either `TIDB_DSN` or the individual `TIDB_HOST` / `TIDB_USER` / … variables in your `.env`.

---

### Google Cloud — credentials.json

You need an OAuth 2.0 Client ID of type **Desktop app** (not a Service Account).

1. Open [Google Cloud Console](https://console.cloud.google.com) and select (or create) a project.
2. **Enable the Gmail API**  
   Navigate to **APIs & Services → Library**, search for **Gmail API**, and click **Enable**.
3. **Configure the OAuth consent screen**  
   Go to **APIs & Services → OAuth consent screen**.  
   - Choose **External** (or Internal for Google Workspace orgs).  
   - Fill in App name, user support email, and developer contact email.  
   - Under **Scopes**, add `https://www.googleapis.com/auth/gmail.send`.  
   - Under **Test users**, add the Gmail address you'll send from.  
   - Save and continue through all steps.
4. **Create an OAuth 2.0 Client ID**  
   Go to **APIs & Services → Credentials → Create Credentials → OAuth client ID**.  
   - Application type: **Desktop app**.  
   - Give it a name (e.g. "web-otp local") and click **Create**.
5. **Download `credentials.json`**  
   Click the download icon next to your new Client ID.  
   Save the file as `credentials.json` in the project root.
6. **Generate `token.json`** — see [Gmail OAuth Setup](#gmail-oauth-setup) below.

---

## Prerequisites

- Go 1.21+
- A [TiDB Cloud](https://tidbcloud.com) cluster (or compatible MySQL database)
- A Google Cloud project with the Gmail API enabled and an OAuth 2.0 Client ID

---

## Environment Configuration

Copy `.env.example` to `.env` and fill in the values.

### TiDB — Option A: Full DSN (takes priority if set)

```env
TIDB_DSN=user:password@tcp(gateway01.ap-southeast-1.prod.aws.tidbcloud.com:4000)/mydb?tls=custom
```

### TiDB — Option B: Individual components

```env
TIDB_HOST=gateway01.ap-southeast-1.prod.aws.tidbcloud.com
TIDB_PORT=4000
TIDB_USER=your_user
TIDB_PASSWORD=your_password
TIDB_DATABASE=your_database
TIDB_CA_CERT=/path/to/isrgrootx1.pem   # or raw PEM content
```

> The `isrgrootx1.pem` file is included in the repo root for TiDB Cloud TLS verification.

### Gmail — Option A: JSON files from Google Cloud Console

```env
GMAIL_FROM_EMAIL=you@gmail.com
GMAIL_CREDENTIALS=/path/to/credentials.json   # or raw JSON content
GMAIL_TOKEN=/path/to/token.json               # or raw JSON content
```

### Gmail — Option B: Individual OAuth values

```env
GMAIL_FROM_EMAIL=you@gmail.com
GMAIL_CLIENT_ID=your_client_id
GMAIL_CLIENT_SECRET=your_client_secret
GMAIL_REFRESH_TOKEN=your_refresh_token
```

> All env vars that accept a file path also accept the raw file content directly — useful for Vercel environment variables where files aren't available.

---

## Database Setup

Run the schema once against your TiDB (or MySQL-compatible) database:

```bash
mysql -h <TIDB_HOST> -P <TIDB_PORT> -u <TIDB_USER> -p <TIDB_DATABASE> < schema.sql
```

The schema creates the `otps` table with indexes on `email`, `expires_at`, and `created_at`.

---

## Gmail OAuth Setup

You need a `token.json` containing a refresh token before the service can send email. Run the helper once on a machine with a browser:

```bash
go run ./cmd/gettoken -credentials credentials.json -out token.json
```

This opens a browser, completes the Google OAuth flow, and writes `token.json`. Then reference it in your `.env`:

```env
GMAIL_CREDENTIALS=credentials.json
GMAIL_TOKEN=token.json
```

For Vercel (no filesystem), paste the raw JSON content of each file into the env var instead of a path.

---

## Running Locally

```bash
# Install dependencies
go mod download

# Start the server (reads .env automatically)
go run ./cmd/server
```

The server listens on `:8080` by default. API endpoints are available at `http://localhost:8080/api/`.

---

## Deploying to Vercel

The `vercel.json` routes all `/api/*` requests to the serverless handler and exposes two frontend pages:

| URL path | File served |
|---|---|
| `/api/:path*` | `api/handler.go` (serverless) |
| `/view` | `public/view.html` |
| `/docs` | `public/docs.html` |

Set all environment variables in the Vercel project settings. Use raw JSON content (not file paths) for `GMAIL_CREDENTIALS` and `GMAIL_TOKEN`.

```bash
vercel deploy
```

---

## API Reference

Base URL: `/api`

All responses are JSON. Errors follow `{ "error": "message" }`.

---

### 1. Issue OTP

Generates a 6-digit OTP for an email address and sends it via Gmail. The OTP is valid for **1 hour**.

**`POST /api/otp`**

#### Request body

```json
{
  "email": "user@example.com",
  "session_token": "550e8400-e29b-41d4-a716-446655440000"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `email` | string | yes | Recipient email address |
| `session_token` | string | yes | Client-generated token (min 8 chars). Must be resent on verify to bind the OTP to this session. Use `crypto.randomUUID()` on the client. |

#### Success response `200`

```json
{ "message": "OTP sent to your email" }
```

#### Example — curl

```bash
curl -X POST https://your-app.vercel.app/api/otp \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "session_token": "550e8400-e29b-41d4-a716-446655440000"
  }'
```

#### Example — JavaScript (fetch)

```js
const sessionToken = crypto.randomUUID();

const res = await fetch('/api/otp', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ email, session_token: sessionToken }),
});
const data = await res.json();
// data.message === "OTP sent to your email"
```

---

### 2. Verify OTP

Checks whether the submitted OTP is valid. On success the OTP is **consumed** and cannot be reused.

**`POST /api/otp/verify`**

#### Request body

```json
{
  "email": "user@example.com",
  "otp": "482910",
  "session_token": "550e8400-e29b-41d4-a716-446655440000"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `email` | string | yes | Same email used when issuing |
| `otp` | string | yes | The 6-digit code the user entered |
| `session_token` | string | yes | The same token sent when issuing |

#### Success response `200`

```json
{ "valid": true }
```

`valid` is `false` if the OTP is wrong, expired, already used, or the session token does not match.

#### Example — curl

```bash
curl -X POST https://your-app.vercel.app/api/otp/verify \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "otp": "482910",
    "session_token": "550e8400-e29b-41d4-a716-446655440000"
  }'
```

#### Example — JavaScript (fetch)

```js
const res = await fetch('/api/otp/verify', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ email, otp: userInput, session_token: sessionToken }),
});
const { valid } = await res.json();
if (valid) {
  // proceed — user is authenticated
}
```

---

### 3. Usage Statistics

Returns summary counts and a paginated list of OTP records. Supports date range and email filtering.

**`GET /api/stats`**

#### Query parameters

| Parameter | Type | Default | Description |
|---|---|---|---|
| `from` | ISO 8601 datetime | — | Include records created on or after this time |
| `to` | ISO 8601 datetime | — | Include records created on or before this time |
| `email` | string | — | Partial match filter on email address |
| `limit` | integer | `20` | Records per page (1–100) |
| `offset` | integer | `0` | Records to skip (for pagination) |

#### Success response `200`

```json
{
  "summary": {
    "total_issued":   142,
    "total_verified": 98,
    "total_expired":  31,
    "total_pending":  13
  },
  "records": [
    {
      "id": 142,
      "email": "user@example.com",
      "created_at": "2026-06-02T12:00:00Z",
      "expires_at": "2026-06-02T13:00:00Z",
      "used": true,
      "used_at": "2026-06-02T12:04:33Z"
    }
  ],
  "total": 142
}
```

> The OTP code and session token are never returned by the stats endpoint.

#### Example — curl (all records today, filtered by domain)

```bash
curl "https://your-app.vercel.app/api/stats?from=2026-06-02T00:00:00Z&email=example.com&limit=50"
```

#### Example — paginating through all records

```js
async function fetchAllStats(base) {
  const limit = 100;
  let offset = 0;
  let all = [];
  while (true) {
    const res = await fetch(`${base}/api/stats?limit=${limit}&offset=${offset}`);
    const { records, total } = await res.json();
    all = all.concat(records);
    offset += limit;
    if (offset >= total) break;
  }
  return all;
}
```

---

## Frontend Pages

| URL | Description |
|---|---|
| `/docs` | Interactive Swagger UI for all API endpoints |
| `/view` | Simple OTP stats dashboard (summary + record list) |

---

## Error Handling

All API errors return an appropriate HTTP status code and a JSON body:

```json
{ "error": "human-readable message" }
```

| Status | Cause |
|---|---|
| `400` | Missing or invalid field (e.g. bad email format, `session_token` under 8 chars) |
| `500` | Database unavailable, Gmail API failure, or other server-side error |

OTP verification failures (wrong code, expired, already used, token mismatch) return **`200`** with `{ "valid": false }` — not an error status.
