import { useEffect, useState } from "react";
import { startAuthentication, startRegistration } from "@simplewebauthn/browser";
import * as passkeyApi from "../api/passkey";
import { useApiAction } from "../hooks/useApiAction";
import { RequestResult } from "../components/RequestResult";
import { useToast } from "../context/ToastContext";
import { ApiError } from "../lib/apiClient";

export function PasskeysPage() {
  const listAction = useApiAction(passkeyApi.listPasskeys);
  const deleteAction = useApiAction(passkeyApi.deletePasskey);
  const toast = useToast();

  const [attachment, setAttachment] = useState<passkeyApi.Attachment>("platform");
  const [friendlyName, setFriendlyName] = useState("My Passkey");
  const [registerBusy, setRegisterBusy] = useState(false);
  const [registerError, setRegisterError] = useState<string | null>(null);
  const [registerResult, setRegisterResult] = useState<unknown>(null);

  const [discoverableBusy, setDiscoverableBusy] = useState(false);
  const [discoverableError, setDiscoverableError] = useState<string | null>(null);
  const [discoverableResult, setDiscoverableResult] = useState<unknown>(null);

  const identifiedAction = useApiAction(passkeyApi.beginIdentifiedLogin);
  const [identifiedEmail, setIdentifiedEmail] = useState("");

  useEffect(() => {
    void listAction.run();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const onRegister = async () => {
    setRegisterBusy(true);
    setRegisterError(null);
    setRegisterResult(null);
    try {
      const begin = await passkeyApi.beginRegistration(attachment);
      const credential = await startRegistration(begin.publicKey as Parameters<typeof startRegistration>[0]);
      const passkey = await passkeyApi.finishRegistration(attachment, friendlyName, credential);
      setRegisterResult(passkey);
      toast.success("Passkey registered");
      await listAction.run();
    } catch (err) {
      setRegisterError(err instanceof ApiError ? `${err.status} — ${err.message}` : String(err));
    } finally {
      setRegisterBusy(false);
    }
  };

  const onDiscoverableLogin = async () => {
    setDiscoverableBusy(true);
    setDiscoverableError(null);
    setDiscoverableResult(null);
    try {
      const begin = await passkeyApi.beginDiscoverableLogin();
      const credential = await startAuthentication(
        begin.options.publicKey as Parameters<typeof startAuthentication>[0],
        false,
      );
      const result = await passkeyApi.finishDiscoverableLogin(begin.sessionToken, credential);
      setDiscoverableResult(result);
    } catch (err) {
      setDiscoverableError(err instanceof ApiError ? `${err.status} — ${err.message}` : String(err));
    } finally {
      setDiscoverableBusy(false);
    }
  };

  const onDelete = async (id: string) => {
    try {
      await deleteAction.run(id);
      toast.success("Passkey removed");
      await listAction.run();
    } catch {
      /* surfaced below */
    }
  };

  return (
    <div className="flex max-w-2xl flex-col gap-6">
      <div>
        <h2 className="text-lg font-semibold">Passkeys / WebAuthn</h2>
        <p className="text-sm text-slate-500">
          Uses the browser's real <code>navigator.credentials</code> API via @simplewebauthn/browser. Requires a
          platform authenticator (Touch ID / Windows Hello) or security key, and{" "}
          <code>WEBAUTHN_RP_ORIGIN</code> on the API matching this page's origin.
        </p>
      </div>

      <div className="card">
        <h3 className="mb-2 text-sm font-semibold">My passkeys — GET /v1/me/passkeys</h3>
        <RequestResult loading={listAction.loading} error={listAction.error} result={null} />
        {listAction.result && listAction.result.length > 0 ? (
          <table className="table-base">
            <thead>
              <tr>
                <th>Friendly name</th>
                <th>Attachment</th>
                <th>Biometric</th>
                <th>Created</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {listAction.result.map((pk) => (
                <tr key={pk.id}>
                  <td>{pk.friendlyName}</td>
                  <td>{pk.attachment}</td>
                  <td>{pk.isBiometric ? "yes" : "no"}</td>
                  <td>{pk.createdAt}</td>
                  <td>
                    <button className="btn-danger !px-2 !py-1 text-xs" onClick={() => void onDelete(pk.id)}>
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          !listAction.loading && <p className="text-sm text-slate-500">No passkeys yet.</p>
        )}
      </div>

      <div className="card flex flex-col gap-3">
        <h3 className="text-sm font-semibold">Register a passkey — POST .../register/begin + /finish</h3>
        <div>
          <label className="label">Attachment</label>
          <select className="input" value={attachment} onChange={(e) => setAttachment(e.target.value as passkeyApi.Attachment)}>
            <option value="platform">platform (Touch ID / Face ID / Windows Hello)</option>
            <option value="cross-platform">cross-platform (security key / passkey manager)</option>
          </select>
        </div>
        <div>
          <label className="label">Friendly name</label>
          <input className="input" value={friendlyName} onChange={(e) => setFriendlyName(e.target.value)} />
        </div>
        <button className="btn self-start" onClick={() => void onRegister()} disabled={registerBusy}>
          {registerBusy ? "Waiting for authenticator…" : "Register passkey"}
        </button>
        {registerError && <p className="field-error">{registerError}</p>}
        <RequestResult loading={false} error={null} result={registerResult} />
      </div>

      <div className="card flex flex-col gap-3">
        <h3 className="text-sm font-semibold">Usernameless (discoverable) login — POST /v1/auth/passkey/login/*</h3>
        <p className="text-xs text-slate-500">
          Note: <code>FinishDiscoverableLogin</code> in the backend calls the identified-user{" "}
          <code>FinishLogin</code> with a <code>nil</code> user, so this ceremony is expected to complete in the
          browser but fail server-side with 401 — a known gap in the boilerplate's passkey module, not this client.
        </p>
        <button className="btn self-start" onClick={() => void onDiscoverableLogin()} disabled={discoverableBusy}>
          {discoverableBusy ? "Waiting for authenticator…" : "Start discoverable login"}
        </button>
        {discoverableError && <p className="field-error">{discoverableError}</p>}
        <RequestResult loading={false} error={null} result={discoverableResult} />
      </div>

      <div className="card flex flex-col gap-3">
        <h3 className="text-sm font-semibold">Identified login — POST /v1/auth/passkey/identified/begin</h3>
        <p className="text-xs text-slate-500">
          The backend handler is an unwired stub (returns a fixed message, no real challenge) — included here for
          completeness.
        </p>
        <div>
          <label className="label">Email</label>
          <input className="input" value={identifiedEmail} onChange={(e) => setIdentifiedEmail(e.target.value)} />
        </div>
        <button
          className="btn self-start"
          onClick={() => void identifiedAction.run(identifiedEmail)}
          disabled={identifiedAction.loading}
        >
          Begin
        </button>
        <RequestResult loading={identifiedAction.loading} error={identifiedAction.error} result={identifiedAction.result} />
      </div>
    </div>
  );
}
