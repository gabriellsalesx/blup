// The window's Content-Security-Policy (index.html): what it may load, and
// no inline script in the built page. Run: npm test (node:test, no deps).
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync, existsSync } from "node:fs";

const here = new URL("..", import.meta.url);
const read = (p: string) => readFileSync(new URL(p, here), "utf8");

function policy(html: string): Map<string, string[]> {
  const m = html.match(/<meta\s+http-equiv="Content-Security-Policy"\s+content="([^"]+)"/i);
  assert.ok(m, "index.html has no Content-Security-Policy");
  const out = new Map<string, string[]>();
  for (const d of m[1].split(";")) {
    const [name, ...values] = d.trim().split(/\s+/);
    if (name) out.set(name, values);
  }
  return out;
}

const WANT: Record<string, string[]> = {
  "default-src": ["'self'"],
  "script-src": ["'self'", "'wasm-unsafe-eval'"],
  "style-src": ["'self'", "'unsafe-inline'"],
  "img-src": ["'self'", "data:", "blob:", "https:", "http:"],
  "media-src": ["'self'", "data:", "blob:"],
  "font-src": ["'self'", "data:"],
  // The error reports (only after the person's yes) go to Blup Pulse.
  "connect-src": ["'self'", "https://ingest.getblup.io"],
  "worker-src": ["'self'", "blob:"],
  "frame-src": ["http://127.0.0.1:*"],
  "object-src": ["'none'"],
  "base-uri": ["'none'"],
};

test("index.html sets the window's policy", () => {
  const p = policy(read("index.html"));
  assert.deepEqual(Object.fromEntries(p), WANT);
  // Never a way to run code from text: inline scripts or eval.
  assert.ok(!p.get("script-src")!.includes("'unsafe-inline'"));
  assert.ok(!p.get("script-src")!.includes("'unsafe-eval'"));
});

test("the policy comes before any script", () => {
  const html = read("index.html");
  assert.ok(html.indexOf("Content-Security-Policy") < html.indexOf("<script"));
});

test("the built page keeps the policy and has no inline script", { skip: !existsSync(new URL("dist/index.html", here)) && "no dist (npm run build)" }, () => {
  const html = read("dist/index.html");
  assert.deepEqual(Object.fromEntries(policy(html)), WANT);
  const scripts = [...html.matchAll(/<script\b([^>]*)>([\s\S]*?)<\/script>/gi)];
  assert.ok(scripts.length > 0);
  for (const [, attrs, body] of scripts) {
    assert.match(attrs, /\bsrc=/, "a script without src");
    assert.equal(body.trim(), "", "an inline script");
  }
  assert.ok(!/\son[a-z]+=/i.test(html), "an inline event handler");
});

test("only the dev server may open its WebSocket", () => {
  const cfg = read("vite.config.ts");
  assert.match(cfg, /apply: "serve"/);
  assert.match(cfg, /connect-src 'self' ws:\/\/127\.0\.0\.1:\* ws:\/\/localhost:\*/);
});
