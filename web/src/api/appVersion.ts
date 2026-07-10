import { api } from "../lib/apiClient";

export type Platform = "ios" | "android" | "web";
export type UpdateStatus = "required" | "available" | "up_to_date";

export interface AppVersion {
  id: string;
  platform: Platform;
  currentVersion: string;
  minVersion: string;
  forceUpdate: boolean;
  releaseNotes?: string;
  storeUrl?: string;
  createdAt: string;
  updatedAt: string;
}

export interface VersionCheckResponse {
  status: UpdateStatus;
  currentVersion: string;
  minVersion: string;
  clientVersion: string;
  releaseNotes?: string;
  storeUrl?: string;
  forceUpdate: boolean;
}

export async function checkVersion(platform: string, version: string): Promise<VersionCheckResponse> {
  const res = await api.get<VersionCheckResponse>("/v1/app-version/check", {
    auth: false,
    query: { platform, version },
  });
  return res.data;
}

export async function listAppVersions(): Promise<AppVersion[]> {
  const res = await api.get<AppVersion[]>("/v1/admin/app-versions");
  return res.data;
}

export interface UpsertAppVersionRequest {
  currentVersion: string;
  minVersion: string;
  forceUpdate: boolean;
  releaseNotes?: string;
  storeUrl?: string;
}

export async function upsertAppVersion(platform: Platform, payload: UpsertAppVersionRequest): Promise<AppVersion> {
  const res = await api.put<AppVersion>(`/v1/admin/app-versions/${platform}`, payload);
  return res.data;
}
