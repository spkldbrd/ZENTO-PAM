import type { Pool } from "pg";

export async function logAudit(
  pool: Pool,
  params: {
    actor: string;
    action: string;
    entityType: string;
    entityId: string | null;
    metadata?: Record<string, unknown>;
  }
) {
  await pool.query(
    `INSERT INTO audit_logs (actor, action, entity_type, entity_id, metadata)
     VALUES ($1, $2, $3, $4, $5::jsonb)`,
    [
      params.actor,
      params.action,
      params.entityType,
      params.entityId,
      JSON.stringify(params.metadata ?? {}),
    ]
  );
}
