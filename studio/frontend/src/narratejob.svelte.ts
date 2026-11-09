// The narration of every post of the blog as one job for the whole window,
// like the translation (translatejob): it goes on while the person opens
// notes and other screens, the side strip shows how far it went, and the
// Podcast screen picks it up again when it opens. It runs in Go
// (StartNarrateSite) and is heard by its events. The batch belongs to its
// site (site: the folder): the screens show it only while that site is
// open.

import { Events } from "@wailsio/runtime";
import { StartNarrateSite, NarrateSiteState, CancelNarrateSite } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
import { toast, toastError } from "./toast.svelte";
import { withAIConsent, ConsentCancelled } from "./aiconsentui";
import { t } from "./i18n";

export const narrateJob = $state({
  running: false,
  /** Asked or checking (consent, notes, voice, cost): nothing to stop yet. */
  preparing: false,
  site: "",
  done: 0,
  total: 0,
  note: "",
  /** Just finished, shown with a check for a moment. */
  finished: false,
});

// The Podcast screen hears the end, to read its counts again.
type Heard = () => void;
const listeners: Heard[] = [];
/** Hears the end of a batch; returns the function that stops hearing it. */
export function onNarrateEnd(fn: Heard): () => void {
  listeners.push(fn);
  return () => drop(listeners, fn);
}
const drop = <T>(list: T[], fn: T) => { const i = list.indexOf(fn); if (i >= 0) list.splice(i, 1); };

// Each note the batch went past (narrated or not), so an open note shows
// its new narration.
type HeardNote = (rel: string, site: string) => void;
const noteListeners: HeardNote[] = [];
/** Hears each note the batch finished; returns the function that stops. */
export function onNoteNarrated(fn: HeardNote): () => void {
  noteListeners.push(fn);
  return () => drop(noteListeners, fn);
}
const passed = (rel: string) => noteListeners.forEach((fn) => fn(rel, narrateJob.site));

const noteName = (p: string) => p.split("/").pop()?.replace(/\.md$/i, "") ?? "";
let finishedTimer: ReturnType<typeof setTimeout> | undefined;

// A batch already running in Go when the window (re)loads.
NarrateSiteState().then((st) => { if (st.running) Object.assign(narrateJob, { running: true, preparing: st.preparing, site: st.site, done: st.done, total: st.total, note: st.note }); }).catch(() => {});
Events.On("narrateSite", (ev) => {
  const d = ev.data;
  if (!d) return;
  // The note read before this step is done: it now has its narration.
  if (narrateJob.note && d.done > narrateJob.done) passed(narrateJob.note);
  Object.assign(narrateJob, { running: true, preparing: false, site: d.site, done: d.done, total: d.total, note: d.note });
});
Events.On("narrateSiteDone", (ev) => {
  const d = ev.data;
  if (!d) return;
  if (narrateJob.note) passed(narrateJob.note);
  Object.assign(narrateJob, { running: false, preparing: false, done: 0, total: 0, note: "", site: d.site });
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
  if (d.skipped) toast(a.skipped.replace("{n}", String(d.skipped)), { kind: "info" });
  const failed = d.failed ?? [];
  if (failed.length) {
    const list = failed.map((f) => `${noteName(f.rel)} (${f.error})`).join("; ");
    toast(a.failed.replace("{n}", String(failed.length)).replace("{list}", list), { kind: "error", ms: 15000 });
  }
});

/** Starts narrating the blog's posts of the site at dir (one batch at a time). */
export async function startNarrateSite(dir: string) {
  if (narrateJob.running) return;
  Object.assign(narrateJob, { running: true, preparing: true, site: dir, done: 0, total: 0, note: "", finished: false });
  try {
    await withAIConsent(() => StartNarrateSite(), "narrate"); // asks before the text first goes to the cloud
    const st = await NarrateSiteState();
    // Running: its size now (the first step's event may come later). Not
    // running and nothing heard: there was nothing to narrate.
    if (st.running) Object.assign(narrateJob, { preparing: st.preparing, site: st.site, total: st.total || narrateJob.total, done: st.done, note: st.note || narrateJob.note });
    else if (narrateJob.total === 0) Object.assign(narrateJob, { running: false, preparing: false });
  } catch (e) {
    Object.assign(narrateJob, { running: false, preparing: false });
    if (e instanceof ConsentCancelled) return;
    toast(String(e).includes("no OpenRouter key") ? t.narration.needKey : String(e).replace(/^(Runtime)?Error: /, ""), { kind: "error" });
  }
}

/** Stops the batch now (the note being read is left as it was). */
export const cancelNarrateSite = () => CancelNarrateSite();

/** The batch's progress, 0 to 100. */
export const narratePercent = () => (narrateJob.total ? Math.floor((narrateJob.done / narrateJob.total) * 100) : 0);

/** Whether the batch belongs to the site at dir. */
export const narratesSite = (dir: string) => !!dir && narrateJob.site === dir;
