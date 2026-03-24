import type { FastifyInstance } from "fastify";
import type { Pool } from "pg";
import { approveBodySchema, denyBodySchema } from "../models/schemas.js";
import type { AdminRequestRow } from "../services/elevationRequests.js";
import {
  approveRequest,
  denyRequest,
  listElevationRequests,
} from "../services/elevationRequests.js";

function serializeAdminRequest(row: AdminRequestRow) {
  const created =
    row.created_at instanceof Date
      ? row.created_at.toISOString()
      : String(row.created_at);
  return {
    id: row.id,
    device_id: row.device_id,
    user: row.user,
    exe_path: row.exe_path,
    hash: row.hash,
    publisher: row.publisher,
    status: row.status,
    created_at: created,
  };
}

export async function registerAdminRoutes(app: FastifyInstance, pool: Pool) {
  app.get<{ Querystring: { status?: string } }>(
    "/admin/requests",
    async (request) => {
      const raw = request.query.status;
      const rows =
        raw === "all"
          ? await listElevationRequests(pool, null)
          : await listElevationRequests(pool, raw ?? "pending");
      return { requests: rows.map(serializeAdminRequest) };
    }
  );

  app.get("/admin/audit-logs", async () => {
    const result = await pool.query(
      `SELECT id, actor, action, entity_type, entity_id, metadata, created_at
       FROM audit_logs
       ORDER BY created_at DESC
       LIMIT 200`
    );
    return { logs: result.rows };
  });

  app.post("/admin/approve", async (request, reply) => {
    const parsed = approveBodySchema.safeParse(request.body);
    if (!parsed.success) {
      return reply.status(400).send({ error: parsed.error.flatten() });
    }
    const result = await approveRequest(pool, parsed.data.requestId);
    if (!result.ok) {
      return reply
        .status(409)
        .send({ error: "request not found or not pending" });
    }
    request.log.info(
      {
        endpoint: "POST /admin/approve",
        request_id: result.requestId,
        device_id: result.deviceId,
        decision: result.decision,
      },
      "elevation_request_decided"
    );
    return { ok: true, status: "approved" };
  });

  app.post("/admin/deny", async (request, reply) => {
    const parsed = denyBodySchema.safeParse(request.body);
    if (!parsed.success) {
      return reply.status(400).send({ error: parsed.error.flatten() });
    }
    const result = await denyRequest(pool, parsed.data.requestId);
    if (!result.ok) {
      return reply
        .status(409)
        .send({ error: "request not found or not pending" });
    }
    request.log.info(
      {
        endpoint: "POST /admin/deny",
        request_id: result.requestId,
        device_id: result.deviceId,
        decision: result.decision,
      },
      "elevation_request_decided"
    );
    return { ok: true, status: "denied" };
  });
}
