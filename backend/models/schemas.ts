import { z } from "zod";

export const registerBodySchema = z.object({
  hostname: z.string().min(1),
  agent_version: z.string().optional().default(""),
});

export const elevationRequestBodySchema = z.object({
  device_id: z.string().uuid(),
  user: z.string().min(1),
  exe_path: z.string().min(1),
  hash: z.string().optional().default(""),
  publisher: z.string().optional().default(""),
});

export const approveBodySchema = z.object({
  requestId: z.string().uuid(),
});

export const denyBodySchema = z.object({
  requestId: z.string().uuid(),
});
