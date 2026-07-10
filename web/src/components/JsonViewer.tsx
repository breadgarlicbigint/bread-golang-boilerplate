import { useState } from "react";

export function JsonViewer({ value }: { value: unknown }) {
  const [copied, setCopied] = useState(false);
  const text = typeof value === "string" ? value : JSON.stringify(value, null, 2);

  const copy = async () => {
    await navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  return (
    <div className="relative">
      <button type="button" onClick={copy} className="btn-secondary absolute right-2 top-2 !px-2 !py-1 text-xs">
        {copied ? "Copied" : "Copy"}
      </button>
      <pre className="max-h-96 overflow-auto whitespace-pre-wrap break-all rounded-md bg-slate-900 p-3 pr-16 text-xs text-slate-100">
        {text}
      </pre>
    </div>
  );
}
