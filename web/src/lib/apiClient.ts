import { getSettings, getAccessToken, getRefreshToken, setTokens, clearTokens } from "./storage";
import { addLogEntry } from "./requestLog";

export interface ApiErrorDetail {
  field?: string;
  message: string;
}

export class ApiError extends Error {
  status: number;
  errors: ApiErrorDetail[];
  requestId?: string;

  constructor(status: number, message: string, errors: ApiErrorDetail[] = [], requestId?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.errors = errors;
    this.requestId = requestId;
  }
}

export interface PaginationMeta {
  total: number;
  page: number;
  perPage: number;
  totalPage: number;
  hasNext: boolean;
  hasPrev: boolean;
  cursor?: string;
}

interface Envelope<T> {
  statusCode: number;
  message: string;
  data?: T;
  _metadata?: PaginationMeta;
  timestamp: string;
  path: string;
  requestId: string;
  errors?: ApiErrorDetail[];
}

export interface RequestOptions {
  method?: string;
  body?: unknown;
  query?: Record<string, string | number | boolean | undefined | null>;
  headers?: Record<string, string>;
  /** Attach the Authorization: Bearer header. Default true. */
  auth?: boolean;
  /** Internal — prevents infinite refresh loops. */
  skipRefresh?: boolean;
  /** Send `body` as-is (e.g. URLSearchParams) instead of JSON-encoding it. */
  rawBody?: boolean;
  /** The endpoint doesn't use the standard Envelope wrapper (e.g. /health*) — return the raw parsed JSON as `data`. */
  rawResponse?: boolean;
}

export interface ApiResult<T> {
  data: T;
  meta?: PaginationMeta;
  message: string;
  requestId: string;
  status: number;
}

function buildUrl(path: string, query?: RequestOptions["query"]): string {
  const { apiBaseUrl } = getSettings();
  const base = apiBaseUrl.endsWith("/") ? apiBaseUrl : `${apiBaseUrl}/`;
  const url = new URL(path.replace(/^\//, ""), base);
  if (query) {
    for (const [k, v] of Object.entries(query)) {
      if (v === undefined || v === null || v === "") continue;
      url.searchParams.set(k, String(v));
    }
  }
  return url.toString();
}

let refreshPromise: Promise<boolean> | null = null;

async function tryRefresh(): Promise<boolean> {
  const rt = getRefreshToken();
  if (!rt) return false;
  if (!refreshPromise) {
    refreshPromise = (async () => {
      try {
        const result = await request<{ accessToken: string; refreshToken: string }>("/v1/auth/refresh", {
          method: "POST",
          body: { refreshToken: rt },
          auth: false,
          skipRefresh: true,
        });
        setTokens(result.data.accessToken, result.data.refreshToken);
        return true;
      } catch {
        return false;
      } finally {
        refreshPromise = null;
      }
    })();
  }
  return refreshPromise;
}

export async function request<T = unknown>(path: string, opts: RequestOptions = {}): Promise<ApiResult<T>> {
  const {
    method = "GET",
    body,
    query,
    headers = {},
    auth = true,
    skipRefresh = false,
    rawBody = false,
    rawResponse = false,
  } = opts;
  const settings = getSettings();
  const finalHeaders: Record<string, string> = { ...headers };
  let payload: BodyInit | undefined;

  if (body !== undefined) {
    if (rawBody) {
      payload = body as BodyInit;
    } else {
      finalHeaders["Content-Type"] = "application/json";
      payload = JSON.stringify(body);
    }
  }
  if (auth) {
    const token = getAccessToken();
    if (token) finalHeaders["Authorization"] = `Bearer ${token}`;
  }
  if (settings.lang) finalHeaders["x-custom-lang"] = settings.lang;
  if (settings.tenantId) finalHeaders["X-Tenant-ID"] = settings.tenantId;
  if (settings.appVersion) finalHeaders["X-App-Version"] = settings.appVersion;
  if (settings.appPlatform) finalHeaders["X-App-Platform"] = settings.appPlatform;

  const url = buildUrl(path, query);
  const startedAt = performance.now();
  let res: Response;
  try {
    res = await fetch(url, { method, headers: finalHeaders, body: payload });
  } catch (err) {
    addLogEntry({
      method,
      path,
      status: 0,
      ok: false,
      durationMs: performance.now() - startedAt,
      requestBody: body,
      responseBody: { error: String(err) },
      timestamp: Date.now(),
    });
    throw new ApiError(0, `Network error — is the API reachable at ${settings.apiBaseUrl}?`);
  }

  const durationMs = performance.now() - startedAt;
  const text = await res.text();
  let json: Envelope<T> | null = null;
  if (text) {
    try {
      json = JSON.parse(text) as Envelope<T>;
    } catch {
      json = { statusCode: res.status, message: text, timestamp: "", path, requestId: "" } as Envelope<T>;
    }
  }

  addLogEntry({
    method,
    path,
    status: res.status,
    ok: res.ok,
    durationMs,
    requestBody: body,
    responseBody: json,
    timestamp: Date.now(),
  });

  if (res.status === 401 && auth && !skipRefresh) {
    const refreshed = await tryRefresh();
    if (refreshed) {
      return request<T>(path, { ...opts, skipRefresh: true });
    }
    // Refresh failed too — the session is genuinely invalid (expired/revoked/missing
    // token), not just a one-off 401. Clear it so the app logs the user out.
    clearTokens("expired");
  }

  if (!res.ok) {
    const message = (!rawResponse && json?.message) || res.statusText || "Request failed";
    throw new ApiError(res.status, message, (!rawResponse && json?.errors) || [], json?.requestId);
  }

  if (rawResponse) {
    return { data: json as unknown as T, message: "", requestId: "", status: res.status };
  }

  return {
    data: json?.data as T,
    meta: json?._metadata,
    message: json?.message ?? "",
    requestId: json?.requestId ?? "",
    status: res.status,
  };
}

export const api = {
  get: <T,>(path: string, opts?: Omit<RequestOptions, "method">) => request<T>(path, { ...opts, method: "GET" }),
  post: <T,>(path: string, body?: unknown, opts?: Omit<RequestOptions, "method" | "body">) =>
    request<T>(path, { ...opts, method: "POST", body }),
  patch: <T,>(path: string, body?: unknown, opts?: Omit<RequestOptions, "method" | "body">) =>
    request<T>(path, { ...opts, method: "PATCH", body }),
  put: <T,>(path: string, body?: unknown, opts?: Omit<RequestOptions, "method" | "body">) =>
    request<T>(path, { ...opts, method: "PUT", body }),
  delete: <T,>(path: string, opts?: Omit<RequestOptions, "method">) => request<T>(path, { ...opts, method: "DELETE" }),
};
