// Full-text search with Pagefind, loaded only when the dialog opens. The
// index is generated after the Hugo build by the blup CLI.
export async function init(dialog) {
  const input = dialog.querySelector("input");
  const out = dialog.querySelector(".search-results");
  const t = dialog.dataset;
  let pagefind;
  try {
    pagefind = await import(t.pagefind + "pagefind.js");
    await pagefind.options({ excerptLength: 18 });
    pagefind.init();
  } catch {
    out.innerHTML = `<p class="search-status">${t.unavailable}</p>`;
    return;
  }
  // Enter opens the first result.
  input.form?.addEventListener("submit", () => out.querySelector("a")?.click());
  let timer;
  input.addEventListener("input", () => {
    clearTimeout(timer);
    timer = setTimeout(run, 150);
  });
  async function run() {
    const q = input.value.trim();
    if (!q) { out.replaceChildren(); return; }
    const search = await pagefind.debouncedSearch(q);
    if (!search) return; // superseded by a newer query
    if (!search.results.length) {
      out.innerHTML = `<p class="search-status">${t.empty}</p>`;
      return;
    }
    const items = await Promise.all(search.results.slice(0, 8).map((r) => r.data()));
    out.innerHTML = items.map((d) =>
      `<a href="${d.url}"><strong>${escape(d.meta.title || d.url)}</strong><p>${d.excerpt}</p></a>`).join("");
  }
  const escape = (s) => s.replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" })[c]);
  if (input.value) run(); // typed while this module was loading
}
