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
const nodes = [...document.querySelectorAll("[data-fb-issue]")];
const API = `https://api.github.com/repos/${repo}`;
const EMOJI = { "+1": "👍", heart: "❤️", hooray: "🎉", rocket: "🚀", laugh: "😄", eyes: "👀", confused: "😕", "-1": "👎" };
const TTL = 5 * 60 * 1000; // unauthenticated GitHub API: 60 requests/hour
const API_TTL = 60 * 1000; // the login service caches for a minute
const AUTH = "blup-gh";
const STATE = "blup-gh-state";

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
  if (!want || want !== h.get("blup_state") || !h.get("blup_token")) return false;
  saveAuth({ token: h.get("blup_token"), refresh: h.get("blup_refresh"), expires: Number(h.get("blup_expires")) || 0 });
  return true;
}

function login() {
  const bytes = crypto.getRandomValues(new Uint8Array(18));
  const state = btoa(String.fromCharCode(...bytes)).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
  try { sessionStorage.setItem(STATE, state); } catch {}
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

function clearCache() {
  try {
    for (const k of Object.keys(sessionStorage)) if (k.startsWith("blup-fb:")) sessionStorage.removeItem(k);
  } catch {}
}

async function gh(url, { method = "GET", body, accept = "application/vnd.github+json", cache = method === "GET", ttl = TTL } = {}) {
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
  const res = await fetch(url, { method, headers, body: body && JSON.stringify(body) });
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
  if (!area) return login();
  const lines = (c.body || "").split("\n").filter((l) => !l.startsWith(">")).join(" ").replace(/\s+/g, " ").trim();
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
        clearCache();
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
      clearCache();
    } catch {
      this.data.reactions[k] = this.count(k) + (had ? 1 : -1);
      had ? this.mine.set(k, had) : this.mine.delete(k);
      this.render();
    }
  }

  render() { for (const v of this.views) v(); }
}

let me = null;
async function user() {
  if (!me) me = await gh("https://api.github.com/user", { cache: false });
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

// form is the comment box (or the sign in button) of the full block.
function form(node, issue, lang) {
  const link = node.querySelector(".fb-comment");
  if (!loginAPI || !link) return;
  node.querySelector(".fb-form")?.remove();
  if (!auth) {
    const b = el("button", { type: "button", className: "button fb-comment" }, link.querySelector("svg")?.cloneNode(true), t.login);
    b.onclick = login;
    const box = el("div", { className: "fb-form" }, b);
    link.hidden = true;
    link.before(box);
    return;
  }
  link.hidden = true;
  const area = el("textarea", { rows: 4, placeholder: t.placeholder || "", required: true });
  area.setAttribute("aria-label", t.placeholder || "");
  const send = el("button", { type: "submit", className: "button", textContent: t.post });
  const out = el("button", { type: "button", className: "fb-link", textContent: t.logout });
  out.onclick = logout;
  const who = el("span", { className: "fb-who" });
  const err = el("p", { className: "fb-error", hidden: true, textContent: t.error });
  const f = el("form", { className: "fb-form" }, area, el("div", { className: "fb-actions" }, who, out, send), err);
  user().then((u) => who.replaceChildren(el("img", { src: `${u.avatar_url}&s=48`, alt: "", width: 24, height: 24 }), `${t.as || ""} ${u.login}`)).catch(() => {});
  f.onsubmit = async (e) => {
    e.preventDefault();
    const text = area.value.trim();
    if (!text) return;
    send.disabled = true;
    err.hidden = true;
    try {
      const c = await gh(`${API}/issues/${issue.data.number}/comments`, {
        method: "POST",
        body: { body: text },
        accept: "application/vnd.github.full+json",
      });
      clearCache();
      area.value = "";
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
  link.before(f);
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
// (falling back to GitHub when it fails), straight to GitHub otherwise.
async function read(path, github, accept) {
  if (loginAPI && !auth) {
    try { return await gh(loginAPI + path, { ttl: API_TTL }); } catch {}
  }
  return gh(github, { accept });
}

async function run() {
  if (!repo || !nodes.length) return;
  const back = loginAPI && takeLogin();
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
    if (!issue) continue;
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
    if (node.dataset.fbMode !== "full") continue;
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
  sortBoard();
}

run();
