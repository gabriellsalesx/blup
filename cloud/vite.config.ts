import adapter from "@sveltejs/adapter-static";
import { sveltekit } from "@sveltejs/kit/vite";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";

// The panel is a single-page app served by the Worker's static assets
// (wrangler.jsonc: not_found_handling = single-page-application, which
// answers with index.html). During `vite dev`, /api goes to `wrangler dev`.
export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit({
			adapter: adapter({ fallback: "index.html" }),
			// Content-Security-Policy (as a <meta>, with hashes of SvelteKit's own
			// scripts). Only our files run, plus Cloudflare Turnstile on sign in.
			// Stripe's checkout and portal are pages of their own (a redirect), so
			// nothing of Stripe's loads here. The headers only HTTP can carry
			// (frame-ancestors and others) are in static/_headers and the Worker.
			csp: {
				mode: "hash",
				directives: {
					"default-src": ["self"],
					"script-src": ["self", "https://challenges.cloudflare.com"],
					"frame-src": ["https://challenges.cloudflare.com"],
					"connect-src": ["self", "https://challenges.cloudflare.com"],
					"img-src": ["self", "data:"],
					"style-src": ["self", "unsafe-inline"],
					"font-src": ["self"],
					"object-src": ["none"],
					"base-uri": ["self"],
					"form-action": ["self"],
				},
			},
		}),
	],
	server: { proxy: { "/api": "http://localhost:8787" } },
});
