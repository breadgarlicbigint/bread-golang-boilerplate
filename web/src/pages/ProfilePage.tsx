import { useEffect, useState, type ChangeEvent, type FormEvent } from "react";
import * as meApi from "../api/me";
import { useApiAction } from "../hooks/useApiAction";
import { RequestResult } from "../components/RequestResult";
import { useToast } from "../context/ToastContext";

export function ProfilePage() {
  const getAction = useApiAction(meApi.getMe);
  const updateAction = useApiAction(meApi.updateMe);
  const passwordAction = useApiAction(meApi.changePassword);
  const toast = useToast();

  const [form, setForm] = useState<meApi.UpdateProfileRequest>({});
  const [pwForm, setPwForm] = useState<meApi.ChangePasswordRequest>({ oldPassword: "", newPassword: "" });

  useEffect(() => {
    void getAction.run().then((u) => {
      setForm({ firstName: u.firstName, lastName: u.lastName, phoneNumber: u.phoneNumber, gender: u.gender });
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const setField = (k: keyof meApi.UpdateProfileRequest) => (e: ChangeEvent<HTMLInputElement>) =>
    setForm((f) => ({ ...f, [k]: e.target.value }));

  const onUpdate = async (e: FormEvent) => {
    e.preventDefault();
    try {
      await updateAction.run(form);
      toast.success("Profile updated");
    } catch {
      /* surfaced below */
    }
  };

  const onChangePassword = async (e: FormEvent) => {
    e.preventDefault();
    try {
      await passwordAction.run(pwForm);
      toast.success("Password changed");
      setPwForm({ oldPassword: "", newPassword: "" });
    } catch {
      /* surfaced below */
    }
  };

  return (
    <div className="flex max-w-2xl flex-col gap-6">
      <div>
        <h2 className="text-lg font-semibold">Profile</h2>
        <p className="text-sm text-slate-500">GET /v1/me · PATCH /v1/me · PATCH /v1/me/password</p>
      </div>

      <div className="card">
        <h3 className="mb-2 text-sm font-semibold">Current profile</h3>
        <RequestResult loading={getAction.loading} error={getAction.error} result={getAction.result} />
      </div>

      <form onSubmit={onUpdate} className="card flex flex-col gap-3">
        <h3 className="text-sm font-semibold">Update profile</h3>
        <div>
          <label className="label">First name</label>
          <input className="input" value={form.firstName ?? ""} onChange={setField("firstName")} />
        </div>
        <div>
          <label className="label">Last name</label>
          <input className="input" value={form.lastName ?? ""} onChange={setField("lastName")} />
        </div>
        <div>
          <label className="label">Phone (E.164, e.g. +15551234567)</label>
          <input className="input" value={form.phoneNumber ?? ""} onChange={setField("phoneNumber")} />
        </div>
        <div>
          <label className="label">Gender</label>
          <select
            className="input"
            value={form.gender ?? ""}
            onChange={(e) => setForm((f) => ({ ...f, gender: e.target.value }))}
          >
            <option value="">—</option>
            <option value="male">male</option>
            <option value="female">female</option>
            <option value="other">other</option>
          </select>
        </div>
        <button className="btn self-start" type="submit" disabled={updateAction.loading}>
          Save
        </button>
        <RequestResult loading={false} error={updateAction.error} result={updateAction.result} />
      </form>

      <form onSubmit={onChangePassword} className="card flex flex-col gap-3">
        <h3 className="text-sm font-semibold">Change password</h3>
        <div>
          <label className="label">Old password</label>
          <input
            className="input"
            type="password"
            required
            value={pwForm.oldPassword}
            onChange={(e) => setPwForm((f) => ({ ...f, oldPassword: e.target.value }))}
          />
        </div>
        <div>
          <label className="label">New password (min 8 chars)</label>
          <input
            className="input"
            type="password"
            required
            minLength={8}
            value={pwForm.newPassword}
            onChange={(e) => setPwForm((f) => ({ ...f, newPassword: e.target.value }))}
          />
        </div>
        <button className="btn self-start" type="submit" disabled={passwordAction.loading}>
          Change password
        </button>
        <RequestResult loading={false} error={passwordAction.error} result={passwordAction.result} />
      </form>
    </div>
  );
}
