import { useSettings } from "../context/SettingsContext";
import { useAuth } from "../context/AuthContext";
import { getAccessToken, getRefreshToken } from "../lib/storage";
import { JsonViewer } from "../components/JsonViewer";

export function SettingsPage() {
  const { settings, updateSettings } = useSettings();
  const { user } = useAuth();

  return (
    <div className="flex max-w-2xl flex-col gap-6">
      <div>
        <h2 className="text-lg font-semibold">Settings</h2>
        <p className="text-sm text-slate-500">Headers applied to every request made by this client.</p>
      </div>

      <div className="card flex flex-col gap-3">
        <div>
          <label className="label">API base URL</label>
          <input
            className="input"
            value={settings.apiBaseUrl}
            onChange={(e) => updateSettings({ apiBaseUrl: e.target.value })}
          />
        </div>
        <div>
          <label className="label">x-custom-lang</label>
          <input className="input" value={settings.lang} onChange={(e) => updateSettings({ lang: e.target.value })} />
        </div>
        <div>
          <label className="label">X-Tenant-ID (only used when MULTI_TENANT_ENABLED=true)</label>
          <input
            className="input"
            value={settings.tenantId}
            onChange={(e) => updateSettings({ tenantId: e.target.value })}
          />
        </div>
        <div>
          <label className="label">X-App-Version</label>
          <input
            className="input"
            placeholder="1.0.0"
            value={settings.appVersion}
            onChange={(e) => updateSettings({ appVersion: e.target.value })}
          />
        </div>
        <div>
          <label className="label">X-App-Platform</label>
          <select
            className="input"
            value={settings.appPlatform}
            onChange={(e) => updateSettings({ appPlatform: e.target.value })}
          >
            <option value="">—</option>
            <option value="ios">ios</option>
            <option value="android">android</option>
            <option value="web">web</option>
          </select>
        </div>
      </div>

      <div className="card">
        <h3 className="mb-2 text-sm font-semibold">Current session (stored in localStorage)</h3>
        <JsonViewer
          value={{
            user,
            accessToken: getAccessToken(),
            refreshToken: getRefreshToken(),
          }}
        />
      </div>
    </div>
  );
}
