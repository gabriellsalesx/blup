// The translation on the computer: Markdown in, Markdown out, through
// Firefox's engine in a worker (public/bergamot/worker.js). The engine
// translates HTML keeping its tags, so each line's Markdown becomes tags it
// keeps (bold, italics, links...) and the parts that must not change (code,
// addresses, wikilink targets, images, math) become tags holding their own
// text, only for context: they come back exactly as they were.

import { ReadNote, SaveTranslation, LocalTranslate, TranslationTexts } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
import { look } from "./theme.svelte";
import { translateReply } from "./translatereply";

type Item = { text: string; html: boolean };
let worker: Worker | null = null;
let seq = 0;
const waiting = new Map<number, { n: number; ok: (out: string[]) => void; fail: (e: Error) => void }>();

function engine(): Worker {
  if (worker) return worker;
  worker = new Worker("/bergamot/worker.js");
  worker.onmessage = (e: MessageEvent<unknown>) => {
    const r = translateReply(e.data);
    const w = r && waiting.get(r.id);
    if (!r || !w) return;
    waiting.delete(r.id);
    if ("error" in r) w.fail(new Error(r.error));
    // One text back for each sent: anything else would shift the lines.
    else if (r.out.length !== w.n) w.fail(new Error("translation worker answered out of shape"));
    else w.ok(r.out);
  };
  worker.onerror = (e) => {
    for (const w of waiting.values()) w.fail(new Error(e.message || "translation worker failed"));
    waiting.clear();
    worker?.terminate();
    worker = null;
  };
  return worker;
}

function run(from: string, to: string, items: Item[]): Promise<string[]> {
  if (!items.length) return Promise.resolve([]);
  const id = ++seq;
  return new Promise((ok, fail) => {
    waiting.set(id, { n: items.length, ok, fail });
    engine().postMessage({ id, from, to, items });
  });
}

const esc = (s: string) => s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
const unesc = (s: string) => s.replace(/&lt;/g, "<").replace(/&gt;/g, ">").replace(/&quot;/g, '"').replace(/&#39;/g, "'").replace(/&nbsp;/g, " ").replace(/&amp;/g, "&");

// Bold, italics, strike and highlight as tags (on escaped text).
const marks = (s: string) => s
  .replace(/\*\*([^*]+)\*\*/g, "<b>$1</b>").replace(/__([^_]+)__/g, "<b>$1</b>")
  .replace(/(^|[^*\w])\*([^*\s][^*]*)\*/g, "$1<i>$2</i>").replace(/(^|[^\w])_([^_\s][^_]*)_(?=\W|$)/g, "$1<i>$2</i>")
  .replace(/~~([^~]+)~~/g, "<s>$1</s>").replace(/==([^=]+)==/g, "<mark>$1</mark>");

/** What a line keeps aside: the original, and the spaces it had around. */
export type Kept = { orig: string; before: boolean; after: boolean; lowerAfter: boolean; lowerText: boolean };

/** A line's text as HTML for the engine, and what to put back. */
function toHTML(text: string): { html: string; keep: Kept[] } {
  const keep: Kept[] = [];
  const inner: { tag: string; text: string }[] = [];
  // A part kept aside, marked \u0000n\u0000 until the text is escaped.
  const hold = (tag: string, orig: string, text: string, at: number, all: string, len: number) => {
    // Whether the word after it (and a link's own text) began in lower
    // case: the engine tends to capitalize what follows a tag.
    keep.push({ orig, before: /\s$/.test(all.slice(0, at)), after: /^\s/.test(all.slice(at + len)), lowerAfter: /^\s*\p{Ll}/u.test(all.slice(at + len)), lowerText: /^\p{Ll}/u.test(text) });
    inner.push({ tag, text });
    return `\u0000${keep.length - 1}\u0000`;
  };
  let s = text;
  s = s.replace(/`[^`]+`/g, (m, at, all) => hold("x-k", m, m.slice(1, -1), at, all, m.length));
  s = s.replace(/\$[^$\n]+\$/g, (m, at, all) => hold("x-k", m, m, at, all, m.length));
  s = s.replace(/!\[\[[^\]]+\]\]|!\[[^\]]*\]\([^)]*\)/g, (m, at, all) => hold("x-k", m, "", at, all, m.length));
  // [[Target|alias]]: the alias is translated, the target kept.
  s = s.replace(/\[\[([^\]|]+)\|([^\]]+)\]\]/g, (m, target, alias, at, all) => hold("x-w", `[[${target}|`, alias, at, all, m.length));
  s = s.replace(/\[\[[^\]]+\]\]/g, (m, at, all) => hold("x-k", m, m.slice(2, -2).split("#")[0], at, all, m.length));
  // [text](address): the text is translated, the address kept.
  s = s.replace(/\[([^\]]+)\]\(([^)\s]+(?:\s+"[^"]*")?)\)/g, (m, label, addr, at, all) => hold("x-l", `](${addr})`, label, at, all, m.length));
  s = s.replace(/<[^>\n]+>/g, (m, at, all) => hold("x-k", m, "", at, all, m.length)); // raw HTML
  s = s.replace(/\bhttps?:\/\/\S+/g, (m, at, all) => hold("x-k", m, m, at, all, m.length));
  s = s.replace(/\{\.[^}]+\}/g, (m, at, all) => hold("x-k", m, "", at, all, m.length));
  s = marks(esc(s));
  s = s.replace(/\u0000(\d+)\u0000/g, (_, n) => {
    const { tag, text } = inner[+n];
    // Links' and wikilinks' text is translated (with its bold, italics);
    // the rest is there for context only.
    return `<${tag} n="${n}">${tag === "x-k" ? esc(text) : marks(esc(text))}</${tag}>`;
  });
  return { html: s, keep };
}

/** The engine's HTML back to Markdown. */
function fromHTML(html: string, keep: Kept[]): string {
  let s = html;
  const tags = (t: string) => t.replace(/<\/?b>/g, "**").replace(/<\/?i>/g, "*").replace(/<\/?s>/g, "~~").replace(/<\/?mark>/g, "==").replace(/<[^>]+>/g, "");
  s = s.replace(/<x-w n="(\d+)">([\s\S]*?)<\/x-w>/g, (_, n, t) => `\u0001${n}\u0002${tags(t).trim()}\u0003`);
  s = s.replace(/<x-l n="(\d+)">([\s\S]*?)<\/x-l>/g, (_, n, t) => `\u0004${n}\u0002${tags(t).trim()}\u0003`);
  s = s.replace(/<x-k n="(\d+)">[\s\S]*?<\/x-k>/g, (_, n) => `\u0005${n}\u0005`);
  s = unesc(tags(s));
  // A word that began in lower case in the original stays so.
  // (a one letter word too: "Y", "E").
  const lower = (t: string, yes: boolean) => (yes ? t.replace(/^(\p{Lu})(?=\p{Ll}|[\s,.;:!?]|$)/u, (c) => c.toLowerCase()) : t);
  const after = (rest: string, yes: boolean) => (yes ? rest.replace(/^(\s*)(\p{Lu})(?=\p{Ll}|[\s,.;:!?]|$)/u, (_, sp, c) => sp + c.toLowerCase()) : rest);
  s = s.replace(/\u0001(\d+)\u0002([^\u0003]*)\u0003([^]*)$/, function fix(_m: string, n: string, t: string, rest: string): string {
    const k = keep[+n];
    const tail = after(rest, k.lowerAfter);
    return `${k.orig}${lower(t, k.lowerText)}]]` + tail.replace(/\u0001(\d+)\u0002([^\u0003]*)\u0003([^]*)$/, fix);
  });
  s = s.replace(/\u0004(\d+)\u0002([^\u0003]*)\u0003([^]*)$/, function fix(_m: string, n: string, t: string, rest: string): string {
    const k = keep[+n];
    const tail = after(rest, k.lowerAfter);
    return `[${lower(t, k.lowerText)}${k.orig}` + tail.replace(/\u0004(\d+)\u0002([^\u0003]*)\u0003([^]*)$/, fix);
  });
  // Kept parts exactly as they were, with the spaces they had around (the
  // engine sometimes moves a space to the other side).
  s = s.replace(/ *\u0005(\d+)\u0005 */g, (m, n, at, all) => {
    const k = keep[+n];
    const start = at === 0, end = at + m.length === all.length;
    return `${k.before && !start ? " " : ""}${k.orig}${k.after && !end ? " " : ""}\u0006${n}\u0006`;
  });
  s = s.replace(/\u0006(\d+)\u0006([^]*)$/, function fix(_m: string, n: string, rest: string): string {
    return after(rest, keep[+n].lowerAfter).replace(/\u0006(\d+)\u0006([^]*)$/, fix);
  });
  return s.replace(/ {2,}/g, " ");
}

// A line: its Markdown marks (heading, quote, list, task, callout), kept,
// and its text, translated.
const LEAD = /^(\s*(?:>\s?)*(?:#{1,6}\s+|[-*+]\s+(?:\[[ xX]\]\s+)?|\d+[.)]\s+)?(?:\[![\w-]+\][+-]?\s*)?)(.*)$/;

/**
 * Translates a note's text (Markdown, no front matter) from one language
 * to another, keeping its structure: code blocks, tables' shape, marks.
 */
async function translateMarkdown(md: string, from: string, to: string, progress?: (done: number, total: number) => void): Promise<string> {
  const lines = md.split("\n");
  const jobs: { line: number; cell?: number; lead: string; keep: Kept[] }[] = [];
  const items: Item[] = [];
  const cells: Record<number, string[]> = {};
  let fence = "";
  lines.forEach((line, i) => {
    const t = line.trim();
    const f = /^(```|~~~)/.exec(t);
    if (f) { fence = fence ? (t.startsWith(fence) ? "" : fence) : f[1]; return; }
    if (fence || !t || /^(-{3,}|\*{3,}|_{3,})$/.test(t) || /^\|?\s*:?-{2,}/.test(t)) return;
    if (/^\s*\|.*\|\s*$/.test(line)) {
      // A table row: each cell on its own.
      cells[i] = line.trim().replace(/^\||\|$/g, "").split("|");
      cells[i].forEach((c, k) => {
        if (!c.trim()) return;
        const { html, keep } = toHTML(c.trim());
        jobs.push({ line: i, cell: k, lead: "", keep });
        items.push({ text: html, html: true });
      });
      return;
    }
    const m = LEAD.exec(line)!;
    if (!m[2].trim()) return;
    const { html, keep } = toHTML(m[2]);
    jobs.push({ line: i, lead: m[1], keep });
    items.push({ text: html, html: true });
  });
  // In batches, so a long note shows its progress and the worker answers.
  const out: string[] = [];
  const BATCH = 40;
  for (let i = 0; i < items.length; i += BATCH) {
    progress?.(i, items.length);
    out.push(...(await run(from, to, items.slice(i, i + BATCH))));
  }
  progress?.(items.length, items.length);
  jobs.forEach((j, k) => {
    const text = fromHTML(out[k] ?? "", j.keep);
    if (j.cell === undefined) lines[j.line] = j.lead + text;
    else cells[j.line][j.cell] = ` ${text} `;
  });
  for (const [i, c] of Object.entries(cells)) lines[+i] = `|${c.join("|")}|`;
  return lines.join("\n");
}

/** A short text (a title, a description). */
async function translateText(text: string, from: string, to: string): Promise<string> {
  if (!text.trim()) return text;
  const { html, keep } = toHTML(text.trim());
  const [out] = await run(from, to, [{ text: html, html: true }]);
  return fromHTML(out ?? "", keep);
}


/**
 * Whether notes are translated on the computer: installed, and chosen (or
 * nothing chosen yet). Otherwise the AI in the cloud the person picked
 * ("openrouter" or "blup"); the older "cloud" let the Studio pick, so it
 * goes to Go, which asks the person to choose.
 */
export async function useLocal(): Promise<boolean> {
  if (look.translateAI !== "local" && look.translateAI !== "") return false;
  return (await LocalTranslate().catch(() => null))?.installed ?? false;
}

/**
 * Translates the note original (in from) into to on the computer and
 * saves it next to it; returns the translation's path.
 */
export async function translateNoteLocal(original: string, from: string, to: string, progress?: (done: number, total: number) => void, follow = true): Promise<string> {
  const n = await ReadNote(original);
  const props = (n.props ?? {}) as Record<string, unknown>;
  const name = original.split("/").pop()!.replace(/\.md$/i, "");
  const title = await translateText(String(props.title || name), from, to);
  const desc = props.description ? await translateText(String(props.description), from, to) : "";
  const body = await translateMarkdown(n.body ?? "", from, to, progress);
  // The other texts the site shows (menu labels, the site's description...).
  const texts: Record<string, string[]> = {};
  const found = (await TranslationTexts(original).catch(() => null)) ?? {};
  for (const [k, list] of Object.entries(found)) {
    texts[k] = await Promise.all((list ?? []).map((x: string) => translateText(x, from, to)));
  }
  return SaveTranslation(original, to, title, desc, body, texts, follow);
}
