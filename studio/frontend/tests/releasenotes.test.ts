// The notes of a version as the update notice draws them: headings, items
// and paragraphs, bold and code; never HTML.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readNotes } from "../src/releasenotes.ts";

test("headings, items and paragraphs", () => {
  const blocks = readNotes("### Blup Notes (celular)\n- Pronto para a **Google Play**.\n* Use `blup publish`.\n\nUm parágrafo\nem duas linhas.");
  assert.deepEqual(blocks, [
    { kind: "heading", spans: [{ text: "Blup Notes (celular)" }] },
    { kind: "item", spans: [{ text: "Pronto para a " }, { text: "Google Play", bold: true }, { text: "." }] },
    { kind: "item", spans: [{ text: "Use " }, { text: "blup publish", code: true }, { text: "." }] },
    { kind: "para", spans: [{ text: "Um parágrafo em duas linhas." }] },
  ]);
});

test("markup in the notes stays text", () => {
  const [b] = readNotes("- <img src=x onerror=alert(1)> **<b>x</b>**");
  assert.equal(b.kind, "item");
  assert.equal(b.spans[0].text, "<img src=x onerror=alert(1)> ");
  assert.deepEqual(b.spans[1], { text: "<b>x</b>", bold: true });
});
