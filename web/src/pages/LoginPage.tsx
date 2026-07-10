import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useAuth } from "../context/AuthContext";
import { useToast } from "../context/ToastContext";
import { useApiAction } from "../hooks/useApiAction";
import { RequestResult } from "../components/RequestResult";

export function LoginPage() {
  const { login } = useAuth();
  const toast = useToast();
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const action = useApiAction(login);

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    try {
      await action.run(email, password);
      toast.success("Logged in");
      navigate("/");
    } catch {
      /* surfaced via RequestResult */
    }
  };

  return (
    <div className="mx-auto max-w-sm">
      <h2 className="mb-4 text-lg font-semibold">Login</h2>
      <form onSubmit={onSubmit} className="card flex flex-col gap-3">
        <div>
          <label className="label">Email</label>
          <input className="input" type="email" required value={email} onChange={(e) => setEmail(e.target.value)} />
        </div>
        <div>
          <label className="label">Password</label>
          <input
            className="input"
            type="password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </div>
        <button className="btn" type="submit" disabled={action.loading}>
          {action.loading ? "Logging in…" : "Login"}
        </button>
      </form>
      <div className="mt-3">
        <RequestResult loading={false} error={action.error} result={null} />
      </div>
      <p className="mt-4 text-sm text-slate-500">
        No account?{" "}
        <Link className="text-slate-900 underline" to="/register">
          Register
        </Link>
      </p>
    </div>
  );
}
