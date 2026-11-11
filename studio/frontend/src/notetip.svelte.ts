// The dates of a note shown while the mouse rests on it in the file list
// (Tree.svelte): when it was last changed and when it was made, read from
// Go only then (NoteDates). One for the whole window (NoteTip.svelte).
import { NoteDates } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";

export const noteTip = $state({ shown: false, x: 0, y: 0, modified: "", created: "" });

const REST_MS = 500;
let timer: ReturnType<typeof setTimeout> | undefined;
let seq = 0;

/** The mouse came to rest on a note's row: its dates show beside it. */
export function showNoteTip(path: string, row: HTMLElement) {
  clearTimeout(timer);
  const mine = ++seq;
  timer = setTimeout(async () => {
    try {
      const d = await NoteDates(path);
      if (mine !== seq || !row.isConnected) return;
      const r = row.getBoundingClientRect();
      Object.assign(noteTip, { shown: true, x: r.right + 8, y: r.top + r.height / 2, modified: d.modified, created: d.created });
    } catch {
      // A note gone meanwhile: nothing to show.
    }
  }, REST_MS);
}

/** The mouse left, clicked or dragged: the dates go. */
export function hideNoteTip() {
  clearTimeout(timer);
  seq++;
  noteTip.shown = false;
}
