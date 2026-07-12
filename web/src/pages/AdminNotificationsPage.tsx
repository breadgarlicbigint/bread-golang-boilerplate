import { useEffect, useState, type FormEvent } from "react";
import * as notifApi from "../api/notifications";
import * as usersApi from "../api/users";
import { useApiAction } from "../hooks/useApiAction";
import { RequestResult } from "../components/RequestResult";
import { useToast } from "../context/ToastContext";

const TYPES = ["system", "auth", "user", "promotion", "alert", "info"] as const;
const CHANNELS = ["email", "push", "in_app", "silent", "whatsapp", "sms"] as const;

export function AdminNotificationsPage() {
  const usersAction = useApiAction(usersApi.listUsers);
  const testEmailAction = useApiAction(notifApi.testEmail);
  const sendAction = useApiAction(notifApi.adminSend);
  const broadcastAction = useApiAction(notifApi.adminBroadcast);
  const toast = useToast();

  useEffect(() => {
    void usersAction.run({ perPage: 50 });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  const users = usersAction.result?.items ?? [];

  // ── Test email ────────────────────────────────────────────────────────────
  const [testTo, setTestTo] = useState("");
  const onTestEmail = async (e: FormEvent) => {
    e.preventDefault();
    try {
      const res = await testEmailAction.run(testTo);
      if (res.sent) toast.success("Test email sent");
      else toast.error(res.error || "Send failed");
    } catch {
      /* surfaced below */
    }
  };

  // ── Single send (transactional-style — always synchronous) ─────────────────
  const [sendForm, setSendForm] = useState({
    userId: "",
    type: "system" as (typeof TYPES)[number],
    channel: "email" as (typeof CHANNELS)[number],
    title: "",
    body: "",
  });
  const onSend = async (e: FormEvent) => {
    e.preventDefault();
    try {
      await sendAction.run(sendForm);
      toast.success("Notification sent");
    } catch {
      /* surfaced below */
    }
  };

  // ── Broadcast (email channel routes through QUEUE_PROMOTIONAL_DRIVER when set) ─
  const [selectedUserIds, setSelectedUserIds] = useState<string[]>([]);
  const [broadcastEmail, setBroadcastEmail] = useState("");
  const [broadcastForm, setBroadcastForm] = useState({
    type: "promotion" as (typeof TYPES)[number],
    channel: "email" as (typeof CHANNELS)[number],
    title: "",
    body: "",
  });
  const toggleUser = (id: string) =>
    setSelectedUserIds((ids) => (ids.includes(id) ? ids.filter((x) => x !== id) : [...ids, id]));

  const onBroadcast = async (e: FormEvent) => {
    e.preventDefault();
    try {
      const result = await broadcastAction.run({
        userIds: selectedUserIds,
        type: broadcastForm.type,
        channel: broadcastForm.channel,
        title: broadcastForm.title,
        body: broadcastForm.body,
        data: broadcastForm.channel === "email" ? { email: broadcastEmail } : undefined,
      });
      toast.success(`Broadcast: ${result.success} success, ${result.failed} failed`);
    } catch {
      /* surfaced below */
    }
  };

  return (
    <div className="flex max-w-2xl flex-col gap-6">
      <div>
        <h2 className="text-lg font-semibold">Admin — Notifications</h2>
        <p className="text-sm text-slate-500">
          POST /v1/admin/notifications/test-email · /send · /broadcast
        </p>
      </div>

      <div className="card flex flex-col gap-3">
        <h3 className="text-sm font-semibold">1. Test email (diagnostic)</h3>
        <p className="text-xs text-slate-500">
          Sends synchronously and reports the raw transport error — use this first to confirm{" "}
          <code>MAIL_DRIVER</code> is actually working before testing queued delivery below.
        </p>
        <form onSubmit={onTestEmail} className="flex flex-col gap-3">
          <div>
            <label className="label">Send to</label>
            <input
              className="input"
              type="email"
              required
              placeholder="you@example.com"
              value={testTo}
              onChange={(e) => setTestTo(e.target.value)}
            />
          </div>
          <button className="btn self-start" type="submit" disabled={testEmailAction.loading}>
            Send test email
          </button>
          <RequestResult loading={false} error={testEmailAction.error} result={testEmailAction.result} />
        </form>
      </div>

      <div className="card flex flex-col gap-3">
        <h3 className="text-sm font-semibold">2. Send to one user (synchronous)</h3>
        <form onSubmit={onSend} className="flex flex-col gap-3">
          <div>
            <label className="label">User</label>
            <select
              className="input"
              required
              value={sendForm.userId}
              onChange={(e) => setSendForm((f) => ({ ...f, userId: e.target.value }))}
            >
              <option value="" disabled>
                Select a user…
              </option>
              {users.map((u) => (
                <option key={u.id} value={u.id}>
                  {u.email}
                </option>
              ))}
            </select>
          </div>
          <div className="flex gap-3">
            <div className="flex-1">
              <label className="label">Type</label>
              <select
                className="input"
                value={sendForm.type}
                onChange={(e) => setSendForm((f) => ({ ...f, type: e.target.value as (typeof TYPES)[number] }))}
              >
                {TYPES.map((t) => (
                  <option key={t} value={t}>
                    {t}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex-1">
              <label className="label">Channel</label>
              <select
                className="input"
                value={sendForm.channel}
                onChange={(e) => setSendForm((f) => ({ ...f, channel: e.target.value as (typeof CHANNELS)[number] }))}
              >
                {CHANNELS.map((c) => (
                  <option key={c} value={c}>
                    {c}
                  </option>
                ))}
              </select>
            </div>
          </div>
          <div>
            <label className="label">Title</label>
            <input
              className="input"
              required
              value={sendForm.title}
              onChange={(e) => setSendForm((f) => ({ ...f, title: e.target.value }))}
            />
          </div>
          <div>
            <label className="label">Body</label>
            <input
              className="input"
              required
              value={sendForm.body}
              onChange={(e) => setSendForm((f) => ({ ...f, body: e.target.value }))}
            />
          </div>
          <button className="btn self-start" type="submit" disabled={sendAction.loading}>
            Send
          </button>
          <RequestResult loading={false} error={sendAction.error} result={sendAction.result} />
        </form>
      </div>

      <div className="card flex flex-col gap-3">
        <h3 className="text-sm font-semibold">3. Broadcast (queue routing)</h3>
        <p className="text-xs text-slate-500">
          Email-channel broadcasts route through <code>QUEUE_PROMOTIONAL_DRIVER</code> when it's
          configured (see <code>pkg/queue/router</code>) — each recipient is enqueued as its own{" "}
          <code>email:send:promotional</code> job instead of sent synchronously, so{" "}
          <strong>success/failed here means "queued", not "delivered"</strong> — check whichever
          worker consumes that driver to confirm actual delivery. Every recipient currently shares
          the single "Recipient email" field below (per-user address lookup isn't wired up yet).
        </p>
        <form onSubmit={onBroadcast} className="flex flex-col gap-3">
          <div>
            <label className="label">Recipients ({selectedUserIds.length} selected)</label>
            <div className="max-h-40 overflow-y-auto rounded-md border border-slate-200 p-2">
              {users.length === 0 && <p className="text-xs text-slate-400">No users loaded</p>}
              {users.map((u) => (
                <label key={u.id} className="flex items-center gap-2 py-0.5 text-sm">
                  <input
                    type="checkbox"
                    checked={selectedUserIds.includes(u.id)}
                    onChange={() => toggleUser(u.id)}
                  />
                  {u.email}
                </label>
              ))}
            </div>
          </div>
          <div className="flex gap-3">
            <div className="flex-1">
              <label className="label">Type</label>
              <select
                className="input"
                value={broadcastForm.type}
                onChange={(e) => setBroadcastForm((f) => ({ ...f, type: e.target.value as (typeof TYPES)[number] }))}
              >
                {TYPES.map((t) => (
                  <option key={t} value={t}>
                    {t}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex-1">
              <label className="label">Channel</label>
              <select
                className="input"
                value={broadcastForm.channel}
                onChange={(e) =>
                  setBroadcastForm((f) => ({ ...f, channel: e.target.value as (typeof CHANNELS)[number] }))
                }
              >
                {CHANNELS.map((c) => (
                  <option key={c} value={c}>
                    {c}
                  </option>
                ))}
              </select>
            </div>
          </div>
          {broadcastForm.channel === "email" && (
            <div>
              <label className="label">Recipient email (used for every selected user)</label>
              <input
                className="input"
                type="email"
                required
                placeholder="promo-test@example.com"
                value={broadcastEmail}
                onChange={(e) => setBroadcastEmail(e.target.value)}
              />
            </div>
          )}
          <div>
            <label className="label">Title</label>
            <input
              className="input"
              required
              value={broadcastForm.title}
              onChange={(e) => setBroadcastForm((f) => ({ ...f, title: e.target.value }))}
            />
          </div>
          <div>
            <label className="label">Body</label>
            <input
              className="input"
              required
              value={broadcastForm.body}
              onChange={(e) => setBroadcastForm((f) => ({ ...f, body: e.target.value }))}
            />
          </div>
          <button className="btn self-start" type="submit" disabled={broadcastAction.loading || selectedUserIds.length === 0}>
            Broadcast to {selectedUserIds.length} user{selectedUserIds.length === 1 ? "" : "s"}
          </button>
          <RequestResult loading={false} error={broadcastAction.error} result={broadcastAction.result} />
        </form>
      </div>
    </div>
  );
}
