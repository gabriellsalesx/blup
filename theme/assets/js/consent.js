// Cookies: loads the site's statistics and ads tools (set in blup
// customize) and YouTube videos. With the banner, only what the visitor
// accepted: "statistics" (Google Analytics, Umami, Plausible), "marketing"
// (Meta Pixel) and "media" (videos). Google Tag Manager can hold both kinds
// of tags: it loads after either is accepted, told what was (Consent Mode),
// and hears every change. The choice stays in this browser for a
// year; "Cookies" in the footer reopens it. Without the banner, everything
// loads at once.
const script = document.querySelector("script[data-consent]");
const cfg = JSON.parse(script?.dataset.consent || "{}");
const KEY = "blup-consent";
const YEAR = 365 * 24 * 60 * 60 * 1000;

const banner = document.querySelector("[data-consent-banner]");
const dialog = document.querySelector("[data-consent-dialog]");

function saved() {
  try {
    const c = JSON.parse(localStorage.getItem(KEY));
    if (c && Date.now() - c.at < YEAR) return c;
  } catch {}
  return null;
}

function add(src, attrs = {}) {
  const s = Object.assign(document.createElement("script"), { src, async: true });
  for (const [k, v] of Object.entries(attrs)) s.setAttribute(k, v);
  document.head.append(s);
}

// --- the tools, loaded once each -------------------------------------------

const loaded = new Set();
const tools = {
  statistics() {
    if (cfg.google_analytics) {
      add(`https://www.googletagmanager.com/gtag/js?id=${encodeURIComponent(cfg.google_analytics)}`);
      window.dataLayer = window.dataLayer || [];
      window.gtag = function () { window.dataLayer.push(arguments); };
      window.gtag("js", new Date());
      window.gtag("config", cfg.google_analytics);
    }
    if (cfg.umami?.id) add(cfg.umami.src, { "data-website-id": cfg.umami.id, defer: "" });
    if (cfg.plausible?.domain) add(cfg.plausible.src, { "data-domain": cfg.plausible.domain, defer: "" });
  },
  marketing() {
    if (!cfg.meta_pixel || window.fbq) return;
    // Meta's loader: queues calls until fbevents.js arrives.
    const fbq = (window.fbq = function () {
      fbq.callMethod ? fbq.callMethod.apply(fbq, arguments) : fbq.queue.push(arguments);
    });
    window._fbq = window._fbq || fbq;
    Object.assign(fbq, { push: fbq, loaded: true, version: "2.0", queue: [] });
    add("https://connect.facebook.net/en_US/fbevents.js");
    fbq("init", cfg.meta_pixel);
    fbq("track", "PageView");
  },
  media() {
    document.querySelectorAll("[data-embed]").forEach(play);
  },
};

function gtag() {
  (window.dataLayer = window.dataLayer || []).push(arguments);
}

// Google's Consent Mode: what Google's tags (and Tag Manager's) may store.
function consentMode(c) {
  const v = (on) => (on ? "granted" : "denied");
  return { analytics_storage: v(c.statistics), ad_storage: v(c.marketing), ad_user_data: v(c.marketing), ad_personalization: v(c.marketing) };
}

function tagManager(choice) {
  if (!cfg.google_tag_manager) return;
  if (loaded.has("gtm")) {
    gtag("consent", "update", consentMode(choice));
    window.dataLayer.push({ event: "blup_consent", ...choice });
    return;
  }
  if (!choice.statistics && !choice.marketing) return;
  loaded.add("gtm");
  gtag("consent", "default", consentMode({}));
  gtag("consent", "update", consentMode(choice));
  window.dataLayer.push({ "gtm.start": Date.now(), event: "gtm.js" });
  window.dataLayer.push({ event: "blup_consent", ...choice });
  add(`https://www.googletagmanager.com/gtm.js?id=${encodeURIComponent(cfg.google_tag_manager)}`);
}

function apply(choice) {
  tagManager(choice);
  for (const k of Object.keys(tools)) {
    if (choice[k] && !loaded.has(k)) {
      loaded.add(k);
      tools[k]();
    }
  }
}

// play swaps a video's button for the player.
function play(box) {
  if (!box.dataset.embed) return;
  const f = document.createElement("iframe");
  Object.assign(f, { src: box.dataset.embed, title: box.dataset.embedTitle || "YouTube", loading: "lazy", allowFullscreen: true });
  f.allow = "accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share";
  f.referrerPolicy = "strict-origin-when-cross-origin";
  box.replaceChildren(f);
  delete box.dataset.embed;
}

// --- the visitor's choice --------------------------------------------------

function choose(choice) {
  const before = saved();
  try { localStorage.setItem(KEY, JSON.stringify({ ...choice, at: Date.now() })); } catch {}
  if (banner) banner.hidden = true;
  // Tools can't be unloaded: taking a "yes" back reloads the page without them.
  if (before && Object.keys(tools).some((k) => before[k] && !choice[k])) return location.reload();
  apply(choice);
}

const all = { statistics: true, marketing: true, media: true };
const none = { statistics: false, marketing: false, media: false };

function open() {
  if (!dialog) return;
  const c = saved() || none;
  for (const box of dialog.querySelectorAll("input[name]")) box.checked = !!c[box.name];
  dialog.returnValue = ""; // Esc or a click outside changes nothing
  dialog.showModal();
}

if (!cfg.banner) {
  apply(all);
} else {
  const c = saved();
  if (c) apply(c);
  else if (banner) banner.hidden = false;

  document.querySelector("[data-consent-all]")?.addEventListener("click", () => choose(all));
  document.querySelector("[data-consent-none]")?.addEventListener("click", () => choose(none));
  document.querySelectorAll("[data-consent-open]").forEach((b) => b.addEventListener("click", open));
  dialog?.addEventListener("close", () => {
    if (dialog.returnValue === "all") return choose(all);
    if (dialog.returnValue !== "save") return;
    const c = { ...none };
    for (const box of dialog.querySelectorAll("input[name]")) c[box.name] = box.checked;
    choose(c);
  });
  // Close by clicking outside the box.
  dialog?.addEventListener("click", (e) => { if (e.target === dialog) dialog.close(); });
  // One video, without accepting all of them.
  document.addEventListener("click", (e) => {
    const b = e.target.closest("[data-embed-load]");
    if (b) play(b.closest("[data-embed]"));
  });
}
