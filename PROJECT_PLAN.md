# PROJECT_PLAN.md

## 1. Mission

Deliver a **Privileged Access Management (PAM) platform for Windows** that lets **standard users** run **approved** applications requiring **administrator privilege** **without** granting **standing** local administrator rights—similar in operator value to **AutoElevate**.

The system must be **auditable**, **policy-driven**, and suitable for **managed service providers (MSPs)** and **IT teams** that need **just-in-time** elevation with **evidence**.

---

## 2. Product description

### 2.1 Actors

- **End user:** Standard (non-admin) Windows user who triggers elevation (installer, management tool, etc.).
- **Technician:** Help-desk or admin who approves/denies requests when policy requires human judgment.
- **Administrator:** Configures policies, devices, integrations, and org-wide settings.

### 2.2 Core product loop

1. User attempts an operation that requires elevation.
2. **Agent** intercepts or detects the need for elevation (phase-dependent; see roadmap).
3. Agent evaluates **local policy** and consults **backend** for authoritative rules and optional approval.
4. If allowed, the **broker service** launches the **approved** process elevated **once** (or for a **scoped** approval).
5. **Backend** retains an **audit record**; **dashboard** displays history and pending items.

---

## 3. System goals

| Goal | Description |
|------|-------------|
| **Just-in-time elevation** | Grant elevated execution only when justified by policy or approval. |
| **No standing admin** | Avoid placing users in persistent high-privilege groups for routine work. |
| **Central policy** | Define what may run elevated, from where, and under which constraints. |
| **Auditability** | Capture who, what, when, where, and outcome for every elevation decision. |
| **Operability** | Deploy with Docker Compose on a Linux VPS; support many endpoints. |
| **Agent resilience** | Degrade safely (deny by default) when offline per documented policy. |

---

## 4. Key features (target end-state)

- **Windows agent (service):** Elevation broker, policy application, backend sync, structured logging.
- **Backend:** Device registry, policy storage, approval workflow, audit log API, Redis-backed rate limiting / queues as needed.
- **Dashboard:** Pending requests, approve/deny, policy editor (phase-dependent), device inventory views.
- **IPC:** Named-pipe protocol between user session helper and privileged service.

---

## 5. Non-goals (early phases)

To prevent scope creep, the following are **explicitly out of scope** until promoted by plan/decisions:

- **Kernel-mode** enforcement or third-party driver dependency for Phase 1.
- **Full privileged access workstation (PAW)** replacement.
- **Enterprise SSO** beyond what Phase 2 defines (Phase 1 may use simpler auth).
- **Mac/Linux** endpoint agents (backend may remain portable).
- **Arbitrary code execution** approval (“run anything elevated”) as a default policy stance.

---

## 6. Phase roadmap

### Phase 1 — Vertical slice (MVP)

**Objective:** Prove safe brokered elevation with central visibility.

**Includes:**

- Agent installer/service skeleton; **LocalSystem** service with documented surface.
- **Named pipe** IPC and versioned message framing.
- Backend **Fastify** API + **PostgreSQL** schema for devices, requests, decisions, audit.
- **Redis** for rate limiting or short-lived approval tokens (minimal use acceptable in P1 if justified).
- Endpoints: register device, submit elevation request, fetch policy snapshot, approve/deny, list requests.
- Dashboard: authenticate technicians (mechanism as per `DECISIONS.md`), view pending requests, approve/deny, basic audit list.
- **Deny-by-default** when policy or backend is unavailable (configurable messaging).

**Excludes:** Advanced policy DSL, hash reputation feeds, full SSO, multi-tenant billing.

### Phase 2 — Operational maturity

**Objective:** Make the product deployable and manageable at scale.

**Includes:**

- Hardened authentication (e.g. OIDC) for dashboard and service accounts.
- Policy versioning, rollout per device/group, improved inventory.
- Agent update channel strategy (documented); signed artifacts where applicable.
- Improved observability: metrics, structured logging correlation, admin health pages.
- Rate limits, abuse controls, backoff, and idempotency on agent APIs.

### Phase 3 — Advanced PAM features

**Objective:** Deepen trust and automation.

**Includes (candidates):** publisher rules, file hash allowlists, integration webhooks, SIEM export, advanced reporting, optional driver-based enforcement **only** if ADR-approved.

---

## 7. Success criteria

| Area | Criterion |
|------|-----------|
| **Security** | No undocumented elevation path from standard user token alone; threats in `SECURITY_MODEL.md` addressed or explicitly accepted. |
| **Correctness** | Policy and server decisions match documented semantics; replay/idempotency behaviors defined in `API_SPEC.md`. |
| **Audit** | Every elevation attempt produces a durable backend record with correlation IDs. |
| **Deployability** | `docker compose up` (or documented equivalent) yields a working dev/prototype stack. |
| **UX** | Clear user messaging on deny/offline; technician workflow completes in under two minutes for common cases. |

---

## 8. Developer workflow

1. **Read:** `AGENT_RULES.md`, `SECURITY_MODEL.md`, and the relevant phase in `TASKS.md`.
2. **Implement:** Small, reviewable changes with tests for sensitive modules.
3. **Record:** Architectural shifts in `DECISIONS.md`; API changes in `API_SPEC.md`.
4. **Validate:** Local stack per `ARCHITECTURE.md`; agent tested on Windows with standard user accounts.
5. **Ship docs with code:** No “docs later” for security or API behavior.

---

## 9. Reference repositories (study)

| Repository | URL | Contribution to our architecture |
|------------|-----|----------------------------------|
| TacticalRMM | https://github.com/amidaware/tacticalrmm | Practical patterns for **fleet agents**, device identity, and **operator dashboards** at MSP scale. |
| gsudo | https://github.com/gerardog/gsudo | Deep familiarity with **UAC**, consent UI, and **sudo-like** elevation UX on Windows—useful for broker design and user messaging. |
| Velociraptor | https://github.com/Velocidex/velociraptor | Lessons in **endpoint agent** design, **telemetry**, and **query/audit** thinking for scalable visibility. |
| JumpServer | https://github.com/jumpserver/jumpserver | Patterns for **approval-oriented** access, **session accountability**, and **administrative workflows** in a web console. |

---

## 10. Document map

- **Architecture & data flow:** `ARCHITECTURE.md`
- **API contracts:** `API_SPEC.md`
- **Threats & controls:** `SECURITY_MODEL.md`
- **ADRs:** `DECISIONS.md`
- **Execution backlog:** `TASKS.md`
- **Agent discipline:** `AGENT_RULES.md`
