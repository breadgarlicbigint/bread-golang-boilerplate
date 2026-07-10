import { useCallback, useState } from "react";
import { ApiError } from "../lib/apiClient";

export interface ActionState<T> {
  loading: boolean;
  error: ApiError | null;
  result: T | null;
}

/** Standardizes the loading/error/result lifecycle for a single API call, used by every page. */
export function useApiAction<Args extends unknown[], T>(fn: (...args: Args) => Promise<T>) {
  const [state, setState] = useState<ActionState<T>>({ loading: false, error: null, result: null });

  const run = useCallback(
    async (...args: Args) => {
      setState({ loading: true, error: null, result: null });
      try {
        const result = await fn(...args);
        setState({ loading: false, error: null, result });
        return result;
      } catch (err) {
        const apiErr = err instanceof ApiError ? err : new ApiError(0, String(err));
        setState({ loading: false, error: apiErr, result: null });
        throw err;
      }
    },
    [fn],
  );

  const reset = useCallback(() => setState({ loading: false, error: null, result: null }), []);

  return { ...state, run, reset };
}
