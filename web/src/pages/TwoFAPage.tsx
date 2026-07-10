import { useState, type FormEvent } from "react";
import * as authApi from "../api/auth";
import { useApiAction } from "../hooks/useApiAction";
import { RequestResult } from "../components/RequestResult";
import { useToast } from "../context/ToastContext";

export function TwoFAPage() {
  const enableAction = useApiAction(authApi.enable2FA);
  const verifyAction = useApiAction(authApi.verify2FA);
  const toast = useToast();
  const [code, setCode] = useState("");

  const onVerify = async (e: FormEvent) => {
    e.preventDefault();
    try {
      await verifyAction.run(code);
      toast.success("2FA verified and activated");
      setCode("");
    } catch {
      /* surfaced below */
    }
  };

  return (
    <div className="flex max-w-lg flex-col gap-6">
      <div>
        <h2 className="text-lg font-semibold">Two-Factor Authentication</h2>
        <p className="text-sm text-slate-500">POST /v1/auth/2fa/enable · POST /v1/auth/2fa/verify</p>
      </div>

      <div className="card flex flex-col gap-3">
        <h3 className="text-sm font-semibold">1. Enable 2FA</h3>
        <p className="text-xs text-slate-500">
          Returns a TOTP secret + QR code URL + backup codes. Scan the QR (or add the secret manually) in an
          authenticator app, then verify below with the 6-digit code.
        </p>
        <button className="btn self-start" onClick={() => void enableAction.run()} disabled={enableAction.loading}>
          Enable 2FA
        </button>
        <RequestResult loading={enableAction.loading} error={enableAction.error} result={enableAction.result} />
      </div>

      <form onSubmit={onVerify} className="card flex flex-col gap-3">
        <h3 className="text-sm font-semibold">2. Verify code</h3>
        <div>
          <label className="label">6-digit code</label>
          <input
            className="input"
            required
            minLength={6}
            maxLength={6}
            value={code}
            onChange={(e) => setCode(e.target.value)}
          />
        </div>
        <button className="btn self-start" type="submit" disabled={verifyAction.loading}>
          Verify
        </button>
        <RequestResult loading={false} error={verifyAction.error} result={verifyAction.result} />
      </form>
    </div>
  );
}
