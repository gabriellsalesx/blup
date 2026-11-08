// Only what observability.ts uses from the Sentry SDK, so the chunk it
// loads later keeps just these (importing the whole package by name
// would bring every integration along).
export { init, captureException } from "@sentry/svelte";
