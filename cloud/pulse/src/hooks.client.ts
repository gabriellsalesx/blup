import { startErrors } from "#lib/observability.ts";

// Errors in the browser go to Blup Pulse (VITE_SENTRY_DSN at build time).
startErrors("pulse");
export { handleError } from "#lib/observability.ts";
