# Phase 1 API contract (implemented)

This document is the **authoritative contract** between the **Windows endpoint agent**, the **backend** (Fastify), and the **dashboard** for Phase 1.

**Base URL:** configurable (e.g. `http://localhost:3001` or `http://YOUR_HOST:3001`). Paths below are **without** a `/v1` prefix.

**Conventions**

- Request and response bodies are **JSON** with `Content-Type: application/json`.
- Timestamps in responses are **RFC 3339** strings in **UTC** (e.g. `2026-03-24T06:11:07.109Z`).
- **Phase 1 does not implement authentication.** All endpoints are open unless you add a reverse proxy or future auth.

**Elevation request `status` values (only these):**

| Value       | Meaning |
| ----------- | ------- |
| `pending`   | Awaiting technician approve/deny. |
| `approved`  | Technician approved; agent may proceed. |
| `denied`    | Technician denied; agent must not elevate. |

**State lifecycle**

1. `POST /agent/elevation-request` creates a row with `status: "pending"`.
2. `POST /admin/approve` transitions **only** `pending` → `approved` (sets `resolved_at`).
3. `POST /admin/deny` transitions **only** `pending` → `denied` (sets `resolved_at`).
4. No API in Phase 1 moves a request back to `pending` or changes `approved`/`denied`.

**Errors (Phase 1 shapes)**

- Validation failure: `400` with `{ "error": <Zod flatten object> }`.
- Missing resource: `404` with `{ "error": "not found" }` or `{ "error": "device not found" }`.
- Invalid transition: `409` with `{ "error": "request not found or not pending" }`.

---

## Debugging / connectivity

### GET /health

**Response `200 OK`:**

```json
{
  "ok": true
}
```

### GET /version

**Response `200 OK`:**

```json
{
  "ok": true,
  "service": "pam-backend",
  "version": "0.1.0",
  "phase": 1
}
```

(`version` comes from `backend/package.json`.)

---

## Agent endpoints

### POST /agent/register

Registers or updates a device by `hostname` (upsert on unique hostname).

**Request JSON:**

```json
{
  "hostname": "DESKTOP-ABC",
  "agent_version": "0.1.0"
}
```

`agent_version` may be omitted (stored as empty string).

**Response `200 OK`:**

```json
{
  "deviceId": "90ded635-68ed-416d-9e8c-b837b7565696"
}
```

`deviceId` is a UUID string. The agent must persist it for subsequent calls.

---

### POST /agent/elevation-request

Creates an elevation request in `pending` state.

**Request JSON:**

```json
{
  "device_id": "90ded635-68ed-416d-9e8c-b837b7565696",
  "user": "CONTOSO\\alice",
  "exe_path": "C:\\Windows\\System32\\cmd.exe",
  "hash": "sha256-or-pe-hash-string",
  "publisher": "Microsoft Corporation"
}
```

- `device_id`: UUID from `POST /agent/register`.
- `user`: Windows username (string).
- `exe_path`: full path to the executable.
- `hash`, `publisher`: optional in schema but should be sent; may be empty strings.

**Response `200 OK`:**

```json
{
  "id": "b008d3ab-c3d4-42f5-897c-14dad70b6c1a",
  "status": "pending"
}
```

**Response `404`:** unknown `device_id`:

```json
{
  "error": "device not found"
}
```

---

### GET /agent/elevation-requests/:id

Poll this URL until `status` is `approved` or `denied`.

**Path:** `:id` = UUID from `POST /agent/elevation-request` (`id` field).

**Response `200 OK`:**

```json
{
  "id": "b008d3ab-c3d4-42f5-897c-14dad70b6c1a",
  "device_id": "90ded635-68ed-416d-9e8c-b837b7565696",
  "user": "CONTOSO\\alice",
  "exe_path": "C:\\Windows\\System32\\cmd.exe",
  "hash": "abc123",
  "publisher": "Microsoft Corporation",
  "status": "approved",
  "created_at": "2026-03-24T06:11:07.109Z",
  "resolved_at": "2026-03-24T06:11:07.189Z",
  "device_hostname": "DESKTOP-ABC"
}
```

While waiting, `status` is `pending` and `resolved_at` is `null`.

**Response `404`:**

```json
{
  "error": "not found"
}
```

---

### GET /agent/policy

**Optional query:** `deviceId=<uuid>` (ignored in Phase 1 logic; reserved for future per-device policy).

**Response `200 OK`:** policy document (JSON object). Default seed includes:

```json
{
  "allowed_publishers": ["Microsoft Corporation"]
}
```

Exact keys may grow over time; agents should treat unknown keys as optional.

---

## Admin / dashboard endpoints

### GET /admin/requests

**Query:** `status` optional.

- Omitted or `pending`: only `pending` rows.
- `all`: all statuses.
- Otherwise: filter by that exact `status` (`approved`, `denied`, etc.).

**Response `200 OK`:**

```json
{
  "requests": [
    {
      "id": "b008d3ab-c3d4-42f5-897c-14dad70b6c1a",
      "device_id": "90ded635-68ed-416d-9e8c-b837b7565696",
      "user": "CONTOSO\\alice",
      "exe_path": "C:\\Windows\\System32\\cmd.exe",
      "hash": "abc123",
      "publisher": "Microsoft Corporation",
      "status": "pending",
      "created_at": "2026-03-24T06:11:07.109Z"
    }
  ]
}
```

Each item **always** includes exactly these fields: `id`, `device_id`, `user`, `exe_path`, `hash`, `publisher`, `status`, `created_at`.

---

### POST /admin/approve

**Request JSON:**

```json
{
  "requestId": "b008d3ab-c3d4-42f5-897c-14dad70b6c1a"
}
```

**Response `200 OK`:**

```json
{
  "ok": true,
  "status": "approved"
}
```

**Response `409`:** not pending or unknown id (same message for both in Phase 1):

```json
{
  "error": "request not found or not pending"
}
```

---

### POST /admin/deny

**Request JSON:**

```json
{
  "requestId": "04715128-5f82-4b5c-a742-d418d2cc5a9b"
}
```

**Response `200 OK`:**

```json
{
  "ok": true,
  "status": "denied"
}
```

**Response `409`:** same as approve.

---

### GET /admin/audit-logs

Returns recent audit rows (newest first, limit 200). Phase 1 storage model:

| Logical field   | Source |
| --------------- | ------ |
| `timestamp`     | `created_at` column |
| `action`        | `action` column |
| `request_id`    | `entity_id` when `entity_type` is `elevation_request`, and duplicated in `metadata.request_id` for approve/deny |
| `admin_user`    | `actor` column for human actions; also `metadata.admin_user` on approve/deny |
| `decision`      | `metadata.decision` (`approved` / `denied`) for approve/deny events |

**Example row (approve) `metadata`:**

```json
{
  "request_id": "b008d3ab-c3d4-42f5-897c-14dad70b6c1a",
  "admin_user": "admin",
  "decision": "approved",
  "device_id": "90ded635-68ed-416d-9e8c-b837b7565696"
}
```

Set `PAM_ADMIN_USER` in the backend environment to change the logged `admin_user` / `actor` for approve/deny (Phase 1 has no login).

---

## Structured server logs (Phase 1)

In addition to Fastify access logs, the backend emits **JSON log lines** (level `info`) with these shapes:

| Event                      | Fields |
| -------------------------- | ------ |
| `device_registered`        | `endpoint`, `device_id` |
| `elevation_request_created`| `endpoint`, `request_id`, `device_id` |
| `elevation_request_decided`| `endpoint`, `request_id`, `device_id`, `decision` (`approved` / `denied`) |

---

## Out of scope for Phase 1

Not implemented: authentication, multi-tenancy, notifications, WebSockets, policy editing UI, `/v1` prefix, idempotency keys, and the broader future API that previously appeared in early drafts of this file. Use this document only for paths and payloads above.
