// A version's notes (Markdown from the GitHub release, in the Studio's
// language: app.UpdateInfo.changes) as plain blocks the update notice
// draws: headings, list items and paragraphs, with **bold** and `code`.
// Never HTML: Svelte writes the text as text, so nothing in the notes can
// become markup or a script.

export type Span = { text: string; bold?: boolean; code?: boolean };
export type Block = { kind: "heading" | "item" | "para"; spans: Span[] };

const INLINE = /(\*\*[^*]+\*\*|`[^`]+`)/;

function spans(text: string): Span[] {
  return text
    .split(INLINE)
    .filter((part) => part !== "")
    .map((part) => {
      if (part.startsWith("**") && part.endsWith("**") && part.length > 4) return { text: part.slice(2, -2), bold: true };
      if (part.startsWith("`") && part.endsWith("`") && part.length > 2) return { text: part.slice(1, -1), code: true };
      return { text: part };
    });
}

/** Turns the notes into blocks; lines of a paragraph join with a space. */
export function readNotes(notes: string): Block[] {
  const out: Block[] = [];
  let para: string[] = [];
  const flush = () => {
    if (para.length) out.push({ kind: "para", spans: spans(para.join(" ")) });
    para = [];
  };
  for (const raw of notes.split(/\r?\n/)) {
    const line = raw.trim();
    const heading = /^#{1,6}\s+(.*)$/.exec(line);
    const item = /^[-*+]\s+(.*)$/.exec(line) ?? /^\d+[.)]\s+(.*)$/.exec(line);
    if (!line) flush();
    else if (heading) { flush(); out.push({ kind: "heading", spans: spans(heading[1]) }); }
    else if (item) { flush(); out.push({ kind: "item", spans: spans(item[1]) }); }
    else para.push(line);
  }
  flush();
  return out;
}
