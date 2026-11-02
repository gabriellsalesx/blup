// The graph's choices read back from localStorage: only what has the kind
// of its default. Run: npm test (node:test, no deps).
import { test } from "node:test";
import assert from "node:assert/strict";
import { readPrefs } from "../src/graph.ts";

const DEFAULTS = { query: "", tagNodes: false, depth: 1, groups: [] as { match: string; color: string }[], open: { filters: true } as Record<string, boolean> };

test("graph prefs: nothing or broken JSON gives the defaults", () => {
  for (const raw of [null, "", "{", "null", "[]", "5", '"x"']) assert.deepEqual(readPrefs(DEFAULTS, raw), DEFAULTS, String(raw));
});

test("graph prefs: values of the wrong kind fall back", () => {
  const p = readPrefs(DEFAULTS, JSON.stringify({ query: 3, tagNodes: "yes", depth: "2", groups: "x", open: [1] }));
  assert.deepEqual(p, DEFAULTS);
  // JSON's numbers too big for a double are Infinity.
  assert.equal(readPrefs(DEFAULTS, '{"depth":1e400}').depth, 1);
});

test("graph prefs: good values are kept, bad items dropped, unknown keys ignored", () => {
  const p = readPrefs(DEFAULTS, JSON.stringify({
    query: "a", tagNodes: true, depth: 3, extra: 1,
    groups: [{ match: "tag:x", color: "#fff", more: 1 }, { match: 1, color: "#000" }, null, "g"],
    open: { filters: false, forces: true, bad: "x" },
  }));
  assert.deepEqual(p, { query: "a", tagNodes: true, depth: 3, groups: [{ match: "tag:x", color: "#fff" }], open: { filters: false, forces: true } });
  assert.equal("extra" in p, false);
});

test("graph prefs: the defaults are not shared with the result", () => {
  const p = readPrefs(DEFAULTS, null);
  p.groups.push({ match: "a", color: "b" });
  assert.equal(DEFAULTS.groups.length, 0);
});
