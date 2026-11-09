// What changed outside a screen, heard from Go: notes on disk (an agent in
// the terminal, another editor, the sync) and the accounts connected
// (GitHub, Cloudflare, OpenRouter). Each returns the function that stops
// hearing, for onDestroy or an $effect's cleanup.

import { Events } from "@wailsio/runtime";

/**
 * Hears the notes that changed on disk, a moment after the last change
 * (wait ms): fn gets every path of the burst (relative to the notes
 * folder, with "/").
 */
export function onNotesChanged(fn: (paths: string[]) => void, wait = 400): () => void {
  let timer: ReturnType<typeof setTimeout> | undefined;
  let paths = new Set<string>();
  const off = Events.On("notesChanged", (ev) => {
    for (const p of ev.data?.paths ?? []) paths.add(p);
    clearTimeout(timer);
    timer = setTimeout(() => {
      const list = [...paths];
      paths = new Set();
      fn(list);
    }, wait);
  });
  return () => {
    off();
    clearTimeout(timer);
  };
}

/** Hears a key connected or forgotten: "github", "cloudflare" or "openrouter". */
export function onConnections(fn: (account: string) => void): () => void {
  return Events.On("connectionsChanged", (ev) => fn(ev.data?.account ?? ""));
}
