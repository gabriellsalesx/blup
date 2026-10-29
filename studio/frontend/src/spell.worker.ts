// The spell check, off the page: Hunspell (the LibreOffice one, in
// WebAssembly) with the Portuguese (Brazil), English (US) and Spanish
// dictionaries. A dictionary loads on first use, in a fraction of a second.
import { createHunspellFromStrings } from "hunspell-wasm";
import ptAff from "../node_modules/dictionary-pt/index.aff?url";
import ptDic from "../node_modules/dictionary-pt/index.dic?url";
import enAff from "../node_modules/dictionary-en/index.aff?url";
import enDic from "../node_modules/dictionary-en/index.dic?url";
import esAff from "../node_modules/dictionary-es/index.aff?url";
import esDic from "../node_modules/dictionary-es/index.dic?url";

const files: Record<string, [string, string]> = { pt: [ptAff, ptDic], en: [enAff, enDic], es: [esAff, esDic] };
type Speller = Awaited<ReturnType<typeof createHunspellFromStrings>>;
const loaded: Record<string, Promise<Speller>> = {};
// Each dictionary's words without their endings, sorted, for completing
// a word as it is typed.
const stems: Record<string, string[]> = {};
let lang = "pt";
let words: string[] = [];
// Words every Blup site uses that the dictionaries don't have.
const BUILT_IN: Record<string, string[]> = {
  all: ["Blup", "Studio", "Obsidian", "Hugo", "GitHub", "Cloudflare", "Markdown", "YouTube", "Instagram", "WhatsApp", "TikTok", "LinkedIn", "Google", "Pagefind", "Bento", "Mochi", "frontmatter", "wikilink", "wikilinks", "callout", "callouts", "changelog", "roadmap", "favicon", "PWA", "RSS", "SEO", "Tangled", "Bluesky", "Mastodon", "Codex", "Claude", "OpenCode", "Gemini", "Antigravity"],
  pt: ["post", "posts", "blog", "blogs", "site", "sites", "link", "links", "email", "e-mail", "online", "feed", "tag", "tags", "layout", "login", "app", "apps", "download", "upload", "newsletter", "podcast", "template", "templates", "dashboard", "backup", "cookie", "cookies", "slug", "deploy", "markdown", "vault", "plugin", "plugins", "card", "cards", "banner", "player", "chat"],
};

function speller(l: string) {
  loaded[l] ??= (async () => {
    const [aff, dic] = await Promise.all(files[l].map((u) => fetch(u).then((r) => r.text())));
    const h = await createHunspellFromStrings(aff, dic);
    stems[l] = [...new Set(dic.split("\n").slice(1).map((x) => x.split("/")[0].trim()).filter((x) => x.length > 3 && !/[\d.]/.test(x)))]
      .sort((a, b) => a.toLowerCase().localeCompare(b.toLowerCase()));
    for (const w of [...BUILT_IN.all, ...(BUILT_IN[l] ?? []), ...words]) h.addWord(w);
    return h;
  })();
  return loaded[l];
}

// A word passes when Hunspell knows it; a hyphenated one when each part does.
function correct(h: Speller, word: string) {
  if (h.testSpelling(word)) return true;
  const parts = word.split("-").filter(Boolean);
  return parts.length > 1 && parts.every((p) => h.testSpelling(p));
}

type Msg =
  | { type: "setup"; lang: string; words: string[] }
  | { type: "check"; id: number; words: string[] }
  | { type: "suggest"; id: number; word: string }
  | { type: "add"; word: string }
  | { type: "complete"; id: number; word: string };

// The first words of the sorted list that start with prefix (any case).
function starting(list: string[], prefix: string, max: number) {
  const p = prefix.toLowerCase();
  let lo = 0, hi = list.length;
  while (lo < hi) {
    const mid = (lo + hi) >> 1;
    if (list[mid].toLowerCase().localeCompare(p) < 0) lo = mid + 1; else hi = mid;
  }
  const out: string[] = [];
  for (let i = lo; i < list.length && out.length < max * 3; i++) {
    const w = list[i];
    if (!w.toLowerCase().startsWith(p)) break;
    if (w.length > prefix.length) out.push(w);
  }
  // Shorter (more common) words first; a lowercase word follows the case
  // of what was typed (Configura -> Configuração).
  return out.sort((a, b) => a.length - b.length).slice(0, max)
    .map((w) => (/^\p{Lu}/u.test(prefix) && /^\p{Ll}/u.test(w) ? w[0].toUpperCase() + w.slice(1) : w));
}

self.onmessage = async (e: MessageEvent<Msg>) => {
  const m = e.data;
  if (m.type === "setup") {
    lang = files[m.lang] ? m.lang : "pt";
    words = m.words;
    return;
  }
  if (m.type === "add") {
    words.push(m.word);
    for (const p of Object.values(loaded)) (await p).addWord(m.word);
    return;
  }
  try {
    const h = await speller(lang);
    if (m.type === "complete") {
      // Corrections that keep one word come first ("facilmente" before
      // "fácil mente").
      const all = correct(h, m.word) ? [] : h.getSpellingSuggestions(m.word);
      const fixes = [...all.filter((w) => !w.includes(" ")), ...all.filter((w) => w.includes(" "))].slice(0, 3);
      postMessage({ id: m.id, fixes, words: starting(stems[lang] ?? [], m.word, 8) });
      return;
    }
    if (m.type === "check") postMessage({ id: m.id, wrong: m.words.filter((w) => !correct(h, w)) });
    else postMessage({ id: m.id, suggestions: h.getSpellingSuggestions(m.word).slice(0, 6) });
  } catch (err) {
    postMessage({ id: m.id, error: String(err) });
  }
};
