// The only script of the blog (~1 KB): theme choice, search dialog and the
// "copy link" button. Search itself is loaded on first use.
const root = document.documentElement;

function syncTheme() {
  const current = root.dataset.theme || "system";
  document.querySelectorAll("[data-set-theme]").forEach((b) =>
    b.setAttribute("aria-checked", String(b.dataset.setTheme === current)));
}

let searchLoaded = null;
function openSearch() {
  const dialog = document.getElementById("search");
  if (!dialog) return;
  dialog.showModal();
  dialog.querySelector("input").focus();
  searchLoaded ??= import(dialog.dataset.script).then((m) => m.init(dialog));
}

document.addEventListener("click", async (e) => {
  const theme = e.target.closest("[data-set-theme]");
  if (theme) {
    const v = theme.dataset.setTheme;
    try { localStorage.setItem("theme", v); } catch {}
    if (v === "system") delete root.dataset.theme; else root.dataset.theme = v;
    document.getElementById("theme-menu")?.hidePopover();
    // Reading progress bar of posts: share of the page already scrolled.
const bar = document.querySelector(".reading-progress");
if (bar) {
  let queued = false;
  const update = () => {
    queued = false;
    const max = document.documentElement.scrollHeight - innerHeight;
    bar.style.setProperty("--progress", max > 0 ? Math.min(1, scrollY / max) : 0);
  };
  const schedule = () => { if (!queued) { queued = true; requestAnimationFrame(update); } };
  addEventListener("scroll", schedule, { passive: true });
  addEventListener("resize", schedule);
  update();
}

// "On this page" of the docs: highlight the heading being read.
const toc = document.querySelector(".docs-toc");
if (toc && "IntersectionObserver" in window) {
  const links = new Map([...toc.querySelectorAll("a[href^='#']")].map((a) => [decodeURIComponent(a.hash.slice(1)), a]));
  const visible = new Set();
  const mark = () => {
    const id = [...links.keys()].find((k) => visible.has(k));
    links.forEach((a, k) => a.toggleAttribute("aria-current", k === id));
  };
  const io = new IntersectionObserver((entries) => {
    entries.forEach((e) => (e.isIntersecting ? visible.add(e.target.id) : visible.delete(e.target.id)));
    if (visible.size) mark();
  }, { rootMargin: "-80px 0px -60% 0px" });
  links.forEach((_, id) => { const h = document.getElementById(id); if (h) io.observe(h); });
}

syncTheme();
    return;
  }
  if (e.target.closest("[data-search-open]")) { openSearch(); return; }
  const copy = e.target.closest("[data-copy]");
  if (copy) {
    try {
      await navigator.clipboard.writeText(copy.dataset.copy);
      copy.classList.add("copied");
      setTimeout(() => copy.classList.remove("copied"), 1500);
    } catch {}
  }
  const dialog = e.target.closest("dialog");
  if (dialog && e.target === dialog) dialog.close(); // click on the backdrop
});

document.addEventListener("keydown", (e) => {
  const typing = /^(INPUT|TEXTAREA|SELECT)$/.test(document.activeElement?.tagName);
  if ((e.key === "k" && (e.metaKey || e.ctrlKey)) || (e.key === "/" && !typing)) {
    e.preventDefault();
    openSearch();
  }
});

// Reading progress bar of posts: share of the page already scrolled.
const bar = document.querySelector(".reading-progress");
if (bar) {
  let queued = false;
  const update = () => {
    queued = false;
    const max = document.documentElement.scrollHeight - innerHeight;
    bar.style.setProperty("--progress", max > 0 ? Math.min(1, scrollY / max) : 0);
  };
  const schedule = () => { if (!queued) { queued = true; requestAnimationFrame(update); } };
  addEventListener("scroll", schedule, { passive: true });
  addEventListener("resize", schedule);
  update();
}

// "On this page" of the docs: highlight the heading being read.
const toc = document.querySelector(".docs-toc");
if (toc && "IntersectionObserver" in window) {
  const links = new Map([...toc.querySelectorAll("a[href^='#']")].map((a) => [decodeURIComponent(a.hash.slice(1)), a]));
  const visible = new Set();
  const mark = () => {
    const id = [...links.keys()].find((k) => visible.has(k));
    links.forEach((a, k) => a.toggleAttribute("aria-current", k === id));
  };
  const io = new IntersectionObserver((entries) => {
    entries.forEach((e) => (e.isIntersecting ? visible.add(e.target.id) : visible.delete(e.target.id)));
    if (visible.size) mark();
  }, { rootMargin: "-80px 0px -60% 0px" });
  links.forEach((_, id) => { const h = document.getElementById(id); if (h) io.observe(h); });
}

syncTheme();
