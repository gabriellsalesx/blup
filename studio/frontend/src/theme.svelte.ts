// The Studio's own choices (light or dark, the editor, the terminal), saved
// in studio-prefs.json.
import { Prefs, SavePrefs } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
import type { Prefs as P } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
import { setupSpell, enableSpell } from "./spell";
import { t } from "./i18n";
import { wordCount, wordRemoved } from "./prefwords";

export const look = $state<P>({ theme: "system", editorFont: 16, editorWidth: "narrow", spellcheck: true, previewOff: false, termFont: 13, spellLang: "", words: [], sideWidth: 0, previewWidth: 0, tray: false, toggleKey: "Alt+Space", keys: {}, shortcutTip: false, lastSite: "", lastNotes: {}, recent: [], voices: {}, welcomed: false, cloudTip: false, backupDir: "", translateAI: "", railNames: false, notifications: true, errorReports: "" });

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

// The choices as Go has them: version goes up after each save reaches
// the file, and when what can narrate changes (a key connected, the local
// voice installed). Screens that ask Go what the choices give (the
// narration's voices, what it costs) read again when it changes.
export const prefs = $state({ version: 0 });
/** Tells the screens that follow prefs.version to read again. */
export const prefsChanged = () => { prefs.version++; };

// The saves go one after the other, each with the choices as they were
// when it was asked: a later one never lands before an earlier one.
let saving: Promise<void> = Promise.resolve();

/** Changes a choice; resolves once it is saved (and prefs.version went up). */
export function setPref<K extends keyof P>(key: K, value: P[K]): Promise<void> {
  look[key] = value;
  apply();
  if (key === "words") {
    const words = value as P["words"];
    if (wordRemoved(words, spellWords)) spell();
    spellWords = wordCount(words);
  } else if (key === "spellLang" || key === "spellcheck") spell();
  const now: P = { ...look };
  saving = saving.then(() => SavePrefs(now)).catch(() => {}).then(prefsChanged);
  return saving;
}

export const setTheme = (theme: string) => setPref("theme", theme);
