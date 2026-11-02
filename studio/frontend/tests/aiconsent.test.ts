// Before a note's text first goes to an AI in the cloud, the person is
// asked; a no sends nothing, a yes is kept and the call runs again.
import { test } from "node:test";
import assert from "node:assert/strict";
import { consentDest, needsProvider, withConsent, ConsentCancelled } from "../src/aiconsent.ts";

test("reads the destination from the Go error", () => {
  assert.equal(consentDest(new Error("ai consent needed: openrouter")), "openrouter");
  assert.equal(consentDest("RuntimeError: ai consent needed: blup"), "blup");
  assert.equal(consentDest(new Error("ai consent needed: evil")), null);
  assert.equal(consentDest(new Error("no OpenRouter key")), null);
  assert.equal(needsProvider(new Error("no AI provider chosen")), true);
  assert.equal(needsProvider(new Error("other")), false);
});

test("a yes is kept and the call runs again", async () => {
  let accepted = "";
  let runs = 0;
  const out = await withConsent(async () => {
    runs++;
    if (!accepted) throw new Error("ai consent needed: blup");
    return "ok";
  }, async (d) => d === "blup", async (d) => { accepted = d; });
  assert.equal(out, "ok");
  assert.equal(accepted, "blup");
  assert.equal(runs, 2);
});

test("a no sends nothing and keeps nothing", async () => {
  let runs = 0;
  let kept = false;
  await assert.rejects(withConsent(async () => { runs++; throw new Error("ai consent needed: openrouter"); }, async () => false, async () => { kept = true; }), ConsentCancelled);
  assert.equal(runs, 1);
  assert.equal(kept, false);
});

test("other errors pass through without asking", async () => {
  let asked = false;
  await assert.rejects(withConsent(async () => { throw new Error("no AI provider chosen"); }, async () => { asked = true; return true; }, async () => {}), /no AI provider chosen/);
  assert.equal(asked, false);
});

test("it does not loop when Go keeps asking", async () => {
  let runs = 0;
  await assert.rejects(withConsent(async () => { runs++; throw new Error("ai consent needed: blup"); }, async () => true, async () => {}), /ai consent needed/);
  assert.equal(runs, 3);
});
