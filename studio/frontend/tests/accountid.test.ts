// The account's id in Settings > Cloud: the help text the owner approved,
// in each language, without dashes.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const src = readFileSync(new URL("../src/i18n.ts", import.meta.url), "utf8");

test("the help text is exactly the approved one", () => {
  for (const help of ["Em caso de ajuda, informe seu ID.", "If you need help, share your ID.", "Si necesitas ayuda, comparte tu ID."])
    assert.ok(src.includes(`accountIdHelp: "${help}"`), help);
  for (const label of ['accountId: "ID da conta"', 'accountId: "Account ID"', 'accountId: "ID de la cuenta"']) assert.ok(src.includes(label), label);
  assert.equal((src.match(/idCopied: "(Copiado|Copied)"/g) ?? []).length, 3);
});

test("no dashes in the new texts", () => {
  for (const m of src.matchAll(/(accountId|accountIdHelp|copyId|idCopied): "([^"]*)"/g)) assert.doesNotMatch(m[2], /[–—]/, m[0]);
});
