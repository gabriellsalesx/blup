// Own domain on Blup Cloud: status colors and the spaced polling.
import { test } from "node:test";
import assert from "node:assert/strict";
import { statusKind, POLL_MS } from "../src/blupdomain.ts";

test("each status of the contract has its color", () => {
  assert.equal(statusKind("active"), "good");
  assert.equal(statusKind("pending_ownership"), "warn");
  assert.equal(statusKind("pending"), "warn");
  assert.equal(statusKind("error"), "bad");
  assert.equal(statusKind("<img src=x>"), "muted");
  assert.equal(statusKind(undefined), "muted");
});

test("polling is spaced (a minute at least)", () => {
  assert.ok(POLL_MS >= 60_000);
});
