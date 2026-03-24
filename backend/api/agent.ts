import type { FastifyInstance } from "fastify";
import type { Pool } from "pg";
import {
  elevationRequestBodySchema,
  registerBodySchema,
} from "../models/schemas.js";
import { registerDevice } from "../services/devices.js";
import {
  createElevationRequest,
  getElevationRequestById,
} from "../services/elevationRequests.js";
import { getAgentPolicy } from "../services/policy.js";

function toIso(v: unknown): string | null {
  if (v == null) return null;
  if (v instanceof Date) return v.toISOString();
  return String(v);
}

export async function registerAgentRoutes(app: FastifyInstance, pool: Pool) {
  app.post("/agent/register", async (request, reply) => {
    const parsed = registerBodySchema.safeParse(request.body);
    if (!parsed.success) {
      return reply.status(400).send({ error: parsed.error.flatten() });
    }
    const { hostname, agent_version } = parsed.data;
    const deviceId = await registerDevice(pool, hostname, agent_version);
    request.log.info(
      { endpoint: "POST /agent/register", device_id: deviceId },
      "device_registered"
    );
    return { deviceId };
  });

  app.post("/agent/elevation-request", async (request, reply) => {
    const parsed = elevationRequestBodySchema.safeParse(request.body);
    if (!parsed.success) {
      return reply.status(400).send({ error: parsed.error.flatten() });
    }
    const b = parsed.data;
    try {
      const row = await createElevationRequest(pool, {
        deviceId: b.device_id,
        user: b.user,
        exePath: b.exe_path,
        hash: b.hash,
        publisher: b.publisher,
      });
      request.log.info(
        {
          endpoint: "POST /agent/elevation-request",
          request_id: row.id,
          device_id: b.device_id,
        },
        "elevation_request_created"
      );
      return { id: row.id, status: row.status };
    } catch (e: unknown) {
      const err = e as { code?: string };
      if (err.code === "23503") {
        return reply.status(404).send({ error: "device not found" });
      }
      throw e;
    }
  });

  app.get<{ Params: { id: string } }>(
    "/agent/elevation-requests/:id",
    async (request, reply) => {
      const { id } = request.params;
      const row = await getElevationRequestById(pool, id);
      if (!row) return reply.status(404).send({ error: "not found" });
      return {
        id: row.id,
        device_id: row.device_id,
        user: row.user,
        exe_path: row.exe_path,
        hash: row.hash,
        publisher: row.publisher,
        status: row.status,
        created_at: toIso(row.created_at) ?? String(row.created_at),
        resolved_at: toIso(row.resolved_at),
        device_hostname: row.device_hostname,
      };
    }
  );

  app.get<{ Querystring: { deviceId?: string } }>(
    "/agent/policy",
    async (request) => {
      const deviceId = request.query.deviceId ?? null;
      const rules = await getAgentPolicy(pool, deviceId);
      return rules;
    }
  );
}
