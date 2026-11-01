// What the files panel creates: notes and folders, and in a vault of notes
// canvases and drawings too (the Studio's own).
export type NewKind = "note" | "folder" | "canvas" | "drawing" | "database";
export type BoardKind = "canvas" | "drawing" | "database";

// The kind of a file of the vault: a canvas, a drawing, or "" for a note.
export function boardKind(path: string): BoardKind | "" {
  const p = path.toLowerCase();
  if (p.endsWith(".canvas")) return "canvas";
  if (p.endsWith(".draw")) return "drawing";
  if (p.endsWith(".table")) return "database";
  return "";
}

// A file's name without its extension.
export const fileTitle = (path: string) => (path.split("/").pop() ?? "").replace(/\.(draw|canvas|table|md)$/i, "");
