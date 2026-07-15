import { useEffect, useState, type FormEvent } from "react";
import * as iotApi from "../api/iot";
import { useApiAction } from "../hooks/useApiAction";
import { RequestResult } from "../components/RequestResult";
import { useToast } from "../context/ToastContext";

export function IotPage() {
  const toast = useToast();

  // ── Simulate telemetry (publishes over MQTT) ─────────────────────────────────
  const simulateAction = useApiAction(iotApi.simulateTelemetry);
  const [simForm, setSimForm] = useState({ deviceId: "sensor-01", metric: "temperature", value: "", unit: "C" });
  const onSimulate = async (e: FormEvent) => {
    e.preventDefault();
    try {
      await simulateAction.run(simForm.deviceId, {
        metric: simForm.metric,
        value: simForm.value === "" ? undefined : Number(simForm.value),
        unit: simForm.unit || undefined,
      });
      toast.success("Telemetry published — watch it arrive on the Realtime page (topic \"iot:telemetry\")");
      setTimeout(() => void refreshTelemetry(), 500); // MQTT round trip is async
    } catch {
      /* surfaced below */
    }
  };

  // ── Send command ─────────────────────────────────────────────────────────────
  const commandAction = useApiAction(iotApi.sendCommand);
  const [cmdForm, setCmdForm] = useState({ deviceId: "sensor-01", command: "reboot" });
  const onCommand = async (e: FormEvent) => {
    e.preventDefault();
    try {
      await commandAction.run(cmdForm.deviceId, { command: cmdForm.command });
      toast.success(`Command "${cmdForm.command}" published to devices/${cmdForm.deviceId}/commands`);
    } catch {
      /* surfaced below */
    }
  };

  // ── Telemetry list ────────────────────────────────────────────────────────────
  const listAction = useApiAction(iotApi.listTelemetry);
  const [filterDeviceId, setFilterDeviceId] = useState("");
  const [page, setPage] = useState(1);

  const refreshTelemetry = () => listAction.run({ deviceId: filterDeviceId || undefined, page, perPage: 10 });

  useEffect(() => {
    void refreshTelemetry();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page]);

  return (
    <div className="flex max-w-3xl flex-col gap-6">
      <div>
        <h2 className="text-lg font-semibold">IoT — MQTT device telemetry demo</h2>
        <p className="text-sm text-slate-500">
          POST /v1/admin/iot/devices/:deviceId/simulate · /command · GET /v1/admin/iot/telemetry
        </p>
        <p className="mt-1 text-xs text-slate-500">
          "Simulate" publishes a reading to MQTT topic <code>devices/:deviceId/telemetry</code> exactly as a
          real device would. A subscriber running in the API process picks it up, persists it here, and
          forwards it to the realtime <code>iot:telemetry</code> topic — open the{" "}
          <a className="underline" href="/realtime">
            Realtime page
          </a>{" "}
          and connect WS/SSE to <code>iot:telemetry</code> to watch it arrive live. Requires{" "}
          <code>MQTT_BROKER_URL</code> to be configured (<code>make docker-mqtt</code>) — otherwise both
          forms below return a 503.
        </p>
      </div>

      <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
        <div className="card flex flex-col gap-3">
          <h3 className="text-sm font-semibold">Simulate telemetry</h3>
          <form onSubmit={onSimulate} className="flex flex-col gap-3">
            <div>
              <label className="label">Device ID</label>
              <input
                className="input"
                required
                value={simForm.deviceId}
                onChange={(e) => setSimForm((f) => ({ ...f, deviceId: e.target.value }))}
              />
            </div>
            <div className="flex gap-3">
              <div className="flex-1">
                <label className="label">Metric</label>
                <input
                  className="input"
                  required
                  value={simForm.metric}
                  onChange={(e) => setSimForm((f) => ({ ...f, metric: e.target.value }))}
                />
              </div>
              <div className="w-24">
                <label className="label">Unit</label>
                <input
                  className="input"
                  value={simForm.unit}
                  onChange={(e) => setSimForm((f) => ({ ...f, unit: e.target.value }))}
                />
              </div>
            </div>
            <div>
              <label className="label">Value (blank = random)</label>
              <input
                className="input"
                type="number"
                step="any"
                placeholder="leave blank for a plausible random value"
                value={simForm.value}
                onChange={(e) => setSimForm((f) => ({ ...f, value: e.target.value }))}
              />
            </div>
            <button className="btn self-start" type="submit" disabled={simulateAction.loading}>
              Simulate
            </button>
            <RequestResult loading={false} error={simulateAction.error} result={simulateAction.result} />
          </form>
        </div>

        <div className="card flex flex-col gap-3">
          <h3 className="text-sm font-semibold">Send command</h3>
          <p className="text-xs text-slate-500">
            Fire-and-forget — nothing in this boilerplate subscribes to <code>devices/:deviceId/commands</code>{" "}
            (a real device would). Demonstrates publishing in the other direction.
          </p>
          <form onSubmit={onCommand} className="flex flex-col gap-3">
            <div>
              <label className="label">Device ID</label>
              <input
                className="input"
                required
                value={cmdForm.deviceId}
                onChange={(e) => setCmdForm((f) => ({ ...f, deviceId: e.target.value }))}
              />
            </div>
            <div>
              <label className="label">Command</label>
              <input
                className="input"
                required
                value={cmdForm.command}
                onChange={(e) => setCmdForm((f) => ({ ...f, command: e.target.value }))}
              />
            </div>
            <button className="btn self-start" type="submit" disabled={commandAction.loading}>
              Send command
            </button>
            <RequestResult loading={false} error={commandAction.error} result={commandAction.result} />
          </form>
        </div>
      </div>

      <div className="card flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold">Persisted telemetry</h3>
          <div className="flex gap-2">
            <input
              className="input !w-40 !py-1 text-xs"
              placeholder="filter by device ID"
              value={filterDeviceId}
              onChange={(e) => setFilterDeviceId(e.target.value)}
            />
            <button
              className="btn-secondary !px-2 !py-1 text-xs"
              onClick={() => {
                setPage(1);
                void refreshTelemetry();
              }}
            >
              Refresh
            </button>
          </div>
        </div>

        {listAction.loading && <p className="text-sm text-slate-500">Loading…</p>}
        {listAction.error && <RequestResult loading={false} error={listAction.error} result={null} />}
        {listAction.result && listAction.result.items.length > 0 ? (
          <>
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-slate-200 text-xs uppercase text-slate-400">
                  <th className="py-1 pr-2">Device</th>
                  <th className="py-1 pr-2">Metric</th>
                  <th className="py-1 pr-2">Value</th>
                  <th className="py-1 pr-2">Recorded at</th>
                </tr>
              </thead>
              <tbody>
                {listAction.result.items.map((r) => (
                  <tr key={r.id} className="border-b border-slate-100">
                    <td className="py-1 pr-2 font-mono text-xs">{r.deviceId}</td>
                    <td className="py-1 pr-2">{r.metric}</td>
                    <td className="py-1 pr-2">
                      {r.value} {r.unit}
                    </td>
                    <td className="py-1 pr-2 text-xs text-slate-500">{r.recordedAt}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            <div className="flex items-center gap-3 text-xs text-slate-500">
              <button className="btn-secondary !px-2 !py-1" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
                Prev
              </button>
              <span>
                Page {listAction.result.meta?.page} / {listAction.result.meta?.totalPage} (total{" "}
                {listAction.result.meta?.total})
              </span>
              <button
                className="btn-secondary !px-2 !py-1"
                disabled={!listAction.result.meta?.hasNext}
                onClick={() => setPage((p) => p + 1)}
              >
                Next
              </button>
            </div>
          </>
        ) : (
          !listAction.loading && <p className="text-sm text-slate-500">No telemetry recorded yet.</p>
        )}
      </div>
    </div>
  );
}
