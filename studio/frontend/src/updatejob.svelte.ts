// The Studio's own update, downloaded in the window like the local AIs:
// it goes on while Settings is closed, and Settings shows how far it went
// when it opens again. Go downloads and checks it (DownloadUpdate, heard
// by its "studioUpdate" events) and puts it in place on restart
// (InstallUpdate).

import { Events } from "@wailsio/runtime";
import { DownloadUpdate, InstallUpdate } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
import { toast } from "./toast.svelte";
import { t } from "./i18n";

type Phase = "idle" | "downloading" | "ready" | "installing" | "failed";

export const updateJob = $state({
  phase: "idle" as Phase,
  done: 0,
  total: 0,
  /** The version downloaded, ready to install. */
  version: "",
  error: "",
});

Events.On("studioUpdate", (ev) => {
  if (ev.data && updateJob.phase === "downloading") Object.assign(updateJob, { done: ev.data.done, total: ev.data.total });
});

const message = (e: unknown) => (e instanceof Error ? e.message : String(e));

/** Downloads and checks the new version (one at a time). */
export async function downloadUpdate() {
  if (updateJob.phase === "downloading" || updateJob.phase === "installing") return;
  Object.assign(updateJob, { phase: "downloading", done: 0, total: 0, version: "", error: "" });
  try {
    const version = await DownloadUpdate();
    Object.assign(updateJob, { phase: "ready", version });
    toast(t.prefs.updateReady.replace("{v}", version));
  } catch (e) {
    Object.assign(updateJob, { phase: "failed", error: message(e) });
  }
}

/** Puts the new version in place and restarts the Studio. */
export async function installUpdate() {
  if (updateJob.phase !== "ready") return;
  updateJob.phase = "installing";
  try {
    await InstallUpdate();
  } catch (e) {
    Object.assign(updateJob, { phase: "failed", error: message(e) });
  }
}

/** How much came, from 0 to 100 (0 while the size is unknown). */
export const updatePercent = () => (updateJob.total ? Math.min(100, Math.round((updateJob.done / updateJob.total) * 100)) : 0);
