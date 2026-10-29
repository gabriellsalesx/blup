// Light, dark or the system's: the Studio's look, saved in its choices.
import { Prefs, SavePrefs } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";

export const look = $state({ theme: "system" });

function apply() {
  const root = document.documentElement;
  if (look.theme === "system") delete root.dataset.theme;
  else root.dataset.theme = look.theme;
}

export async function loadTheme() {
  try { look.theme = (await Prefs()).theme || "system"; } catch {}
  apply();
}

export function setTheme(theme: string) {
  look.theme = theme;
  apply();
  SavePrefs({ theme }).catch(() => {});
}
