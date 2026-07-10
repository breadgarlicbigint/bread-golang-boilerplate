import { useEffect, useState, type FormEvent } from "react";
import * as appVersionApi from "../api/appVersion";
import { useApiAction } from "../hooks/useApiAction";
import { RequestResult } from "../components/RequestResult";
import { useToast } from "../context/ToastContext";

const PLATFORMS: appVersionApi.Platform[] = ["ios", "android", "web"];

export function AdminAppVersionsPage() {
  const listAction = useApiAction(appVersionApi.listAppVersions);
  const upsertAction = useApiAction(appVersionApi.upsertAppVersion);
  const toast = useToast();

  const [platform, setPlatform] = useState<appVersionApi.Platform>("ios");
  const [form, setForm] = useState<appVersionApi.UpsertAppVersionRequest>({
    currentVersion: "",
    minVersion: "",
    forceUpdate: false,
    releaseNotes: "",
    storeUrl: "",
  });

  const refresh = () => void listAction.run();

  useEffect(() => {
    refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    try {
      await upsertAction.run(platform, form);
      toast.success(`${platform} version policy saved`);
      refresh();
    } catch {
      /* surfaced below */
    }
  };

  return (
    <div className="flex max-w-2xl flex-col gap-6">
      <div>
        <h2 className="text-lg font-semibold">Admin — App Versions</h2>
        <p className="text-sm text-slate-500">GET /v1/admin/app-versions · PUT /v1/admin/app-versions/:platform</p>
      </div>

      <div className="card">
        <h3 className="mb-2 text-sm font-semibold">Current policies</h3>
        <RequestResult loading={listAction.loading} error={listAction.error} result={listAction.result} />
      </div>

      <form onSubmit={onSubmit} className="card flex flex-col gap-3">
        <h3 className="text-sm font-semibold">Create / update policy</h3>
        <div>
          <label className="label">Platform</label>
          <select className="input" value={platform} onChange={(e) => setPlatform(e.target.value as appVersionApi.Platform)}>
            {PLATFORMS.map((p) => (
              <option key={p} value={p}>
                {p}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="label">Current version</label>
          <input
            className="input"
            required
            placeholder="2.5.0"
            value={form.currentVersion}
            onChange={(e) => setForm((f) => ({ ...f, currentVersion: e.target.value }))}
          />
        </div>
        <div>
          <label className="label">Min version</label>
          <input
            className="input"
            required
            placeholder="2.0.0"
            value={form.minVersion}
            onChange={(e) => setForm((f) => ({ ...f, minVersion: e.target.value }))}
          />
        </div>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={form.forceUpdate}
            onChange={(e) => setForm((f) => ({ ...f, forceUpdate: e.target.checked }))}
          />
          Force update below min version
        </label>
        <div>
          <label className="label">Release notes</label>
          <input
            className="input"
            value={form.releaseNotes}
            onChange={(e) => setForm((f) => ({ ...f, releaseNotes: e.target.value }))}
          />
        </div>
        <div>
          <label className="label">Store URL</label>
          <input className="input" value={form.storeUrl} onChange={(e) => setForm((f) => ({ ...f, storeUrl: e.target.value }))} />
        </div>
        <button className="btn self-start" type="submit" disabled={upsertAction.loading}>
          Save
        </button>
        <RequestResult loading={false} error={upsertAction.error} result={upsertAction.result} />
      </form>
    </div>
  );
}
