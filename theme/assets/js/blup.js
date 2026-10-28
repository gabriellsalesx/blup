// The only script of the blog (~1 KB): theme choice, search dialog and the
// "copy link" button, "load more" and the reading bar. Search itself is
// loaded on first use.
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
    syncTheme();
    return;
  }
  if (e.target.closest("[data-search-open]")) { openSearch(); return; }
  if (e.target.closest("[data-search-close]")) { e.target.closest("dialog")?.close(); return; }
  const copy = e.target.closest("[data-copy]");
  if (copy) {
    // Feedback where the click happened: the icon turns into a check and a
    // small bubble (also read by screen readers) says it worked.
    const tip = copy.parentElement.querySelector(".copy-tip");
    let ok = true;
    try { await navigator.clipboard.writeText(copy.dataset.copy); } catch { ok = false; }
    const state = ok ? "copied" : "copy-failed";
    copy.classList.remove("copied", "copy-failed");
    copy.classList.add(state);
    if (tip) tip.textContent = ok ? copy.dataset.copied : copy.dataset.failed;
    clearTimeout(copy._timer);
    copy._timer = setTimeout(() => {
      copy.classList.remove(state);
      if (tip) tip.textContent = "";
    }, 2000);
    if (!ok) prompt(copy.dataset.failed, copy.dataset.copy); // copy by hand
  }
  const more = e.target.closest(".load-more a[rel=next]");
  if (more) loadMore(e, more);
  const dialog = e.target.closest("dialog");
  if (dialog && e.target === dialog) dialog.close(); // click on the backdrop
});

// "Load more": bring the next page's cards into this list, like the
// reference blog. Without JavaScript the link opens the next page.
async function loadMore(e, link) {
  const grid = link.closest("section, main")?.querySelector(".post-grid, .changelog");
  if (!grid) return;
  e.preventDefault();
  if (link.ariaBusy === "true") return;
  link.ariaBusy = "true";
  try {
    const res = await fetch(link.href);
    if (!res.ok) throw new Error(res.status);
    const doc = new DOMParser().parseFromString(await res.text(), "text/html");
    const next = doc.querySelector(grid.matches(".post-grid") ? ".post-grid" : ".changelog");
    const first = next?.firstElementChild;
    const scripts = [...(next?.querySelectorAll("script[type=module][src]") ?? [])].map((s) => s.src);
    grid.append(...(next ? next.children : []));
    // Scripts in the new items (the lock of protected posts) don't run by
    // themselves: load them and tell them about the new items.
    await Promise.all(scripts.map((src) => import(src).catch(() => {})));
    grid.dispatchEvent(new Event("blup:added", { bubbles: true }));
    first?.querySelector(".card-title, h2 a")?.focus({ preventScroll: true });
    const after = doc.querySelector(".load-more a[rel=next]");
    if (after) { link.href = after.href; link.ariaBusy = "false"; }
    else link.closest(".load-more").remove();
  } catch {
    location.href = link.href; // fall back to opening the page
  }
}

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
