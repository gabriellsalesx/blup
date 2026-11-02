// The preview's address changed (another note opened, or the preview
// started again): is it still the page the reader shows, on the same
// server? A restarted preview has a new address (another port) with the
// same path: the old server is gone, so its page must load again.
export function samePage(next: string, before: string, path: string): boolean {
  try {
    const a = new URL(next, location.href), b = new URL(before, location.href);
    return a.origin === b.origin && a.pathname === path;
  } catch {
    return false;
  }
}

// What a preview page tells the Studio (the theme's preview script,
// mochi/scripts.html): it loaded, it scrolled, or it is ready after a
// scroll the Studio asked for; with its path, query and scroll.
export type PreviewMessage = { blup: "loaded" | "scroll" | "ready"; path: string; q: string; y: number };

// previewMessage reads a message from a preview page, or null when it is
// not one: the origin says who sent it, not that its shape is right.
export function previewMessage(d: unknown): PreviewMessage | null {
  if (typeof d !== "object" || d === null) return null;
  const { blup, path, q, y } = d as Record<string, unknown>;
  if (blup !== "loaded" && blup !== "scroll" && blup !== "ready") return null;
  if (typeof path !== "string" || !path.startsWith("/") || path.startsWith("//") || path.length > 2048 || /[\u0000-\u001f\u007f\\]/.test(path)) return null;
  if (typeof q !== "string" || (q !== "" && !q.startsWith("?")) || q.length > 4096) return null;
  if (typeof y !== "number" || !Number.isFinite(y)) return null;
  return { blup, path, q, y: Math.max(0, y) };
}
