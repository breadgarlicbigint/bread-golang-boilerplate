export interface LogEntry {
  id: string;
  method: string;
  path: string;
  status: number;
  ok: boolean;
  durationMs: number;
  requestBody?: unknown;
  responseBody?: unknown;
  timestamp: number;
}

const MAX_ENTRIES = 200;
let entries: LogEntry[] = [];
type Listener = (entries: LogEntry[]) => void;
const listeners = new Set<Listener>();

export function addLogEntry(e: Omit<LogEntry, "id">) {
  const entry: LogEntry = { ...e, id: crypto.randomUUID() };
  entries = [entry, ...entries].slice(0, MAX_ENTRIES);
  listeners.forEach((l) => l(entries));
}

export function getLogEntries(): LogEntry[] {
  return entries;
}

export function clearLog() {
  entries = [];
  listeners.forEach((l) => l(entries));
}

export function subscribeLog(listener: Listener): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}
