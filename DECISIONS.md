# DECISIONS.md

Architecture Decision Records (ADRs) for the PAM platform. New decisions append with the next available **ADR** number. **Superseded** decisions are retained for history.

---

## ADR-001 — Implement the Windows agent in Go

**Decision:** Build the Windows endpoint agent (service, pipe server, backend client) primarily in **Go**.

**Rationale:**

- Strong story for **single static binary** deployment on Windows endpoints.
- Mature ecosystem for **cross-platform tooling** (still Windows-targeted here) and **HTTP/TLS** clients.
- Simpler supply chain for a long-running service compared to mixing multiple runtimes on the agent.

**Alternatives considered:**

- **Rust:** Excellent safety/performance; higher onboarding friction for some teams, longer iteration for Win32 interop in early phases.
- **C#/.NET:** Very productive for Windows services; larger runtime dependency and update surface on endpoints—acceptable but heavier for minimal agent footprint.
- **C++:** Maximum control; slower safe iteration and more footguns for memory safety.

---

## ADR-002 — Use Fastify for the backend HTTP server

**Decision:** Use **Fastify** (Node.js, TypeScript) as the primary API framework.

**Rationale:**

- Predictable performance for JSON APIs with **schema-based validation** (e.g. TypeBox/Zod integration patterns).
- Low ceremony routing and middleware model suitable for **agent high-throughput** + **dashboard** traffic profiles.
- Aligns with team stack choice for rapid API iteration in early phases.

**Alternatives considered:**

- **NestJS:** Strong structure for large apps; more boilerplate than needed for Phase 1 slice.
- **Express:** Ubiquitous; weaker out-of-the-box performance and schema ergonomics compared to Fastify defaults.
- **Go (backend):** Would split language between agent and API; acceptable but reduces shared TypeScript types with dashboard.

---

## ADR-003 — Use PostgreSQL as system of record

**Decision:** Use **PostgreSQL** for durable entities: devices, policies, requests, decisions, audit.

**Rationale:**

- ACID transactions for **workflow state** (pending → approved/denied).
- Rich indexing and JSONB if needed for flexible policy documents without losing queryability.

**Alternatives considered:**

- **MySQL/MariaDB:** Viable; slightly weaker JSON ergonomics in some deployments.
- **MongoDB:** Flexible documents; weaker relational integrity for approval workflows without careful schema discipline.

---

## ADR-004 — Use Redis for rate limiting and ephemeral state

**Decision:** Use **Redis** for **rate limiting**, short-lived tokens/locks, and optional background job bookkeeping—not as the primary audit store.

**Rationale:**

- Fast, TTL-native primitives to protect agent endpoints from abuse and accidental retry storms.
- Keeps **authoritative** audit data in Postgres.

**Alternatives considered:**

- **In-memory only:** Lost on restart; poor for multi-instance API in Phase 2+.
- **Postgres-only rate limits:** Possible with careful design; higher DB load and more complex GC of counters.

---

## ADR-005 — Use named pipes for local IPC

**Decision:** Use **Windows named pipes** for communication between the user session component and the **LocalSystem** broker service.

**Rationale:**

- **Local-only** channel with **explicit DACLs**; avoids exposing elevation workflow on TCP/localhost sockets.
- Well-supported in Go via `winio` or analogous libraries and native Win32 for edge cases.

**Alternatives considered:**

- **ALPC (local RPC):** More complex to implement correctly; higher security review burden.
- **TCP localhost:** Wider attack surface; harder to reason about port conflicts and firewall rules.
- **Shared memory + events:** Fast but error-prone for authentication and framing.

---

## ADR-006 — Broker elevation model (no standing admin for users)

**Decision:** All elevated launches are performed by a **trusted broker service** after **policy + server evaluation** (with defined offline behavior).

**Rationale:**

- Prevents equating “user consent” with **permanent privilege**.
- Centralizes **auditing** and **policy interpretation** in one highly scrutinized component.
- Aligns with PAM product goals comparable to AutoElevate-style workflows.

**Alternatives considered:**

- **gsudo-style delegated elevation for users:** Excellent UX for power users; weaker centralized control and audit unless heavily wrapped.
- **Temporary membership in Administrators group:** Simple but dangerous; easy to leave users over-privileged; poor audit granularity.

---

## ADR-007 — Avoid kernel drivers in Phase 1 (and default roadmap)

**Decision:** Phase 1 **must not** require a **kernel driver** or minifilter for core functionality.

**Rationale:**

- Kernel code dramatically increases **attack surface**, **code signing**, **stability**, and **compliance** burden.
- User-mode broker can deliver MVP value for **explicit** elevation flows with simpler rollout.

**Alternatives considered:**

- **Minifilter to intercept process creation:** Strong enforcement; defer to Phase 3+ with full ADR, signing pipeline, and recovery plan.
- **ELAM / early launch drivers:** Unnecessary for initial product slice.

---

## ADR-008 — Dashboard in Next.js with TailwindCSS

**Decision:** Build the technician/admin UI with **Next.js** and **TailwindCSS**.

**Rationale:**

- Productive React model for data tables, filters, and auth flows.
- Tailwind enables consistent spacing/typography with minimal custom CSS drift across agents’ contributions.

**Alternatives considered:**

- **Vite + React:** Lighter; lose some integrated routing/server patterns if SSR/Route Handlers desired later.
- **Blazor / WPF webview:** Less aligned with chosen stack and hiring patterns.

---

## ADR-010 — Phase 1 agent HTTP contract follows `API_SPEC.md` (no `/v1` prefix)

**Decision:** The Windows agent’s backend client MUST use the same paths and JSON shapes as `API_SPEC.md` for Phase 1: `POST /agent/register`, `POST /agent/elevation-request`, `GET /agent/elevation-requests/:id`, with statuses `pending` / `approved` / `denied`, and **no** `/v1` URL prefix, enrollment keys, or bearer tokens unless a later ADR promotes them.

**Rationale:**

- The backend and dashboard were implemented against this contract; a parallel “draft v1” shape in the agent blocked the end-to-end approval loop entirely.
- Keeping one authoritative document (`API_SPEC.md`) avoids split-brain between components.

**Alternatives considered:**

- **Add a `/v1` compatibility shim on the backend:** Duplicates routes and prolongs two contracts; rejected for Phase 1 minimal surface.
- **Rewrite `API_SPEC.md` to match the agent:** Would force backend/dashboard changes and contradict the already-deployed Phase 1 API.

---

## ADR-009 — Containerized deployment target (Linux VPS)

**Decision:** Target **Docker Compose** on a **Linux VPS** for production-like deployments in early phases.

**Rationale:**

- Repeatable deployments for small teams; easy to add Postgres/Redis sidecars.
- Clear separation: Windows agents are remote endpoints; backend is centralized.

**Alternatives considered:**

- **Kubernetes:** Powerful; operational overhead too high for Phase 1 MVP.
- **Windows Server for API:** Higher cost; less common for small MSP backends.

---

## Reference repositories (study, not dependencies)

| Repository | URL | Why listed |
|------------|-----|------------|
| TacticalRMM | https://github.com/amidaware/tacticalrmm | Real-world agent + MSP operations patterns. |
| gsudo | https://github.com/gerardog/gsudo | UAC and user-mode elevation patterns to contrast with broker service. |
| Velociraptor | https://github.com/Velocidex/velociraptor | Endpoint agent rigor and scalable telemetry. |
| JumpServer | https://github.com/jumpserver/jumpserver | Approval-centric PAM-style UX and audit modeling. |
