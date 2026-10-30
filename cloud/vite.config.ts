import adapter from "@sveltejs/adapter-static";
import { sveltekit } from "@sveltejs/kit/vite";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";

// The panel is a single-page app served by the Worker's static assets
// (wrangler.jsonc: not_found_handling = single-page-application, which
// answers with index.html). During `vite dev`, /api goes to `wrangler dev`.
export default defineConfig({
	plugins: [tailwindcss(), sveltekit({ adapter: adapter({ fallback: "index.html" }) })],
	server: { proxy: { "/api": "http://localhost:8787" } },
});
