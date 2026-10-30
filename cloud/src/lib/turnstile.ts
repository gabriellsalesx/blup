// Cloudflare Turnstile: the script loads once, on the sign-in page only.
type Turnstile = {
	render: (el: HTMLElement, opts: Record<string, unknown>) => string;
	reset: (id: string) => void;
	remove: (id: string) => void;
};
declare global {
	interface Window { turnstile?: Turnstile }
}

let loading: Promise<Turnstile> | null = null;

export function loadTurnstile(): Promise<Turnstile> {
	loading ??= new Promise((resolve, reject) => {
		const s = document.createElement("script");
		s.src = "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit";
		s.async = true;
		s.onload = () => (window.turnstile ? resolve(window.turnstile) : reject(new Error("turnstile")));
		s.onerror = () => { loading = null; reject(new Error("turnstile")); };
		document.head.append(s);
	});
	return loading;
}
