import type { ReactNode } from "react";
import { NavLink, Outlet } from "react-router-dom";
import { useAuth } from "../context/AuthContext";
import { useSettings } from "../context/SettingsContext";

const linkClass = ({ isActive }: { isActive: boolean }) =>
  "block rounded-md px-3 py-1.5 text-sm " + (isActive ? "bg-slate-900 text-white" : "text-slate-600 hover:bg-slate-200");

function NavSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="mb-4">
      <p className="mb-1 px-3 text-xs font-semibold uppercase tracking-wide text-slate-400">{title}</p>
      <nav className="flex flex-col gap-0.5">{children}</nav>
    </div>
  );
}

export function Layout() {
  const { user, isAdmin, isAuthenticated, logout } = useAuth();
  const { settings } = useSettings();

  return (
    <div className="flex min-h-screen">
      <aside className="w-60 shrink-0 border-r border-slate-200 bg-white p-3">
        <h1 className="mb-4 px-3 text-base font-bold">ACK API Console</h1>

        <NavSection title="Account">
          <NavLink to="/" end className={linkClass}>
            Dashboard
          </NavLink>
          <NavLink to="/profile" className={linkClass}>
            Profile
          </NavLink>
          <NavLink to="/2fa" className={linkClass}>
            Two-Factor Auth
          </NavLink>
          <NavLink to="/passkeys" className={linkClass}>
            Passkeys
          </NavLink>
          <NavLink to="/mobile" className={linkClass}>
            Mobile Numbers
          </NavLink>
          <NavLink to="/notifications" className={linkClass}>
            Notifications
          </NavLink>
          <NavLink to="/realtime" className={linkClass}>
            Realtime (WS/SSE)
          </NavLink>
          <NavLink to="/oauth" className={linkClass}>
            OAuth / Social
          </NavLink>
        </NavSection>

        {isAdmin && (
          <NavSection title="Admin">
            <NavLink to="/admin/users" className={linkClass}>
              Users
            </NavLink>
            <NavLink to="/admin/app-versions" className={linkClass}>
              App Versions
            </NavLink>
            <NavLink to="/admin/analytics" className={linkClass}>
              Analytics
            </NavLink>
            <NavLink to="/admin/notifications" className={linkClass}>
              Notifications
            </NavLink>
            <NavLink to="/admin/iot" className={linkClass}>
              IoT (MQTT)
            </NavLink>
          </NavSection>
        )}

        <NavSection title="Tools">
          <NavLink to="/app-version-check" className={linkClass}>
            App Version Check
          </NavLink>
          <NavLink to="/health" className={linkClass}>
            Health
          </NavLink>
          <NavLink to="/console" className={linkClass}>
            API Console
          </NavLink>
          <NavLink to="/i18n-compare" className={linkClass}>
            i18n Compare
          </NavLink>
          <NavLink to="/activity" className={linkClass}>
            Activity Log
          </NavLink>
          <NavLink to="/settings" className={linkClass}>
            Settings
          </NavLink>
        </NavSection>
      </aside>

      <div className="flex flex-1 flex-col">
        <header className="flex items-center justify-between border-b border-slate-200 bg-white px-4 py-2">
          <span className="text-xs text-slate-500">{settings.apiBaseUrl}</span>
          <div className="flex items-center gap-3 text-sm">
            {isAuthenticated ? (
              <>
                <span className="text-slate-600">
                  {user?.email} <span className="text-xs text-slate-400">({user?.role})</span>
                </span>
                <button className="btn-secondary" onClick={() => void logout()}>
                  Logout
                </button>
              </>
            ) : (
              <NavLink to="/login" className="btn">
                Login
              </NavLink>
            )}
          </div>
        </header>

        <main className="flex-1 p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
