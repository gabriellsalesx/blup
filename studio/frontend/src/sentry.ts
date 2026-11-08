// Only what errorreports.ts uses of the SDK, so the chunk it loads on
// demand keeps just that (a dynamic import of the whole package takes all).
export { init, close } from "@sentry/svelte";
