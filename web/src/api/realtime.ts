import { api } from "../lib/apiClient";
import { getAccessToken, getSettings } from "../lib/storage";

export interface RealtimeEvent {
  topic: string;
  type: string;
  title?: string;
  body?: string;
  data?: Record<string, unknown>;
  timestamp: string;
}

export interface PublishRequest {
  topic: string;
  type: string;
  title?: string;
  body?: string;
  data?: Record<string, unknown>;
}

export interface PublishResult {
  delivered: number;
}

export async function publish(payload: PublishRequest): Promise<PublishResult> {
  const res = await api.post<PublishResult>("/v1/admin/realtime/publish", payload);
  return res.data;
}

export interface RealtimeStats {
  topicCount: number;
  clientCount: number;
  topics: Record<string, number>;
}

export async function getStats(): Promise<RealtimeStats> {
  const res = await api.get<RealtimeStats>("/v1/admin/realtime/stats");
  return res.data;
}

/** GET /v1/me/ws user-topic naming convention, mirrored client-side purely
 * for display — the server auto-subscribes every connection to this topic,
 * nothing here needs to send it. */
export function userTopic(userId: string): string {
  return `user:${userId}`;
}

/** Opens a WebSocket to GET /v1/me/ws, auto-subscribed server-side to the
 * caller's private channel. The token goes in ?token= because the native
 * WebSocket API can't set an Authorization header. */
export function openWebSocket(): WebSocket {
  const { apiBaseUrl } = getSettings();
  const url = new URL("v1/me/ws", apiBaseUrl.endsWith("/") ? apiBaseUrl : `${apiBaseUrl}/`);
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  url.searchParams.set("token", getAccessToken() ?? "");
  return new WebSocket(url.toString());
}

/** Sends a {"action":"subscribe"|"unsubscribe","topic":"..."} control frame
 * over an open WebSocket — the generic pub/sub demo on top of the
 * auto-subscribed private channel. */
export function wsControlMessage(action: "subscribe" | "unsubscribe", topic: string): string {
  return JSON.stringify({ action, topic });
}

/** Opens an SSE stream to GET /v1/me/events. Same ?token= constraint as
 * WebSocket (EventSource can't set headers either). Pass `topic` to watch
 * an arbitrary topic instead of the caller's private channel — unlike
 * WebSocket, SSE is one-directional so the topic is fixed for the
 * connection's lifetime. */
export function openEventSource(topic?: string): EventSource {
  const { apiBaseUrl } = getSettings();
  const url = new URL("v1/me/events", apiBaseUrl.endsWith("/") ? apiBaseUrl : `${apiBaseUrl}/`);
  url.searchParams.set("token", getAccessToken() ?? "");
  if (topic) url.searchParams.set("topic", topic);
  return new EventSource(url.toString());
}
