import { api, type PaginationMeta } from "../lib/apiClient";

export interface NotificationResponse {
  id: string;
  type: string;
  channel: string;
  status: string;
  title: string;
  body: string;
  imageUrl?: string;
  data?: Record<string, unknown>;
  actionUrl?: string;
  isRead: boolean;
  createdAt: string;
}

export interface ListNotificationsQuery {
  [key: string]: string | number | boolean | undefined;
  page?: number;
  perPage?: number;
  unreadOnly?: boolean;
}

export async function listNotifications(
  query: ListNotificationsQuery = {},
): Promise<{ items: NotificationResponse[]; meta?: PaginationMeta }> {
  const res = await api.get<NotificationResponse[]>("/v1/me/notifications", { query });
  return { items: res.data, meta: res.meta };
}

export async function unreadCount(): Promise<number> {
  const res = await api.get<{ unread: number }>("/v1/me/notifications/unread-count");
  return res.data.unread;
}

export async function markRead(id: string): Promise<void> {
  await api.patch(`/v1/me/notifications/${encodeURIComponent(id)}/read`);
}

export async function markAllRead(): Promise<void> {
  await api.patch("/v1/me/notifications/read-all");
}

export interface NotificationPreferences {
  channels: Record<string, boolean>;
  types: Record<string, Record<string, boolean>>;
}

export async function getPreferences(): Promise<NotificationPreferences> {
  const res = await api.get<NotificationPreferences>("/v1/me/notifications/preferences");
  return res.data;
}

export async function updatePreferences(payload: NotificationPreferences): Promise<void> {
  await api.patch("/v1/me/notifications/preferences", payload);
}

export interface RegisterDeviceRequest {
  token: string;
  platform: "ios" | "android" | "web";
  deviceModel?: string;
  appVersion?: string;
}

export async function registerDevice(payload: RegisterDeviceRequest): Promise<void> {
  await api.post("/v1/me/notifications/devices", payload);
}

export async function removeDevice(token: string): Promise<void> {
  await api.delete(`/v1/me/notifications/devices/${encodeURIComponent(token)}`);
}

export interface AdminSendRequest {
  userId: string;
  type: string;
  channel: string;
  title: string;
  body: string;
  imageUrl?: string;
  data?: Record<string, unknown>;
  actionUrl?: string;
}

export async function adminSend(payload: AdminSendRequest): Promise<void> {
  await api.post("/v1/admin/notifications/send", payload);
}

export interface AdminBroadcastRequest {
  userIds: string[];
  type: string;
  channel: string;
  title: string;
  body: string;
  data?: Record<string, unknown>;
}

export async function adminBroadcast(payload: AdminBroadcastRequest): Promise<{ sent: number; failed: number } | unknown> {
  const res = await api.post("/v1/admin/notifications/broadcast", payload);
  return res.data;
}
