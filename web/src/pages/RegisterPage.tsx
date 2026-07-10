import { useState, type ChangeEvent, type FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useAuth } from "../context/AuthContext";
import { useToast } from "../context/ToastContext";
import { useApiAction } from "../hooks/useApiAction";
import { RequestResult } from "../components/RequestResult";
import type { RegisterRequest } from "../api/auth";

const EMPTY: RegisterRequest = { email: "", username: "", password: "", firstName: "", lastName: "" };

export function RegisterPage() {
  const { register } = useAuth();
  const toast = useToast();
  const navigate = useNavigate();
  const [form, setForm] = useState<RegisterRequest>(EMPTY);
  const action = useApiAction(register);

  const set = (k: keyof RegisterRequest) => (e: ChangeEvent<HTMLInputElement>) =>
    setForm((f) => ({ ...f, [k]: e.target.value }));

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    try {
      await action.run(form);
      toast.success("Registered — you can now log in");
      navigate("/login");
    } catch {
      /* surfaced via RequestResult */
    }
  };

  return (
    <div className="mx-auto max-w-sm">
      <h2 className="mb-4 text-lg font-semibold">Register</h2>
      <form onSubmit={onSubmit} className="card flex flex-col gap-3">
        <div>
          <label className="label">Email</label>
          <input className="input" type="email" required value={form.email} onChange={set("email")} />
        </div>
        <div>
          <label className="label">Username</label>
          <input className="input" required minLength={3} value={form.username} onChange={set("username")} />
        </div>
        <div>
          <label className="label">Password (min 8 chars)</label>
          <input className="input" type="password" required minLength={8} value={form.password} onChange={set("password")} />
        </div>
        <div>
          <label className="label">First name</label>
          <input className="input" required value={form.firstName} onChange={set("firstName")} />
        </div>
        <div>
          <label className="label">Last name</label>
          <input className="input" required value={form.lastName} onChange={set("lastName")} />
        </div>
        <button className="btn" type="submit" disabled={action.loading}>
          {action.loading ? "Registering…" : "Register"}
        </button>
      </form>
      <div className="mt-3">
        <RequestResult loading={false} error={action.error} result={null} />
      </div>
      <p className="mt-4 text-sm text-slate-500">
        Already have an account?{" "}
        <Link className="text-slate-900 underline" to="/login">
          Login
        </Link>
      </p>
    </div>
  );
}
