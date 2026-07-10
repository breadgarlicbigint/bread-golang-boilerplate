import { useEffect } from "react";
import * as healthApi from "../api/health";
import { useApiAction } from "../hooks/useApiAction";
import { RequestResult } from "../components/RequestResult";

export function HealthPage() {
  const healthAction = useApiAction(healthApi.health);
  const liveAction = useApiAction(healthApi.live);
  const readyAction = useApiAction(healthApi.ready);

  useEffect(() => {
    void healthAction.run();
    void liveAction.run();
    void readyAction.run();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="flex max-w-2xl flex-col gap-6">
      <div>
        <h2 className="text-lg font-semibold">Health</h2>
        <p className="text-sm text-slate-500">GET /health · /health/live · /health/ready</p>
      </div>

      <div className="card">
        <div className="mb-2 flex items-center justify-between">
          <h3 className="text-sm font-semibold">GET /health (Mongo + Redis + memory)</h3>
          <button className="btn-secondary !px-2 !py-1 text-xs" onClick={() => void healthAction.run()}>
            Refresh
          </button>
        </div>
        <RequestResult loading={healthAction.loading} error={healthAction.error} result={healthAction.result} />
      </div>

      <div className="card">
        <div className="mb-2 flex items-center justify-between">
          <h3 className="text-sm font-semibold">GET /health/live</h3>
          <button className="btn-secondary !px-2 !py-1 text-xs" onClick={() => void liveAction.run()}>
            Refresh
          </button>
        </div>
        <RequestResult loading={liveAction.loading} error={liveAction.error} result={liveAction.result} />
      </div>

      <div className="card">
        <div className="mb-2 flex items-center justify-between">
          <h3 className="text-sm font-semibold">GET /health/ready</h3>
          <button className="btn-secondary !px-2 !py-1 text-xs" onClick={() => void readyAction.run()}>
            Refresh
          </button>
        </div>
        <RequestResult loading={readyAction.loading} error={readyAction.error} result={readyAction.result} />
      </div>
    </div>
  );
}
