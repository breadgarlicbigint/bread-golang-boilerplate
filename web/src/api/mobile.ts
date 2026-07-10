import { api } from "../lib/apiClient";

export interface UserMobile {
  id: string;
  userId: string;
  tenantId?: string;
  countryCode: string;
  number: string;
  e164: string;
  isVerified: boolean;
  isPrimary: boolean;
  verifiedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export async function listMobiles(): Promise<UserMobile[]> {
  const res = await api.get<UserMobile[]>("/v1/me/mobiles");
  return res.data;
}

export async function sendOTP(e164: string, channel: "sms" | "whatsapp" = "sms"): Promise<{ channel: string; e164: string }> {
  const res = await api.post<{ channel: string; e164: string }>("/v1/me/mobiles/send-otp", { e164, channel });
  return res.data;
}

export async function verifyOTP(e164: string, code: string): Promise<void> {
  await api.post("/v1/me/mobiles/verify", { e164, code });
}

export async function setPrimary(e164: string): Promise<void> {
  await api.patch(`/v1/me/mobiles/${encodeURIComponent(e164)}/primary`);
}

export async function deleteMobile(e164: string): Promise<void> {
  await api.delete(`/v1/me/mobiles/${encodeURIComponent(e164)}`);
}
