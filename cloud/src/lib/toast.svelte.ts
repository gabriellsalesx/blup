// Short notices at the bottom of the page, the same as Blup Studio's:
// every screen tells what just happened through toast().
type Toast = { id: number; text: string|(()=>string); kind: "ok" | "error" | "info"; action?: { label: string; run: () => void } };

export const toasts = $state<Toast[]>([]);
let next = 0;

export function toast(text: string|(()=>string), opts: { kind?: Toast["kind"]; action?: Toast["action"]; ms?: number } = {}) {
	const id = ++next;
	toasts.push({ id, text, kind: opts.kind ?? "ok", action: opts.action });
	if (toasts.length > 4) toasts.shift();
	setTimeout(() => dismiss(id), opts.ms ?? (opts.action ? 7000 : opts.kind === "error" ? 6000 : 3500));
}

export function dismiss(id: number) {
	const i = toasts.findIndex((t) => t.id === id);
	if (i >= 0) toasts.splice(i, 1);
}
