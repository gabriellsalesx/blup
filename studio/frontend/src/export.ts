// A note as a page of its own, to save as HTML or print (and save as PDF
// from the system's print window): headings, lists and tasks, quotes and
// callouts, code, tables, rules, images (embedded), bold, italics,
// highlights and links. Everything is escaped first; nothing runs.

const esc = (s: string) => s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");

// image(src) gives an image a note shows as a data: URL ("" when missing).
type Images = (src: string) => Promise<string>;

async function inline(s: string, image: Images): Promise<string> {
  const keep: string[] = [];
  const hold = (html: string) => `\u0000${keep.push(html) - 1}\u0000`;
  s = s.replace(/`([^`]+)`/g, (_, c) => hold(`<code>${esc(c)}</code>`));
  // Images first (their address is not text), then links.
  const imgs: Promise<void>[] = [];
  s = s.replace(/!\[\[([^\]|]+)(?:\|[^\]]*)?\]\]|!\[([^\]]*)\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g, (_, wiki, alt, src) => {
    const i = keep.push("") - 1;
    // Only an embedded image (data:image/...), escaped: an address from
    // anywhere else never reaches the page as HTML.
    imgs.push(image(wiki ? `[[${wiki}]]` : src).then((url) => { keep[i] = /^data:image\//i.test(url ?? "") ? `<img src="${esc(url)}" alt="${esc(alt ?? wiki ?? "")}">` : ""; }));
    return `\u0000${i}\u0000`;
  });
  await Promise.all(imgs);
  s = s.replace(/\[\[([^\]|#]+)(?:#[^\]|]*)?(?:\|([^\]]*))?\]\]/g, (_, name, alias) => hold(esc(alias || name)));
  s = s.replace(/\[([^\]]+)\]\(([^)\s]+)\)/g, (_, text, href) =>
    /^(https?:|mailto:)/.test(href) ? hold(`<a href="${esc(href)}">${esc(text)}</a>`) : hold(esc(text)));
  s = esc(s);
  s = s.replace(/\*\*([^*]+)\*\*/g, "<b>$1</b>").replace(/__([^_]+)__/g, "<b>$1</b>");
  s = s.replace(/(^|[^*])\*([^*]+)\*/g, "$1<i>$2</i>").replace(/(^|\W)_([^_]+)_(?=\W|$)/g, "$1<i>$2</i>");
  s = s.replace(/~~([^~]+)~~/g, "<s>$1</s>").replace(/==([^=]+)==/g, "<mark>$1</mark>");
  return s.replace(/\u0000(\d+)\u0000/g, (_, i) => keep[+i]);
}

/** The note's text (no front matter) as HTML. */
export async function noteHTML(md: string, image: Images): Promise<string> {
  const lines = md.replace(/\r/g, "").replace(/^---\n[\s\S]*?\n---\n/, "").split("\n");
  const out: string[] = [];
  let list: string[] = []; // open lists, innermost last
  const closeLists = () => { while (list.length) out.push(`</${list.pop()}>`); };
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    let m: RegExpExecArray | null;
    if ((m = /^\s*(```|~~~)\s*(\S*)/.exec(line))) {
      closeLists();
      const code: string[] = [];
      for (i++; i < lines.length && !lines[i].trim().startsWith(m[1]); i++) code.push(lines[i]);
      out.push(`<pre><code>${esc(code.join("\n"))}</code></pre>`);
      continue;
    }
    if ((m = /^(#{1,6})\s+(.*)$/.exec(line))) { closeLists(); out.push(`<h${m[1].length}>${await inline(m[2], image)}</h${m[1].length}>`); continue; }
    if (/^\s*(-{3,}|\*{3,}|_{3,})\s*$/.test(line)) { closeLists(); out.push("<hr>"); continue; }
    if (/^\s*\|.*\|\s*$/.test(line) && i + 1 < lines.length && /^\s*\|?\s*:?-{2,}/.test(lines[i + 1])) {
      closeLists();
      const cells = (l: string) => l.trim().replace(/^\||\|$/g, "").split("|").map((c) => c.trim());
      const head = cells(line);
      const rows: string[][] = [];
      for (i += 2; i < lines.length && /^\s*\|/.test(lines[i]); i++) rows.push(cells(lines[i]));
      i--;
      const th = await Promise.all(head.map((c) => inline(c, image)));
      const body = await Promise.all(rows.map(async (r) => `<tr>${(await Promise.all(r.map((c) => inline(c, image)))).map((c) => `<td>${c}</td>`).join("")}</tr>`));
      out.push(`<table><thead><tr>${th.map((c) => `<th>${c}</th>`).join("")}</tr></thead><tbody>${body.join("")}</tbody></table>`);
      continue;
    }
    if ((m = /^>\s?(.*)$/.exec(line))) {
      closeLists();
      const quote: string[] = [m[1]];
      for (i++; i < lines.length && /^>/.test(lines[i]); i++) quote.push(lines[i].replace(/^>\s?/, ""));
      i--;
      const callout = /^\[!(\w+)\][+-]?\s*(.*)$/.exec(quote[0]);
      if (callout) {
        const title = callout[2] || callout[1][0].toUpperCase() + callout[1].slice(1);
        out.push(`<aside class="callout"><b>${await inline(title, image)}</b>${await noteHTML(quote.slice(1).join("\n"), image)}</aside>`);
      } else out.push(`<blockquote>${await noteHTML(quote.join("\n"), image)}</blockquote>`);
      continue;
    }
    if ((m = /^(\s*)([-*+]|\d+[.)])\s+(\[( |x|X)\]\s+)?(.*)$/.exec(line))) {
      const depth = Math.floor(m[1].replace(/\t/g, "  ").length / 2) + 1;
      const kind = /\d/.test(m[2]) ? "ol" : "ul";
      while (list.length > depth) out.push(`</${list.pop()}>`);
      while (list.length < depth) { out.push(`<${kind}>`); list.push(kind); }
      const box = m[3] ? `<input type="checkbox" disabled${m[4] === " " ? "" : " checked"}> ` : "";
      out.push(`<li>${box}${await inline(m[5], image)}</li>`);
      continue;
    }
    if (!line.trim()) { closeLists(); continue; }
    closeLists();
    // A paragraph: the lines until a blank one or another block.
    const para = [line];
    while (i + 1 < lines.length && lines[i + 1].trim() && !/^(#{1,6}\s|>|\s*([-*+]|\d+[.)])\s|\s*(```|~~~)|\s*\||\s*(-{3,}|\*{3,}|_{3,})\s*$)/.test(lines[i + 1])) para.push(lines[++i]);
    out.push(`<p>${(await Promise.all(para.map((p) => inline(p, image)))).join("<br>")}</p>`);
  }
  closeLists();
  return out.join("\n");
}

const STYLE = `body{max-width:46rem;margin:2.5rem auto;padding:0 1.25rem;font:16px/1.65 system-ui,-apple-system,"Segoe UI",sans-serif;color:#18181b}
h1,h2,h3{line-height:1.25}img{max-width:100%;border-radius:6px}pre{background:#f4f4f5;padding:.8rem 1rem;border-radius:6px;overflow:auto}
code{font-family:ui-monospace,monospace;font-size:.9em}blockquote{margin:1rem 0;padding:.1rem 1rem;border-inline-start:3px solid #d4d4d8;color:#52525b}
.callout{margin:1rem 0;padding:.6rem 1rem;border-radius:6px;background:#eff6ff;border-inline-start:3px solid #3b82f6}
table{border-collapse:collapse;width:100%}th,td{border:1px solid #e4e4e7;padding:.4rem .6rem;text-align:start}mark{background:#fef08a}
a{color:#2563eb}hr{border:0;border-top:1px solid #e4e4e7;margin:2rem 0}`;

/** A whole HTML file of the note. */
export async function noteDocument(title: string, md: string, lang: string, image: Images): Promise<string> {
  return `<!doctype html>\n<html lang="${esc(lang)}">\n<head>\n<meta charset="utf-8">\n<meta name="viewport" content="width=device-width, initial-scale=1">\n<title>${esc(title)}</title>\n<style>${STYLE}</style>\n</head>\n<body>\n<h1>${esc(title)}</h1>\n${await noteHTML(md, image)}\n</body>\n</html>\n`;
}

/** Prints the note alone (the system's print window also saves a PDF). */
export async function printNote(title: string, md: string, image: Images) {
  const root = document.createElement("div");
  root.id = "print-note";
  root.innerHTML = `<style>@media screen{#print-note{display:none}}@media print{body>*:not(#print-note){display:none!important}#print-note{display:block}${STYLE.replace(/body\{/, "#print-note{")}}</style><h1>${esc(title)}</h1>${await noteHTML(md, image)}`;
  document.body.append(root);
  const done = () => { root.remove(); window.removeEventListener("afterprint", done); };
  window.addEventListener("afterprint", done);
  window.print();
  // Some web views don't send afterprint: cleaned up anyway.
  setTimeout(done, 60_000);
}
