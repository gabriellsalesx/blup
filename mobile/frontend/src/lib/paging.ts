import type { Attachment } from "svelte/attachments";

// A long list is put on screen a piece at a time: the first rows show at
// once, and more come as the list nears its end. Thousands of notes do not
// build thousands of rows to show ten.
export const PAGE = 60;

// On an empty element after the last row: calls more when that element
// comes within reach of the scrolling list it is in.
export function nearEnd(more: () => void): Attachment<HTMLElement> {
  return (el) => {
    const seen = new IntersectionObserver((entries) => {
      if (entries.some((e) => e.isIntersecting)) more();
    }, { root: el.closest(".list"), rootMargin: "800px 0px" });
    seen.observe(el);
    return () => seen.disconnect();
  };
}
