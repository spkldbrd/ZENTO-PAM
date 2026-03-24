#!/usr/bin/env python3
"""Run against a running Phase 1 API (default http://127.0.0.1:3001). Requires curl."""
import json
import subprocess
import sys

API = sys.argv[1] if len(sys.argv) > 1 else "http://127.0.0.1:3001"


def curl(method: str, path: str, data: dict | None = None) -> tuple[int, str]:
    cmd = ["curl", "-sS", "-w", "\n%{http_code}", "-X", method, f"{API}{path}"]
    if data is not None:
        cmd.extend(["-H", "Content-Type: application/json", "-d", json.dumps(data)])
    p = subprocess.run(cmd, capture_output=True, text=True)
    out = p.stdout.strip()
    if "\n" not in out:
        return 0, out
    body, _, code = out.rpartition("\n")
    return int(code), body


def main() -> None:
    lines: list[str] = []
    lines.append(f"API={API}\n")

    code, body = curl("GET", "/health", None)
    lines.append(f"health http={code} {body}\n")
    code, body = curl("GET", "/version", None)
    lines.append(f"version http={code} {body}\n")

    code, body = curl("POST", "/agent/register", {"hostname": "validate-pc", "agent_version": "0.0.1"})
    lines.append(f"register http={code} {body}\n")
    if code != 200:
        print("".join(lines))
        sys.exit(1)
    device_id = json.loads(body)["deviceId"]

    code, body = curl(
        "POST",
        "/agent/elevation-request",
        {
            "device_id": device_id,
            "user": "DOMAIN\\user",
            "exe_path": "C:\\Windows\\System32\\cmd.exe",
            "hash": "abc123",
            "publisher": "Microsoft Corporation",
        },
    )
    lines.append(f"elevation1 http={code} {body}\n")
    rid1 = json.loads(body)["id"]

    code, body = curl(
        "POST",
        "/agent/elevation-request",
        {
            "device_id": device_id,
            "user": "DOMAIN\\user2",
            "exe_path": "C:\\Windows\\System32\\notepad.exe",
            "hash": "def456",
            "publisher": "Microsoft Corporation",
        },
    )
    lines.append(f"elevation2 http={code} {body}\n")
    rid2 = json.loads(body)["id"]

    code, body = curl("GET", "/admin/requests?status=pending", None)
    lines.append(f"admin/requests http={code} {body[:500]}\n")

    code, body = curl("POST", "/admin/approve", {"requestId": rid1})
    lines.append(f"approve http={code} {body}\n")

    code, body = curl("GET", f"/agent/elevation-requests/{rid1}", None)
    lines.append(f"poll approved http={code} {body}\n")

    code, body = curl("POST", "/admin/deny", {"requestId": rid2})
    lines.append(f"deny http={code} {body}\n")

    code, body = curl("GET", f"/agent/elevation-requests/{rid2}", None)
    lines.append(f"poll denied http={code} {body}\n")

    code, body = curl("GET", "/agent/policy", None)
    lines.append(f"policy http={code} {body}\n")

    code, body = curl("GET", "/admin/audit-logs", None)
    lines.append(f"audit-logs http={code} len={len(body)}\n")

    code, body = curl("GET", "/health", None)
    lines.append(f"health http={code} {body}\n")

    print("".join(lines))


if __name__ == "__main__":
    main()
