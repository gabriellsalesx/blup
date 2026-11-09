// The site's publish as one job for the whole window, like the sync and
// the translation: it goes on while the person opens notes and other
// screens, the side strip shows it, and the Publish screen picks up its
// lines (and how it ended) when it opens again. It runs in Go
// (StartPublish) and is heard by its events.

import { Events } from "@wailsio/runtime";
import { StartPublish, PublishState } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
import type { Progress, PublishDone } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
import { toast } from "./toast.svelte";
import { t } from "./i18n";

export const publishJob = $state({
  running: false,
  lines: [] as Progress[],
  /** Sending the files to Blup Cloud: done of total. */
  done: 0,
  total: 0,
  /** Just published, shown with a check for a moment. */
  finished: false,
  /** How the last one ended, for a Publish screen opened after it. */
  last: null as PublishDone | null,
});

let finishedTimer: ReturnType<typeof setTimeout> | undefined;

PublishState().then((st) => { if (st.running) Object.assign(publishJob, { running: true, lines: st.lines ?? [], done: st.done, total: st.total }); }).catch(() => {});
Events.On("publishCount", (ev) => { if (ev.data) Object.assign(publishJob, { done: ev.data.done, total: ev.data.total }); });
Events.On("publishLine", (ev) => { if (ev.data) publishJob.lines.push(ev.data); });
Events.On("publishDone", (ev) => {
  const end = ev.data;
  if (!end) return;
  Object.assign(publishJob, { running: false, last: end, done: 0, total: 0 });
  if (end.error) {
    toast(t.toast.publishFailed, { kind: "error" });
    return;
  }
  publishJob.finished = true;
  clearTimeout(finishedTimer);
  finishedTimer = setTimeout(() => (publishJob.finished = false), 8000);
  toast(t.toast.published);
});

/** Starts publishing the open site (one at a time). */
export async function startPublish() {
  if (publishJob.running) return;
  Object.assign(publishJob, { running: true, lines: [], finished: false, last: null, done: 0, total: 0 });
  try {
    await StartPublish();
  } catch (e) {
    publishJob.running = false;
    throw e;
  }
}

/** The last line, for the side strip. */
export function publishStep(): string {
  if (publishJob.total) return t.pubs.uploadCount.replace("{done}", publishJob.done.toLocaleString()).replace("{total}", publishJob.total.toLocaleString());
  return publishJob.lines.at(-1)?.text ?? t.pubs.publishing;
}
