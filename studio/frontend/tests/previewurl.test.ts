// The preview follows a restart (a new address) even on the same page.
import { test } from "node:test";
import assert from "node:assert/strict";

(globalThis as { location?: unknown }).location = { href: "http://127.0.0.1:9000/" };
const { samePage } = await import("../src/previewurl.ts");

test("the page the reader went to is not loaded twice", () => {
  assert.equal(samePage("http://127.0.0.1:4000/docs/", "http://127.0.0.1:4000/", "/docs/"), true);
});

test("a restarted preview loads the page again", () => {
  assert.equal(samePage("http://127.0.0.1:5000/", "http://127.0.0.1:4000/", "/"), false);
  assert.equal(samePage("http://127.0.0.1:5000/docs/", "http://127.0.0.1:4000/", "/docs/"), false);
});

test("another page loads", () => {
  assert.equal(samePage("http://127.0.0.1:4000/blog/", "http://127.0.0.1:4000/", "/docs/"), false);
  assert.equal(samePage("not a url", "", "/"), false);
});

const { previewMessage } = await import("../src/previewurl.ts");

test("a preview page's messages are read", () => {
  assert.deepEqual(previewMessage({ blup: "loaded", y: 120, path: "/docs/", q: "?_blup=3" }), { blup: "loaded", y: 120, path: "/docs/", q: "?_blup=3" });
  assert.deepEqual(previewMessage({ blup: "scroll", y: -4, path: "/", q: "" }), { blup: "scroll", y: 0, path: "/", q: "" });
});

test("anything else from the preview's origin is ignored", () => {
  for (const d of [
    null, "loaded", 3, [], {},
    { blup: "scrollTo", y: 0, path: "/", q: "" },
    { blup: "loaded", y: 0, path: 5, q: "" },
    { blup: "loaded", y: 0, path: "//evil.com/", q: "" },
    { blup: "loaded", y: 0, path: "https://evil.com/", q: "" },
    { blup: "loaded", y: 0, path: "/a\nb", q: "" },
    { blup: "loaded", y: 0, path: "/", q: { _blup: "1" } },
    { blup: "loaded", y: 0, path: "/", q: "_blup=1" },
    { blup: "loaded", y: Number.NaN, path: "/", q: "" },
    { blup: "loaded", y: Infinity, path: "/", q: "" },
    { blup: "loaded", y: "10", path: "/", q: "" },
  ]) assert.equal(previewMessage(d), null, JSON.stringify(d));
});
