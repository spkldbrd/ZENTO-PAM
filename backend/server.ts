import cors from "@fastify/cors";
import Fastify from "fastify";
import { registerAdminRoutes } from "./api/admin.js";
import { registerAgentRoutes } from "./api/agent.js";
import { pool } from "./db/pool.js";
import { getApiVersionPayload } from "./lib/versionInfo.js";

const port = Number(process.env.PORT ?? 3001);
const host = process.env.HOST ?? "0.0.0.0";

async function main() {
  const app = Fastify({ logger: true });

  await app.register(cors, {
    origin: true,
    methods: ["GET", "POST", "OPTIONS"],
  });

  await registerAgentRoutes(app, pool);
  await registerAdminRoutes(app, pool);

  app.get("/health", async () => ({ ok: true }));

  app.get("/version", async () => {
    const { name, version } = getApiVersionPayload();
    return {
      ok: true,
      service: name,
      version,
      phase: 1,
    };
  });

  try {
    await app.listen({ port, host });
  } catch (err) {
    app.log.error(err);
    process.exit(1);
  }
}

main();
