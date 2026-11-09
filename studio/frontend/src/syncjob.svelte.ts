// The sync of the notes as one job for the whole window, like the site's
// translation (translatejob): it goes on while the person opens notes and
// other screens, the side strip shows how far it went, and the Sync screen
// picks it up (and how it ended) when it opens again. Never two at once.
// It runs in Go (StartSync) and is heard by its events.

import { Events } from "@wailsio/runtime";
import { StartSync, SyncState } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
import type { SyncDone } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
import { toast, toastError } from "./toast.svelte";
import { t } from "./i18n";

export const syncJob = $state({
  running: false,
  phase: "",
  done: 0,
  total: 0,
  /** Just finished well, shown with a check for a moment. */
  finished: false,
  /** How the last one ended, for a Sync screen opened after it. */
  last: null as SyncDone | null,
});

// The Sync screen hears each end, to show the result or the error.
type Heard = (end: SyncDone) => void;
const listeners = new Set<Heard>();
/** Hears each end of a sync; returns the function that stops hearing. */
export function onSyncDone(fn: Heard): () => void {
  listeners.add(fn);
  return () => listeners.delete(fn);
}

let finishedTimer: ReturnType<typeof setTimeout> | undefined;
let waiting: ((end: SyncDone) => void)[] = [];

SyncState().then((st) => { if (st.running) Object.assign(syncJob, { running: true, phase: st.phase, done: st.done, total: st.total }); }).catch(() => {});
Events.On("syncProgress", (ev) => {
  const d = ev.data;
  if (d) Object.assign(syncJob, { running: true, phase: d.phase, done: d.done, total: d.total });
});
Events.On("syncDone", (ev) => {
  const end = ev.data;
  if (!end) return;
  Object.assign(syncJob, { running: false, phase: "", done: 0, total: 0, last: end });
  if (!end.error) {
    syncJob.finished = true;
    clearTimeout(finishedTimer);
    finishedTimer = setTimeout(() => (syncJob.finished = false), 8000);
    // "Compare from scratch" says it in its own words.
    if (!end.fresh) toast(t.toast.synced);
  } else if (!end.stuck && !listeners.size) {
    // No Sync screen open to show it: a notice instead.
    toastError(end.error);
  }
  listeners.forEach((fn) => fn(end));
  const w = waiting;
  waiting = [];
  w.forEach((fn) => fn(end));
});

/**
 * Starts syncing (fresh: "Compare from scratch") and resolves with how it
 * ended; while one runs, resolves with that one's end.
 */
export async function startSync(fresh = false): Promise<SyncDone> {
  const end = new Promise<SyncDone>((ok) => waiting.push(ok));
  if (!syncJob.running) {
    Object.assign(syncJob, { running: true, phase: "", done: 0, total: 0, finished: false, last: null });
    try {
      await StartSync(fresh);
    } catch (e) {
      Object.assign(syncJob, { running: false });
      waiting = [];
      throw e;
    }
  }
  return end;
}

/** The text of a step, for the screen and the side strip. */
export function syncStepText(phase: string, done: number, total: number): string {
  const p = t.pubs;
  const n = (s: string) => s.replace("{done}", String(done)).replace("{total}", String(total));
  switch (phase) {
    case "scan": return p.stepScan;
    case "compare": return p.stepCompare;
    case "upload": return n(p.stepUpload);
    case "download": return n(p.stepDownload);
    case "commit": return p.stepCommit;
    case "retry": return p.stepRetry;
    default: return p.syncing;
  }
}
