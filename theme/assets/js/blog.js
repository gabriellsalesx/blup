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

syncTheme();
