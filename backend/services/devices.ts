import type { Pool } from "pg";

export async function registerDevice(
  pool: Pool,
  hostname: string,
  agentVersion: string
): Promise<string> {
  const result = await pool.query(
    `INSERT INTO devices (hostname, agent_version, last_seen)
     VALUES ($1, $2, now())
     ON CONFLICT (hostname) DO UPDATE SET
       agent_version = EXCLUDED.agent_version,
       last_seen = now()
     RETURNING id`,
    [hostname, agentVersion]
  );
  return result.rows[0].id as string;
}
