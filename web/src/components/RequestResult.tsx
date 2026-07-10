import type { ApiError } from "../lib/apiClient";
import { JsonViewer } from "./JsonViewer";

export function RequestResult({
  loading,
  error,
  result,
}: {
  loading: boolean;
  error: ApiError | null;
  result: unknown;
}) {
  if (loading) return <p className="text-sm text-slate-500">Loading…</p>;

  if (error) {
    return (
      <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">
        <p className="font-medium">
          {error.status || "Network error"} — {error.message}
        </p>
        {error.errors.length > 0 && (
          <ul className="mt-1 list-disc pl-5">
            {error.errors.map((e, i) => (
              <li key={i}>
                {e.field ? `${e.field}: ` : ""}
                {e.message}
              </li>
            ))}
          </ul>
        )}
      </div>
    );
  }

  if (result === null || result === undefined) return null;
  return <JsonViewer value={result} />;
}
