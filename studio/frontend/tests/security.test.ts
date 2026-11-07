// Proofs of concept against XSS in the Markdown the Studio turns into HTML
// itself (canvas cards and export/print). Run: npm test (node:test, no deps).
import { test } from "node:test";
import assert from "node:assert/strict";
import { cardHTML } from "../src/canvas.ts";
import { noteHTML, noteDocument } from "../src/export.ts";
import { resolveHref } from "../src/notelinks.ts";

const ATTACKS = [
  `<img src=x onerror="alert(1)">`,
  `<svg onload=alert(1)><script>alert(1)</script></svg>`,
  `<iframe src="javascript:alert(1)"></iframe>`,
  `[x](javascript:alert(1))`,
  `[x](data:text/html,<script>alert(1)</script>)`,
  `[x](vbscript:msgbox(1))`,
  `[a"><img src=x onerror=alert(1)>](https://e.com/"onmouseover="alert(1))`,
  `[[Nota<img src=x onerror=alert(1)>|<b onclick=alert(1)>a</b>]]`,
  `**<i onclick=alert(1)>x</i>** _"'_ ==<a href=javascript:1>m</a>==`,
  "`<script>alert(1)</script>`",
  "```\n</code></pre><script>alert(1)</script>\n```",
  `> [!note] <img src=x onerror=alert(1)>\n> <svg/onload=alert(1)>`,
  `| <img src=x onerror=1> | b |\n|---|---|\n| <script>1</script> | [x](javascript:1) |`,
  `- [ ] <style>*{background:url(https://evil)}</style>`,
  `# <a href="javascript:alert(1)">t</a>`,
  `[x](https://a.com/*b*"c) *y* _z_ ~~w~~`,
];

// The tags and attributes the renderers are allowed to produce.
const TAGS = new Set(["h1", "h2", "h3", "h4", "h5", "h6", "p", "ul", "ol", "li", "b", "i", "s", "mark", "code", "pre", "blockquote", "hr", "br", "button", "a", "img", "aside", "table", "thead", "tbody", "tr", "th", "td", "input"]);
const ATTRS = new Set(["class", "data-href", "href", "src", "alt", "type", "disabled", "checked"]);

// A tiny tokenizer good enough for HTML these renderers build: every tag
// and its attributes, with quoted values.
function check(html: string) {
  const tag = /<\/?([a-zA-Z][\w-]*)((?:\s+[^\s=>"'/]+(?:\s*=\s*(?:"[^"]*"|'[^']*'|[^\s>"']+))?)*)\s*\/?>/g;
  for (const m of html.matchAll(tag)) {
    assert.ok(TAGS.has(m[1].toLowerCase()), `tag not allowed: <${m[1]}> in ${html}`);
    for (const a of m[2].matchAll(/([^\s=>"'/]+)(?:\s*=\s*("[^"]*"|'[^']*'|[^\s>"']+))?/g)) {
      const name = a[1].toLowerCase();
      assert.ok(ATTRS.has(name), `attribute not allowed: ${name} in ${m[0]}`);
      const v = (a[2] ?? "").replace(/^["']|["']$/g, "");
      if (name === "href") assert.match(v, /^(https?:|mailto:)/i, `bad href ${v}`);
      if (name === "src") assert.match(v, /^data:image\//i, `bad src ${v}`);
    }
  }
  // Nothing left that looks like a raw tag the tokenizer missed.
  assert.doesNotMatch(html.replace(tag, ""), /<[a-zA-Z/!?]/, `stray markup in ${html}`);
  assert.doesNotMatch(html, /<script|<svg|<iframe|<style/i);
}

test("canvas cards: attacker Markdown never becomes active HTML", () => {
  for (const md of ATTACKS) check(cardHTML(md));
});

test("canvas cards: link targets stay data (no javascript: execution path)", () => {
  const html = cardHTML(`[x](javascript:alert)`);
  assert.match(html, /<button class="lnk" data-href="javascript:alert">x<\/button>/);
  // The click handler sends non-note addresses through resolveHref, which
  // only lets http(s)/mailto out.
  assert.equal(resolveHref("javascript:alert(1)", "a.md", []), null);
  assert.equal(resolveHref("data:text/html,x", "a.md", []), null);
  assert.equal(resolveHref("vbscript:x", "a.md", []), null);
  assert.deepEqual(resolveHref("https://e.com", "a.md", []), { url: "https://e.com" });
});

test("export/print: attacker Markdown never becomes active HTML", async () => {
  const image = async () => "";
  for (const md of ATTACKS) check(await noteHTML(md, image));
});

test("export/print: only data:image/ URLs are embedded, escaped", async () => {
  const evil = async () => `x" onerror="alert(1)`;
  assert.equal(await noteHTML(`![a](p.png)`, evil), "<p></p>");
  const js = async () => "javascript:alert(1)";
  assert.equal(await noteHTML(`![a](p.png)`, js), "<p></p>");
  const ok = async () => `data:image/png;base64,AAAA"x`;
  const html = await noteHTML(`![a"b](p.png)`, ok);
  assert.equal(html, `<p><img src="data:image/png;base64,AAAA&quot;x" alt="a&quot;b"></p>`);
  check(html);
});

test("export: title and language are escaped", async () => {
  const doc = await noteDocument(`</title><script>alert(1)</script>`, "x", `"><script>`, async () => "");
  assert.doesNotMatch(doc, /<script/);
});

test("canvas cards: an image of the vault is a mark to load, never an <img> from the text", () => {
  assert.match(cardHTML(`![foto](../Anexos/a%20b.png)`), /<span class="card-img" data-src="\.\.\/Anexos\/a%20b\.png">foto<\/span>/);
  assert.match(cardHTML(`![[foto.png|300]]`), /<span class="card-img" data-src="\[\[foto\.png\]\]">foto\.png<\/span>/);
  // On the web: a link, so opening a note tells no server anything.
  assert.match(cardHTML(`![x](https://example.com/p.png)`), /<button class="lnk" data-href="https:\/\/example\.com\/p\.png">x<\/button>/);
  assert.doesNotMatch(cardHTML(`![x](https://example.com/p.png)`), /card-img/);
  // Not an image: the link it always was.
  assert.match(cardHTML(`![[Outra nota]]`), /<button class="lnk"/);
  for (const md of [`![a"onerror="x](b.png)`, `![x](b.png"onerror="alert(1))`, `![[a"onload="x.png]]`]) assert.doesNotMatch(cardHTML(md), /<img|\sonerror=|\sonload=/);
});
