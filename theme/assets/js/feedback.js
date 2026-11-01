// Feedback of the Bento theme: votes, reactions and comments live in GitHub
// issues of the site's public "-feedback" repository (one issue per roadmap
// item, changelog entry and blog post, opened by the blup CLI). This reads
// them from GitHub's public API.
//
// With a login service (data-fb-api, github.com/gabriellsalesx/blup-api),
// visitors sign in with GitHub and vote, react and comment right here: the
// service only trades the login for the visitor's key, kept in this
// browser; everything else goes straight to GitHub. Without it, the links
// in the page take visitors to GitHub.
//
// Visitors who are not signed in read through the login service too: it
// reads GitHub with its own key and caches for a minute, so they don't hit
// GitHub's limit of 60 reads per hour (shared by everyone behind the same
// address, as on mobile networks). Signed in, reads go straight to GitHub
// with the visitor's key, to show their own changes at once.
const script = document.querySelector("script[data-fb-repo]");
const repo = script?.dataset.fbRepo;
const loginAPI = script?.dataset.fbApi;
const t = JSON.parse(script?.dataset.fbText || "{}");
// The page's language: a comment written in another version of the page
// (they share one conversation) says which.
const pageLang = script?.dataset.fbLang || "";
const LANG_NAMES = { pt: "Português", en: "English", es: "Español" };
const LANG_MARK = /<!--\s*blup:lang=([a-z]{2})\s*-->/;
const nodes = [...document.querySelectorAll("[data-fb-issue]")];
const API = `https://api.github.com/repos/${repo}`;
const EMOJI = { "+1": "👍", heart: "❤️", hooray: "🎉", rocket: "🚀", laugh: "😄", eyes: "👀", confused: "😕", "-1": "👎" };
const TTL = 5 * 60 * 1000; // unauthenticated GitHub API: 60 requests/hour
const API_TTL = 60 * 1000; // the login service caches for a minute
const AUTH = "blup-gh";
const STATE = "blup-gh-state";
const HASH = "blup-gh-hash";

// --- the visitor's GitHub key -------------------------------------------

let auth = null;
try { auth = JSON.parse(localStorage.getItem(AUTH)); } catch {}

function saveAuth(a) {
  auth = a;
  try { a ? localStorage.setItem(AUTH, JSON.stringify(a)) : localStorage.removeItem(AUTH); } catch {}
}

// takeLogin reads the key the login service handed back after "#".
function takeLogin() {
  const h = new URLSearchParams(location.hash.slice(1));
  if (!h.has("blup_state")) return false;
  history.replaceState(null, "", location.pathname + location.search);
  let want = null;
  try { want = sessionStorage.getItem(STATE); sessionStorage.removeItem(STATE); } catch {}
  // Back where the visitor was (an open request of the feedback board).
  try { const back = sessionStorage.getItem(HASH); sessionStorage.removeItem(HASH); if (back) history.replaceState(null, "", location.pathname + location.search + back); } catch {}
  if (!want || want !== h.get("blup_state") || !h.get("blup_token")) return false;
  saveAuth({ token: h.get("blup_token"), refresh: h.get("blup_refresh"), expires: Number(h.get("blup_expires")) || 0 });
  return true;
}

function login() {
  const bytes = crypto.getRandomValues(new Uint8Array(18));
  const state = btoa(String.fromCharCode(...bytes)).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
  try { sessionStorage.setItem(STATE, state); sessionStorage.setItem(HASH, location.hash); } catch {}
  const q = new URLSearchParams({ repo, return: location.origin + location.pathname + location.search, state });
  location.href = `${loginAPI}/login?${q}`;
}

function logout() {
  saveAuth(null);
  clearCache();
  location.reload();
}

// key returns a valid key, renewing it when it is about to expire.
async function key() {
  if (!auth) return null;
  if (auth.expires && auth.expires - 60 < Date.now() / 1000) {
    try {
      const res = await fetch(`${loginAPI}/refresh`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refresh: auth.refresh }),
      });
      if (!res.ok) throw new Error(res.status);
      const v = await res.json();
      saveAuth({ ...auth, token: v.token, refresh: v.refresh || auth.refresh, expires: Number(v.expires) || 0 });
    } catch {
      saveAuth(null);
      return null;
    }
  }
  return auth.token;
}

// --- GitHub API ----------------------------------------------------------

// changed records that the visitor just voted, reacted or commented: for
// a few minutes this browser reads straight from GitHub, skipping every
// cache, so they see their own change even after reloading.
const CHANGED = "blup-fb-changed";
const FRESH = 3 * 60 * 1000;

function changed() {
  clearCache();
  try { localStorage.setItem(CHANGED, String(Date.now())); } catch {}
}

function fresh() {
  try { return Date.now() - Number(localStorage.getItem(CHANGED)) < FRESH; } catch { return false; }
}

function clearCache() {
  try {
    for (const k of Object.keys(sessionStorage)) if (k.startsWith("blup-fb:")) sessionStorage.removeItem(k);
  } catch {}
}

async function gh(url, { method = "GET", body, accept = "application/vnd.github+json", cache = method === "GET" && !fresh(), ttl = TTL } = {}) {
  const ck = `blup-fb:${url}`;
  if (cache) {
    try {
      const hit = JSON.parse(sessionStorage.getItem(ck));
      if (hit && Date.now() - hit.at < ttl) return hit.data;
    } catch {}
  }
  const headers = { Accept: accept };
  // The visitor's key only ever goes to GitHub.
  const k = loginAPI && url.startsWith("https://api.github.com/") && (await key());
  if (k) headers.Authorization = `Bearer ${k}`;
  if (body) headers["Content-Type"] = "application/json";
  // After a change, ask GitHub again instead of the browser's own copy
  // (GitHub lets browsers keep answers for a minute).
  const res = await fetch(url, { method, headers, body: body && JSON.stringify(body), cache: fresh() ? "no-cache" : "default" });
  if (res.status === 401 && k) saveAuth(null); // revoked on GitHub
  if (!res.ok) throw new Error(res.status);
  const data = res.status === 204 ? null : await res.json();
  if (cache) try { sessionStorage.setItem(ck, JSON.stringify({ at: Date.now(), data })); } catch {}
  return data;
}

// --- rendering -----------------------------------------------------------

function el(tag, props = {}, ...kids) {
  const e = Object.assign(document.createElement(tag), props);
  e.append(...kids.filter((k) => k != null));
  return e;
}

function comment(c, lang) {
  const img = el("img", { src: `${c.user.avatar_url}&s=64`, alt: "", width: 32, height: 32, loading: "lazy" });
  const who = el("a", { href: c.user.html_url, rel: "noopener", textContent: c.user.login });
  const when = el("time", {
    dateTime: c.created_at,
    textContent: new Date(c.created_at).toLocaleDateString(lang, { day: "numeric", month: "short", year: "numeric" }),
  });
  const body = el("div", { className: "fb-body prose" });
  // body_html is rendered and sanitized by GitHub.
  body.innerHTML = c.body_html || "";
  const head = el("div", { className: "fb-head" }, img, who, when);
  const written = LANG_MARK.exec(c.body || "")?.[1];
  if (written && pageLang && written !== pageLang && t.inLang) {
    head.append(el("span", { className: "fb-lang", textContent: t.inLang.replace("{lang}", LANG_NAMES[written] || written) }));
  }
  if (loginAPI) {
    const b = el("button", { type: "button", className: "fb-link fb-reply", textContent: t.reply });
    b.onclick = () => reply(li, c);
    head.append(b);
  }
  const li = el("li", { className: "fb-item" }, head, body);
  li.dataset.id = c.id;
  li.dataset.author = c.user.id;
  return li;
}

// reply fills the comment box with a quote of c and a mention of its author
// (GitHub notifies them). Comments on an issue are a single list, so the
// quote shows what the answer is about.
function reply(li, c) {
  const area = li.closest("[data-fb-issue]")?.querySelector(".fb-form textarea");
  if (!area || area.disabled) return login();
  area.closest(".fb-form")?.querySelector('[data-fb-tab="write"]')?.click();
  const lines = (c.body || "").replace(LANG_MARK, "").split("\n").filter((l) => !l.startsWith(">")).join(" ").replace(/\s+/g, " ").trim();
  const quote = lines.length > 200 ? `${lines.slice(0, 200)}…` : lines;
  area.value = `${quote ? `> ${quote}\n\n` : ""}@${c.user.login} ${area.value}`;
  area.scrollIntoView({ block: "center" });
  area.focus();
  area.setSelectionRange(area.value.length, area.value.length);
}

// deletable adds "Delete" to the visitor's own comments, and to every
// comment when the visitor owns the repository (GitHub allows both).
async function deletable(node, issue) {
  if (!loginAPI || !auth) return;
  let u;
  try { u = await user(); } catch { return; }
  const owner = u.login.toLowerCase() === repo.split("/")[0].toLowerCase();
  for (const li of node.querySelectorAll(".fb-item")) {
    if (li.querySelector(".fb-delete") || !(owner || li.dataset.author === String(u.id))) continue;
    const b = el("button", { type: "button", className: "fb-link fb-delete", textContent: t.delete });
    b.onclick = async () => {
      if (!confirm(t.deleteQ)) return;
      b.disabled = true;
      try {
        await gh(`${API}/issues/comments/${li.dataset.id}`, { method: "DELETE" });
        changed();
        li.remove();
        issue.data.comments = Math.max(0, (issue.data.comments || 1) - 1);
        issue.render();
      } catch {
        b.disabled = false;
        alert(t.error);
      }
    };
    li.querySelector(".fb-head").append(b);
  }
}

// Issue is the live state of one page's issue, shared by every element of
// the page that shows it.
class Issue {
  constructor(data) {
    this.data = data;
    this.mine = null; // the visitor's reactions: content -> reaction id
    this.views = [];
  }

  get url() { return this.data.html_url; }
  count(k) { return this.data.reactions?.[k] || 0; }

  async loadMine() {
    if (!auth || this.mine) return;
    try {
      const me = await user();
      const list = await gh(`${API}/issues/${this.data.number}/reactions?per_page=100`, { cache: false });
      this.mine = new Map(list.filter((r) => r.user?.id === me.id).map((r) => [r.content, r.id]));
    } catch {
      this.mine = new Map();
    }
  }

  async toggle(k) {
    if (!loginAPI) return void (location.href = this.url);
    if (!(await key())) return login();
    await this.loadMine();
    const had = this.mine.get(k);
    // Show it at once; GitHub confirms below.
    this.data.reactions[k] = this.count(k) + (had ? -1 : 1);
    had ? this.mine.delete(k) : this.mine.set(k, 0);
    this.render();
    try {
      if (had) {
        await gh(`${API}/issues/${this.data.number}/reactions/${had}`, { method: "DELETE" });
      } else {
        const r = await gh(`${API}/issues/${this.data.number}/reactions`, { method: "POST", body: { content: k } });
        this.mine.set(k, r.id);
      }
      changed();
    } catch {
      this.data.reactions[k] = this.count(k) + (had ? 1 : -1);
      had ? this.mine.set(k, had) : this.mine.delete(k);
      this.render();
    }
  }

  render() { for (const v of this.views) v(); }
}

// user is the signed in visitor, asked once per page (shared promise).
let me = null;
function user() {
  me ||= gh("https://api.github.com/user", { cache: false }).catch((e) => { me = null; throw e; });
  return me;
}

// chips shows the reaction counts; with the login service each one is a
// button that adds or removes the visitor's reaction.
function chips(box, issue, skipVotes) {
  box.replaceChildren();
  for (const [k, emoji] of Object.entries(EMOJI)) {
    const n = issue.count(k);
    if (!n || (skipVotes && k === "+1")) continue; // 👍 already shown as votes
    const text = `${emoji} ${n}`;
    if (loginAPI) {
      const b = el("button", { type: "button", className: "fb-chip", textContent: text });
      b.setAttribute("aria-pressed", String(!!issue.mine?.has(k)));
      b.onclick = () => issue.toggle(k);
      box.append(b);
    } else {
      box.append(el("a", { className: "fb-chip", href: issue.url, rel: "noopener", textContent: text }));
    }
  }
}

// picker turns the "react" link into a small menu of reactions.
function picker(add, issue) {
  if (!loginAPI || add.dataset.fbPicker) return;
  add.dataset.fbPicker = "1";
  add.querySelector("span")?.replaceChildren(t.react || "");
  const menu = el("div", { className: "fb-picker", hidden: true, role: "menu" });
  for (const [k, emoji] of Object.entries(EMOJI)) {
    const b = el("button", { type: "button", textContent: emoji, role: "menuitem" });
    b.setAttribute("aria-label", k);
    b.onclick = () => { menu.hidden = true; issue.toggle(k); };
    menu.append(b);
  }
  add.after(menu);
  add.addEventListener("click", (e) => {
    e.preventDefault();
    if (!auth) return login();
    menu.hidden = !menu.hidden;
    if (!menu.hidden) issue.loadMine().then(() => issue.render());
  });
  document.addEventListener("click", (e) => {
    if (!menu.hidden && !menu.contains(e.target) && !add.contains(e.target)) menu.hidden = true;
  });
}

// form drives the comment box of the full block. The page draws it locked
// (a sample of where comments go); signed in, it unlocks: write, preview
// (GitHub renders the Markdown) and post.
function form(node, issue, lang) {
  const box = node.querySelector("[data-fb-box]");
  if (!loginAPI || !box) return;
  const area = box.querySelector("textarea");
  const button = box.querySelector(".fb-comment");
  if (!auth) {
    button.addEventListener("click", (e) => { e.preventDefault(); login(); });
    return;
  }
  const send = el("button", { type: "submit", className: "button fb-comment" }, button.querySelector("svg")?.cloneNode(true) || "", t.post);
  button.replaceWith(send);
  area.disabled = false;
  area.placeholder = t.placeholder || "";
  const who = box.querySelector(".fb-who");
  const out = box.querySelector("[data-fb-logout]");
  out.hidden = false;
  out.onclick = logout;
  user().then((u) => { who.replaceChildren(el("img", { src: `${u.avatar_url}&s=48`, alt: "", width: 20, height: 20 }), `${t.as || ""} ${u.login}`); who.hidden = false; }).catch(() => {});
  // Write and Preview tabs.
  const preview = box.querySelector(".fb-preview");
  const tabs = [...box.querySelectorAll("[data-fb-tab]")];
  tabs.forEach((b) => { b.disabled = false; });
  box.querySelector(".fb-tabs").addEventListener("click", async (e) => {
    const tab = e.target.closest("[data-fb-tab]");
    if (!tab || tab.getAttribute("aria-selected") === "true") return;
    tabs.forEach((b) => b.setAttribute("aria-selected", String(b === tab)));
    const show = tab.dataset.fbTab === "preview";
    area.hidden = show;
    preview.hidden = !show;
    if (!show) return area.focus();
    if (!area.value.trim()) return void preview.replaceChildren(el("p", { className: "fb-empty", textContent: t.nothing || "" }));
    preview.setAttribute("aria-busy", "true");
    try {
      const res = await fetch("https://api.github.com/markdown", {
        method: "POST",
        headers: { Accept: "application/vnd.github+json", "Content-Type": "application/json", Authorization: `Bearer ${await key()}` },
        body: JSON.stringify({ text: area.value, mode: "gfm", context: repo }),
      });
      if (!res.ok) throw new Error(res.status);
      preview.innerHTML = await res.text(); // GitHub's own sanitized HTML
    } catch {
      preview.replaceChildren(el("p", { className: "fb-error", textContent: t.error || "" }));
    }
    preview.removeAttribute("aria-busy");
  });
  const err = box.querySelector(".fb-error");
  box.onsubmit = async (e) => {
    e.preventDefault();
    const text = area.value.trim();
    if (!text) return;
    send.disabled = true;
    err.hidden = true;
    try {
      const c = await gh(`${API}/issues/${issue.data.number}/comments`, {
        method: "POST",
        // An invisible mark of the page's language (an HTML comment, which
        // GitHub doesn't show).
        body: { body: pageLang ? `${text}\n\n<!-- blup:lang=${pageLang} -->` : text },
        accept: "application/vnd.github.full+json",
      });
      changed();
      area.value = "";
      tabs[0].click();
      issue.data.comments = (issue.data.comments || 0) + 1;
      const ol = node.querySelector("[data-fb-list]");
      ol.querySelector("[data-fb-empty]")?.remove();
      ol.append(comment(c, lang));
      deletable(node, issue);
      issue.render();
    } catch {
      err.hidden = false;
    }
    send.disabled = false;
  };
}

// sortBoard orders the roadmap's open columns by votes (a stable sort: the
// weight order breaks ties) and marks the most voted card.
function sortBoard() {
  const board = document.querySelector(".board[data-fb-sort]");
  if (!board) return;
  const votes = (li) => {
    const mini = li.querySelector(".fb-mini");
    return mini && !mini.hidden ? Number(mini.querySelector("[data-fb-votes]")?.textContent) || 0 : 0;
  };
  let top = null;
  for (const ul of board.querySelectorAll('.column:not([data-status="done"]) ul')) {
    const items = [...ul.children].sort((a, b) => votes(b) - votes(a));
    ul.append(...items);
    for (const li of items) if (votes(li) > (top ? votes(top) : 0)) top = li;
  }
  if (top && t.top) top.querySelector(".roadmap-card")?.prepend(el("span", { className: "fb-top", textContent: t.top }));
}

// read goes through the login service for visitors who are not signed in
// (falling back to GitHub when it fails), straight to GitHub otherwise and
// right after the visitor's own change (the service caches for a minute).
async function read(path, github, accept) {
  if (loginAPI && !auth && !fresh()) {
    try { return await gh(loginAPI + path, { ttl: API_TTL }); } catch {}
  }
  return gh(github, { accept });
}

// attach brings one block of the page to life with its issue: votes,
// reactions and, for the full block, the comments and the comment box.
async function attach(node, issue, lang, back) {
  const vote = node.querySelector(".fb-vote");
  const box = node.querySelector("[data-fb-chips]");
  const view = () => {
    const votes = issue.count("+1");
    node.querySelectorAll("[data-fb-votes]").forEach((e) => { e.textContent = votes; });
    node.querySelectorAll("[data-fb-ccount]").forEach((e) => { e.textContent = issue.data.comments || (node.dataset.fbMode === "card" ? 0 : ""); });
    vote?.setAttribute("aria-pressed", String(!!issue.mine?.has("+1")));
    if (box) chips(box, issue, !!vote);
  };
  issue.views.push(view);
  if (vote && loginAPI) vote.onclick = (e) => { e.preventDefault(); issue.toggle("+1"); };
  const add = node.querySelector(".fb-add");
  if (add) picker(add, issue);
  view();
  node.hidden = false;
  if (node.dataset.fbMode !== "full") return;
  if (loginAPI) node.querySelector(".fb-note")?.replaceChildren(t.note || "");
  form(node, issue, lang);
  if (auth) issue.loadMine().then(() => issue.render());
  if (issue.data.comments) {
    try {
      const n = issue.data.number;
      const comments = await read(
        `/comments?repo=${encodeURIComponent(repo)}&issue=${n}`,
        `${API}/issues/${n}/comments?per_page=100`,
        "application/vnd.github.full+json",
      );
      node.querySelector("[data-fb-list]").replaceChildren(...comments.map((c) => comment(c, lang)));
      deletable(node, issue);
    } catch {}
  }
  if (back) node.scrollIntoView({ block: "start" });
}

async function run() {
  if (!repo || !nodes.length) return;
  let list;
  try {
    list = await read(`/issues?repo=${encodeURIComponent(repo)}`, `${API}/issues?labels=blup&state=all&per_page=100`);
  } catch {
    return; // offline or rate limited: the links to GitHub still work
  }
  const issues = new Map(list.map((i) => [i.number, new Issue(i)]));
  const lang = document.documentElement.lang || undefined;
  for (const node of nodes) {
    const issue = issues.get(Number(node.dataset.fbIssue));
    if (issue) await attach(node, issue, lang, back);
  }
  sortBoard();
  sortTabs(issues);
}

// sortTabs: Nori's "Latest / Top / Discussions" tabs above the list of
// posts. They show only when the posts have reactions and comments; Top
// orders by reactions and Discussions by comments, after bringing in the
// posts of the next pages (the list is paginated).
function sortTabs(issues) {
  const bar = document.querySelector("[data-fb-tabs]");
  const list = document.querySelector(".nori-list");
  if (!bar || !list) return;
  const all = [...issues.values()];
  if (!all.some((i) => i.data.reactions?.total_count) || !all.some((i) => i.data.comments)) return;
  const score = (li, kind) => {
    const i = issues.get(Number(li.dataset.fbIssue));
    if (!i) return 0;
    return kind === "top" ? i.data.reactions?.total_count || 0 : i.data.comments || 0;
  };
  let order = 0;
  const number = () => list.querySelectorAll(":scope > li:not([data-order])").forEach((li) => { li.dataset.order = order++; });
  number();
  let loaded = null;
  const loadAll = () => (loaded ||= (async () => {
    let next = document.querySelector(".load-more a[rel=next]");
    while (next) {
      const doc = new DOMParser().parseFromString(await (await fetch(next.href)).text(), "text/html");
      list.append(...doc.querySelectorAll(".nori-list > li"));
      next = doc.querySelector(".load-more a[rel=next]");
    }
    document.querySelector(".load-more")?.remove();
    number();
  })());
  bar.addEventListener("click", async (e) => {
    const tab = e.target.closest("[data-fb-sort]");
    if (!tab || tab.getAttribute("aria-selected") === "true") return;
    bar.querySelectorAll("[data-fb-sort]").forEach((b) => b.setAttribute("aria-selected", String(b === tab)));
    const kind = tab.dataset.fbSort;
    if (kind !== "recent") {
      list.setAttribute("aria-busy", "true");
      try { await loadAll(); } catch {}
      list.removeAttribute("aria-busy");
    }
    const rows = [...list.children];
    rows.sort((a, b) => (kind === "recent" ? 0 : score(b, kind) - score(a, kind)) || a.dataset.order - b.dataset.order);
    list.append(...rows);
  });
  bar.hidden = false;
}

// The login handed back after "#", read once for the whole page.
const back = loginAPI && takeLogin();
run();
board();

// --- the feedback board (Bento's feedback section) -------------------------

// A request of the board is an issue a visitor opened from it, marked in
// its text (an HTML comment GitHub doesn't show) with its type and the
// language of the page it came from. Its status is a "status: ..." label
// the site's owner sets in Blup Studio; a closed request without one is done.
const POST_MARK = /<!--\s*blup:feedback\s+kind=([a-z]+)(?:\s+lang=([a-z]{2}))?\s*-->/;
const KINDS = ["feature", "bug", "improvement", "other"];

function postKind(p) { const k = POST_MARK.exec(p.body || "")?.[1]; return KINDS.includes(k) ? k : "other"; }
function postLang(p) { return POST_MARK.exec(p.body || "")?.[2] || ""; }
function postStatus(p) {
  const l = (p.labels || []).map((x) => (typeof x === "string" ? x : x?.name || "")).find((n) => n.startsWith("status:"));
  if (l) return l.slice(7).trim();
  return p.state === "closed" ? "done" : "new";
}

async function board() {
  const root = document.querySelector("[data-fb-board]");
  if (!root || !repo) return;
  const bt = JSON.parse(root.dataset.fbbText || "{}");
  const lang = document.documentElement.lang || undefined;
  const list = root.querySelector("[data-fbb-list]");
  const detail = root.querySelector("[data-fbb-detail]");
  const bar = root.querySelector(".fbb-bar");
  const state = { kind: "all", status: "", sort: "top", q: "" };
  let posts = [];
  // In Blup Studio's preview, example requests (the section note's
  // "examples") fill the board until a real one comes.
  const examples = [...(root.querySelector("[data-fbb-examples]")?.content.children || [])];
  const note = root.querySelector("[data-fbb-note]");

  try {
    const raw = await read(
      `/posts?repo=${encodeURIComponent(repo)}`,
      `${API}/issues?state=all&sort=created&direction=desc&per_page=100`,
      "application/vnd.github.full+json",
    );
    posts = raw.filter((p) => !p.pull_request && POST_MARK.test(p.body || "")).map((p) => new Issue(p));
  } catch {
    if (!examples.length) {
      list.replaceChildren(el("li", { className: "fbb-empty", textContent: bt.error }));
      return;
    }
  }

  // The list: the requests that pass the filters, in the chosen order.
  const chip = (cls, text) => el("span", { className: `fbb-chip ${cls}`, textContent: text });
  const plain = (html) => { const d = document.createElement("div"); d.innerHTML = html || ""; return d.textContent || ""; };
  const row = (issue) => {
    const p = issue.data;
    const kind = postKind(p), status = postStatus(p), written = postLang(p);
    const vote = el("button", { type: "button", className: "fbb-vote" }, el("span", { className: "fbb-up", textContent: "▲" }), el("b", { textContent: String(issue.count("+1")) }));
    vote.setAttribute("aria-label", `${bt.votes}: ${issue.count("+1")}`);
    vote.setAttribute("aria-pressed", String(!!issue.mine?.has("+1")));
    vote.onclick = () => issue.toggle("+1");
    const open = el("a", { href: `#p-${p.number}`, className: "fbb-title", textContent: p.title });
    const meta = el("div", { className: "fbb-meta" },
      chip(`k-${kind}`, bt.kinds?.[kind] || kind),
      chip(`s-${status}`, bt.statuses?.[status] || status),
      el("span", { textContent: p.user?.login || "" }),
      el("span", { textContent: `💬 ${p.comments || 0}` }));
    if (written && written !== pageLang) meta.append(el("span", { className: "fb-lang", textContent: (bt.written || "").replace("{lang}", LANG_NAMES[written] || written) }));
    return el("li", { className: "fbb-item" }, vote, el("div", { className: "fbb-main" }, open, meta));
  };
  const counts = () => {
    for (const tab of root.querySelectorAll(".fbb-tab")) {
      const k = tab.dataset.kind;
      const n = posts.length ? posts.filter((i) => k === "all" || postKind(i.data) === k).length : examples.filter((x) => k === "all" || x.dataset.kind === k).length;
      tab.querySelector("[data-count]").textContent = String(n);
    }
  };
  const draw = () => {
    const q = state.q.trim().toLowerCase();
    if (note) note.hidden = !!posts.length;
    if (!posts.length && examples.length) {
      const shown = examples
        .filter((x) => state.kind === "all" || x.dataset.kind === state.kind)
        .filter((x) => !state.status || x.dataset.status === state.status)
        .filter((x) => !q || x.textContent.toLowerCase().includes(q));
      list.replaceChildren(...(shown.length ? shown : [el("li", { className: "fbb-empty", textContent: bt.nothing })]));
      counts();
      return;
    }
    const shown = posts
      .filter((i) => state.kind === "all" || postKind(i.data) === state.kind)
      .filter((i) => !state.status || postStatus(i.data) === state.status)
      .filter((i) => !q || i.data.title.toLowerCase().includes(q) || plain(i.data.body_html).toLowerCase().includes(q))
      .sort((a, b) => (state.sort === "top" ? b.count("+1") - a.count("+1") : 0) || Date.parse(b.data.created_at) - Date.parse(a.data.created_at));
    list.replaceChildren(...(shown.length ? shown.map(row) : [el("li", { className: "fbb-empty", textContent: posts.length ? bt.nothing : bt.empty })]));
    counts();
  };
  for (const i of posts) i.views.push(() => { if (!list.hidden) draw(); });

  root.querySelector(".fbb-tabs").addEventListener("click", (e) => {
    const tab = e.target.closest(".fbb-tab");
    if (!tab) return;
    root.querySelectorAll(".fbb-tab").forEach((b) => b.setAttribute("aria-selected", String(b === tab)));
    state.kind = tab.dataset.kind;
    draw();
  });
  root.querySelector("[data-fbb-status]").addEventListener("change", (e) => { state.status = e.target.value; draw(); });
  root.querySelector(".fbb-sort").addEventListener("click", (e) => {
    const b = e.target.closest("[data-sort]");
    if (!b) return;
    root.querySelectorAll(".fbb-sort [data-sort]").forEach((x) => x.setAttribute("aria-selected", String(x === b)));
    state.sort = b.dataset.sort;
    draw();
  });
  root.querySelector("[data-fbb-search]").addEventListener("input", (e) => { state.q = e.target.value; draw(); });

  // One request on its own (#p-12): its text, then the block of votes,
  // reactions and comments every page of the site uses.
  const show = async () => {
    const n = Number(/^#p-(\d+)$/.exec(location.hash)?.[1]);
    const issue = n && posts.find((i) => i.data.number === n);
    if (!issue) {
      detail.hidden = true;
      detail.replaceChildren();
      list.hidden = bar.hidden = false;
      draw();
      return;
    }
    list.hidden = bar.hidden = true;
    const p = issue.data;
    const kind = postKind(p), status = postStatus(p), written = postLang(p);
    const backLink = el("a", { href: "#", className: "fbb-back", textContent: `← ${bt.back}` });
    backLink.onclick = (e) => { e.preventDefault(); history.pushState(null, "", location.pathname + location.search); show(); };
    const meta = el("div", { className: "fbb-meta" }, chip(`k-${kind}`, bt.kinds?.[kind] || kind), chip(`s-${status}`, bt.statuses?.[status] || status),
      el("a", { href: p.user?.html_url, rel: "noopener", textContent: p.user?.login || "" }),
      el("time", { dateTime: p.created_at, textContent: new Date(p.created_at).toLocaleDateString(lang, { day: "numeric", month: "short", year: "numeric" }) }));
    if (written && written !== pageLang) meta.append(el("span", { className: "fb-lang", textContent: (bt.written || "").replace("{lang}", LANG_NAMES[written] || written) }));
    const body = el("div", { className: "prose fbb-body" });
    body.innerHTML = p.body_html || ""; // rendered and sanitized by GitHub
    const block = root.querySelector("[data-fbb-block]").content.firstElementChild.cloneNode(true);
    block.dataset.fbIssue = String(p.number);
    block.querySelectorAll('a[href^="https://github.com/"]').forEach((a) => { a.href = a.href.replace(/\/issues(#.*)?$/, `/issues/${p.number}$1`); });
    detail.replaceChildren(backLink, el("h2", { className: "fbb-detail-title", textContent: p.title }), meta, body, block);
    detail.hidden = false;
    detail.scrollIntoView({ block: "start" });
    await attach(block, issue, lang, false);
  };
  addEventListener("hashchange", show);
  addEventListener("popstate", show);

  // A new request: signed in, a small form; without the login service, the
  // new issue form on GitHub.
  const dialog = root.querySelector("[data-fbb-dialog]");
  const form = dialog.querySelector("form");
  const err = form.querySelector(".fb-error");
  root.querySelector("[data-fbb-new]").addEventListener("click", async (e) => {
    if (!loginAPI) return; // the link goes to GitHub
    e.preventDefault();
    if (!(await key())) return login();
    err.hidden = true;
    dialog.showModal();
    form.elements.title.focus();
  });
  form.querySelector("[data-fbb-cancel]").onclick = () => dialog.close();
  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    // form.elements: form.title is the form's own title attribute.
    const f = form.elements;
    const title = f.title.value.trim();
    if (!title) return;
    const kind = f.kind.value || "other";
    const send = form.querySelector("[data-fbb-send]");
    send.disabled = true;
    err.hidden = true;
    try {
      const mark = `<!-- blup:feedback kind=${kind}${pageLang ? ` lang=${pageLang}` : ""} -->`;
      const p = await gh(`${API}/issues`, {
        method: "POST",
        body: { title, body: `${f.body.value.trim()}\n\n${mark}` },
        accept: "application/vnd.github.full+json",
      });
      changed();
      const issue = new Issue({ ...p, reactions: p.reactions || {} });
      issue.views.push(() => { if (!list.hidden) draw(); });
      posts.unshift(issue);
      form.reset();
      dialog.close();
      location.hash = `p-${p.number}`;
    } catch {
      err.hidden = false;
    }
    send.disabled = false;
  });

  await show();
  if (back) root.scrollIntoView({ block: "start" });
}
