import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useToast } from "../context/ToastContext";
import type { AuthClearReason } from "../lib/storage";

/**
 * Mounted once inside <BrowserRouter>. Reacts to "ack:auth-cleared" fired by
 * lib/storage.clearTokens — redirects to /login from anywhere in the app
 * (not just pages behind ProtectedRoute), and toasts only when the session
 * was cleared automatically (401 after a failed refresh), not on a manual
 * Logout click.
 */
export function SessionWatcher() {
  const navigate = useNavigate();
  const toast = useToast();

  useEffect(() => {
    const handler = (e: Event) => {
      const reason = (e as CustomEvent<{ reason: AuthClearReason }>).detail?.reason;
      if (reason === "expired") {
        toast.error("Your session has expired — please log in again.");
      }
      if (window.location.pathname !== "/login") {
        navigate("/login", { replace: true });
      }
    };
    window.addEventListener("ack:auth-cleared", handler);
    return () => window.removeEventListener("ack:auth-cleared", handler);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [navigate]);

  return null;
}
