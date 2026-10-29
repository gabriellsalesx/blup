// The Studio's keyboard shortcuts: what each one does, its default keys,
// and the keys of a key press in the same form ("Ctrl+Shift+F"), which is
// also how Wails writes the global shortcut.
export type Action = "terminal" | "preview" | "newNote" | "search" | "sidebar" | "settings";

export const ACTIONS: { key: Action; keys: string }[] = [
  { key: "newNote", keys: "Ctrl+N" },
  { key: "search", keys: "Ctrl+Shift+F" },
  { key: "sidebar", keys: "Ctrl+Alt+S" },
  { key: "preview", keys: "Ctrl+Alt+P" },
  { key: "terminal", keys: "Ctrl+`" },
  { key: "settings", keys: "Ctrl+," },
];

const named: Record<string, string> = { " ": "Space", Escape: "Esc", ArrowUp: "Up", ArrowDown: "Down", ArrowLeft: "Left", ArrowRight: "Right" };

// The keys of a press, or "" while only modifiers are down.
export function comboOf(e: KeyboardEvent): string {
  if (["Control", "Shift", "Alt", "Meta", "AltGraph"].includes(e.key)) return "";
  let k = named[e.key] ?? e.key;
  if (/^Key[A-Z]$/.test(e.code)) k = e.code.slice(3); // a letter, whatever the layout or Shift
  else if (/^Digit\d$/.test(e.code)) k = e.code.slice(5);
  else if (e.code === "Backquote") k = "`";
  else if (k.length === 1) k = k.toUpperCase();
  return [e.ctrlKey && "Ctrl", e.altKey && "Alt", e.shiftKey && "Shift", e.metaKey && "Super", k].filter(Boolean).join("+");
}

// The keys of an action: the user's, else the default ("" = turned off).
export function keysOf(saved: Record<string, string | undefined> | null | undefined, a: Action): string {
  return saved && a in saved ? saved[a] ?? "" : ACTIONS.find((x) => x.key === a)!.keys;
}
