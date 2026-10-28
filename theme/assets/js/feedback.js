// Feedback of the Bento theme: votes, reactions and comments live in GitHub
// issues of the site's public "-feedback" repository (one issue per roadmap
// item, changelog entry and blog post, opened by the blup CLI). This reads
// them from GitHub's public API, without login; voting, reacting and
// commenting happen on GitHub, through the links already in the page.
const script = document.querySelector("script[data-fb-repo]");
const repo = script?.dataset.fbRepo;
const nodes = [...document.querySelectorAll("[data-fb-issue]")];
const API = `https://api.github.com/repos/${repo}`;
const EMOJI = { "+1": "👍", heart: "❤️", hooray: "🎉", rocket: "🚀", laugh: "😄", eyes: "👀", confused: "😕", "-1": "👎" };
const TTL = 5 * 60 * 1000; // unauthenticated GitHub API: 60 requests/hour

async function get(url, accept = "application/vnd.github+json") {
  const key = `blup-fb:${url}`;
  try {
    const hit = JSON.parse(sessionStorage.getItem(key));
    if (hit && Date.now() - hit.at < TTL) return hit.data;
  } catch {}
  const res = await fetch(url, { headers: { Accept: accept } });
  if (!res.ok) throw new Error(res.status);
  const data = await res.json();
  try { sessionStorage.setItem(key, JSON.stringify({ at: Date.now(), data })); } catch {}
  return data;
}

function chips(box, reactions, url, skipVotes) {
  box.replaceChildren();
  for (const [k, emoji] of Object.entries(EMOJI)) {
    const n = reactions?.[k] || 0;
    if (!n || (skipVotes && k === "+1")) continue; // 👍 already shown as votes
    const a = document.createElement("a");
    a.className = "fb-chip";
    a.href = url;
    a.rel = "noopener";
    a.textContent = `${emoji} ${n}`;
    box.append(a);
  }
}

function comment(c, lang) {
  const li = document.createElement("li");
  li.className = "fb-item";
  const head = document.createElement("div");
  head.className = "fb-head";
  const img = document.createElement("img");
  img.src = `${c.user.avatar_url}&s=64`;
  img.alt = "";
  img.width = img.height = 32;
  img.loading = "lazy";
  const who = document.createElement("a");
  who.href = c.user.html_url;
  who.rel = "noopener";
  who.textContent = c.user.login;
  const when = document.createElement("time");
  when.dateTime = c.created_at;
  when.textContent = new Date(c.created_at).toLocaleDateString(lang, { day: "numeric", month: "short", year: "numeric" });
  head.append(img, who, when);
  const body = document.createElement("div");
  body.className = "fb-body prose";
  // body_html is rendered and sanitized by GitHub.
  body.innerHTML = c.body_html || "";
  li.append(head, body);
  return li;
}

async function run() {
  if (!repo || !nodes.length) return;
  let list;
  try {
    list = await get(`${API}/issues?labels=blup&state=all&per_page=100`);
  } catch {
    return; // offline or rate limited: the links to GitHub still work
  }
  const issues = new Map(list.map((i) => [i.number, i]));
  const lang = document.documentElement.lang || undefined;
  for (const node of nodes) {
    const issue = issues.get(Number(node.dataset.fbIssue));
    if (!issue) continue;
    const votes = issue.reactions?.["+1"] || 0;
    node.querySelectorAll("[data-fb-votes]").forEach((el) => { el.textContent = votes; });
    node.querySelectorAll("[data-fb-ccount]").forEach((el) => { el.textContent = issue.comments || (node.dataset.fbMode === "card" ? 0 : ""); });
    const box = node.querySelector("[data-fb-chips]");
    if (box) chips(box, issue.reactions, issue.html_url, !!node.querySelector(".fb-vote"));
    node.hidden = false;
    if (node.dataset.fbMode === "full" && issue.comments) {
      try {
        const comments = await get(`${API}/issues/${issue.number}/comments?per_page=100`, "application/vnd.github.html+json");
        const ol = node.querySelector("[data-fb-list]");
        ol.replaceChildren(...comments.map((c) => comment(c, lang)));
      } catch {}
    }
  }
}

run();
