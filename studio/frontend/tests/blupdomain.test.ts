// Own domain on Blup Cloud: status colors and the spaced polling.
import { test } from "node:test";
import assert from "node:assert/strict";
import { statusKind, polls, graceDate, POLL_MS } from "../src/blupdomain.ts";

test("each status of the contract has its color", () => {
  assert.equal(statusKind("active"), "good");
  assert.equal(statusKind("pending_ownership"), "warn");
  assert.equal(statusKind("pending"), "warn");
  assert.equal(statusKind("error"), "bad");
  assert.equal(statusKind("disabled"), "bad");
  assert.equal(statusKind("<img src=x>"), "muted");
  assert.equal(statusKind(undefined), "muted");
});

test("polling is spaced (a minute at least)", () => {
  assert.ok(POLL_MS >= 60_000);
});

test("only a domain on its way is asked again", () => {
  for (const s of ["pending_ownership", "pending", "error"]) assert.equal(polls(s), true, s);
  for (const s of ["active", "disabled", "", null, undefined]) assert.equal(polls(s), false, String(s));
});

test("the grace day after a downgrade", () => {
  assert.equal(graceDate(null, "pt-BR"), "");
  assert.equal(graceDate("amanhã", "pt-BR"), "");
  assert.match(graceDate("2026-11-16T12:00:00Z", "en-US"), /November 16, 2026/);
});
