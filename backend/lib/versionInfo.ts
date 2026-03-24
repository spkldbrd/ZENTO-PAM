import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));

export function getApiVersionPayload(): { version: string; name: string } {
  let version = "0.0.0";
  try {
    const pkgPath = join(__dirname, "..", "..", "package.json");
    const pkg = JSON.parse(readFileSync(pkgPath, "utf8")) as { version?: string; name?: string };
    version = pkg.version ?? version;
    return { version, name: pkg.name ?? "pam-backend" };
  } catch {
    return { version, name: "pam-backend" };
  }
}
