// The narration of every post of the blog as one job for the whole window,
// like the translation (translatejob): it goes on while the person opens
// notes and other screens, the side strip shows how far it went, and the
// Podcast screen picks it up again when it opens. It runs in Go
// (StartNarrateSite) and is heard by its events.

import { Events } from "@wailsio/runtime";
import { StartNarrateSite, NarrateSiteState, CancelNarrateSite } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
import { toast, toastError } from "./toast.svelte";
import { withAIConsent, ConsentCancelled } from "./aiconsentui";
import { t } from "./i18n";

export const narrateJob = $state({
  running: false,
  done: 0,
  total: 0,
  note: "",
  /** Just finished, shown with a check for a moment. */
  finished: false,
});

// The Podcast screen hears the end, to read its counts again.
type Heard = () => void;
const listeners = new Set<Heard>();
/** Hears the end of a batch; returns the function that stops hearing it. */
export function onNarrateEnd(fn: Heard): () => void {
  listeners.add(fn);
  return () => listeners.delete(fn);
}

// Each note the batch went past (narrated or not), so an open note shows
// its new narration.
type HeardNote = (rel: string) => void;
const noteListeners = new Set<HeardNote>();
/** Hears each note the batch finished; returns the function that stops. */
export function onNoteNarrated(fn: HeardNote): () => void {
  noteListeners.add(fn);
  return () => noteListeners.delete(fn);
}
const passed = (rel: string) => noteListeners.forEach((fn) => fn(rel));

const noteName = (p: string) => p.split("/").pop()?.replace(/\.md$/i, "") ?? "";
let finishedTimer: ReturnType<typeof setTimeout> | undefined;

// A batch already running in Go when the window (re)loads.
NarrateSiteState().then((st) => { if (st.running) Object.assign(narrateJob, { running: true, done: st.done, total: st.total, note: st.note }); }).catch(() => {});
Events.On("narrateSite", (ev) => {
  const d = ev.data;
  if (!d) return;
  // The note read before this step is done: it now has its narration.
  if (narrateJob.note && d.done > narrateJob.done) passed(narrateJob.note);
  Object.assign(narrateJob, { running: true, done: d.done, total: d.total, note: d.note });
});
Events.On("narrateSiteDone", (ev) => {
  const d = ev.data;
  if (!d) return;
  if (narrateJob.note) passed(narrateJob.note);
  Object.assign(narrateJob, { running: false, done: 0, total: 0, note: "" });
  listeners.forEach((fn) => fn());
  const a = t.narrAll;
  if (d.cancelled) toast(a.cancelled, { kind: "info" });
  else if (d.error) toastError(d.error);
  if (d.done) {
    narrateJob.finished = true;
    clearTimeout(finishedTimer);
    finishedTimer = setTimeout(() => (narrateJob.finished = false), 8000);
    toast(a.finished.replace("{n}", String(d.done)));
  }
  const failed = d.failed ?? [];
  if (failed.length) toast(a.failed.replace("{n}", String(failed.length)).replace("{names}", failed.map(noteName).join(", ")), { kind: "error" });
});

/** Starts narrating the blog's posts (one batch at a time). */
export async function startNarrateSite() {
  if (narrateJob.running) return;
  Object.assign(narrateJob, { running: true, done: 0, total: 0, note: "", finished: false });
  try {
    await withAIConsent(() => StartNarrateSite(), "narrate"); // asks before the text first goes to the cloud
    const st = await NarrateSiteState();
    // Running: its size now (the first step's event may come later).
    // Not running and nothing heard: there was nothing to narrate.
    if (st.running) Object.assign(narrateJob, { total: st.total || narrateJob.total, done: st.done, note: st.note || narrateJob.note });
    else if (narrateJob.total === 0) narrateJob.running = false;
  } catch (e) {
    narrateJob.running = false;
    if (e instanceof ConsentCancelled) return;
    toast(String(e).includes("no OpenRouter key") ? t.narration.needKey : String(e).replace(/^(Runtime)?Error: /, ""), { kind: "error" });
  }
}

/** Stops the batch after the note being read. */
export const cancelNarrateSite = () => CancelNarrateSite();

/** The batch's progress, 0 to 100. */
export const narratePercent = () => (narrateJob.total ? Math.floor((narrateJob.done / narrateJob.total) * 100) : 0);
