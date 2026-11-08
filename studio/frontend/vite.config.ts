import { defineConfig, type Plugin } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import wails from "@wailsio/runtime/plugins/vite";

// Under wails3 dev, Vite's client reloads the window through a WebSocket of
// its dev server: index.html's Content-Security-Policy lets it connect there
// in dev only (the built Studio keeps connect-src 'self' plus the error reports' ingest).
const devSocket = (): Plugin => ({
  name: "blup-dev-csp",
  apply: "serve",
  transformIndexHtml: (html) => html.replace("connect-src 'self'", "connect-src 'self' ws://127.0.0.1:* ws://localhost:*"),
});

// https://vitejs.dev/config/
export default defineConfig({
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  plugins: [svelte(), wails("./bindings"), devSocket()],
  build: {
    rolldownOptions: {
      output: {
        codeSplitting: {
          // The error reports' SDK in a chunk of its own, loaded only when
          // they are on (errorreports.ts). Svelte's runtime goes first in
          // a chunk of its own: else the group takes it along (a
          // dependency of @sentry/svelte) and the opening fetches the SDK.
          groups: [
            { name: "svelte", test: /node_modules[\\/]svelte[\\/]/, priority: 2 },
            { name: "sentry", test: /node_modules[\\/]@sentry[\\/]/, priority: 1 },
          ],
        },
      },
    },
  },
});
