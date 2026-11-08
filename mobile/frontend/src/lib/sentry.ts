// Only what errorreports.ts uses of the SDK, so the build leaves out the rest
// (replay, feedback). Loaded on its own, after the person's yes.
export { close, init } from "@sentry/svelte";
