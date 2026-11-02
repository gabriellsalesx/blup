// The local translation's worker answers are checked before use.
import { test } from "node:test";
import assert from "node:assert/strict";
import { translateReply } from "../src/translatereply.ts";

test("answers of the worker are read", () => {
  assert.deepEqual(translateReply({ id: 3, out: ["Olá", ""] }), { id: 3, out: ["Olá", ""] });
  assert.deepEqual(translateReply({ id: 3, error: "no model" }), { id: 3, error: "no model" });
});

test("anything else is not an answer", () => {
  for (const d of [null, "x", {}, { id: "3", out: [] }, { id: 1.5, out: [] }, { id: 0, out: [] }, { id: 3 }, { id: 3, out: "texto" }, { id: 3, out: ["a", 2] }, { id: 3, error: { m: 1 } }])
    assert.equal(translateReply(d), null, JSON.stringify(d));
});
