// The screens that are not the first one come when they are needed: the app
// opens with only the list of notes, and the editor (CodeMirror), the
// canvas, the drawing, the databases, the calendar and the account load on
// their first use. A module loads once; after that it is at hand at once.
const once = <T>(load: () => Promise<T>) => {
  let p: Promise<T> | undefined;
  return () => (p ??= load().catch((e) => { p = undefined; throw e; }));
};

export const screens = {
  editor: once(() => import("../screens/Editor.svelte")),
  board: once(() => import("../screens/Board.svelte")),
  calendar: once(() => import("../screens/Calendar.svelte")),
  account: once(() => import("../screens/Account.svelte")),
};

// After the list shows, the editor loads while the phone is idle: the first
// note opens as fast as the next ones.
export function warmEditor() {
  const go = () => void screens.editor().catch(() => {});
  if ("requestIdleCallback" in window) requestIdleCallback(go, { timeout: 4000 });
  else setTimeout(go, 1500);
}
