import * as Sentry from "@sentry/svelte";
import type { HandleClientError } from "@sveltejs/kit/hooks";
import { BLUP_VERSION } from "../../shared/version";

// The panels' errors in the browser (Blup Cloud, Pulse, Deals), sent to
// Blup Pulse with the official Sentry SDK: errors only (no tracing, no
// replay), nothing personal. The DSN of each panel comes at build time
// (VITE_SENTRY_DSN); without it, as on this computer, nothing is sent.
// https://docs.sentry.io/platforms/javascript/guides/svelte/

export type Panel = "cloud" | "pulse" | "deals" | "staff";

export function startErrors(panel: Panel, dsn: string | undefined = import.meta.env.VITE_SENTRY_DSN) {
	if (!dsn || typeof window === "undefined") return;
	const local = ["localhost", "127.0.0.1"].includes(location.hostname) || location.hostname.endsWith(".localhost");
	Sentry.init({
		dsn,
		environment: local ? "development" : "production",
		// The project's version (package.json, through shared/version.ts).
		release: `blup-paineis@${BLUP_VERSION}`,
		initialScope: { tags: { panel } },
		// No user, cookies, headers, bodies or query strings.
		dataCollection: { userInfo: false, cookies: false, httpHeaders: false, httpBodies: [], urlQueryParams: false },
		sendClientReports: false,
		// The default integrations, without the console's (a log line may
		// carry what the person typed).
		integrations: (defaults) => defaults.filter((i) => !/console/i.test(i.name)),
		beforeSend(event) {
			if (event.request?.url) event.request = { url: event.request.url.split(/[?#]/)[0] };
			delete event.user;
			return event;
		},
	});
}

// SvelteKit's own hook: an error thrown while loading or showing a page.
// Only the unexpected ones ("unknown"); a 404 or an error(...) of the app
// is not a bug.
export const handleError: HandleClientError = ({ kind, error }) => {
	if (kind === "unknown") Sentry.captureException(error);
};
