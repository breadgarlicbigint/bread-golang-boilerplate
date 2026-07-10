import { useState, type FormEvent } from "react";
import * as authApi from "../api/auth";
import { useApiAction } from "../hooks/useApiAction";
import { RequestResult } from "../components/RequestResult";
import { useSettings } from "../context/SettingsContext";
import { useAuth } from "../context/AuthContext";
import { useToast } from "../context/ToastContext";

export function OAuthPage() {
  const { settings } = useSettings();
  const { importSession } = useAuth();
  const toast = useToast();

  const [appleForm, setAppleForm] = useState<authApi.AppleCallbackRequest>({});
  const appleAction = useApiAction(authApi.appleCallback);

  const [pasted, setPasted] = useState("");
  const [pasteError, setPasteError] = useState<string | null>(null);

  const onAppleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    try {
      await appleAction.run(appleForm);
    } catch {
      /* surfaced below */
    }
  };

  const onImportPasted = () => {
    setPasteError(null);
    try {
      const parsed = JSON.parse(pasted);
      const data = parsed.data ?? parsed;
      if (!data.accessToken || !data.refreshToken || !data.user) {
        throw new Error("Expected fields: accessToken, refreshToken, user");
      }
      importSession(data.accessToken, data.refreshToken, data.user);
      toast.success("Session imported");
      setPasted("");
    } catch (err) {
      setPasteError(err instanceof Error ? err.message : String(err));
    }
  };

  return (
    <div className="flex max-w-2xl flex-col gap-6">
      <div>
        <h2 className="text-lg font-semibold">OAuth / Social Login</h2>
        <p className="text-sm text-slate-500">
          GET /v1/auth/github · GET /v1/auth/github/callback · POST /v1/auth/apple/callback
        </p>
      </div>

      <div className="card flex flex-col gap-3">
        <h3 className="text-sm font-semibold">GitHub</h3>
        <p className="text-xs text-slate-500">
          The GitHub callback returns the login JSON directly instead of redirecting back to this app, so it opens in
          a new tab. Requires <code>GITHUB_CLIENT_ID</code>/<code>GITHUB_CLIENT_SECRET</code> configured on the API.
          Copy the JSON body from that tab and paste it below to import the session here.
        </p>
        <a
          className="btn self-start"
          href={authApi.githubRedirectUrl(settings.apiBaseUrl)}
          target="_blank"
          rel="noreferrer"
        >
          Login with GitHub ↗
        </a>
      </div>

      <div className="card flex flex-col gap-3">
        <h3 className="text-sm font-semibold">Import session from pasted JSON</h3>
        <textarea
          className="input h-28 font-mono text-xs"
          placeholder='{"data": {"accessToken": "...", "refreshToken": "...", "user": {...}}}'
          value={pasted}
          onChange={(e) => setPasted(e.target.value)}
        />
        <button className="btn self-start" type="button" onClick={onImportPasted}>
          Import
        </button>
        {pasteError && <p className="field-error">{pasteError}</p>}
      </div>

      <form onSubmit={onAppleSubmit} className="card flex flex-col gap-3">
        <h3 className="text-sm font-semibold">Apple Sign In callback</h3>
        <p className="text-xs text-slate-500">
          Sign in with Apple JS runs on a registered Apple Service ID and POSTs here — it can't be triggered from an
          arbitrary dev origin. If you have a real <code>code</code>/<code>id_token</code> pair, submit it directly
          against the raw endpoint.
        </p>
        <div>
          <label className="label">code</label>
          <input
            className="input"
            value={appleForm.code ?? ""}
            onChange={(e) => setAppleForm((f) => ({ ...f, code: e.target.value }))}
          />
        </div>
        <div>
          <label className="label">id_token</label>
          <input
            className="input"
            value={appleForm.id_token ?? ""}
            onChange={(e) => setAppleForm((f) => ({ ...f, id_token: e.target.value }))}
          />
        </div>
        <div>
          <label className="label">firstName (first sign-in only)</label>
          <input
            className="input"
            value={appleForm.firstName ?? ""}
            onChange={(e) => setAppleForm((f) => ({ ...f, firstName: e.target.value }))}
          />
        </div>
        <div>
          <label className="label">lastName (first sign-in only)</label>
          <input
            className="input"
            value={appleForm.lastName ?? ""}
            onChange={(e) => setAppleForm((f) => ({ ...f, lastName: e.target.value }))}
          />
        </div>
        <button className="btn self-start" type="submit" disabled={appleAction.loading}>
          Submit
        </button>
        <RequestResult loading={appleAction.loading} error={appleAction.error} result={appleAction.result} />
      </form>
    </div>
  );
}
