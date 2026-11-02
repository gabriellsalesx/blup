// The Studio's own choices (light or dark, the editor, the terminal), saved
// in studio-prefs.json.
import { Prefs, SavePrefs } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
import type { Prefs as P } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
import { setupSpell, enableSpell } from "./spell";
import { t } from "./i18n";

export const look = $state<P>({ theme: "system", editorFont: 16, editorWidth: "narrow", spellcheck: true, previewOff: false, termFont: 13, spellLang: "", words: [], sideWidth: 0, previewWidth: 0, tray: false, toggleKey: "Alt+Space", keys: {}, shortcutTip: false, lastSite: "", lastNotes: {}, recent: [], voices: {}, welcomed: false, cloudTip: false, backupDir: "", translateAI: "", railNames: false, notifications: true });

// The spell check's dictionary: the one chosen, else the Studio's language.
export const spellLang = () => look.spellLang || t.lang;
let spellWords: number | undefined;
// Also run when the Studio's language changes (its dictionary may follow).
export const spell = () => { setupSpell(spellLang(), look.words ?? []); enableSpell(look.spellcheck); };

// Whether the Studio is dark now (chosen, or the system's), for what draws
// its own colors (the drawings).
const media = matchMedia("(prefers-color-scheme: dark)");
export const scheme = $state({ dark: media.matches });

function apply() {
  const root = document.documentElement;
  if (look.theme === "system") delete root.dataset.theme;
  else root.dataset.theme = look.theme;
  scheme.dark = look.theme === "dark" || (look.theme !== "light" && media.matches);
}
media.addEventListener("change", apply);

export async function loadTheme() {
  try { Object.assign(look, await Prefs()); } catch {}
  look.theme ||= "system";
  apply();
  spell();
}

export function setPref<K extends keyof P>(key: K, value: P[K]) {
  look[key] = value;
  apply();
  // A word taken out of the list needs a fresh dictionary; one added is
  // already in it (spell.ts addWord).
  if (key === "spellLang" || key === "spellcheck" || (key === "words" && (value as string[]).length < (spellWords ?? 0))) spell();
  if (key === "words") spellWords = (value as string[]).length;
  SavePrefs({ ...look }).catch(() => {});
}

export const setTheme = (theme: string) => setPref("theme", theme);
