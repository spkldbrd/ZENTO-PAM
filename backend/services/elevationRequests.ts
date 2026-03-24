import type { Pool } from "pg";
import { logAudit } from "./audit.js";
import { getAdminUser } from "./adminConfig.js";

export async function createElevationRequest(
  pool: Pool,
  input: {
    deviceId: string;
    user: string;
    exePath: string;
    hash: string;
    publisher: string;
  }
) {
  const result = await pool.query(
    `INSERT INTO elevation_requests (device_id, "user", exe_path, hash, publisher, status)
     VALUES ($1, $2, $3, $4, $5, 'pending')
     RETURNING id, status`,
    [input.deviceId, input.user, input.exePath, input.hash, input.publisher]
  );
  const row = result.rows[0] as { id: string; status: string };
  await logAudit(pool, {
    actor: "system",
    action: "elevation_request_created",
    entityType: "elevation_request",
    entityId: row.id,
    metadata: {
      request_id: row.id,
      device_id: input.deviceId,
      exe_path: input.exePath,
    },
  });
  return row;
}

export async function getElevationRequestById(pool: Pool, id: string) {
  const result = await pool.query(
    `SELECT er.id, er.device_id, er."user", er.exe_path, er.hash, er.publisher, er.status, er.created_at, er.resolved_at,
            d.hostname AS device_hostname
     FROM elevation_requests er
     JOIN devices d ON d.id = er.device_id
     WHERE er.id = $1`,
    [id]
  );
  return result.rows[0] ?? null;
}

/** Pass `null` for no status filter (all requests). Stable fields for GET /admin/requests. */
export async function listElevationRequests(pool: Pool, status: string | null) {
  const params: string[] = [];
  let where = "";
  if (status !== null) {
    params.push(status);
    where = `WHERE er.status = $1`;
  }
  const result = await pool.query(
    `SELECT er.id, er.device_id, er."user", er.exe_path, er.hash, er.publisher, er.status, er.created_at
     FROM elevation_requests er
     ${where}
     ORDER BY er.created_at DESC`,
    params
  );
  return result.rows as AdminRequestRow[];
}

export type AdminRequestRow = {
  id: string;
  device_id: string;
  user: string;
  exe_path: string;
  hash: string;
  publisher: string;
  status: string;
  created_at: Date | string;
};

export type DecisionResult =
  | { ok: false }
  | { ok: true; requestId: string; deviceId: string; decision: "approved" | "denied" };

export async function approveRequest(pool: Pool, requestId: string): Promise<DecisionResult> {
  const adminUser = getAdminUser();
  const result = await pool.query(
    `UPDATE elevation_requests
     SET status = 'approved', resolved_at = now()
     WHERE id = $1 AND status = 'pending'
     RETURNING id, device_id`,
    [requestId]
  );
  if (result.rowCount === 0) return { ok: false };
  const row = result.rows[0] as { id: string; device_id: string };
  await logAudit(pool, {
    actor: adminUser,
    action: "request_approved",
    entityType: "elevation_request",
    entityId: row.id,
    metadata: {
      request_id: row.id,
      admin_user: adminUser,
      decision: "approved",
      device_id: row.device_id,
    },
  });
  return { ok: true, requestId: row.id, deviceId: row.device_id, decision: "approved" };
}

export async function denyRequest(pool: Pool, requestId: string): Promise<DecisionResult> {
  const adminUser = getAdminUser();
  const result = await pool.query(
    `UPDATE elevation_requests
     SET status = 'denied', resolved_at = now()
     WHERE id = $1 AND status = 'pending'
     RETURNING id, device_id`,
    [requestId]
  );
  if (result.rowCount === 0) return { ok: false };
  const row = result.rows[0] as { id: string; device_id: string };
  await logAudit(pool, {
    actor: adminUser,
    action: "request_denied",
    entityType: "elevation_request",
    entityId: row.id,
    metadata: {
      request_id: row.id,
      admin_user: adminUser,
      decision: "denied",
      device_id: row.device_id,
    },
  });
  return { ok: true, requestId: row.id, deviceId: row.device_id, decision: "denied" };
}
