// A field of a database's record saves what was typed, in its kind.
import { test } from "node:test";
import assert from "node:assert/strict";
import { typedValue } from "../src/fieldvalue.ts";

test("a number column keeps a number, only when it changed", () => {
  assert.equal(typedValue("number", "42", ""), 42);
  assert.equal(typedValue("number", "4.50", "4"), 4.5);
  assert.equal(typedValue("number", "42", "42"), undefined);
  assert.equal(typedValue("number", "1e400", ""), undefined);
  assert.equal(typedValue("number", "abc", "1"), undefined);
  assert.equal(typedValue("number", "", "5"), "");
});

test("other columns keep the text, only when it changed", () => {
  assert.equal(typedValue("text", "oi", ""), "oi");
  assert.equal(typedValue("text", "oi", "oi"), undefined);
  assert.equal(typedValue("email", "", "a@b.c"), "");
});
