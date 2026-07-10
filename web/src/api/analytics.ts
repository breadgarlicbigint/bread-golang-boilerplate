import { api } from "../lib/apiClient";

export type DateRangeMode = "none" | "range" | "rangeGranularity";

export interface AnalyticsEndpoint {
  key: string;
  label: string;
  path: string;
  dateRange: DateRangeMode;
}

/** Mirrors modules/analytics/handler/analytics.handler.go RegisterRoutes exactly. */
export const ANALYTICS_ENDPOINTS: AnalyticsEndpoint[] = [
  { key: "registrations", label: "User registrations", path: "/v1/admin/analytics/users/registrations", dateRange: "rangeGranularity" },
  { key: "churn", label: "User churn", path: "/v1/admin/analytics/users/churn", dateRange: "range" },
  { key: "signup-methods", label: "Signup methods", path: "/v1/admin/analytics/users/signup-methods", dateRange: "none" },
  { key: "blocked-trend", label: "Blocked user trend", path: "/v1/admin/analytics/users/blocked-trend", dateRange: "range" },
  { key: "login-frequency", label: "Login frequency", path: "/v1/admin/analytics/auth/login-frequency", dateRange: "rangeGranularity" },
  { key: "login-methods", label: "Login methods", path: "/v1/admin/analytics/auth/login-methods", dateRange: "range" },
  { key: "lockout", label: "Lockout stats", path: "/v1/admin/analytics/auth/lockout", dateRange: "none" },
  { key: "passkey-adoption", label: "Passkey adoption", path: "/v1/admin/analytics/passkeys/adoption", dateRange: "none" },
  { key: "mobile-verification", label: "Mobile verification", path: "/v1/admin/analytics/mobile/verification", dateRange: "none" },
  { key: "credential-stuffing", label: "Credential stuffing anomalies", path: "/v1/admin/analytics/anomalies/credential-stuffing", dateRange: "none" },
  { key: "device-proliferation", label: "Device proliferation anomalies", path: "/v1/admin/analytics/anomalies/device-proliferation", dateRange: "none" },
  { key: "fraud-signals", label: "Fraud signals", path: "/v1/admin/analytics/fraud/signals", dateRange: "none" },
];

export async function fetchAnalytics(
  path: string,
  params: { startDate?: string; endDate?: string; granularity?: string } = {},
): Promise<unknown> {
  const res = await api.get<unknown>(path, {
    query: { startDate: params.startDate, endDate: params.endDate, granularity: params.granularity },
  });
  return res.data;
}
