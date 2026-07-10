export interface Settings {
  apiBaseUrl: string;
  tenantId: string;
  lang: string;
  appVersion: string;
  appPlatform: string;
}

export interface StoredUser {
  id: string;
  email: string;
  role: string;
}

const KEYS = {
  accessToken: "ack.accessToken",
  refreshToken: "ack.refreshToken",
  user: "ack.user",
  settings: "ack.settings",
};

const DEFAULT_SETTINGS: Settings = {
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL || "http://localhost:3000",
  tenantId: "",
  lang: "en",
  appVersion: "",
  appPlatform: "",
};

export function getSettings(): Settings {
  const raw = localStorage.getItem(KEYS.settings);
  if (!raw) return DEFAULT_SETTINGS;
  try {
    return { ...DEFAULT_SETTINGS, ...JSON.parse(raw) };
  } catch {
    return DEFAULT_SETTINGS;
  }
}

export function setSettings(s: Settings) {
  localStorage.setItem(KEYS.settings, JSON.stringify(s));
}

export function getAccessToken(): string | null {
  return localStorage.getItem(KEYS.accessToken);
}

export function getRefreshToken(): string | null {
  return localStorage.getItem(KEYS.refreshToken);
}

export function setTokens(accessToken: string, refreshToken: string) {
  localStorage.setItem(KEYS.accessToken, accessToken);
  localStorage.setItem(KEYS.refreshToken, refreshToken);
}

export type AuthClearReason = "manual" | "expired";

/** reason "expired" (session invalid — 401 after a failed refresh) triggers a toast + redirect; "manual" (Logout button) just clears state. */
export function clearTokens(reason: AuthClearReason = "manual") {
  localStorage.removeItem(KEYS.accessToken);
  localStorage.removeItem(KEYS.refreshToken);
  localStorage.removeItem(KEYS.user);
  window.dispatchEvent(new CustomEvent<{ reason: AuthClearReason }>("ack:auth-cleared", { detail: { reason } }));
}

export function getStoredUser(): StoredUser | null {
  const raw = localStorage.getItem(KEYS.user);
  if (!raw) return null;
  try {
    return JSON.parse(raw);
  } catch {
    return null;
  }
}

export function setStoredUser(u: StoredUser) {
  localStorage.setItem(KEYS.user, JSON.stringify(u));
}
