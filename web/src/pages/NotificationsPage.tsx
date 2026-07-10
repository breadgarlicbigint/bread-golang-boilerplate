import { useEffect, useState } from "react";
import * as notifApi from "../api/notifications";
import { useApiAction } from "../hooks/useApiAction";
import { RequestResult } from "../components/RequestResult";
import { useToast } from "../context/ToastContext";

export function NotificationsPage() {
  const listAction = useApiAction(notifApi.listNotifications);
  const unreadAction = useApiAction(notifApi.unreadCount);
  const markReadAction = useApiAction(notifApi.markRead);
  const markAllAction = useApiAction(notifApi.markAllRead);
  const prefsAction = useApiAction(notifApi.getPreferences);
  const updatePrefsAction = useApiAction(notifApi.updatePreferences);
  const registerDeviceAction = useApiAction(notifApi.registerDevice);
  const removeDeviceAction = useApiAction(notifApi.removeDevice);
  const toast = useToast();

  const [unreadOnly, setUnreadOnly] = useState(false);
  const [prefsJson, setPrefsJson] = useState("");
  const [deviceToken, setDeviceToken] = useState("");
  const [devicePlatform, setDevicePlatform] = useState<"ios" | "android" | "web">("web");

  const refresh = () => {
    void listAction.run({ unreadOnly });
    void unreadAction.run();
  };

  useEffect(() => {
    refresh();
    void prefsAction.run().then((p) => setPrefsJson(JSON.stringify(p, null, 2)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    void listAction.run({ unreadOnly });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [unreadOnly]);

  const onSavePrefs = async () => {
    try {
      const parsed = JSON.parse(prefsJson) as notifApi.NotificationPreferences;
      await updatePrefsAction.run(parsed);
      toast.success("Preferences updated");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Invalid JSON");
    }
  };

  return (
    <div className="flex max-w-2xl flex-col gap-6">
      <div>
        <h2 className="text-lg font-semibold">Notifications</h2>
        <p className="text-sm text-slate-500">GET/PATCH /v1/me/notifications/*</p>
      </div>

      <div className="card flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold">
            My notifications {unreadAction.result !== null ? `(${unreadAction.result} unread)` : ""}
          </h3>
          <div className="flex gap-2">
            <label className="flex items-center gap-1 text-xs text-slate-600">
              <input type="checkbox" checked={unreadOnly} onChange={(e) => setUnreadOnly(e.target.checked)} />
              Unread only
            </label>
            <button className="btn-secondary !px-2 !py-1 text-xs" onClick={refresh}>
              Refresh
            </button>
            <button
              className="btn-secondary !px-2 !py-1 text-xs"
              onClick={() => markAllAction.run().then(refresh).catch(() => undefined)}
            >
              Mark all read
            </button>
          </div>
        </div>
        <RequestResult loading={listAction.loading} error={listAction.error} result={null} />
        {listAction.result && listAction.result.items.length > 0 ? (
          <table className="table-base">
            <thead>
              <tr>
                <th>Title</th>
                <th>Type</th>
                <th>Read</th>
                <th>Created</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {listAction.result.items.map((n) => (
                <tr key={n.id}>
                  <td>{n.title}</td>
                  <td>{n.type}</td>
                  <td>{n.isRead ? "yes" : "no"}</td>
                  <td>{n.createdAt}</td>
                  <td>
                    {!n.isRead && (
                      <button
                        className="btn-secondary !px-2 !py-1 text-xs"
                        onClick={() => markReadAction.run(n.id).then(refresh).catch(() => undefined)}
                      >
                        Mark read
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          !listAction.loading && <p className="text-sm text-slate-500">No notifications.</p>
        )}
      </div>

      <div className="card flex flex-col gap-3">
        <h3 className="text-sm font-semibold">Preferences — GET/PATCH .../preferences</h3>
        <textarea className="input h-40 font-mono text-xs" value={prefsJson} onChange={(e) => setPrefsJson(e.target.value)} />
        <button className="btn self-start" onClick={() => void onSavePrefs()} disabled={updatePrefsAction.loading}>
          Save preferences
        </button>
        <RequestResult loading={prefsAction.loading} error={prefsAction.error ?? updatePrefsAction.error} result={null} />
      </div>

      <div className="card flex flex-col gap-3">
        <h3 className="text-sm font-semibold">Device tokens (push) — POST/DELETE .../devices</h3>
        <div>
          <label className="label">Device token</label>
          <input className="input" value={deviceToken} onChange={(e) => setDeviceToken(e.target.value)} />
        </div>
        <div>
          <label className="label">Platform</label>
          <select className="input" value={devicePlatform} onChange={(e) => setDevicePlatform(e.target.value as typeof devicePlatform)}>
            <option value="web">web</option>
            <option value="ios">ios</option>
            <option value="android">android</option>
          </select>
        </div>
        <div className="flex gap-2">
          <button
            className="btn"
            onClick={() =>
              registerDeviceAction
                .run({ token: deviceToken, platform: devicePlatform })
                .then(() => toast.success("Device registered"))
                .catch(() => undefined)
            }
            disabled={registerDeviceAction.loading}
          >
            Register
          </button>
          <button
            className="btn-danger"
            onClick={() =>
              removeDeviceAction
                .run(deviceToken)
                .then(() => toast.success("Device removed"))
                .catch(() => undefined)
            }
            disabled={removeDeviceAction.loading}
          >
            Remove
          </button>
        </div>
        <RequestResult
          loading={registerDeviceAction.loading || removeDeviceAction.loading}
          error={registerDeviceAction.error ?? removeDeviceAction.error}
          result={null}
        />
      </div>
    </div>
  );
}
