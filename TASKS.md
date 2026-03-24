# TASKS.md

Organized backlog for multi-agent development. **Status** values: `todo` | `in_progress` | `blocked` | `done`.

**Owner type:** which subsystem primarily owns the work (others may assist).

**Source of truth:** [ZENTO-PAM on GitHub](https://github.com/spkldbrd/ZENTO-PAM). Integration work should land there; local clones should track `origin/main` (or the agreed default branch) after the initial commit is pushed.

---

## Phase 1 tasks

| ID | Description | Owner type | Status | Notes |
|----|-------------|------------|--------|-------|
| P1-001 | Define versioned **named pipe** frame + JSON schemas shared between agent and docs (`ARCHITECTURE.md` / `API_SPEC.md`). | Windows agent | in_progress | **Implemented:** newline-delimited JSON (`ARCHITECTURE.md` §5.2). **Future:** `PAM1` magic+length framing still open. |
| P1-002 | Implement **Windows service** skeleton (install, start, stop, recovery) in Go; structured logs to Windows Event Log + optional file sink. | Windows agent | in_progress | Service + console mode exist; Event Log sink still open. |
| P1-003 | Implement pipe server with **locked-down DACL**; validate connecting process identity; basic abuse limits per user SID. | Windows agent | in_progress | DACL via SDDL; full client attestation / rate limits incomplete. |
| P1-004 | Implement session **client library** (Go or small C#/C++ shim if required) to send `ElevationRequest` to broker. | Windows agent | done | `pam-request` CLI + JSON request shape. |
| P1-005 | Implement broker **policy evaluation** stub: local cache + match rules (path prefix, publisher stub, or hash placeholder). | Windows agent | done | `policy/policy.json` + publisher/hash evaluation (Windows). |
| P1-006 | Implement **elevated launch** path from broker with tests on standard user + deny scenarios. | Windows agent | in_progress | `broker.Launch` exists; session-0 / UX limitations documented in `agent/README.md`. |
| P1-007 | Backend: **PostgreSQL** schema for `devices`, `elevation_requests`, `decisions`, `audit_events`, `policy_versions`. | Backend | done | `backend/db/migrations/` (core tables present). |
| P1-008 | Backend: Fastify app scaffold—config loader, logger (pino), error handler, zod validation. | Backend | done | `backend/server.ts`, zod in `models/schemas.ts`. |
| P1-009 | Backend: `POST /agent/register` with enrollment key issuance flow (admin creates key in dashboard or seed script). | Backend | blocked | **Superseded for Phase 1** by `API_SPEC.md` (open registration, no enrollment key). Defer to Phase 2 or rewrite task when device auth returns. |
| P1-010 | Backend: `POST /agent/elevation-request` + `GET` polling for request status. | Backend | done | Matches `API_SPEC.md`; agent client aligned (see `DECISIONS.md` ADR-010). |
| P1-011 | Backend: `GET /agent/policy` with version field; support delta or full document. | Backend / Windows agent | in_progress | Backend serves policy JSON; **agent does not fetch it yet** (uses local `policy/policy.json` only). Add agent pull + cache when ready; versioning/delta still open. |
| P1-012 | Backend: `POST /admin/approve`, `POST /admin/deny`, `GET /admin/requests`. | Backend | done | Phase 1: no dashboard login; `PAM_ADMIN_USER` labels audit actor. |
| P1-013 | Redis: rate limit agent endpoints per `device_id` + IP fallback. | Backend | todo | Redis present in `infra/docker/docker-compose.yml`; **not wired** in API code yet. |
| P1-014 | Dashboard: technician login, pending requests view, approve/deny actions. | Dashboard | in_progress | Approve/deny + lists work against API; **no auth** (per Phase 1 spec). |
| P1-015 | Dashboard: audit log viewer (read-only) with filters by device and time range. | Dashboard | in_progress | Recent 200 rows shown; filters/pagination open. |
| P1-016 | Infra: **Docker Compose** for API + Postgres + Redis + dashboard static/server build. | Infra | done | `infra/docker/docker-compose.yml`. **VPS:** run only from Git clone at `/opt/ZENTO-PAM` per **ADR-011**. |
| P1-017 | End-to-end test script: agent (or simulator) registers, submits request, approves, observes decision. | Cross-cutting | todo | **Integration coordinator:** run manually—backend+dashboard+agent with `config.json` `backend_base_url`. |
| P1-018 | Security review pass: **STRIDE** walkthrough vs `SECURITY_MODEL.md`; fix gaps or record accepts. | Cross-cutting | todo | Blocking before “MVP done” claim. |
| P1-INT-001 | **Integration:** Agent `config.json` + `backend_base_url`; verify register → elevation-request → dashboard approve/deny → poll → launch. | Cross-cutting | in_progress | Client paths match `API_SPEC.md` (ADR-010). **Manual E2E:** `agent/README.md`. Audit log includes `device_id`, `working_directory`, `arguments_present`, `backend_request_id`, `correlation_id` where applicable. |

---

## Phase 2 tasks

| ID | Description | Owner type | Status | Notes |
|----|-------------|------------|--------|-------|
| P2-001 | OIDC/OAuth2 login for dashboard; service accounts for automation. | Backend / Dashboard | todo | Refresh token rotation; CSRF strategy documented. |
| P2-002 | Policy editor UI + policy rollout by device groups; version history and rollback. | Dashboard / Backend | todo | Diff view between versions. |
| P2-003 | Agent updater channel: signed releases, staged rollout percentage. | Windows agent | todo | Threat model supply-chain controls. |
| P2-004 | Heartbeat + inventory: OS version, patch level, agent version, network info (minimal PII). | Windows agent / Backend | todo | Align fields with dashboard inventory page. |
| P2-005 | Observability: metrics (Prometheus), request tracing IDs, structured correlation across agent and API. | Cross-cutting | todo | Redact sensitive fields at collection. |
| P2-006 | mTLS or token-based **device authentication** hardening; key rotation. | Backend / Windows agent | todo | Prefer automated rotation with grace period. |
| P2-007 | Webhooks for approval events to external PSA/RMM (optional). | Backend | todo | Signed webhook payloads. |
| P2-008 | Backup/restore runbook for Postgres; documented RPO/RTO targets. | Infra | todo | Encrypt backups at rest. |

---

## Future tasks (Phase 3+ / backlog)

| ID | Description | Owner type | Status | Notes |
|----|-------------|------------|--------|-------|
| F-001 | Publisher-based rules (Authenticode chain) with trust store updates. | Windows agent | todo | High test burden; handle offline. |
| F-002 | File hash allowlists with cloud hash sources and caching. | Backend / Windows agent | todo | Watch for TOCTOU; combine with path rules. |
| F-003 | Optional kernel filter driver for **pre-launch** hooking (only if ADR-approved). | Windows agent | todo | Significant compliance/maintenance cost. |
| F-004 | SIEM export (CEF/JSON), scheduled reports, anomaly detection. | Backend | todo | Privacy review for log fields. |
| F-005 | Multi-tenant billing, organizations, delegated admin roles. | Backend / Dashboard | todo | Row-level security in Postgres. |
| F-006 | macOS/Linux investigation agent or alternate architecture. | Cross-cutting | todo | Not currently in scope. |

---

## Reference repositories (study)

Use for patterns only; do not treat as submodules without legal review.

| Repository | URL | Notes for task execution |
|------------|-----|---------------------------|
| TacticalRMM | https://github.com/amidaware/tacticalrmm | Study agent command dispatch and fleet UX when implementing P2 inventory/heartbeat. |
| gsudo | https://github.com/gerardog/gsudo | Study UAC edge cases when hardening P1 elevation launcher. |
| Velociraptor | https://github.com/Velocidex/velociraptor | Study agent performance patterns for telemetry volume in P2+. |
| JumpServer | https://github.com/jumpserver/jumpserver | Study approval UX and audit presentation for dashboard iterations. |

---

## How agents should use this file

- Pick tasks matching their **owner type** and Phase.
- Mark **blocked** with a short reason (e.g. “waiting on API_SPEC envelope format”).
- When completing tasks that change behavior, update **API_SPEC.md** / **ARCHITECTURE.md** in the same merge/set of commits.
