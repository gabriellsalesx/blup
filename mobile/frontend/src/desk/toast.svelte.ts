// The Studio's toasts, on the phone's.
import { app, message } from "../lib/app.svelte";

export function toast(text: string, opts: { action?: { label: string; run: () => void } } = {}) { app.say({ text, action: opts.action }); }
export function toastError(e: unknown) { app.say({ text: message(e) }); }
