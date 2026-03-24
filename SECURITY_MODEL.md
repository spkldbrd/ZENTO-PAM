# SECURITY_MODEL.md

Security model for the PAM platform: trust boundaries, privilege rules, threat considerations, and secure development guidance. This document complements `ARCHITECTURE.md` and `AGENT_RULES.md`.

---

## 1. Trust boundaries

```
                    Untrusted
                        │
          Internet / User desktop
                        │
    ┌───────────────────┼───────────────────┐
    │                   │                   │
    ▼                   ▼                   ▼
 Browser            Standard user        Malicious local
 (dashboard)        + session tools      processes (assume)
    │                   │                   │
    │ HTTPS             │ Named pipe        │ Pipe ACL +
    │ auth              │ (authenticated    │ broker checks
    │                   │  session client)  │
    ▼                   ▼                   ▼
┌─────────┐         ┌─────────────────────────────┐
│ Backend │◄───────►│ Broker service (LocalSystem)│
│  API    │  TLS    │  + policy + launcher        │
└─────────┘         └─────────────────────────────┘
    │                           │
    ▼                           ▼
 Postgres / Redis            OS kernel / UAC
```

**Boundary rules:**

- **Browser ↔ API:** Authenticate technicians; enforce RBAC; never expose device bearer tokens to the browser.
- **Agent ↔ API:** Authenticate devices; least-privilege scopes; rotate credentials; rate-limit.
- **User session ↔ Broker:** Treat as **semi-trusted**; attacker may control the session helper binary—**do not** trust arguments blindly; validate against policy and cryptographic/file evidence where available.
- **Broker ↔ OS:** Highest trust in user space; must be minimal, auditable, and hardened.

---

## 2. Endpoint security assumptions

### 2.1 Windows endpoint

- **Standard users** may run arbitrary unprivileged code; they may attempt to **spoof** IPC clients or **replay** messages.
- **Administrators** on the endpoint can bypass user-mode controls; PAM goals focus on **standard user** threat model and **operator accountability**, not anti-admin forensics.
- **Malware** may try to piggyback approved launches; mitigate with **one-shot grants**, **hash/path constraints**, and **short TTL**.

### 2.2 Backend VPS

- OS and container images are **kept patched**.
- **TLS** terminates at a reverse proxy or API with modern cipher suites.
- **Database** not exposed publicly; reachable only on private network / docker network.

---

## 3. Privilege model

| Principal | Typical privilege | Capabilities |
|-----------|-------------------|--------------|
| End user | Standard user | Request elevation; cannot directly obtain SYSTEM |
| Session helper | Standard user | Send requests; display UX |
| Broker service | LocalSystem (or elevated service account) | Validate, decide local path, create elevated process per grant |
| Technician | Org-authenticated user | Approve/deny; edit policy (RBAC) |
| Device credential | Machine identity | Register, submit requests, fetch policy—not human admin |

**Principle:** **No standing admin rights** for end users; elevated rights are **broker-mediated**, **scoped**, and **audited**.

---

## 4. Elevation restrictions

- **Policy default:** `DENY` unless an explicit rule matches.
- **Grants** must bind to **constraints** (at minimum target path; preferred + file hash or publisher).
- **TTL:** Approvals expire; brokers refuse expired grants.
- **One-shot:** Prefer single launch per grant unless a documented exception exists in `DECISIONS.md`.
- **No elevation of generic hosts** by default (e.g. `powershell.exe`, `cmd.exe`) unless tightly constrained via ADR-approved patterns.

---

## 5. Credential handling

- **Enrollment keys:** Short-lived or revocable; never embed in agent binaries; distribute via MDM/RMM or secure onboarding.
- **Device access tokens:** Opaque, rotatable, stored in ACL-protected Windows storage (DPAPI-protected file or service vault); **never** world-readable.
- **Technician sessions:** HttpOnly cookies or Bearer tokens per chosen auth; enforce logout and idle timeout on dashboard.
- **Secrets in dev:** Use `.env` excluded from git; provide `.env.example` without real values.

**Forbidden:**

- Committing private keys, JWT signing secrets, or production DB passwords.
- Committing **host SSH passwords** or embedding them in repository scripts (e.g. Paramiko `password=` literals). Prefer **SSH keys** (`ssh -i …`, `ssh-agent`) for operator access to VPS; if a tool must use a password, read it from a **local, gitignored** env file or secret store—never from tracked source.
- Logging full tokens or refresh tokens.

---

## 6. Audit logging

### 6.1 Events (minimum)

- Device registration / rotation failures
- Policy fetch (version, outcome)
- Elevation request created (correlation ID, hashes, paths, user SID)
- Decision (auto/technician), approver ID, comment (optional), TTL
- Broker launch success/failure with Windows error codes (sanitized)
- Deny reasons (policy id, rule id)

### 6.2 Integrity

- **Authoritative** log is **server-side**; agent logs are supplementary.
- Consider **hash-chaining** or external SIEM export in later phases—not required Phase 1, but design tables for immutability-friendly storage.

### 6.3 Redaction

- Redact secrets, tokens, and password-like arguments.
- If logging command lines, apply **allowlist/denylist** redaction rules.

---

## 7. Attack surface considerations

| Threat | Example | Mitigation |
|--------|---------|------------|
| Rogue pipe client | Malware connects to broker pipe | Strong DACL; validate client PID/token/image path; rate limits |
| Token theft | Steal device access token | Short TTL + rotation; bind token to device key; anomaly detection (Phase 2+) |
| Replay | Resubmit old approved request | One-time grants; server tracks `request_id` state; idempotency keys |
| MITM on agent TLS | Corporate SSL inspection misuse | Pinning options documented; monitor for broken chains |
| Technician account takeover | Stolen dashboard creds | MFA (Phase 2), RBAC, session hardening, audit alerts |
| SQL injection | Malicious filters | Parameterized queries; zod/type validation on all inputs |
| SSRF from dashboard | Fetch internal metadata | Block internal IPs in any outbound fetch features |
| Supply chain | Compromised dependency | Pin deps; audit; signed releases for agent |

**STRIDE snapshot:**

- **Spoofing:** Device enrollment keys, session auth—use strong auth artifacts.
- **Tampering:** TLS + code signing for agent binaries (roadmap).
- **Repudiation:** Server audit with technician attribution.
- **Information disclosure:** Minimize PII in logs; tighten error messages for unprivileged callers.
- **Denial of service:** Redis rate limits; payload size caps; DB indexes.
- **Elevation of privilege:** Core asset—broker must not trust user-supplied paths without policy matching.

---

## 8. Secure development guidelines

- **Threat model changes** require updates to this document and possibly a new ADR.
- **Security-sensitive PRs** need: tests, rollback plan, and explicit consideration of **least privilege**.
- **Dependencies:** Run vulnerability scanning in CI; block merges on critical issues in agent/auth libraries when practical.
- **Code review focus:** IPC validation, process creation parameters, auth middleware, SQL, file path handling.
- **Windows specifics:** Avoid **DLL planting** (use full paths, SafeDllSearchMode awareness), beware **TOCTOU** between check and use—prefer handles and transactional patterns where possible.

---

## 9. Reference repositories (study)

| Repository | URL | Security-relevant lesson |
|------------|-----|---------------------------|
| TacticalRMM | https://github.com/amidaware/tacticalrmm | Operational security for agent fleets; least-privilege remote admin patterns. |
| gsudo | https://github.com/gerardog/gsudo | UAC nuances; careful handling of elevated tokens and user expectations. |
| Velociraptor | https://github.com/Velocidex/velociraptor | Defensive agent engineering; high-value endpoint monitoring patterns. |
| JumpServer | https://github.com/jumpserver/jumpserver | Session-centric access control; strong audit and approval models. |

---

## 10. Alignment

- **Controls → tasks:** See `TASKS.md` security review item P1-018.
- **Architecture:** See trust boundaries in `ARCHITECTURE.md`.
- **Non-negotiables:** See `AGENT_RULES.md`.
