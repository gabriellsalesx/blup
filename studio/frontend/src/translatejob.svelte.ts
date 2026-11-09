// The site's translation as one job for the whole window: it goes on while
// the person opens notes and other screens, the side strip shows how far
// it went, and the Languages screen picks it up again when it opens. Never
// two at once: a second click only shows the one running.
//
// In the cloud the job runs in Go (StartTranslateSite) and is heard by its
// events; on the computer it runs here, note by note in the window.

import { Events } from "@wailsio/runtime";
import { StartTranslateSite, TranslateSiteState, TranslationTodo, FollowTranslations } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
import { toast, toastError } from "./toast.svelte";
import { t } from "./i18n";
import { useLocal, translateNoteLocal } from "./localtranslate";
import { withAIConsent, ConsentCancelled, needsProvider } from "./aiconsentui";

export const LANG_NAMES: Record<string, string> = { pt: "Português", en: "English", es: "Español" };

export const job = $state({
  running: false,
  lang: "",
  done: 0,
  total: 0,
  note: "",
  /** The language just finished, shown with a check for a moment. */
  finished: "",
});

// The screens that list translations hear each note done ("progress")
// and the end ("ended"), to load their counts again.
type Heard = (what: "progress" | "ended") => void;
const listeners = new Set<Heard>();
/** Hears the job; returns the function that stops hearing it. */
export function onJob(fn: Heard): () => void {
  listeners.add(fn);
  return () => listeners.delete(fn);
}
const tell = (what: "progress" | "ended") => listeners.forEach((fn) => fn(what));

const noteName = (p: string) => p.split("/").pop()?.replace(/\.md$/i, "") ?? "";
let finishedTimer: ReturnType<typeof setTimeout> | undefined;

function finish(lang: string, done: number, skipped: number, failed: string[], error: string) {
  Object.assign(job, { running: false, done: 0, total: 0, note: "" });
  tell("ended");
  if (error) {
    toastError(error);
    return;
  }
  job.finished = lang;
  clearTimeout(finishedTimer);
  finishedTimer = setTimeout(() => (job.finished = ""), 8000);
  const sl = t.sitelangs;
  toast(sl.finished.replace("{n}", String(done)).replace("{lang}", LANG_NAMES[lang] ?? lang));
  if (skipped) toast(t.aip.skipped.replace("{n}", String(skipped)));
  if (failed.length) toast(sl.failed.replace("{n}", String(failed.length)).replace("{names}", failed.map(noteName).join(", ")), { kind: "error" });
}

// A job already running in Go when the window (re)loads.
TranslateSiteState().then((st) => { if (st.running) Object.assign(job, { running: true, lang: st.lang, done: st.done, total: st.total, note: st.note }); }).catch(() => {});
Events.On("translateSite", (ev) => {
  const d = ev.data;
  if (d) {
    Object.assign(job, { running: true, lang: d.lang, done: d.done, total: d.total, note: d.note });
    tell("progress");
  }
});
Events.On("translateSiteDone", (ev) => {
  const d = ev.data;
  if (d) finish(d.lang, d.done, d.skipped, d.failed ?? [], d.error);
});

/** Translates what the site misses in lang; siteLang is its main language. */
export async function translateSite(lang: string, siteLang: string) {
  if (job.running) return;
  job.finished = "";
  try {
    if (await useLocal()) {
      // On the computer: note by note in the window.
      Object.assign(job, { running: true, lang, done: 0, total: 0, note: "" });
      const todo = (await TranslationTodo(lang)) ?? [];
      const failed: string[] = [];
      let n = 0;
      for (const [i, rel] of todo.entries()) {
        Object.assign(job, { done: i, total: todo.length, note: rel });
        tell("progress");
        try {
          await translateNoteLocal(rel, siteLang, lang, undefined, false);
          n++;
        } catch {
          failed.push(rel);
        }
      }
      await FollowTranslations();
      finish(lang, n, 0, failed, "");
      return;
    }
    // In the cloud: Go runs it and tells (events above).
    Object.assign(job, { running: true, lang, done: 0, total: 0, note: "" });
    await withAIConsent(() => StartTranslateSite(lang), "translate");
  } catch (e) {
    Object.assign(job, { running: false, done: 0, total: 0, note: "" });
    if (e instanceof ConsentCancelled) return;
    if (needsProvider(e)) toast(t.aip.chooseWho, { kind: "error" });
    else toastError(e);
  }
}
