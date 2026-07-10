import { useState } from "react";
import { ApiError, request } from "../lib/apiClient";
import { RequestResult } from "../components/RequestResult";

const METHODS = ["GET", "POST", "PATCH", "PUT", "DELETE"] as const;

export function ApiConsolePage() {
  const [method, setMethod] = useState<(typeof METHODS)[number]>("GET");
  const [path, setPath] = useState("/v1/me");
  const [body, setBody] = useState("");
  const [auth, setAuth] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);
  const [result, setResult] = useState<unknown>(null);

  const send = async () => {
    setLoading(true);
    setError(null);
    setResult(null);
    try {
      let parsedBody: unknown;
      if (body.trim()) {
        try {
          parsedBody = JSON.parse(body);
        } catch {
          throw new ApiError(0, "Request body is not valid JSON");
        }
      }
      const res = await request(path, { method, body: parsedBody, auth });
      setResult(res);
    } catch (err) {
      setError(err instanceof ApiError ? err : new ApiError(0, String(err)));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex max-w-3xl flex-col gap-6">
      <div>
        <h2 className="text-lg font-semibold">API Console</h2>
        <p className="text-sm text-slate-500">
          Fire an arbitrary request against the API — a catch-all for anything not covered by a dedicated page.
          Settings (tenant/lang/app-version headers) and the stored access token are applied automatically.
        </p>
      </div>

      <div className="card flex flex-col gap-3">
        <div className="flex gap-2">
          <select className="input !w-32" value={method} onChange={(e) => setMethod(e.target.value as typeof method)}>
            {METHODS.map((m) => (
              <option key={m} value={m}>
                {m}
              </option>
            ))}
          </select>
          <input className="input flex-1" value={path} onChange={(e) => setPath(e.target.value)} placeholder="/v1/me" />
        </div>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={auth} onChange={(e) => setAuth(e.target.checked)} />
          Include Authorization: Bearer header
        </label>
        <div>
          <label className="label">JSON body (optional)</label>
          <textarea
            className="input h-32 font-mono text-xs"
            placeholder='{"key": "value"}'
            value={body}
            onChange={(e) => setBody(e.target.value)}
          />
        </div>
        <button className="btn self-start" onClick={() => void send()} disabled={loading}>
          {loading ? "Sending…" : "Send"}
        </button>
      </div>

      <div className="card">
        <RequestResult loading={loading} error={error} result={result} />
      </div>
    </div>
  );
}
