import { goto } from "$app/navigation";

// A card that opens a page when clicked anywhere. Its link covers it for
// the keyboard but lets the mouse through (pointer-events-none), so hover
// tips inside still work; clicks on links, buttons and menus inside do
// their own thing. Ctrl/⌘ or the middle button open a new tab.
export function openCard(event: MouseEvent, href: string) {
	if (event.target instanceof Element && event.target.closest("a,button,input,select,textarea,[popover]")) return;
	if (window.getSelection()?.toString()) return;
	if (event.ctrlKey || event.metaKey || event.button === 1) {
		window.open(href, "_blank", "noopener");
		return;
	}
	void goto(href);
}
export const cardLink = "pointer-events-none absolute inset-0 rounded-[inherit] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blup";
