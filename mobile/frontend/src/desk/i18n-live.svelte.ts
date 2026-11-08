// The interface's texts as state (i18n.ts): a new language changes them in
// place, and every screen showing them follows, with no reload.

export function live<T extends object>(texts: T): T {
  const s = $state(texts);
  return s;
}

// The language shown, for what builds its texts once (the editor's menus).
export const shown = $state({ lang: "" });
