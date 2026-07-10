import { useEffect, useState } from "react";
import { clearLog, getLogEntries, subscribeLog, type LogEntry } from "../lib/requestLog";
import { JsonViewer } from "../components/JsonViewer";

function statusColor(entry: LogEntry): string {
  if (!entry.ok) return "text-red-600";
  if (entry.status >= 200 && entry.status < 300) return "text-green-600";
  return "text-slate-600";
}

export function ActivityLogPage() {
  const [entries, setEntries] = useState<LogEntry[]>(() => getLogEntries());
  const [selected, setSelected] = useState<LogEntry | null>(null);

  useEffect(() => subscribeLog(setEntries), []);

  return (
    <div className="flex max-w-5xl gap-4">
      <div className="flex-1">
        <div className="mb-2 flex items-center justify-between">
          <h2 className="text-lg font-semibold">Activity Log</h2>
          <button
            className="btn-secondary text-xs"
            onClick={() => {
              clearLog();
              setSelected(null);
            }}
          >
            Clear
          </button>
        </div>
        <p className="mb-3 text-sm text-slate-500">
          Every request made by this client, most recent first — client-side only, mirrors the backend's own request
          logging concept for testing purposes.
        </p>
        <div className="card max-h-[70vh] overflow-auto p-0">
          <table className="table-base">
            <thead>
              <tr>
                <th>Method</th>
                <th>Path</th>
                <th>Status</th>
                <th>ms</th>
                <th>Time</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((e) => (
                <tr key={e.id} className="cursor-pointer hover:bg-slate-50" onClick={() => setSelected(e)}>
                  <td>{e.method}</td>
                  <td className="max-w-xs truncate">{e.path}</td>
                  <td className={statusColor(e)}>{e.status || "ERR"}</td>
                  <td>{Math.round(e.durationMs)}</td>
                  <td>{new Date(e.timestamp).toLocaleTimeString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {entries.length === 0 && <p className="p-3 text-sm text-slate-500">No requests yet.</p>}
        </div>
      </div>

      <div className="w-96 shrink-0">
        <h3 className="mb-2 text-sm font-semibold">Detail</h3>
        {selected ? (
          <div className="flex flex-col gap-3">
            <div className="card">
              <p className="text-xs text-slate-500">Request</p>
              <JsonViewer value={{ method: selected.method, path: selected.path, body: selected.requestBody }} />
            </div>
            <div className="card">
              <p className="text-xs text-slate-500">Response ({selected.status})</p>
              <JsonViewer value={selected.responseBody} />
            </div>
          </div>
        ) : (
          <p className="text-sm text-slate-500">Select a row to inspect it.</p>
        )}
      </div>
    </div>
  );
}
