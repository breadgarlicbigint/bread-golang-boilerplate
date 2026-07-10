import { useState } from "react";
import { ANALYTICS_ENDPOINTS, fetchAnalytics } from "../api/analytics";
import { useApiAction } from "../hooks/useApiAction";
import { RequestResult } from "../components/RequestResult";

function todayMinus(days: number): string {
  const d = new Date();
  d.setDate(d.getDate() - days);
  return d.toISOString().slice(0, 10);
}

export function AdminAnalyticsPage() {
  const [startDate, setStartDate] = useState(todayMinus(30));
  const [endDate, setEndDate] = useState(todayMinus(0));
  const [granularity, setGranularity] = useState("day");
  const action = useApiAction(fetchAnalytics);
  const [activeLabel, setActiveLabel] = useState<string | null>(null);

  const run = (path: string, label: string, needsRange: boolean) => {
    setActiveLabel(label);
    void action.run(path, needsRange ? { startDate, endDate, granularity } : {});
  };

  return (
    <div className="flex max-w-3xl flex-col gap-6">
      <div>
        <h2 className="text-lg font-semibold">Admin — Analytics</h2>
        <p className="text-sm text-slate-500">/v1/admin/analytics/* — cached server-side, X-Cache header shows HIT/MISS</p>
      </div>

      <div className="card flex flex-wrap items-end gap-3">
        <div>
          <label className="label">Start date</label>
          <input className="input" type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} />
        </div>
        <div>
          <label className="label">End date</label>
          <input className="input" type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} />
        </div>
        <div>
          <label className="label">Granularity</label>
          <select className="input" value={granularity} onChange={(e) => setGranularity(e.target.value)}>
            <option value="day">day</option>
            <option value="week">week</option>
            <option value="month">month</option>
          </select>
        </div>
      </div>

      <div className="card">
        <h3 className="mb-2 text-sm font-semibold">Endpoints</h3>
        <div className="flex flex-wrap gap-2">
          {ANALYTICS_ENDPOINTS.map((ep) => (
            <button
              key={ep.key}
              className="btn-secondary text-xs"
              onClick={() => run(ep.path, ep.label, ep.dateRange !== "none")}
            >
              {ep.label}
            </button>
          ))}
        </div>
      </div>

      <div className="card">
        <h3 className="mb-2 text-sm font-semibold">{activeLabel ?? "Result"}</h3>
        <RequestResult loading={action.loading} error={action.error} result={action.result} />
      </div>
    </div>
  );
}
