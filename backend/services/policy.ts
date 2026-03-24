import type { Pool } from "pg";

type PolicyRules = {
  allowed_publishers?: string[];
  [key: string]: unknown;
};

const fallback: PolicyRules = {
  allowed_publishers: ["Microsoft Corporation"],
};

export async function getAgentPolicy(pool: Pool, _deviceId?: string | null): Promise<PolicyRules> {
  const result = await pool.query(
    `SELECT rules FROM policies WHERE name = 'default' ORDER BY created_at ASC LIMIT 1`
  );
  if (result.rows.length === 0) return fallback;
  const rules = result.rows[0].rules as PolicyRules;
  return rules && typeof rules === "object" ? rules : fallback;
}
