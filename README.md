# PAM Platform (Phase 1)

Privileged Access Management backend: Fastify + PostgreSQL + Redis, technician dashboard (Next.js). Windows agent is out of scope for this repo.

## Prerequisites

- Docker and Docker Compose
- Optional: Node.js 22+ for local `npm run dev` without containers

## Configuration

Copy [.env.example](.env.example) to `.env` at the repository root and adjust secrets for non-local use.

- `NEXT_PUBLIC_API_URL` must be the URL **the browser** uses to reach the API (e.g. `http://localhost:3001` on your machine, or `http://YOUR_PUBLIC_IP:3001` on a VPS). The dashboard image bakes this in at **build** time—after changing it, run `docker compose ... build dashboard` (or `up --build`).

Compose **does not publish** Postgres or Redis on the host (`5432` / `6379`), so the stack can coexist with OS-level PostgreSQL and Redis. Access the DB from the host with:

`docker compose -f infra/docker/docker-compose.yml exec postgres psql -U pam -d pam`

## Run with Docker

From the **repository root**:

```bash
docker compose -f infra/docker/docker-compose.yml --env-file .env up --build -d
```

Foreground (logs in terminal):

```bash
docker compose -f infra/docker/docker-compose.yml --env-file .env up --build
```

Or use the helpers:

- Linux/macOS: [infra/scripts/dev-up.sh](infra/scripts/dev-up.sh)
- Windows PowerShell: [infra/scripts/dev-up.ps1](infra/scripts/dev-up.ps1)

Stop:

```bash
docker compose -f infra/docker/docker-compose.yml down
```

### Validated startup and health check

The following was verified on an Ubuntu 24.04 host with Docker 29.x (2026-03-24):

1. Create `.env` from `.env.example` (set `NEXT_PUBLIC_API_URL` to the URL your browser will use for the API).
2. `docker compose -f infra/docker/docker-compose.yml --env-file .env up --build -d`
3. Wait until all services report healthy:

```bash
docker compose -f infra/docker/docker-compose.yml ps
```

Expect **(healthy)** for `postgres`, `redis`, `backend`, and `dashboard`. Backend and dashboard healthchecks call `GET http://127.0.0.1:3001/health` and `GET http://127.0.0.1:3000/` from inside the containers. From the host you can also call `GET http://127.0.0.1:3001/version` for build metadata.

4. Confirm migrations ran on first backend start:

```bash
docker compose -f infra/docker/docker-compose.yml logs backend | head -20
```

You should see lines such as `Applied migration: 001_init.sql` and `Applied migration: 002_seed_policy.sql` once per fresh volume.

5. Automated API check (requires `curl` and Python 3 on the machine where you run it; default API URL `http://127.0.0.1:3001`):

```bash
python3 infra/scripts/validate-phase1-api.py http://127.0.0.1:3001
```

6. Dashboard HTTP check from the host:

```bash
curl -sS -o /dev/null -w '%{http_code}\n' http://127.0.0.1:3000/
```

Expect `200`. Open `http://localhost:3000` (or your host/IP) in a browser: pending requests load from the API; Approve/Deny call the admin endpoints (CORS is enabled with `origin: true` on the API).

### Windows / CRLF note

If `docker logs` shows `exec ./docker-entrypoint.sh: no such file or directory`, the entrypoint likely has Windows CRLF line endings. The backend image runs `sed -i 's/\r$//' docker-entrypoint.sh` after copy; keeping [`.gitattributes`](.gitattributes) (`*.sh text eol=lf`) avoids the issue at the source.

Services (default host ports):

| Service     | Host port (default) |
| ----------- | -------------------- |
| Dashboard   | 3000                 |
| Backend API | 3001                 |
| PostgreSQL  | (not published)      |
| Redis       | (not published)      |

Phase 1 does **not** enforce authentication on `/agent/*` or `/admin/*`.

The HTTP contract for the Windows agent and dashboard is locked in **[API_SPEC.md](API_SPEC.md)** (request/response shapes, `status` values, lifecycle, audit field mapping).

## Phase 1 Windows Agent Integration

Use this sequence when wiring the Windows agent to a running backend (same host or LAN/VPS; use your real base URL instead of `http://127.0.0.1:3001`).

1. **Start the stack** (from repo root):  
   `docker compose -f infra/docker/docker-compose.yml --env-file .env up --build -d`  
   Wait until `docker compose ... ps` shows `backend` and `dashboard` as **healthy**.

2. **Verify connectivity:**  
   `curl -sS http://127.0.0.1:3001/health` → `{"ok":true}`  
   `curl -sS http://127.0.0.1:3001/version` → JSON with `version`, `phase`, `service`.

3. **Register the agent:**  
   `POST /agent/register` with `hostname` and `agent_version`. Store the returned `deviceId` (UUID) locally.

4. **Submit an elevation request:**  
   `POST /agent/elevation-request` with `device_id`, `user`, `exe_path`, and optional `hash` / `publisher`. Store the returned `id` (request UUID) and `status` (`pending`).

5. **Approve or deny (technician):**  
   From the dashboard at `http://localhost:3000` (or your host), or via API:  
   `POST /admin/approve` or `POST /admin/deny` with body `{ "requestId": "<uuid>" }`.

6. **Poll until terminal:**  
   `GET /agent/elevation-requests/<id>` until `status` is `approved` or `denied`, then launch or block per your agent logic.

7. **Policy (optional):**  
   `GET /agent/policy` for JSON rules (e.g. `allowed_publishers`).

Optional: run `python3 infra/scripts/validate-phase1-api.py http://127.0.0.1:3001` on a machine that can reach the API to exercise the same flow.

## Phase 1 Verified Flow

End-to-end check (same sequence as [infra/scripts/validate-phase1-api.py](infra/scripts/validate-phase1-api.py)):

1. `POST /agent/register` with `hostname` and `agent_version` → receive `deviceId`.
2. `POST /agent/elevation-request` twice (same `device_id`) → receive two request `id` values, both `status: pending`.
3. `GET /admin/requests?status=pending` → both requests listed; each item includes `id`, `device_id`, `user`, `exe_path`, `hash`, `publisher`, `status`, `created_at`.
4. `POST /admin/approve` with `requestId` of the first → `{"ok":true,"status":"approved"}`.
5. `GET /agent/elevation-requests/:id` for that id → `status` is `approved` and `resolved_at` is set.
6. `POST /admin/deny` with `requestId` of the second → `{"ok":true,"status":"denied"}`.
7. `GET /agent/elevation-requests/:id` for the second id → `status` is `denied`.
8. `GET /agent/policy` → JSON including `allowed_publishers` (seeded default).
9. `GET /admin/audit-logs` → includes `elevation_request_created`, `request_approved`, and `request_denied` actions; approve/deny rows include `metadata.decision`, `metadata.request_id`, and `metadata.admin_user` (see [API_SPEC.md](API_SPEC.md)).

This flow was executed successfully against a running compose stack (HTTP 200 on all steps).

## Reference repositories (read-only)

Study only; do not modify after cloning:

```bash
mkdir -p references
git clone https://github.com/amidaware/tacticalrmm references/tacticalrmm
git clone https://github.com/jumpserver/jumpserver references/jumpserver
```

The [`.gitignore`](.gitignore) excludes `references/` so large clones stay out of version control.

## Example flow (curl)

Replace `API` with your API base URL (e.g. `http://localhost:3001`).

**0. Health and version**

```bash
curl -sS "$API/health"
curl -sS "$API/version"
```

**1. Register device**

```bash
curl -sS -X POST "$API/agent/register" \
  -H "Content-Type: application/json" \
  -d '{"hostname":"PC-01","agent_version":"0.0.1"}'
```

Save `deviceId` from the response.

**2. Submit elevation request**

```bash
curl -sS -X POST "$API/agent/elevation-request" \
  -H "Content-Type: application/json" \
  -d "{\"device_id\":\"DEVICE_UUID\",\"user\":\"DOMAIN\\\\user\",\"exe_path\":\"C:\\\\Windows\\\\System32\\\\cmd.exe\",\"hash\":\"abc123\",\"publisher\":\"Microsoft Corporation\"}"
```

Save `id` from the response.

**3. List pending requests (admin)**

```bash
curl -sS "$API/admin/requests?status=pending"
```

**4. Approve or deny**

```bash
curl -sS -X POST "$API/admin/approve" \
  -H "Content-Type: application/json" \
  -d '{"requestId":"REQUEST_UUID"}'

curl -sS -X POST "$API/admin/deny" \
  -H "Content-Type: application/json" \
  -d '{"requestId":"REQUEST_UUID"}'
```

**5. Poll request status (agent)**

```bash
curl -sS "$API/agent/elevation-requests/REQUEST_UUID"
```

**6. Policy for agents**

```bash
curl -sS "$API/agent/policy"
```

**7. Audit log (dashboard API)**

```bash
curl -sS "$API/admin/audit-logs"
```

## Project layout

```
backend/          # Fastify API (TypeScript)
  api/            # Route registration
  lib/            # Shared helpers (e.g. version payload)
  models/         # Zod schemas
  services/       # Business logic
  db/             # Pool, migration runner, SQL migrations (db/migrations/)
dashboard/        # Next.js + Tailwind (App Router)
infra/docker/     # Compose + Dockerfiles
infra/scripts/    # dev-up / dev-down, validate-phase1-api.py
```

## Local development (without Docker for apps)

**Backend**

```bash
cd backend
export DATABASE_URL=postgresql://pam:pam_dev_change_me@localhost:5432/pam
npm install
npm run build
node dist/db/migrate.js
npm run dev
```

**Dashboard**

```bash
cd dashboard
export NEXT_PUBLIC_API_URL=http://localhost:3001
npm install
npm run dev
```

PostgreSQL and Redis must be running (e.g. only the `postgres` and `redis` services from this Compose file, or local installs). If you map DB/Redis ports in a custom override, adjust `DATABASE_URL` accordingly.
