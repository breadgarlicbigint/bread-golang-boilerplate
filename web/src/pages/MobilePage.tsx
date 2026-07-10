import { useEffect, useState } from "react";
import * as mobileApi from "../api/mobile";
import { useApiAction } from "../hooks/useApiAction";
import { RequestResult } from "../components/RequestResult";
import { useToast } from "../context/ToastContext";

export function MobilePage() {
  const listAction = useApiAction(mobileApi.listMobiles);
  const sendAction = useApiAction(mobileApi.sendOTP);
  const verifyAction = useApiAction(mobileApi.verifyOTP);
  const primaryAction = useApiAction(mobileApi.setPrimary);
  const deleteAction = useApiAction(mobileApi.deleteMobile);
  const toast = useToast();

  const [e164, setE164] = useState("");
  const [channel, setChannel] = useState<"sms" | "whatsapp">("sms");
  const [code, setCode] = useState("");

  useEffect(() => {
    void listAction.run();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const refresh = () => void listAction.run();

  return (
    <div className="flex max-w-2xl flex-col gap-6">
      <div>
        <h2 className="text-lg font-semibold">Mobile Numbers</h2>
        <p className="text-sm text-slate-500">GET/POST/PATCH/DELETE /v1/me/mobiles</p>
      </div>

      <div className="card">
        <h3 className="mb-2 text-sm font-semibold">My mobiles</h3>
        <RequestResult loading={listAction.loading} error={listAction.error} result={null} />
        {listAction.result && listAction.result.length > 0 ? (
          <table className="table-base">
            <thead>
              <tr>
                <th>E.164</th>
                <th>Verified</th>
                <th>Primary</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {listAction.result.map((m) => (
                <tr key={m.id}>
                  <td>{m.e164}</td>
                  <td>{m.isVerified ? "yes" : "no"}</td>
                  <td>{m.isPrimary ? "yes" : "no"}</td>
                  <td className="flex gap-2">
                    {!m.isPrimary && (
                      <button
                        className="btn-secondary !px-2 !py-1 text-xs"
                        onClick={() => primaryAction.run(m.e164).then(refresh)}
                      >
                        Make primary
                      </button>
                    )}
                    <button className="btn-danger !px-2 !py-1 text-xs" onClick={() => deleteAction.run(m.e164).then(refresh)}>
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          !listAction.loading && <p className="text-sm text-slate-500">No mobile numbers yet.</p>
        )}
        <RequestResult loading={false} error={primaryAction.error ?? deleteAction.error} result={null} />
      </div>

      <div className="card flex flex-col gap-3">
        <h3 className="text-sm font-semibold">Send OTP — POST /v1/me/mobiles/send-otp</h3>
        <div>
          <label className="label">E.164 phone number</label>
          <input className="input" placeholder="+15551234567" value={e164} onChange={(e) => setE164(e.target.value)} />
        </div>
        <div>
          <label className="label">Channel</label>
          <select className="input" value={channel} onChange={(e) => setChannel(e.target.value as "sms" | "whatsapp")}>
            <option value="sms">sms</option>
            <option value="whatsapp">whatsapp</option>
          </select>
        </div>
        <button
          className="btn self-start"
          onClick={() =>
            sendAction
              .run(e164, channel)
              .then(() => toast.success("OTP sent"))
              .catch(() => undefined)
          }
          disabled={sendAction.loading}
        >
          Send OTP
        </button>
        <RequestResult loading={sendAction.loading} error={sendAction.error} result={sendAction.result} />
      </div>

      <div className="card flex flex-col gap-3">
        <h3 className="text-sm font-semibold">Verify OTP — POST /v1/me/mobiles/verify</h3>
        <div>
          <label className="label">E.164 phone number</label>
          <input className="input" placeholder="+15551234567" value={e164} onChange={(e) => setE164(e.target.value)} />
        </div>
        <div>
          <label className="label">6-digit code</label>
          <input className="input" maxLength={6} value={code} onChange={(e) => setCode(e.target.value)} />
        </div>
        <button
          className="btn self-start"
          onClick={() =>
            verifyAction
              .run(e164, code)
              .then(() => {
                toast.success("Mobile verified");
                refresh();
              })
              .catch(() => undefined)
          }
          disabled={verifyAction.loading}
        >
          Verify
        </button>
        <RequestResult loading={verifyAction.loading} error={verifyAction.error} result={verifyAction.result} />
      </div>
    </div>
  );
}
