// The logo's own link: only web addresses or paths of the site.
import { test } from "node:test";
import assert from "node:assert/strict";
import { validLogoLink } from "../src/logolink.ts";

test("web addresses and paths of the site are accepted", () => {
  for (const s of ["https://empresa.com.br", "http://empresa.com/loja?x=1#a", "/sobre/", "/"]) assert.equal(validLogoLink(s), true, s);
});

test("anything else keeps the home page", () => {
  for (const s of ["", "javascript:alert(1)", "data:text/html,x", "//evil.com", "empresa.com", "ftp://x.com", "https://", "https://user:pw@x.com", "https://x.com/a b", "/a\nb", "https://x.com/\"onmouseover=1", "mailto:a@b.c", "\\\\evil"]) assert.equal(validLogoLink(s), false, s);
});
