// The Studio's own choices (light or dark, the editor, the terminal), saved
// in studio-prefs.json.
import { Prefs, SavePrefs } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
import type { Prefs as P } from "../bindings/github.com/gabriellsalesx/blup/studio/models";

export const look = $state<P>({ theme: "system", editorFont: 16, editorWidth: "narrow", spellcheck: true, previewOff: false, termFont: 13 });

function apply() {
  const root = document.documentElement;
  if (look.theme === "system") delete root.dataset.theme;
  else root.dataset.theme = look.theme;
}

export async function loadTheme() {
  try { Object.assign(look, await Prefs()); } catch {}
  look.theme ||= "system";
  apply();
}

export function setPref<K extends keyof P>(key: K, value: P[K]) {
  look[key] = value;
  apply();
  SavePrefs({ ...look }).catch(() => {});
}

export const setTheme = (theme: string) => setPref("theme", theme);
