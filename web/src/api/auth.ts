import { api } from "../lib/apiClient";
import type { StoredUser } from "../lib/storage";

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  accessToken: string;
  refreshToken: string;
  tokenType: string;
  expiresIn: number;
  user: StoredUser;
}

export interface RegisterRequest {
  email: string;
  username: string;
  password: string;
  firstName: string;
  lastName: string;
}

export interface RefreshResponse {
  accessToken: string;
  refreshToken: string;
  tokenType: string;
  expiresIn: number;
}

export interface Enable2FAResponse {
  secret: string;
  qrCodeUrl: string;
  backupCodes: string[];
}

export async function login(payload: LoginRequest): Promise<LoginResponse> {
  const res = await api.post<LoginResponse>("/v1/auth/login", payload, { auth: false });
  return res.data;
}

export async function register(payload: RegisterRequest): Promise<void> {
  await api.post("/v1/auth/register", payload, { auth: false });
}

export async function refresh(refreshToken: string): Promise<RefreshResponse> {
  const res = await api.post<RefreshResponse>("/v1/auth/refresh", { refreshToken }, { auth: false });
  return res.data;
}

export async function logout(): Promise<void> {
  await api.delete("/v1/auth/logout");
}

export async function logoutAll(): Promise<void> {
  await api.delete("/v1/auth/logout-all");
}

export async function enable2FA(): Promise<Enable2FAResponse> {
  const res = await api.post<Enable2FAResponse>("/v1/auth/2fa/enable");
  return res.data;
}

export async function verify2FA(code: string): Promise<void> {
  await api.post("/v1/auth/2fa/verify", { code });
}

export function githubRedirectUrl(apiBaseUrl: string): string {
  return `${apiBaseUrl.replace(/\/$/, "")}/v1/auth/github`;
}

export interface AppleCallbackRequest {
  code?: string;
  id_token?: string;
  firstName?: string;
  lastName?: string;
}

export async function appleCallback(payload: AppleCallbackRequest): Promise<LoginResponse> {
  const form = new URLSearchParams();
  Object.entries(payload).forEach(([k, v]) => {
    if (v) form.set(k, v);
  });
  const res = await api.post<LoginResponse>("/v1/auth/apple/callback", form, { auth: false, rawBody: true });
  return res.data;
}
