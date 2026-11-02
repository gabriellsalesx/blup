// Light, dark or the system's, chosen per browser (a convenience: it lives
// in localStorage and the page works without it). app.html applies it
// before the first paint, so nothing flashes.
export type Theme = "system" | "light" | "dark";

const KEY = "blup-theme";

function read(): Theme {
	try {
		const v = localStorage.getItem(KEY);
		return v === "light" || v === "dark" ? v : "system";
	} catch {
		return "system";
	}
}

export const theme = $state({ value: read() });

const media = () => window.matchMedia("(prefers-color-scheme: dark)");

function applyTheme() {
	const dark = theme.value === "dark" || (theme.value === "system" && media().matches);
	document.documentElement.dataset.theme = dark ? "dark" : "light";
}

export function setTheme(v: Theme) {
	theme.value = v;
	try {
		if (v === "system") localStorage.removeItem(KEY);
		else localStorage.setItem(KEY, v);
	} catch {}
	applyTheme();
}

// The system's changes count while "system" is chosen.
if (typeof window !== "undefined") media().addEventListener("change", () => theme.value === "system" && applyTheme());
