import { defineConfig, type Plugin } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import wails from "@wailsio/runtime/plugins/vite";

// The page's Content Security Policy, only in the build: the dev server
// (wails3 dev) needs its own scripts and socket. Scripts only from the app;
// styles inline too (Svelte's style: and CodeMirror's); images from the app,
// data: (the notes' pictures come so), blob: and https: (a web picture in a
// note, loaded on a tap); connections to the app and over HTTPS (error
// reports, after the yes).
const csp = [
  "default-src 'self'",
  "script-src 'self'",
  "style-src 'self' 'unsafe-inline'",
  "img-src 'self' data: blob: https:",
  "font-src 'self'",
  "connect-src 'self' https:",
  "object-src 'none'",
  "base-uri 'none'",
  "form-action 'none'",
  "frame-src 'none'",
].join("; ");

const contentSecurityPolicy = (): Plugin => ({
  name: "blup-csp",
  apply: "build",
  transformIndexHtml: () => [{ tag: "meta", attrs: { "http-equiv": "Content-Security-Policy", content: csp }, injectTo: "head-prepend" }],
});

// https://vitejs.dev/config/
export default defineConfig({
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  plugins: [svelte(), wails("./bindings"), contentSecurityPolicy()],
});
