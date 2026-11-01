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
    // Relative addresses (a site without its domain yet) become full ones.
    const text = new URL(copy.dataset.copy, location.href).href;
    try { await navigator.clipboard.writeText(text); } catch { ok = false; }
    const state = ok ? "copied" : "copy-failed";
    copy.classList.remove("copied", "copy-failed");
    copy.classList.add(state);
    if (tip) tip.textContent = ok ? copy.dataset.copied : copy.dataset.failed;
    clearTimeout(copy._timer);
    copy._timer = setTimeout(() => {
      copy.classList.remove(state);
      if (tip) tip.textContent = "";
    }, 2000);
    if (!ok) prompt(copy.dataset.failed, text); // copy by hand
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

// Post narration player (listen.html): play and pause, skip 15 s, seek,
// speed. The audio loads only on the first play.
for (const box of document.querySelectorAll("[data-listen]")) {
  const audio = box.querySelector("audio");
  const play = box.querySelector("[data-listen-play]");
  const seek = box.querySelector("[data-listen-seek]");
  const now = box.querySelector("[data-listen-now]");
  const len = box.querySelector("[data-listen-len]");
  const speedBtn = box.querySelector("[data-listen-speed]");
  const speeds = [1, 1.25, 1.5, 2, 0.75];
  let speed = 0;
  audio.controls = false;
  box.classList.add("listen-ready");
  const clock = (s) => {
    s = Math.max(0, Math.floor(s || 0));
    const h = Math.floor(s / 3600), m = Math.floor((s % 3600) / 60), ss = String(s % 60).padStart(2, "0");
    return h ? `${h}:${String(m).padStart(2, "0")}:${ss}` : `${m}:${ss}`;
  };
  let dragging = false;
  const paint = () => {
    const d = audio.duration;
    if (Number.isFinite(d) && d > 0) {
      len.textContent = clock(d);
      len.hidden = false;
      if (!dragging) seek.value = String(Math.round((audio.currentTime / d) * 1000));
    }
    now.textContent = clock(audio.currentTime);
    seek.style.setProperty("--fill", `${seek.value / 10}%`);
  };
  const state = () => {
    const on = !audio.paused;
    box.classList.toggle("playing", on);
    play.setAttribute("aria-label", on ? play.dataset.pause : play.dataset.play);
  };
  play.addEventListener("click", () => {
    if (audio.paused) {
      // One narration at a time.
      document.querySelectorAll("[data-listen] audio").forEach((a) => a !== audio && a.pause());
      audio.play().catch(() => {});
    } else audio.pause();
  });
  box.querySelectorAll("[data-listen-skip]").forEach((b) => b.addEventListener("click", () => {
    audio.currentTime = Math.max(0, Math.min((audio.duration || Infinity) - 0.1, audio.currentTime + Number(b.dataset.listenSkip)));
    paint();
  }));
  seek.addEventListener("input", () => {
    dragging = true;
    if (Number.isFinite(audio.duration)) now.textContent = clock((seek.value / 1000) * audio.duration);
    seek.style.setProperty("--fill", `${seek.value / 10}%`);
  });
  seek.addEventListener("change", () => {
    dragging = false;
    if (Number.isFinite(audio.duration)) audio.currentTime = (seek.value / 1000) * audio.duration;
  });
  speedBtn.addEventListener("click", () => {
    speed = (speed + 1) % speeds.length;
    audio.playbackRate = speeds[speed];
    speedBtn.textContent = `${String(speeds[speed]).replace(".", document.documentElement.lang.startsWith("pt") ? "," : ".")}×`;
  });
  for (const ev of ["timeupdate", "loadedmetadata", "durationchange", "seeked"]) audio.addEventListener(ev, paint);
  saveButton(box.querySelector("[data-listen-save]"), audio.src);
  for (const ev of ["play", "pause", "ended"]) audio.addEventListener(ev, state);
  // The title scrolls when it doesn't fit, at a steady reading pace.
  const name = box.querySelector("[data-listen-name]");
  if (name) {
    const text = name.querySelector(".listen-track > span");
    const fit = () => {
      const w = text.offsetWidth;
      const over = w > name.clientWidth;
      name.classList.toggle("scrolling", over);
      if (over) name.style.setProperty("--marquee", `${Math.max(8, (w + 48) / 35)}s`);
    };
    fit();
    // The box, and the title itself (it widens when the font loads).
    const ro = new ResizeObserver(fit);
    ro.observe(name);
    ro.observe(text);
  }
}

// saveButton keeps a narration on the device to listen offline, in the
// cache the site's service worker plays it from ("blup-audio", see the
// worker in internal/pwa). Shown only when the worker is on (published
// site); a second tap deletes the copy.
async function saveButton(btn, src) {
  if (!btn || !("caches" in window) || !navigator.serviceWorker) return;
  if (!navigator.serviceWorker.controller) {
    // First visit: the worker takes the page over once it is installed.
    navigator.serviceWorker.addEventListener("controllerchange", () => saveButton(btn, src), { once: true });
    return;
  }
  const CACHE = "blup-audio";
  const pct = btn.querySelector(".listen-pct");
  const show = (state, text) => {
    btn.dataset.state = state;
    btn.setAttribute("aria-label", text);
    btn.title = text;
    if (state !== "saving") pct.textContent = "";
  };
  const cache = await caches.open(CACHE);
  show((await cache.match(src)) ? "saved" : "", (await cache.match(src)) ? btn.dataset.saved : btn.dataset.save);
  btn.hidden = false;
  btn.addEventListener("click", async () => {
    if (btn.dataset.state === "saving") return;
    if (btn.dataset.state === "saved") {
      await cache.delete(src);
      return show("", btn.dataset.save);
    }
    show("saving", btn.dataset.saving);
    try {
      const res = await fetch(src, { cache: "no-store" });
      if (!res.ok) throw new Error(res.status);
      // Read it piece by piece to show how much came.
      const total = Number(res.headers.get("Content-Length")) || 0;
      const reader = res.body.getReader();
      const parts = [];
      let got = 0;
      for (;;) {
        const { done, value } = await reader.read();
        if (done) break;
        parts.push(value);
        got += value.length;
        if (total) pct.textContent = `${Math.floor((got / total) * 100)}%`;
      }
      const type = res.headers.get("Content-Type") || "audio/mpeg";
      // Older versions of this narration (another ?v=) go away.
      const path = new URL(src).pathname;
      for (const req of await cache.keys()) if (new URL(req.url).pathname === path && req.url !== src) await cache.delete(req);
      await cache.put(src, new Response(new Blob(parts, { type }), { headers: { "Content-Type": type, "Content-Length": String(got) } }));
      show("saved", btn.dataset.saved);
    } catch {
      show("", btn.dataset.failed);
    }
  });
}

// The changelog's map: hovering (or focusing) a day shows its versions in
// a small box over the page (not clipped by the map's scroll).
const clMap = document.querySelector(".cl-map");
if (clMap) {
  // On a narrow screen the map scrolls: it starts at the newest months.
  const scroller = clMap.querySelector(".cl-scroll");
  if (scroller) scroller.scrollLeft = scroller.scrollWidth;
  let tip = null;
  const show = (e) => {
    const day = e.target.closest?.(".cl-day[data-tip]");
    if (!day) return;
    tip ??= document.body.appendChild(Object.assign(document.createElement("div"), { className: "cl-tip", role: "tooltip" }));
    tip.textContent = day.dataset.tip;
    tip.hidden = false;
    const r = day.getBoundingClientRect(), t = tip.getBoundingClientRect();
    const x = Math.min(Math.max(8, r.left + r.width / 2 - t.width / 2), innerWidth - t.width - 8);
    const y = r.top - t.height - 8 < 8 ? r.bottom + 8 : r.top - t.height - 8;
    tip.style.left = `${x}px`;
    tip.style.top = `${y}px`;
  };
  const hide = () => { if (tip) tip.hidden = true; };
  clMap.addEventListener("pointerover", show);
  clMap.addEventListener("focusin", show);
  clMap.addEventListener("pointerout", hide);
  clMap.addEventListener("focusout", hide);
  addEventListener("scroll", hide, { passive: true });
}
