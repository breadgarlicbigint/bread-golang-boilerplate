import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";
import * as authApi from "../api/auth";
import { clearTokens, getAccessToken, getStoredUser, setStoredUser, setTokens, type StoredUser } from "../lib/storage";

interface AuthContextValue {
  user: StoredUser | null;
  isAuthenticated: boolean;
  isAdmin: boolean;
  login: (email: string, password: string) => Promise<authApi.LoginResponse>;
  register: (payload: authApi.RegisterRequest) => Promise<void>;
  logout: () => Promise<void>;
  logoutAll: () => Promise<void>;
  importSession: (accessToken: string, refreshToken: string, user: StoredUser) => void;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<StoredUser | null>(() => getStoredUser());

  useEffect(() => {
    const handler = () => setUser(null);
    window.addEventListener("ack:auth-cleared", handler);
    return () => window.removeEventListener("ack:auth-cleared", handler);
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    const res = await authApi.login({ email, password });
    setTokens(res.accessToken, res.refreshToken);
    setStoredUser(res.user);
    setUser(res.user);
    return res;
  }, []);

  const register = useCallback(async (payload: authApi.RegisterRequest) => {
    await authApi.register(payload);
  }, []);

  const logout = useCallback(async () => {
    try {
      await authApi.logout();
    } finally {
      clearTokens();
      setUser(null);
    }
  }, []);

  const logoutAll = useCallback(async () => {
    try {
      await authApi.logoutAll();
    } finally {
      clearTokens();
      setUser(null);
    }
  }, []);

  const importSession = useCallback((accessToken: string, refreshToken: string, u: StoredUser) => {
    setTokens(accessToken, refreshToken);
    setStoredUser(u);
    setUser(u);
  }, []);

  const value: AuthContextValue = {
    user,
    isAuthenticated: !!user && !!getAccessToken(),
    isAdmin: user?.role === "admin",
    login,
    register,
    logout,
    logoutAll,
    importSession,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
