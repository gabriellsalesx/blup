// The theme before the first paint (src/lib/theme.svelte.ts keeps it after).
// A file of its own, not inline, so the Content-Security-Policy needs no
// exception for it.
try {
	var t = localStorage.getItem("blup-theme");
	document.documentElement.dataset.theme = t === "light" || t === "dark" ? t : matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
} catch (e) {}
