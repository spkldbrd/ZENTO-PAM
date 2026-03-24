# PAM Windows agent (Phase 1)

Local elevation broker: a **LocalSystem** Windows service exposes a secured named pipe (`\\.\pipe\pam_elevation`), evaluates [policy/policy.json](../policy/policy.json) when running **local-only** or **local fallback**, and launches allowed executables with **CreateProcessAsUser** using a **DuplicateTokenEx** primary token from the service process (same pattern as **CreateProcessWithTokenW** for a primary token). The child runs in high-privilege **session 0** service context—not the interactive user desktop.

When `backend_base_url` is set in `config.json`, the agent follows [API_SPEC.md](../API_SPEC.md): it registers on startup (`POST /agent/register`), submits each elevation as `POST /agent/elevation-request`, and polls `GET /agent/elevation-requests/:id` until `approved`, `denied`, or timeout. If the backend is unreachable and `local_fallback` is `true`, the agent evaluates local policy instead.

## Build

Requires Go 1.22+ on Windows:

```powershell
cd agent
go build -o ..\build\pam-agent.exe .\cmd\pam-agent
go build -o ..\build\pam-request.exe .\cmd\pam-request
```

## Configuration

Copy `config.example.json` to `config.json` in the **same directory as `pam-agent.exe`** (the agent’s base directory). Fields:

| Field | Meaning |
| ----- | ------- |
| `backend_base_url` | Backend origin, e.g. `http://127.0.0.1:3001`. Empty or omitted → local policy only. |
| `enrollment_key` | Reserved; Phase 1 backend does not use it. |
| `polling_interval_ms` | Delay between polls while status is `pending` (default 2000). |
| `request_timeout_seconds` | Max wait for `approved`/`denied` after submit (default 180). |
| `local_fallback` | If `true`, use local policy when backend registration or elevation API fails. If `false` and backend is configured, elevation is denied when the backend is unavailable. |
| `agent_version` | Sent as `agent_version` on register (default `0.2.0`). |

## Install

From an elevated PowerShell:

```powershell
Set-ExecutionPolicy -Scope Process Bypass -Force
.\scripts\install.ps1
```

Optional: `-SourceDir` (defaults to `..\build` from `scripts`) and `-InstallDir` (defaults to `C:\Program Files\PamAgent`). Place `config.json` next to the installed `pam-agent.exe` if you use the backend.

## End-to-end integration test (agent + backend + dashboard)

Prerequisites: PostgreSQL reachable by the backend, `DATABASE_URL` set for the backend, migrations applied, Node.js installed.

1. **Backend** — from repo root `backend/`:
   - `npm install`
   - `npm run build`
   - `npm run db:migrate` (or your documented migrate command for this repo)
   - `npm run dev` (listens on `PORT`, default **3001**; override with `PORT` / `HOST` if needed).
   - Verify: `curl http://127.0.0.1:3001/health` returns `{"ok":true}`.

2. **Dashboard** — from `dashboard/`:
   - `npm install`
   - `npm run dev` (default **http://localhost:3000**).
   - Open the requests UI so you can approve or deny pending items.

3. **Agent config** — next to `pam-agent.exe`, create `config.json` (see `config.example.json`):
   - Set `backend_base_url` to the backend origin (e.g. `http://127.0.0.1:3001`).
   - Set `local_fallback` to `true` while testing so a backend outage still allows local policy; set `false` to require backend registration.

4. **Start the agent** — elevated console or installed LocalSystem service:
   - Run `pam-agent.exe` (or start the service).
   - In `logs/agent.log`, confirm a `registration` event with `decision":"allowed"` and `result` containing `device_id=...`.

5. **Request elevation** — in a user session (or same machine):
   - `pam-request.exe C:\Windows\System32\notepad.exe`
   - In the dashboard, confirm a new **pending** request for this host’s device.

6. **Approve path** — approve the request in the dashboard.
   - The agent should poll until status is `approved`, then launch via the broker.
   - `pam-request` should exit successfully with a PID in the JSON response.
   - `logs/agent.log` should show an `elevation` line with `backend_request_id`, `backend_status":"ALLOWED"`, and `result` like `launched pid=...`.

7. **Deny path** — run `pam-request.exe` again for another executable (or repeat after policy allows local test); **deny** in the dashboard.
   - `pam-request` should print a failure response; message should indicate backend denial.
   - `logs/agent.log` should show `backend_status` matching the denial path (`DENIED`) and `decision":"denied"`.

8. **Backend unavailable** — stop the backend process; with `local_fallback: true`, repeat `pam-request` for a path allowed by `policy/policy.json`; elevation should succeed via local policy and `mode` in the log should reflect fallback. With `local_fallback: false`, expect denial when the agent could not register or cannot reach the backend for that flow.

## Manual test (local policy only, no backend)

1. Omit `backend_base_url` in `config.json` or use a fresh `config.json` with only defaults.
2. Ensure `policy\policy.json` exists **next to** `pam-agent.exe`.
3. Run `pam-agent.exe` in a console; it listens on the pipe.
4. In another terminal: `pam-request.exe C:\Windows\System32\notepad.exe`
5. Check `logs\agent.log` for JSON audit lines (`mode` should be `local_only`).

## Policy

- **allowed_hashes** empty: allow only if Authenticode **simple display name** matches **allowed_publishers** (case-insensitive).
- **allowed_hashes** non-empty: allow if SHA-256 (hex, lowercase) matches **or** publisher matches. If only **allowed_hashes** is set, the publisher is never resolved (fast path).
- Both lists empty: deny all.

Publisher resolution uses **CryptQueryObject** / **WinTrust** helpers where possible; **catalog-signed** OS binaries (e.g. `notepad.exe`) fall back to a short **PowerShell** `Get-AuthenticodeSignature` call so the simple name (often `Microsoft Windows`) can match **allowed_publishers**.

Compute a file hash for allowlisting:

```powershell
Get-FileHash -Algorithm SHA256 .\path\to\file.exe
```

## Limitations (Phase 1)

- Launched processes run as **SYSTEM** in session **0**; some installers need the user’s interactive session (future work).
- **CreateProcessAsUser** requires **SeAssignPrimaryTokenPrivilege** / **SeIncreaseQuotaPrivilege** (held by **LocalSystem** and elevated admins). Run the service as LocalSystem via `install.ps1`, or run `pam-agent.exe` from an **elevated** console for local tests.
- Phase 1 backend contract does not carry arguments, working directory, or user SID on the wire; those remain in local audit context where applicable. UAC integration is not implemented.
