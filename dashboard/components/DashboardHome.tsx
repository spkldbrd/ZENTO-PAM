"use client";

import { getApiBase } from "@/lib/api";
import { useCallback, useEffect, useState } from "react";

type ElevationRequest = {
  id: string;
  device_id: string;
  user: string;
  exe_path: string;
  hash: string;
  publisher: string;
  status: string;
  created_at: string;
};

type AuditLog = {
  id: string;
  actor: string;
  action: string;
  entity_type: string;
  entity_id: string | null;
  metadata: Record<string, unknown>;
  created_at: string;
};

export function DashboardHome() {
  const api = getApiBase();
  const [requests, setRequests] = useState<ElevationRequest[]>([]);
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [filter, setFilter] = useState<"pending" | "all">("pending");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    try {
      const q = filter === "all" ? "all" : "pending";
      const [rRes, lRes] = await Promise.all([
        fetch(`${api}/admin/requests?status=${q}`),
        fetch(`${api}/admin/audit-logs`),
      ]);
      if (!rRes.ok) throw new Error(`Requests failed: ${rRes.status}`);
      if (!lRes.ok) throw new Error(`Audit failed: ${lRes.status}`);
      const rJson = (await rRes.json()) as { requests: ElevationRequest[] };
      const lJson = (await lRes.json()) as { logs: AuditLog[] };
      setRequests(rJson.requests);
      setLogs(lJson.logs);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Load failed");
    } finally {
      setLoading(false);
    }
  }, [api, filter]);

  useEffect(() => {
    setLoading(true);
    void load();
  }, [load]);

  async function approve(id: string) {
    setBusyId(id);
    setError(null);
    try {
      const res = await fetch(`${api}/admin/approve`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ requestId: id }),
      });
      if (!res.ok) {
        const j = await res.json().catch(() => ({}));
        throw new Error((j as { error?: string }).error ?? res.statusText);
      }
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Approve failed");
    } finally {
      setBusyId(null);
    }
  }

  async function deny(id: string) {
    setBusyId(id);
    setError(null);
    try {
      const res = await fetch(`${api}/admin/deny`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ requestId: id }),
      });
      if (!res.ok) {
        const j = await res.json().catch(() => ({}));
        throw new Error((j as { error?: string }).error ?? res.statusText);
      }
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Deny failed");
    } finally {
      setBusyId(null);
    }
  }

  return (
    <main className="mx-auto max-w-6xl px-4 py-10">
      <header className="mb-10 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-white">
            Elevation requests
          </h1>
          <p className="mt-1 text-sm text-slate-400">
            Phase 1 dashboard — no authentication
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            onClick={() => {
              setFilter("pending");
            }}
            className={`rounded-md px-3 py-1.5 text-sm font-medium ${
              filter === "pending"
                ? "bg-sky-600 text-white"
                : "bg-slate-800 text-slate-300 hover:bg-slate-700"
            }`}
          >
            Pending
          </button>
          <button
            type="button"
            onClick={() => setFilter("all")}
            className={`rounded-md px-3 py-1.5 text-sm font-medium ${
              filter === "all"
                ? "bg-sky-600 text-white"
                : "bg-slate-800 text-slate-300 hover:bg-slate-700"
            }`}
          >
            All
          </button>
          <button
            type="button"
            onClick={() => void load()}
            className="rounded-md bg-slate-800 px-3 py-1.5 text-sm font-medium text-slate-200 hover:bg-slate-700"
          >
            Refresh
          </button>
        </div>
      </header>

      {error && (
        <div className="mb-6 rounded-md border border-red-900/60 bg-red-950/40 px-4 py-3 text-sm text-red-200">
          {error}
        </div>
      )}

      <section className="mb-12 overflow-hidden rounded-lg border border-slate-800 bg-slate-900/50">
        <div className="border-b border-slate-800 px-4 py-3">
          <h2 className="text-sm font-medium text-slate-200">Requests</h2>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full min-w-[800px] text-left text-sm">
            <thead className="bg-slate-900 text-xs uppercase text-slate-500">
              <tr>
                <th className="px-4 py-3">Time</th>
                <th className="px-4 py-3">Device ID</th>
                <th className="px-4 py-3">User</th>
                <th className="px-4 py-3">Executable</th>
                <th className="px-4 py-3">Publisher</th>
                <th className="px-4 py-3">Hash</th>
                <th className="px-4 py-3">Status</th>
                <th className="px-4 py-3 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800">
              {loading ? (
                <tr>
                  <td colSpan={8} className="px-4 py-8 text-center text-slate-500">
                    Loading…
                  </td>
                </tr>
              ) : requests.length === 0 ? (
                <tr>
                  <td colSpan={8} className="px-4 py-8 text-center text-slate-500">
                    No requests
                  </td>
                </tr>
              ) : (
                requests.map((row) => (
                  <tr key={row.id} className="text-slate-300">
                    <td className="whitespace-nowrap px-4 py-3 text-slate-400">
                      {new Date(row.created_at).toLocaleString()}
                    </td>
                    <td
                      className="max-w-[200px] truncate px-4 py-3 font-mono text-xs text-slate-400"
                      title={row.device_id}
                    >
                      {row.device_id}
                    </td>
                    <td className="px-4 py-3">{row.user}</td>
                    <td className="max-w-xs truncate px-4 py-3 font-mono text-xs">
                      {row.exe_path}
                    </td>
                    <td className="px-4 py-3">{row.publisher || "—"}</td>
                    <td className="max-w-[120px] truncate px-4 py-3 font-mono text-xs text-slate-500">
                      {row.hash || "—"}
                    </td>
                    <td className="px-4 py-3">
                      <span
                        className={`rounded px-2 py-0.5 text-xs font-medium ${
                          row.status === "pending"
                            ? "bg-amber-950 text-amber-200"
                            : row.status === "approved"
                              ? "bg-emerald-950 text-emerald-200"
                              : "bg-slate-800 text-slate-300"
                        }`}
                      >
                        {row.status}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-right">
                      {row.status === "pending" ? (
                        <div className="flex justify-end gap-2">
                          <button
                            type="button"
                            disabled={busyId === row.id}
                            onClick={() => void approve(row.id)}
                            className="rounded-md bg-emerald-700 px-2.5 py-1 text-xs font-medium text-white hover:bg-emerald-600 disabled:opacity-50"
                          >
                            Approve
                          </button>
                          <button
                            type="button"
                            disabled={busyId === row.id}
                            onClick={() => void deny(row.id)}
                            className="rounded-md bg-slate-700 px-2.5 py-1 text-xs font-medium text-white hover:bg-red-900/80 disabled:opacity-50"
                          >
                            Deny
                          </button>
                        </div>
                      ) : (
                        <span className="text-slate-600">—</span>
                      )}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </section>

      <section className="overflow-hidden rounded-lg border border-slate-800 bg-slate-900/50">
        <div className="border-b border-slate-800 px-4 py-3">
          <h2 className="text-sm font-medium text-slate-200">Audit log</h2>
          <p className="text-xs text-slate-500">Most recent 200 events</p>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full min-w-[640px] text-left text-sm">
            <thead className="bg-slate-900 text-xs uppercase text-slate-500">
              <tr>
                <th className="px-4 py-3">Time</th>
                <th className="px-4 py-3">Actor</th>
                <th className="px-4 py-3">Action</th>
                <th className="px-4 py-3">Entity</th>
                <th className="px-4 py-3">Details</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800">
              {logs.map((log) => (
                <tr key={log.id} className="text-slate-300">
                  <td className="whitespace-nowrap px-4 py-3 text-slate-400">
                    {new Date(log.created_at).toLocaleString()}
                  </td>
                  <td className="px-4 py-3">{log.actor}</td>
                  <td className="px-4 py-3">{log.action}</td>
                  <td className="px-4 py-3 font-mono text-xs text-slate-500">
                    {log.entity_type}
                    {log.entity_id ? ` / ${log.entity_id.slice(0, 8)}…` : ""}
                  </td>
                  <td className="max-w-md truncate px-4 py-3 font-mono text-xs text-slate-500">
                    {JSON.stringify(log.metadata)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </main>
  );
}
