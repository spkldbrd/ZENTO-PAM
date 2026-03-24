/**
 * API base URL for browser `fetch` calls.
 *
 * `NEXT_PUBLIC_*` is inlined at **build** time. A default of `http://localhost:3001`
 * breaks remote users (their browser calls *their* localhost). When the baked URL is
 * empty or points at localhost/127.0.0.1, we derive the API from `window.location`
 * (same host as the dashboard, port 3001). Set `NEXT_PUBLIC_API_URL` to an explicit
 * non-local URL at build time if the API is on a different host or path (e.g. TLS).
 */
function isLocalPlaceholder(url: string | undefined): boolean {
  const u = url?.trim();
  if (!u) return true;
  try {
    const parsed = new URL(u);
    return parsed.hostname === "localhost" || parsed.hostname === "127.0.0.1";
  } catch {
    return false;
  }
}

export function getApiBase(): string {
  const env = process.env.NEXT_PUBLIC_API_URL?.trim() ?? "";

  if (typeof window !== "undefined") {
    if (!isLocalPlaceholder(env)) {
      return env;
    }
    const { protocol, hostname } = window.location;
    return `${protocol}//${hostname}:3001`;
  }

  if (env && !isLocalPlaceholder(env)) {
    return env;
  }
  return "http://localhost:3001";
}
