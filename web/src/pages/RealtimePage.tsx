import { useEffect, useRef, useState, type FormEvent } from "react";
import * as realtimeApi from "../api/realtime";
import type { RealtimeEvent } from "../api/realtime";
import { useApiAction } from "../hooks/useApiAction";
import { RequestResult } from "../components/RequestResult";
import { useAuth } from "../context/AuthContext";
import { useToast } from "../context/ToastContext";

const MAX_LOG = 50;

type ConnState = "disconnected" | "connecting" | "connected" | "error";

function StatusBadge({ state }: { state: ConnState }) {
  const styles: Record<ConnState, string> = {
    disconnected: "bg-slate-100 text-slate-600",
    connecting: "bg-amber-100 text-amber-700",
    connected: "bg-emerald-100 text-emerald-700",
    error: "bg-red-100 text-red-700",
  };
  return <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${styles[state]}`}>{state}</span>;
}

function EventLog({ events }: { events: RealtimeEvent[] }) {
  if (events.length === 0) return <p className="text-xs text-slate-400">No events received yet.</p>;
  return (
    <ul className="flex max-h-56 flex-col-reverse gap-1 overflow-y-auto text-xs">
      {events.map((evt, i) => (
        <li key={i} className="rounded border border-slate-200 bg-slate-50 p-2">
          <div className="flex items-center justify-between">
            <span className="font-mono font-medium">{evt.type}</span>
            <span className="text-slate-400">{new Date(evt.timestamp).toLocaleTimeString()}</span>
          </div>
          <div className="text-slate-500">topic: {evt.topic}</div>
          {evt.title && <div>{evt.title}</div>}
          {evt.body && <div className="text-slate-600">{evt.body}</div>}
          {evt.data && <pre className="mt-1 whitespace-pre-wrap break-all text-[10px] text-slate-500">{JSON.stringify(evt.data)}</pre>}
        </li>
      ))}
    </ul>
  );
}

export function RealtimePage() {
  const { user } = useAuth();
  const toast = useToast();

  // ── WebSocket ────────────────────────────────────────────────────────────────
  const wsRef = useRef<WebSocket | null>(null);
  const [wsState, setWsState] = useState<ConnState>("disconnected");
  const [wsEvents, setWsEvents] = useState<RealtimeEvent[]>([]);
  const [wsTopic, setWsTopic] = useState("");

  const connectWs = () => {
    if (wsRef.current) return;
    setWsState("connecting");
    const socket = realtimeApi.openWebSocket();
    socket.onopen = () => setWsState("connected");
    socket.onerror = () => setWsState("error");
    socket.onclose = () => {
      setWsState("disconnected");
      wsRef.current = null;
    };
    socket.onmessage = (msg) => {
      try {
        const evt = JSON.parse(msg.data) as RealtimeEvent;
        setWsEvents((prev) => [evt, ...prev].slice(0, MAX_LOG));
      } catch {
        /* ignore malformed frames */
      }
    };
    wsRef.current = socket;
  };

  const disconnectWs = () => {
    wsRef.current?.close();
    wsRef.current = null;
    setWsState("disconnected");
  };

  const wsSubscribe = () => {
    if (!wsRef.current || !wsTopic) return;
    wsRef.current.send(realtimeApi.wsControlMessage("subscribe", wsTopic));
    toast.success(`Subscribed to "${wsTopic}"`);
  };

  const wsUnsubscribe = () => {
    if (!wsRef.current || !wsTopic) return;
    wsRef.current.send(realtimeApi.wsControlMessage("unsubscribe", wsTopic));
    toast.success(`Unsubscribed from "${wsTopic}"`);
  };

  // ── SSE ──────────────────────────────────────────────────────────────────────
  const sseRef = useRef<EventSource | null>(null);
  const [sseState, setSseState] = useState<ConnState>("disconnected");
  const [sseEvents, setSseEvents] = useState<RealtimeEvent[]>([]);
  const [sseTopic, setSseTopic] = useState("");

  const connectSse = () => {
    if (sseRef.current) return;
    setSseState("connecting");
    const source = realtimeApi.openEventSource(sseTopic || undefined);
    source.onopen = () => setSseState("connected");
    source.onerror = () => setSseState("error");
    source.addEventListener("keepalive", () => {
      /* ignore — just proof the stream is alive */
    });
    // Every non-keepalive server-sent event is dispatched under its own
    // event name (see modules/realtime/handler SSE — c.SSEvent(evtType, evt)).
    // Without an "onmessage"-style catch-all, we listen once and rely on
    // MessageEvent.type to tell them apart.
    const handle = (msg: MessageEvent) => {
      try {
        const evt = JSON.parse(msg.data) as RealtimeEvent;
        setSseEvents((prev) => [evt, ...prev].slice(0, MAX_LOG));
      } catch {
        /* ignore malformed frames */
      }
    };
    source.addEventListener("notification", handle);
    source.addEventListener("iot.telemetry", handle);
    source.addEventListener("custom", handle);
    source.addEventListener("message", handle);
    sseRef.current = source;
  };

  const disconnectSse = () => {
    sseRef.current?.close();
    sseRef.current = null;
    setSseState("disconnected");
  };

  useEffect(() => {
    return () => {
      wsRef.current?.close();
      sseRef.current?.close();
    };
  }, []);

  // ── Admin publish (generic pub/sub) ───────────────────────────────────────────
  const publishAction = useApiAction(realtimeApi.publish);
  const [publishForm, setPublishForm] = useState({
    topic: "",
    type: "custom",
    title: "",
    body: "",
  });
  const onPublish = async (e: FormEvent) => {
    e.preventDefault();
    try {
      const res = await publishAction.run({
        topic: publishForm.topic,
        type: publishForm.type,
        title: publishForm.title,
        body: publishForm.body,
      });
      toast.success(`Delivered to ${res.delivered} connection(s)`);
    } catch {
      /* surfaced below */
    }
  };

  // ── Stats ──────────────────────────────────────────────────────────────────────
  const statsAction = useApiAction(realtimeApi.getStats);

  const myTopic = user ? realtimeApi.userTopic(user.id) : "";

  return (
    <div className="flex max-w-3xl flex-col gap-6">
      <div>
        <h2 className="text-lg font-semibold">Realtime — WebSocket / SSE / Pub-Sub</h2>
        <p className="text-sm text-slate-500">
          GET /v1/me/ws · GET /v1/me/events · POST /v1/admin/realtime/publish · GET /v1/admin/realtime/stats
        </p>
        {user && (
          <p className="mt-1 text-xs text-slate-500">
            Your private channel is <code>{myTopic}</code> — every notification sent to you (see Admin →
            Notifications) is pushed here live in addition to being persisted.
          </p>
        )}
      </div>

      <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
        <div className="card flex flex-col gap-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold">WebSocket</h3>
            <StatusBadge state={wsState} />
          </div>
          <p className="text-xs text-slate-500">
            Auto-subscribed to your private channel on connect. Send subscribe/unsubscribe frames to also
            watch an arbitrary topic (e.g. <code>iot:telemetry</code> or a topic you publish to below).
          </p>
          <div className="flex gap-2">
            <button className="btn" onClick={connectWs} disabled={wsState === "connected" || wsState === "connecting"}>
              Connect
            </button>
            <button className="btn-secondary" onClick={disconnectWs} disabled={wsState !== "connected"}>
              Disconnect
            </button>
          </div>
          <div className="flex gap-2">
            <input
              className="input"
              placeholder="topic to join/leave"
              value={wsTopic}
              onChange={(e) => setWsTopic(e.target.value)}
            />
            <button className="btn-secondary shrink-0" onClick={wsSubscribe} disabled={wsState !== "connected" || !wsTopic}>
              Join
            </button>
            <button className="btn-secondary shrink-0" onClick={wsUnsubscribe} disabled={wsState !== "connected" || !wsTopic}>
              Leave
            </button>
          </div>
          <EventLog events={wsEvents} />
        </div>

        <div className="card flex flex-col gap-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold">Server-Sent Events</h3>
            <StatusBadge state={sseState} />
          </div>
          <p className="text-xs text-slate-500">
            One-directional — the topic is fixed at connect time. Leave blank for your private channel, or
            set it (e.g. <code>iot:telemetry</code>) before connecting.
          </p>
          <div className="flex gap-2">
            <input
              className="input"
              placeholder="topic (blank = your private channel)"
              value={sseTopic}
              onChange={(e) => setSseTopic(e.target.value)}
              disabled={sseState === "connected" || sseState === "connecting"}
            />
          </div>
          <div className="flex gap-2">
            <button className="btn" onClick={connectSse} disabled={sseState === "connected" || sseState === "connecting"}>
              Connect
            </button>
            <button className="btn-secondary" onClick={disconnectSse} disabled={sseState !== "connected"}>
              Disconnect
            </button>
          </div>
          <EventLog events={sseEvents} />
        </div>
      </div>

      <div className="card flex flex-col gap-3">
        <h3 className="text-sm font-semibold">Publish to an arbitrary topic (admin)</h3>
        <p className="text-xs text-slate-500">
          Generic pub/sub test, independent of the notification system — delivers to every WebSocket/SSE
          connection currently subscribed to Topic.
        </p>
        <form onSubmit={onPublish} className="flex flex-col gap-3">
          <div className="flex gap-3">
            <div className="flex-1">
              <label className="label">Topic</label>
              <input
                className="input"
                required
                placeholder={myTopic || "e.g. demo-topic"}
                value={publishForm.topic}
                onChange={(e) => setPublishForm((f) => ({ ...f, topic: e.target.value }))}
              />
            </div>
            <div className="flex-1">
              <label className="label">Type</label>
              <input
                className="input"
                required
                value={publishForm.type}
                onChange={(e) => setPublishForm((f) => ({ ...f, type: e.target.value }))}
              />
            </div>
          </div>
          <div>
            <label className="label">Title</label>
            <input
              className="input"
              value={publishForm.title}
              onChange={(e) => setPublishForm((f) => ({ ...f, title: e.target.value }))}
            />
          </div>
          <div>
            <label className="label">Body</label>
            <input
              className="input"
              value={publishForm.body}
              onChange={(e) => setPublishForm((f) => ({ ...f, body: e.target.value }))}
            />
          </div>
          <button className="btn self-start" type="submit" disabled={publishAction.loading}>
            Publish
          </button>
          <RequestResult loading={false} error={publishAction.error} result={publishAction.result} />
        </form>
      </div>

      <div className="card flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold">Connection stats</h3>
          <button className="btn-secondary" onClick={() => void statsAction.run()} disabled={statsAction.loading}>
            Refresh
          </button>
        </div>
        <RequestResult loading={statsAction.loading} error={statsAction.error} result={statsAction.result} />
      </div>
    </div>
  );
}
