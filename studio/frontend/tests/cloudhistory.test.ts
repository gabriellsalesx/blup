// Deleting the old history on Blup Cloud: what the screen shows after.
import { test } from "node:test";
import assert from "node:assert/strict";
import { bytes, afterPurge } from "../src/cloudhistory.ts";

test("sizes like the settings' usage", () => {
  assert.equal(bytes(0), "0.0 MB");
  assert.equal(bytes(5 << 20), "5.0 MB");
  assert.equal(bytes(300 << 20), "300 MB");
  assert.equal(bytes(3 * (1 << 30)), "3.0 GB");
});

test("the use after a purge comes from the cloud's answer", () => {
  const a = { storageBytes: 900, historyBytes: 400, plan: "pro" };
  assert.deepEqual(afterPurge(a, { freedBytes: 300, storageBytes: 600 }), { storageBytes: 600, historyBytes: 100, plan: "pro" });
  // Again: nothing freed, nothing changes.
  assert.deepEqual(afterPurge(a, { freedBytes: 0, storageBytes: 900 }), a);
  // Freed more than the history known here (other vaults): not below 0.
  assert.equal(afterPurge(a, { freedBytes: 999, storageBytes: 1 }).historyBytes, 0);
  // Unknown before: still unknown, never a made up 0.
  assert.equal(afterPurge({ ...a, historyBytes: null }, { freedBytes: 300, storageBytes: 600 }).historyBytes, null);
});

test("the banner shows only above the limit, with the grace while it runs", async () => {
  const { overLimit } = await import("../src/cloudhistory.ts");
  const now = new Date("2026-11-02T12:00:00Z");
  const a = { signedIn: true, storageBytes: 900, storageLimit: 1000, graceUntil: "", graceLimit: 0 };
  assert.equal(overLimit(a, now), null);
  assert.equal(overLimit({ ...a, signedIn: false, storageBytes: 2000 }, now), null);
  assert.equal(overLimit({ ...a, storageLimit: 0, storageBytes: 2000 }, now), null);
  assert.deepEqual(overLimit({ ...a, storageBytes: 1500 }, now), { used: 1500, limit: 1000, graceUntil: null, graceLimit: 0 });
  const g = overLimit({ ...a, storageBytes: 1500, graceUntil: "2026-11-16T00:00:00Z", graceLimit: 5000 }, now);
  assert.equal(g?.graceUntil?.toISOString(), "2026-11-16T00:00:00.000Z");
  assert.equal(g?.graceLimit, 5000);
  // A grace over, or not a date: none.
  assert.equal(overLimit({ ...a, storageBytes: 1500, graceUntil: "2026-10-01T00:00:00Z", graceLimit: 5000 }, now)?.graceUntil, null);
  assert.equal(overLimit({ ...a, storageBytes: 1500, graceUntil: "amanhã", graceLimit: 5000 }, now)?.graceUntil, null);
});
