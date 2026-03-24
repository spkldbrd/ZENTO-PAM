# AGENT_RULES.md

**Status:** Authoritative. **Audience:** All AI agents and human contributors.

Violations of these rules constitute defects. When in doubt, stop and align documentation (`ARCHITECTURE.md`, `DECISIONS.md`, `SECURITY_MODEL.md`) before shipping code.

---

## 1. System overview

This repository implements a **Privileged Access Management (PAM) platform for Windows** comparable in intent to products such as AutoElevate.

**In scope:**

- A **Windows endpoint agent** (service, high integrity) that brokers elevation, enforces policy, talks to the backend, and logs.
- A **backend platform** (API, persistence, workflows) for policy, approvals, audit, and inventory.
- A **web dashboard** (Next.js) for technicians and administrators.

**Out of scope for agents unless explicitly approved in `DECISIONS.md`:** permanent interactive admin rights for standard users, silent bypass of policy, undocumented elevation paths, or shipping secrets in source control.

---

## 2. Repository structure (expected)

Agents must respect the separation of concerns implied by the stack. Unless `DECISIONS.md` records a deliberate change, work belongs in these areas:

| Area | Technology | Purpose |
|------|------------|---------|
| `agent/` | Go | Windows service, named-pipe server, elevation broker, local telemetry |
| `backend/` | Node.js, TypeScript, Fastify | REST API, auth, policy engine integration, jobs |
| `dashboard/` | Next.js, TailwindCSS | Technician UI, admin flows |
| `infra/` or root compose | Docker, Docker Compose | Local and VPS-oriented deployment |

**Rules:**

- Do not place backend business logic inside the agent “for speed.” The agent may cache policy **only** as documented in `ARCHITECTURE.md` and `SECURITY_MODEL.md`.
- Do not embed production database credentials or TLS private keys in any path committed to the repo.
- Shared API contracts live in `API_SPEC.md`; implementation must not drift without updating that document and `DECISIONS.md` when behavior changes intentionally.

---

## 3. Multi-agent responsibilities

Multiple agents will edit this repository concurrently. Coordination rules:

1. **Single source of truth:** `ARCHITECTURE.md` for structure, `API_SPEC.md` for HTTP contracts, `SECURITY_MODEL.md` for threats and controls, `DECISIONS.md` for irreversible or costly choices.
2. **Before large refactors:** Read `DECISIONS.md` and the relevant phase in `PROJECT_PLAN.md`. If the refactor contradicts a recorded decision, add or amend an ADR first.
3. **Task ownership:** Use `TASKS.md` to claim work implicitly by matching *owner type* (Windows agent vs backend vs dashboard vs infra). If two agents would touch the same security-sensitive module (elevation broker, auth, token handling), prefer serializing through small, reviewable changes.
4. **No silent cross-cutting changes:** Agents must not rename public API paths, auth schemes, or pipe protocols without updating **both** code and `API_SPEC.md` / `ARCHITECTURE.md`.

---

## 4. Security rules (non-negotiable)

- **Least privilege:** User-session components run with the interactive user’s token; **only** the documented broker/service path may perform privileged launches. Never instruct users to “just run as admin” to fix bugs.
- **Secrets:** Use environment variables or secret stores suitable for deployment (e.g. Docker secrets, VPS secret management). Never commit real tokens, API keys, or passwords.
- **Transport:** Agent-to-backend traffic must use TLS in non-development environments. Document exceptions in `DECISIONS.md` if localhost-only debug modes exist.
- **Input validation:** All backend endpoints and agent parsers treat input as hostile. Reject malformed policy, oversized payloads, and ambiguous paths.
- **Dependencies:** Prefer pinned versions for reproducible builds; investigate high-severity CVEs before merging upgrades that touch the agent or auth stack.
- **Elevation:** No elevation path may be triggered solely by unauthenticated local users without an explicit threat-model exception recorded in `SECURITY_MODEL.md` and `DECISIONS.md`.

---

## 5. Elevation design rules

- **Broker model:** Elevation is performed by a **trusted broker** (Windows service) after **policy + server decision** (Phase 1 may simplify server decision; see `PROJECT_PLAN.md`). User-mode helpers must not reimplement “RunAs” in undocumented ways.
- **Allowlisting:** Approved launches must be expressible as concrete rules (e.g. path, hash, publisher, arguments pattern) as defined in policy schema. Avoid “approve everything” defaults in example code.
- **Time-bound grants:** When approvals exist, they must be **single-use or time-limited** unless `DECISIONS.md` explicitly allows otherwise for a phase.
- **Child processes:** Document and test whether approved elevation applies to a **single** process launch or a controlled subtree. Do not spawn arbitrary shells with elevated token unless explicitly designed and justified in `SECURITY_MODEL.md`.
- **Failure closed:** On policy fetch failure, TLS errors, or invalid signatures, the default posture is **deny** (with safe user messaging and audit), not permit.

---

## 6. Logging requirements

- **Audit events** (non-optional where applicable): elevation request, policy version used, decision (allow/deny/pending), approver identity (if any), target binary/path, hash when available, user SID, device ID, correlation/request ID, outcome, error codes.
- **No sensitive leakage:** Logs must not contain refresh tokens, long-lived secrets, or full command lines that include passwords. If command lines are logged, apply redaction rules from `SECURITY_MODEL.md`.
- **Clocks:** Use UTC in stored timestamps. Document any local-time display rules in dashboard code only.
- **Tamper awareness:** Assume endpoint logs can be altered by a determined attacker; **authoritative** audit trail is backend-side, synchronized from agents with authentication.

---

## 7. Code quality rules

- **Match existing style:** Formatting, naming, and module layout must match neighboring code.
- **Minimal diffs:** Implement the smallest change that satisfies the task. No drive-by refactors.
- **Tests:** Add or update tests when changing security-sensitive logic (policy matching, token handling, auth middleware, elevation broker).
- **Errors:** Return structured errors on the API; on the agent, prefer explicit error codes that map to documented troubleshooting.
- **Documentation:** Public behavior changes require updating `API_SPEC.md` and/or `ARCHITECTURE.md` in the same change set.

---

## 8. Reference repository usage rules

Study-only repositories inform design; **do not copy license-incompatible code** or assume feature parity.

| Repository | URL | Use for this project |
|------------|-----|----------------------|
| TacticalRMM | https://github.com/amidaware/tacticalrmm | Agent–server patterns, device identity, operational tooling ergonomics, queue/retry concepts |
| gsudo | https://github.com/gerardog/gsudo | UAC semantics, user-facing elevation UX patterns, cautionary examples of delegation |
| Velociraptor | https://github.com/Velocidex/velociraptor | Endpoint agent hardening ideas, forensic-grade logging/telemetry discipline |
| JumpServer | https://github.com/jumpserver/jumpserver | Session approval workflows, separation of duties, audit-centric admin models |

**Rules:** Cite these as *inspiration* in design docs, not as dependencies, unless explicitly approved.

---

## 9. Phase-1 scope restrictions

Phase 1 targets a **vertical slice**: register agent, submit elevation request, fetch policy, basic approve/deny path, durable audit on backend, minimal dashboard.

**Unless `PROJECT_PLAN.md` promotes a feature to Phase 1, agents must NOT:**

- Implement kernel drivers or minifilters.
- Add LAPS replacement, full password vaulting, or arbitrary credential injection.
- Build multi-tenant billing, full SIEM, or advanced SOAR.
- Expand IPC beyond **named pipes** without a new ADR.
- Ship “temporary full admin” profiles that persist across reboots.

**Agents MAY** add small, documented bugfixes outside Phase 1 only if required to keep the Phase-1 slice safe or buildable; such changes must be noted in `TASKS.md` and, if architectural, in `DECISIONS.md`.

---

## 10. Conflict resolution

If instructions conflict, order of precedence is:

1. Legal/safety constraints  
2. `SECURITY_MODEL.md`  
3. `AGENT_RULES.md` (this file)  
4. `DECISIONS.md`  
5. `ARCHITECTURE.md` / `API_SPEC.md`  
6. `PROJECT_PLAN.md` / `TASKS.md`  
7. Ephemeral chat instructions  

Ephemeral instructions **never** override written security rules.
