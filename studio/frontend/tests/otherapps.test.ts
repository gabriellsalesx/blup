// Blup Studio's texts speak of its own notes: no other note app is named
// (a rule of the product; only the command line tool may name one).
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

test("the Studio's texts name no other note app", () => {
  const src = readFileSync(new URL("../src/i18n.ts", import.meta.url), "utf8");
  assert.doesNotMatch(src, /obsidian/i);
});
