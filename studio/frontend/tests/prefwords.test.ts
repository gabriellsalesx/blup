import { test } from "node:test";
import assert from "node:assert/strict";
import { wordCount, wordRemoved } from "../src/prefwords.ts";

test("words from the binding may be null", () => {
  assert.equal(wordCount(null), 0);
  assert.equal(wordCount(undefined), 0);
  assert.equal(wordCount([]), 0);
  assert.equal(wordCount(["a", "b"]), 2);
});

test("a word taken out asks for a fresh dictionary, null included", () => {
  assert.equal(wordRemoved(null, 2), true);
  assert.equal(wordRemoved([], 1), true);
  assert.equal(wordRemoved(["a"], 2), true);
  assert.equal(wordRemoved(["a", "b"], 1), false);
  assert.equal(wordRemoved(null, undefined), false);
  assert.equal(wordRemoved([], 0), false);
});
