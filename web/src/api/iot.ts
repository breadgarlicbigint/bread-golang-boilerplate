import { api, type PaginationMeta } from "../lib/apiClient";

export interface SimulateRequest {
  metric: string;
  value?: number;
  unit?: string;
}

export interface SimulateResult {
  published: boolean;
}

export async function simulateTelemetry(deviceId: string, payload: SimulateRequest): Promise<SimulateResult> {
  const res = await api.post<SimulateResult>(
    `/v1/admin/iot/devices/${encodeURIComponent(deviceId)}/simulate`,
    payload,
  );
  return res.data;
}

export interface CommandRequest {
  command: string;
  data?: Record<string, unknown>;
}

export interface CommandResult {
  published: boolean;
}

export async function sendCommand(deviceId: string, payload: CommandRequest): Promise<CommandResult> {
  const res = await api.post<CommandResult>(
    `/v1/admin/iot/devices/${encodeURIComponent(deviceId)}/command`,
    payload,
  );
  return res.data;
}

export interface TelemetryReading {
  id: string;
  deviceId: string;
  metric: string;
  value: number;
  unit?: string;
  recordedAt: string;
}

export interface ListTelemetryQuery {
  [key: string]: string | number | boolean | undefined;
  deviceId?: string;
  page?: number;
  perPage?: number;
}

export async function listTelemetry(
  query: ListTelemetryQuery = {},
): Promise<{ items: TelemetryReading[]; meta?: PaginationMeta }> {
  const res = await api.get<TelemetryReading[]>("/v1/admin/iot/telemetry", { query });
  return { items: res.data, meta: res.meta };
}
