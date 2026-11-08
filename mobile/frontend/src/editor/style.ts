import { HighlightStyle } from "@codemirror/language";
import { tags } from "@lezer/highlight";

// The Markdown colors of the Blup Studio.
export const markdownStyle = HighlightStyle.define([
  { tag: tags.heading1, fontWeight: "800", fontSize: "1.5em" },
  { tag: tags.heading2, fontWeight: "700", fontSize: "1.3em" },
  { tag: tags.heading3, fontWeight: "700", fontSize: "1.15em" },
  { tag: [tags.heading4, tags.heading5, tags.heading6], fontWeight: "700" },
  { tag: tags.strong, fontWeight: "700" },
  { tag: tags.emphasis, fontStyle: "italic" },
  { tag: tags.strikethrough, textDecoration: "line-through" },
  { tag: [tags.link, tags.url], color: "var(--blue)" },
  { tag: tags.monospace, fontFamily: "ui-monospace, monospace", fontSize: ".9em", backgroundColor: "var(--soft)", borderRadius: "4px" },
  { tag: [tags.processingInstruction, tags.meta, tags.contentSeparator], color: "var(--muted)" },
  { tag: tags.quote, color: "var(--muted)", fontStyle: "italic" },
]);
