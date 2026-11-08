// Blup Studio's texts speak of its own notes: no other note app is named
// (a rule of the product; only the command line tool may name one).
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

test("the Studio's texts name no other note app", () => {
  const src = ["i18n.ts", "i18n-en.ts", "i18n-es.ts"].map((f) => readFileSync(new URL(`../src/${f}`, import.meta.url), "utf8")).join("\n");
  assert.doesNotMatch(src, /obsidian/i);
});
