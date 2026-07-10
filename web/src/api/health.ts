import { api } from "../lib/apiClient";

export async function health(): Promise<Record<string, unknown>> {
  const res = await api.get<Record<string, unknown>>("/health", { auth: false, rawResponse: true });
  return res.data;
}

export async function live(): Promise<Record<string, unknown>> {
  const res = await api.get<Record<string, unknown>>("/health/live", { auth: false, rawResponse: true });
  return res.data;
}

export async function ready(): Promise<Record<string, unknown>> {
  const res = await api.get<Record<string, unknown>>("/health/ready", { auth: false, rawResponse: true });
  return res.data;
}
