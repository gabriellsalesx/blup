// The blog's podcast page (podcast/page.html): the posts' narrations as
// episodes, played one after the other by a single player at the bottom.
//
// One <audio> element plays every episode: when one ends, the next one goes
// into the same element. Phones keep playing it with the screen off, and
// iPhones only move on by themselves when it is the same element. The
// Media Session puts the episode (title and cover) and the buttons on the
// lock screen. Where the reader stopped is kept on this device, so the
// player comes back paused on that episode, at that time.

const root = document.querySelector("[data-podcast]");
const player = document.querySelector("[data-pod-player]");

if (root && player) {
  const text = JSON.parse(root.dataset.podText || "{}");
  const audio = player.querySelector("audio");
  const toggle = player.querySelector("[data-pod-toggle]");
  const nowTitle = player.querySelector("[data-pod-now]");
  const seek = player.querySelector("[data-pod-seek]");
  const cur = player.querySelector("[data-pod-cur]");
  const len = player.querySelector("[data-pod-len]");
  const speedBtn = player.querySelector("[data-pod-speed]");
  const mute = player.querySelector("[data-pod-mute]");
  const queueBtn = player.querySelector("[data-pod-queue-btn]");
  const queue = player.querySelector("[data-pod-queue]");
  const queueList = player.querySelector("[data-pod-queue-list]");
  const eps = [...root.querySelectorAll("[data-pod-ep]")];
  const KEY = `blup-podcast:${location.pathname}`;
  const speeds = [1, 1.25, 1.5, 2, 0.75];
  let speed = 0;
  let current = -1;
  let dragging = false;

  const clock = (s) => {
    if (!Number.isFinite(s)) return "0:00";
    s = Math.max(0, Math.floor(s));
    const h = Math.floor(s / 3600), m = Math.floor((s % 3600) / 60), r = String(s % 60).padStart(2, "0");
    return h ? `${h}:${String(m).padStart(2, "0")}:${r}` : `${m}:${r}`;
  };
  const ep = (i) => eps[i]?.dataset ?? {};
  const absolute = (url) => new URL(url, location.href).href;

  // --- where the reader stopped ---------------------------------------
  const save = () => {
    if (current < 0) return;
    try { localStorage.setItem(KEY, JSON.stringify({ src: ep(current).src, t: Math.floor(audio.currentTime || 0) })); } catch {}
  };
  const saved = () => { try { return JSON.parse(localStorage.getItem(KEY) || "null"); } catch { return null; } };

  // --- the episode buttons and the queue ------------------------------
  const paint = () => {
    const playing = !audio.paused && !audio.ended;
    player.classList.toggle("playing", playing);
    toggle.setAttribute("aria-label", playing ? text.stop : text.play);
    eps.forEach((li, i) => {
      const on = i === current && playing;
      const b = li.querySelector("[data-pod-play]");
      b.setAttribute("aria-pressed", String(on));
      b.querySelector("[data-pod-label]").textContent = on ? text.pause : text.listen;
    });
    queueList.querySelectorAll("button").forEach((b) => b.setAttribute("aria-current", String(Number(b.dataset.i) === current)));
  };
  const fillQueue = () => {
    queueList.replaceChildren(...eps.map((li, i) => {
      const b = document.createElement("button");
      b.type = "button";
      b.dataset.i = String(i);
      const n = document.createElement("span");
      n.textContent = String(i + 1);
      const t = document.createElement("span");
      t.textContent = li.dataset.title;
      const d = document.createElement("span");
      d.textContent = li.dataset.secs ? clock(Number(li.dataset.secs)) : "";
      b.append(n, t, d);
      const item = document.createElement("li");
      item.append(b);
      return item;
    }));
  };
  queueList.addEventListener("click", (e) => {
    const b = e.target.closest("button[data-i]");
    if (b) play(Number(b.dataset.i));
  });
  queueBtn.addEventListener("click", () => {
    const open = queue.hidden;
    queue.hidden = !open;
    queueBtn.setAttribute("aria-expanded", String(open));
  });

  // --- the lock screen ------------------------------------------------
  const session = "mediaSession" in navigator ? navigator.mediaSession : null;
  const position = () => {
    if (!session?.setPositionState || !Number.isFinite(audio.duration)) return;
    try { session.setPositionState({ duration: audio.duration, playbackRate: audio.playbackRate, position: Math.min(audio.currentTime, audio.duration) }); } catch {}
  };
  const describe = () => {
    if (!session || current < 0) return;
    const d = ep(current);
    const site = document.querySelector('meta[property="og:site_name"]')?.content || document.title;
    session.metadata = new MediaMetadata({
      title: d.title,
      artist: site,
      album: document.querySelector(".pod-page h1")?.textContent?.trim() || "",
      artwork: d.art ? [{ src: absolute(d.art), sizes: "512x512", type: "image/jpeg" }] : [],
    });
  };
  if (session) {
    const on = (action, fn) => { try { session.setActionHandler(action, fn); } catch {} };
    on("play", () => audio.play());
    on("pause", () => audio.pause());
    on("previoustrack", () => (audio.currentTime > 5 || current <= 0 ? (audio.currentTime = 0) : play(current - 1)));
    on("nexttrack", () => current < eps.length - 1 && play(current + 1));
    on("seekbackward", (d) => (audio.currentTime = Math.max(0, audio.currentTime - (d.seekOffset || 15))));
    on("seekforward", (d) => (audio.currentTime = Math.min(audio.duration || Infinity, audio.currentTime + (d.seekOffset || 15))));
    on("seekto", (d) => { if (Number.isFinite(d.seekTime)) audio.currentTime = d.seekTime; });
  }

  // --- playing --------------------------------------------------------
  // load puts episode i in the player, at time t, without playing.
  function load(i, t = 0) {
    current = i;
    const d = ep(i);
    if (audio.getAttribute("src") !== d.src) {
      audio.src = d.src;
      audio.load();
    }
    nowTitle.textContent = d.title;
    len.textContent = d.secs ? clock(Number(d.secs)) : "0:00";
    player.hidden = false;
    if (t > 0) {
      const at = () => { audio.currentTime = t; audio.removeEventListener("loadedmetadata", at); };
      if (audio.readyState >= 1) audio.currentTime = t; else audio.addEventListener("loadedmetadata", at);
    }
    describe();
    paint();
    page();
  }
  function play(i) {
    if (i !== current) load(i);
    audio.playbackRate = speeds[speed];
    audio.play().catch(() => paint());
  }

  root.addEventListener("click", (e) => {
    const b = e.target.closest("[data-pod-play]");
    if (!b) return;
    const i = eps.indexOf(b.closest("[data-pod-ep]"));
    if (i === current && !audio.paused) audio.pause();
    else play(i);
  });
  toggle.addEventListener("click", () => (audio.paused ? play(current < 0 ? 0 : current) : audio.pause()));
  audio.addEventListener("play", () => { paint(); position(); if (session) session.playbackState = "playing"; });
  audio.addEventListener("pause", () => { paint(); save(); if (session) session.playbackState = "paused"; });
  // The next episode, in the same element (see above).
  audio.addEventListener("ended", () => {
    if (current < eps.length - 1) play(current + 1);
    else { paint(); try { localStorage.removeItem(KEY); } catch {} }
  });
  let lastSave = 0;
  audio.addEventListener("timeupdate", () => {
    const d = audio.duration;
    if (Number.isFinite(d) && d > 0) {
      if (!dragging) seek.value = String(Math.round((audio.currentTime / d) * 1000));
      seek.style.setProperty("--fill", `${(audio.currentTime / d) * 100}%`);
    }
    if (!dragging) cur.textContent = clock(audio.currentTime);
    if (Date.now() - lastSave > 5000) { lastSave = Date.now(); save(); }
  });
  audio.addEventListener("loadedmetadata", () => { len.textContent = clock(audio.duration); position(); });
  audio.addEventListener("ratechange", position);

  seek.addEventListener("input", () => {
    dragging = true;
    if (Number.isFinite(audio.duration)) cur.textContent = clock((seek.value / 1000) * audio.duration);
    seek.style.setProperty("--fill", `${seek.value / 10}%`);
  });
  seek.addEventListener("change", () => {
    dragging = false;
    if (Number.isFinite(audio.duration)) audio.currentTime = (seek.value / 1000) * audio.duration;
    position();
  });
  player.querySelectorAll("[data-pod-skip]").forEach((b) => b.addEventListener("click", () => {
    audio.currentTime = Math.max(0, Math.min((audio.duration || Infinity) - 0.1, audio.currentTime + Number(b.dataset.podSkip)));
    position();
  }));
  speedBtn.addEventListener("click", () => {
    speed = (speed + 1) % speeds.length;
    audio.playbackRate = speeds[speed];
    speedBtn.textContent = `${String(speeds[speed]).replace(".", document.documentElement.lang.startsWith("en") ? "." : ",")}×`;
  });
  mute.addEventListener("click", () => {
    audio.muted = !audio.muted;
    mute.setAttribute("aria-pressed", String(audio.muted));
    mute.setAttribute("aria-label", audio.muted ? text.unmute : text.mute);
  });
  addEventListener("pagehide", save);

  // --- show notes: the post's text, brought in here -------------------
  root.addEventListener("click", async (e) => {
    const b = e.target.closest("[data-pod-notes]");
    if (!b) return;
    const li = b.closest("[data-pod-ep]");
    const box = li.querySelector("[data-pod-notes-box]");
    const open = box.hidden;
    box.hidden = !open;
    b.setAttribute("aria-expanded", String(open));
    b.textContent = open ? text.notesHide : text.notes;
    if (!open || box.dataset.loaded) return;
    box.dataset.loaded = "1";
    const link = document.createElement("a");
    link.className = "pod-open";
    link.href = li.dataset.url;
    link.textContent = text.openPost;
    try {
      const res = await fetch(li.dataset.url);
      if (!res.ok) throw new Error(String(res.status));
      // The post is a page of this same site: its text, as the theme
      // wrote it (a protected post never becomes an episode).
      const doc = new DOMParser().parseFromString(await res.text(), "text/html");
      const prose = doc.querySelector("article .prose, .nori-prose, .prose");
      if (!prose) throw new Error("no text");
      prose.querySelectorAll("script, .listen, .feedback").forEach((n) => n.remove());
      const body = document.createElement("div");
      body.className = prose.className;
      body.removeAttribute("data-pagefind-body");
      body.append(...[...prose.childNodes].map((n) => document.importNode(n, true)));
      box.replaceChildren(body, link);
    } catch {
      const p = document.createElement("p");
      p.textContent = text.notesError;
      box.replaceChildren(p, link);
    }
  });

  // --- ten at a time ------------------------------------------------------
  const more = root.querySelector("[data-pod-more]");
  const PAGE = 10;
  let shown = PAGE;
  function page() {
    eps.forEach((li, i) => (li.hidden = i >= shown && i !== current));
    if (more) more.parentElement.hidden = shown >= eps.length;
  }
  more?.addEventListener("click", (e) => {
    e.preventDefault();
    shown += PAGE;
    page();
  });

  fillQueue();
  page();
  // Back where the reader stopped: that episode, paused, at that time.
  const last = saved();
  const i = last ? eps.findIndex((li) => li.dataset.src === last.src) : -1;
  if (i >= 0) load(i, last.t || 0);
}
