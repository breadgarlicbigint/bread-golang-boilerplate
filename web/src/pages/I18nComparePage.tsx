import { useState } from "react";
import { ApiError, request } from "../lib/apiClient";
import { RequestResult } from "../components/RequestResult";

const METHODS = ["GET", "POST", "PATCH", "PUT", "DELETE"] as const;

interface Preset {
  label: string;
  method: (typeof METHODS)[number];
  path: string;
  body?: string;
  auth: boolean;
}

// Public endpoints only (auth: false) so the preset works without logging in first.
const PRESETS: Preset[] = [
  {
    label: "Invalid login credentials",
    method: "POST",
    path: "/v1/auth/login",
    body: JSON.stringify({ email: "nobody@example.com", password: "WrongPassword123" }, null, 2),
    auth: false,
  },
  {
    label: "Missing registration fields",
    method: "POST",
    path: "/v1/auth/register",
    body: "{}",
    auth: false,
  },
  {
    label: "Weak password on register",
    method: "POST",
    path: "/v1/auth/register",
    body: JSON.stringify(
      { email: "test@example.com", username: "testuser99", password: "short", firstName: "Test", lastName: "User" },
      null,
      2,
    ),
    auth: false,
  },
];

interface ColumnState {
  loading: boolean;
  error: ApiError | null;
  result: unknown;
}

const emptyColumn: ColumnState = { loading: false, error: null, result: null };

export function I18nComparePage() {
  const [method, setMethod] = useState<(typeof METHODS)[number]>(PRESETS[0].method);
  const [path, setPath] = useState(PRESETS[0].path);
  const [body, setBody] = useState(PRESETS[0].body ?? "");
  const [auth, setAuth] = useState(PRESETS[0].auth);
  const [langA, setLangA] = useState("en");
  const [langB, setLangB] = useState("id");
  const [colA, setColA] = useState<ColumnState>(emptyColumn);
  const [colB, setColB] = useState<ColumnState>(emptyColumn);

  const applyPreset = (p: Preset) => {
    setMethod(p.method);
    setPath(p.path);
    setBody(p.body ?? "");
    setAuth(p.auth);
    setColA(emptyColumn);
    setColB(emptyColumn);
  };

  const fireOne = async (lang: string, setCol: (c: ColumnState) => void) => {
    setCol({ loading: true, error: null, result: null });
    try {
      let parsedBody: unknown;
      if (body.trim()) {
        try {
          parsedBody = JSON.parse(body);
        } catch {
          throw new ApiError(0, "Request body is not valid JSON");
        }
      }
      const res = await request(path, { method, body: parsedBody, auth, lang });
      setCol({ loading: false, error: null, result: res });
    } catch (err) {
      setCol({ loading: false, error: err instanceof ApiError ? err : new ApiError(0, String(err)), result: null });
    }
  };

  const compare = () => {
    void fireOne(langA, setColA);
    void fireOne(langB, setColB);
  };

  const busy = colA.loading || colB.loading;

  return (
    <div className="flex max-w-5xl flex-col gap-6">
      <div>
        <h2 className="text-lg font-semibold">i18n Compare</h2>
        <p className="text-sm text-slate-500">
          Fires the same request twice with different <code>x-custom-lang</code> header values (this overrides the
          Settings page language for these two requests only) so the translated <code>message</code>/
          <code>errors[]</code> text can be compared side by side. A locale missing a key falls back to English
          automatically — try comparing "Missing registration fields" to see it, since <code>id.json</code> has no{" "}
          <code>validation.*</code> section yet.
        </p>
      </div>

      <div className="card flex flex-col gap-3">
        <div>
          <label className="label">Presets</label>
          <div className="flex flex-wrap gap-2">
            {PRESETS.map((p) => (
              <button key={p.label} className="btn-secondary" onClick={() => applyPreset(p)}>
                {p.label}
              </button>
            ))}
          </div>
        </div>

        <div className="flex gap-2">
          <select className="input !w-32" value={method} onChange={(e) => setMethod(e.target.value as typeof method)}>
            {METHODS.map((m) => (
              <option key={m} value={m}>
                {m}
              </option>
            ))}
          </select>
          <input
            className="input flex-1"
            value={path}
            onChange={(e) => setPath(e.target.value)}
            placeholder="/v1/auth/login"
          />
        </div>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={auth} onChange={(e) => setAuth(e.target.checked)} />
          Include Authorization: Bearer header
        </label>
        <div>
          <label className="label">JSON body (optional)</label>
          <textarea
            className="input h-28 font-mono text-xs"
            placeholder='{"key": "value"}'
            value={body}
            onChange={(e) => setBody(e.target.value)}
          />
        </div>

        <div className="flex gap-2">
          <div className="flex-1">
            <label className="label">Language A</label>
            <input className="input" value={langA} onChange={(e) => setLangA(e.target.value)} placeholder="en" />
          </div>
          <div className="flex-1">
            <label className="label">Language B</label>
            <input className="input" value={langB} onChange={(e) => setLangB(e.target.value)} placeholder="id" />
          </div>
        </div>

        <button className="btn self-start" onClick={compare} disabled={busy}>
          {busy ? "Sending…" : "Compare"}
        </button>
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div className="card">
          <h3 className="mb-2 text-sm font-semibold uppercase tracking-wide text-slate-400">
            x-custom-lang: {langA || "(none)"}
          </h3>
          <RequestResult loading={colA.loading} error={colA.error} result={colA.result} />
        </div>
        <div className="card">
          <h3 className="mb-2 text-sm font-semibold uppercase tracking-wide text-slate-400">
            x-custom-lang: {langB || "(none)"}
          </h3>
          <RequestResult loading={colB.loading} error={colB.error} result={colB.result} />
        </div>
      </div>
    </div>
  );
}
