// The panels load their data once, when a page opens; a tab left in the
// background goes stale (a device signed in, a new issue, a deal sent).
// When the tab is seen again, the page's data loads again (`refresh`, the
// panel's refreshAll), at most once every `gap` milliseconds, only on the
// pages that just show data (`when`), and never while a dialog is open or
// a field has the focus (nothing typed is lost).
type Page = Pick<Document, "visibilityState" | "activeElement" | "addEventListener" | "removeEventListener" | "querySelector">;
export const REFRESH_GAP = 30_000;

export function refreshOnReturn(refresh: () => Promise<unknown>, when: () => boolean, page: Page = document, gap = REFRESH_GAP, now = () => Date.now()): () => void {
	let last = now(), running = false;
	const seen = () => {
		if (page.visibilityState !== "visible" || running || now() - last < gap || !when()) return;
		if (page.querySelector("dialog[open]") || page.activeElement?.matches("input,textarea,select,[contenteditable]:not([contenteditable=false])")) return;
		running = true;
		last = now();
		refresh().catch(() => {}).finally(() => { running = false; last = now(); });
	};
	page.addEventListener("visibilitychange", seen);
	return () => page.removeEventListener("visibilitychange", seen);
}
