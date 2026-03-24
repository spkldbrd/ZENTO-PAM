# ARCHITECTURE.md

Deep technical architecture for the PAM platform: Windows agent (Go service), backend (Fastify/TypeScript), dashboard (Next.js), and infrastructure (Docker Compose on Linux VPS).

---

## 1. High-level system diagram

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                           Windows endpoint                                    │
│  ┌────────────────────┐         named pipe          ┌────────────────────────┐ │
│  │ User session       │  JSON-RPC or framed msgs   │ Windows Service        │ │
│  │ (standard user)    │ ─────────────────────────► │ (LocalSystem broker)   │ │
│  │ Optional UI/helper │ ◄───────────────────────── │ Policy cache, launcher │ │
│  └─────────┬──────────┘                             └───────────┬────────────┘ │
│            │                                                     │              │
└────────────┼─────────────────────────────────────────────────────┼────────────┘
             │                                                     │
             │                     HTTPS (mTLS optional future)      │
             └─────────────────────────────────────────────────────┘
                                        │
                                        ▼
                             ┌──────────────────────┐
                             │  Load balancer / TLS │
                             │  (VPS, e.g. Caddy)   │
                             └──────────┬───────────┘
                                        │
                    ┌───────────────────┼───────────────────┐
                    ▼                   ▼                   ▼
             ┌─────────────┐      ┌─────────────┐    ┌─────────────┐
             │  Fastify API│      │  PostgreSQL │    │   Redis     │
             │  (Node/TS)  │      │  (system of │    │ rate limit, │
             └──────┬──────┘      │   record)   │    │ sessions,   │
                    │             └─────────────┘    │ pub/sub?    │
                    │                                └─────────────┘
                    ▼
             ┌─────────────┐
             │  Next.js    │
             │  dashboard  │
             └─────────────┘
```

---

## 2. Component responsibilities

### 2.1 Windows endpoint agent

| Subcomponent | Privilege | Responsibility |
|--------------|-----------|----------------|
| **Service (broker)** | LocalSystem | Named-pipe server; validates client; applies policy; performs elevated `CreateProcessAsUser` / staged launch per `SECURITY_MODEL.md`; ships audit events to backend. |
| **Session helper (optional P1)** | User | UX: prompts, “why denied,” copy correlation ID; forwards requests to pipe. |
| **Updater (later phase)** | Mixed | Out-of-band; must not weaken broker integrity. |

### 2.2 Backend platform

- **Device registry:** stable device identity, enrollment keys, version reporting.
- **Policy service:** authoritative policy documents; versioned; scoped to tenant/device/group.
- **Workflow:** pending elevation requests, approve/deny, expiry, technician attribution.
- **Audit service:** append-mostly event store derived from agent reports and admin actions.
- **AuthN/Z:** dashboard users vs agent credentials (see `SECURITY_MODEL.md`).

### 2.3 Web dashboard

- Technician queues, search/filter, device inventory, policy management screens (phase-gated).
- All mutating actions go through backend APIs—**no** direct agent reach from browser.

---

## 3. Windows agent design

### 3.1 Process model

```
┌──────────────────────┐
│ pam-broker.exe       │  Windows Service (LocalSystem)
│  - Pipe server       │
│  - Policy cache      │
│  - Backend client    │
│  - Process launcher  │
└──────────────────────┘
         ▲
         │ \\.\pipe\pam-elevator (example name; implement ONE well-known name)
         │
┌────────┴─────────────┐
│ pam-session.exe      │  Runs in user session (standard user)
│  - CLI/UI shim       │
└──────────────────────┘
```

**Principles:**

- **Single broker** per machine listening on a **secured** pipe ACL (only local interactive users + administrators as needed—not “Everyone”).
- **No** elevation from the session helper directly.

### 3.2 Startup and identity

- **Phase 1 (implemented):** On start, if `config.json` sets `backend_base_url`, the broker calls **`POST /agent/register`** (see `API_SPEC.md`) and persists `device_id` under `data/agent_state.json`. There is **no heartbeat** in Phase 1. Local policy is loaded from disk when evaluating **local-only** or **local_fallback** paths (`agent/policy/policy.json`).
- **Transport:** Development may use plain HTTP to localhost. **Non-development** agent-to-backend traffic must use **TLS** per `AGENT_RULES.md` (typically terminate TLS at the reverse proxy in front of Fastify).
- **Future phases:** Cryptographic device keys, enrollment tokens, and heartbeats are **not** required for Phase 1; see `DECISIONS.md` **ADR-010** and `API_SPEC.md`.

---

## 4. Elevation broker model

The broker is the **only** component allowed to create elevated processes.

**Canonical flow:**

1. Session helper sends `ElevationRequest` over the pipe: target path, optional args (used for launch only), working directory, user SID/username, parent metadata. **Phase 1 backend** request bodies do not carry arguments (see `API_SPEC.md`); the broker logs **`arguments_present`** locally without logging raw argv.
2. Broker evaluates **local policy snapshot** (fast path).
3. If required, broker opens **server-backed** request (pending approval) and polls or receives push (Phase 2+); Phase 1 may use **synchronous** poll loop with backoff—document timeouts in `API_SPEC.md`.
4. On **allow**, broker creates process with elevated token using APIs consistent with `SECURITY_MODEL.md` (e.g. elevated duplicate token, appropriate window station/desktop handling if UI required).
5. On **deny**, broker returns structured denial to session helper and emits audit.

```
  Standard user                Broker (SYSTEM)                 Backend
       │                             │                            │
       │  ElevationRequest           │                            │
       │ ───────────────────────────►│  POST elevation-request    │
       │                             │ ──────────────────────────►│
       │                             │◄────────────────────────── │
       │                             │   decision / pending       │
       │  allow + launch token       │                            │
       │ ◄───────────────────────────│                            │
       │                             │  audit event               │
       │                             │ ──────────────────────────►│
```

---

## 5. Named pipe communication

### 5.1 Why pipes here

- **Local-only** IPC with explicit ACLs.
- **No** network exposure of elevation API.
- Simple framing for Go ↔ native Windows APIs.

### 5.2 Framing (Phase 1 implemented)

- **Newline-delimited JSON:** one UTF-8 JSON object per request, trailing newline on responses. **Maximum request size** is enforced in code (see `agent/ipc/server.go`).
- **Planned (see `TASKS.md` P1-001):** optional **`PAM1` magic + version + length** framing for binary-safe, multi-message sessions; not implemented yet—docs previously described this as normative; implementation intentionally lags.

**Correlation:** The broker generates a **local correlation ID** for file audit lines when using the backend path; Phase 1 **does not** send it to the API (no matching field in `API_SPEC.md`).

### 5.3 Authorization on the pipe

- Broker verifies caller identity via **GetNamedPipeClientProcessId** + **OpenProcess** + **QueryFullProcessImageName** + **token user** cross-checks as defense-in-depth (exact checks documented with tests).
- Reject unknown clients; rate-limit per SID.

---

## 6. Backend API interaction

- **Agent APIs** are called **only** from the broker service account on the host (plus health checks).
- **Admin APIs** are called from the dashboard; **Phase 1** has **no** technician session auth (see `API_SPEC.md` / `TASKS.md` P1-012).
- **Idempotency-Key** (or equivalent) for elevation submission is **not** part of Phase 1; deduplication is a **Phase 2+** concern (see `DECISIONS.md` **ADR-010**).

See `API_SPEC.md` for paths, schemas, and error codes.

---

## 7. Data flow diagrams

### 7.1 Policy distribution

```
Backend (source of truth)
        │
        │  GET /agent/policy (optional `deviceId`; see `API_SPEC.md`)
        ▼
   Broker cache (disk + memory)
        │
        │  inject into evaluation
        ▼
   Local decision (allow/deny/pending)
```

### 7.2 Approval workflow

```
Technician (dashboard)
        │
        │ POST /admin/approve
        ▼
   Backend validates role + request state
        │
        │ state transition: APPROVED, expires_at
        ▼
   Broker poll / next action sees approval
        │
        ▼
   Elevated launch + audit
```

---

## 8. Trust boundaries

| Boundary | Trust level | Controls |
|----------|-------------|----------|
| Internet ↔ API | Low | TLS, auth, rate limits, WAF optional |
| Dashboard ↔ API | Medium | session auth, CSRF protections for cookie models, strict CORS |
| Agent ↔ API | Medium-high | enrollment key/mTLS future, IP allowlist optional, device credentials rotation |
| User session ↔ Pipe | Low | pipe ACL, client validation, binary allowlists |
| Broker ↔ OS | High | service hardening, minimal dependencies, secure defaults |

**Rule:** Treat **compromised standard user** as expected; they must not gain **arbitrary** SYSTEM or admin capability through the pipe alone.

---

## 9. Scaling considerations

- **PostgreSQL:** partition large audit tables by month; index on `(device_id, created_at)` and `(status, created_at)` for queues.
- **Redis:** bounded keys for rate limiting; avoid sticky business logic solely in Redis.
- **Agents:** exponential backoff on errors; **ETag** or version field on policy fetches.
- **Dashboard:** server-side pagination everywhere.

---

## 10. Reference repositories (study)

| Repository | URL | Architectural takeaway |
|------------|-----|------------------------|
| TacticalRMM | https://github.com/amidaware/tacticalrmm | Fleet operations, agent commands, practical MSP deployment patterns. |
| gsudo | https://github.com/gerardog/gsudo | UAC nuances, user-mode elevation helpers (contrast with our broker service model). |
| Velociraptor | https://github.com/Velocidex/velociraptor | Robust agent design, heavy telemetry, performance at scale. |
| JumpServer | https://github.com/jumpserver/jumpserver | Approval workflows and admin audit-first UX patterns. |

---

## 11. Related documents

- `API_SPEC.md` — contracts
- `SECURITY_MODEL.md` — threats and controls
- `DECISIONS.md` — ADRs
- `PROJECT_PLAN.md` — phased scope
