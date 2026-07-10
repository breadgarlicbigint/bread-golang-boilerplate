import { useEffect } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../context/AuthContext";
import { useApiAction } from "../hooks/useApiAction";
import { health } from "../api/health";
import { RequestResult } from "../components/RequestResult";

const QUICK_LINKS: { to: string; label: string; desc: string }[] = [
  { to: "/profile", label: "Profile", desc: "GET/PATCH /v1/me, change password" },
  { to: "/2fa", label: "Two-Factor Auth", desc: "Enable + verify TOTP" },
  { to: "/passkeys", label: "Passkeys", desc: "WebAuthn registration + login" },
  { to: "/mobile", label: "Mobile Numbers", desc: "OTP send/verify via SMS or WhatsApp" },
  { to: "/notifications", label: "Notifications", desc: "List, preferences, device tokens" },
  { to: "/oauth", label: "OAuth / Social", desc: "GitHub + Apple sign-in" },
  { to: "/admin/users", label: "Admin — Users", desc: "CRUD, block/unblock (admin only)" },
  { to: "/admin/analytics", label: "Admin — Analytics", desc: "12 reporting endpoints (admin only)" },
  { to: "/console", label: "API Console", desc: "Fire any raw request" },
];

export function DashboardPage() {
  const { user, isAuthenticated } = useAuth();
  const healthAction = useApiAction(health);

  useEffect(() => {
    void healthAction.run();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h2 className="text-lg font-semibold">Dashboard</h2>
        <p className="text-sm text-slate-500">
          {isAuthenticated ? `Signed in as ${user?.email} (${user?.role})` : "Not signed in — most pages require login."}
        </p>
      </div>

      <div className="card">
        <h3 className="mb-2 text-sm font-semibold">API health</h3>
        <RequestResult loading={healthAction.loading} error={healthAction.error} result={healthAction.result} />
      </div>

      <div>
        <h3 className="mb-2 text-sm font-semibold">Quick links</h3>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {QUICK_LINKS.map((l) => (
            <Link key={l.to} to={l.to} className="card block hover:border-slate-400">
              <p className="font-medium">{l.label}</p>
              <p className="text-xs text-slate-500">{l.desc}</p>
            </Link>
          ))}
        </div>
      </div>
    </div>
  );
}
