// The images of a card's Markdown (cardHTML leaves a mark where each goes).
// Only files of the vault load, as data: URLs, so what a note says never
// reaches a page or a server: an address on the web stays a link.
import { NoteImage } from "./studio";

export function cardImages(el: HTMLElement, opts: { from: string; html?: string }) {
  let from = opts.from;
  const load = () => queueMicrotask(() => {
    for (const mark of el.querySelectorAll<HTMLElement>(".card-img:not(.loading)")) {
      mark.classList.add("loading");
      NoteImage(from, mark.dataset.src ?? "").then((url) => {
        if (!url.startsWith("data:image/")) return;
        const img = document.createElement("img");
        img.decoding = "async";
        img.src = url;
        img.alt = mark.textContent ?? "";
        img.draggable = false;
        mark.replaceChildren(img);
      }).catch(() => {});
    }
  });
  load();
  return { update(next: { from: string; html?: string }) { from = next.from; load(); } };
}
