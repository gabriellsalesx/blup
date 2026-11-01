// Links between notes, as the editor reads them: a Markdown link with a
// path relative to the note ([Name](../Folder/Name.md)), the kind the site
// understands, or an Obsidian [[Name]]. Images (![...]) are not links.

export type Target = { note: string } | { url: string };
export type NoteRef = { name: string; path: string };

const MD_LINK = /!?\[(?:[^\]\\]|\\.)*\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g;
const WIKI_LINK = /!?\[\[([^\]|#\n]+)(#[^\]|\n]*)?(\|[^\]\n]*)?\]\]/g;

const dirOf = (path: string) => path.split("/").slice(0, -1).join("/");
const noExt = (s: string) => s.toLowerCase().replace(/\.md$/, "");

function find(notes: NoteRef[], path: string) {
  const want = path.toLowerCase();
  return notes.find((n) => n.path.toLowerCase() === want || n.path.toLowerCase() === want + ".md")?.path;
}

// resolveHref turns a link's address, relative to the note `from`, into a
// note of the folder or an address on the web.
export function resolveHref(href: string, from: string, notes: NoteRef[]): Target | null {
  if (/^(https?:|mailto:)/i.test(href)) return { url: href };
  let path = href.split("#")[0].split("?")[0];
  if (!path) return null;
  try { path = decodeURIComponent(path); } catch {}
  const parts = path.startsWith("/") ? [] : from.split("/").slice(0, -1);
  for (const seg of path.split("/")) {
    if (seg === "..") parts.pop();
    else if (seg && seg !== ".") parts.push(seg);
  }
  const found = find(notes, parts.join("/"));
  return found ? { note: found } : null;
}

// resolveWiki finds the note of [[name]]: by name or path, one in the same
// folder as `from` first, as Obsidian does.
export function resolveWiki(name: string, from: string, notes: NoteRef[]): Target | null {
  const want = noExt(name.trim());
  const matches = notes.filter((n) => noExt(n.name) === want || noExt(n.path) === want || n.path.toLowerCase().endsWith("/" + want + ".md"));
  const best = matches.find((n) => dirOf(n.path) === dirOf(from)) ?? matches[0];
  return best ? { note: best.path } : null;
}

// targetAt is the link at column `at` of a line of the note `from`.
export function targetAt(line: string, at: number, from: string, notes: NoteRef[]): Target | null {
  for (const m of line.matchAll(MD_LINK)) {
    if (m[0].startsWith("!")) continue;
    if (at >= m.index! && at <= m.index! + m[0].length) return resolveHref(m[1], from, notes);
  }
  for (const m of line.matchAll(WIKI_LINK)) {
    if (m[0].startsWith("!")) continue;
    if (at >= m.index! && at <= m.index! + m[0].length) return resolveWiki(m[1], from, notes);
  }
  return null;
}
