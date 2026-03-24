/** Phase 1: single logical admin identity (no auth). Override with PAM_ADMIN_USER. */
export function getAdminUser(): string {
  return process.env.PAM_ADMIN_USER?.trim() || "admin";
}
