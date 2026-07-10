import { Link } from "react-router-dom";

export function NotFoundPage() {
  return (
    <div>
      <h2 className="text-lg font-semibold">Page not found</h2>
      <Link className="text-slate-900 underline" to="/">
        Back to dashboard
      </Link>
    </div>
  );
}
