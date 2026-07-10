import { useState, type FormEvent } from "react";
import * as appVersionApi from "../api/appVersion";
import { useApiAction } from "../hooks/useApiAction";
import { RequestResult } from "../components/RequestResult";

export function AppVersionCheckPage() {
  const [platform, setPlatform] = useState<appVersionApi.Platform>("web");
  const [version, setVersion] = useState("1.0.0");
  const action = useApiAction(appVersionApi.checkVersion);

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    void action.run(platform, version);
  };

  return (
    <div className="max-w-lg">
      <h2 className="text-lg font-semibold">App Version Check</h2>
      <p className="mb-4 text-sm text-slate-500">
        GET /v1/app-version/check — public endpoint clients call on launch. Also mirrors the{" "}
        <code>X-App-Version</code>/<code>X-App-Platform</code> headers the API middleware reads on every request
        (set them in Settings to see <code>X-Version-Status</code> on all responses).
      </p>
      <form onSubmit={onSubmit} className="card flex flex-col gap-3">
        <div>
          <label className="label">Platform</label>
          <select className="input" value={platform} onChange={(e) => setPlatform(e.target.value as appVersionApi.Platform)}>
            <option value="ios">ios</option>
            <option value="android">android</option>
            <option value="web">web</option>
          </select>
        </div>
        <div>
          <label className="label">Client version</label>
          <input className="input" value={version} onChange={(e) => setVersion(e.target.value)} />
        </div>
        <button className="btn self-start" type="submit" disabled={action.loading}>
          Check
        </button>
      </form>
      <div className="mt-3">
        <RequestResult loading={action.loading} error={action.error} result={action.result} />
      </div>
    </div>
  );
}
