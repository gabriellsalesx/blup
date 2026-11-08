import type { HandleClientError } from "@sveltejs/kit/hooks";
import { BLUP_VERSION } from "../../shared/version";

// The panels' errors in the browser (Blup Cloud, Pulse, Deals), sent to
// Blup Pulse with the official Sentry SDK: errors only (no tracing, no
// replay), nothing personal. The DSN of each panel comes at build time
// (VITE_SENTRY_DSN); without it, as on this computer, nothing is sent.
// https://docs.sentry.io/platforms/javascript/guides/svelte/
// The SDK (about 90 KB) is its own chunk, loaded after the panel starts and
// only when there is a DSN, so it never delays the first screen.

type SentrySDK = typeof import("./sentry-sdk");
let sdk: Promise<SentrySDK> | undefined;

export type Panel = "cloud" | "pulse" | "deals" | "staff";

export function startErrors(panel: Panel, dsn: string | undefined = import.meta.env.VITE_SENTRY_DSN) {
	if (!dsn || typeof window === "undefined") return;
	const local = ["localhost", "127.0.0.1"].includes(location.hostname) || location.hostname.endsWith(".localhost");
	sdk = import("./sentry-sdk").then((Sentry) => { init(Sentry, panel, dsn, local); return Sentry; });
	// A failed download (offline, a new deploy) leaves the panel as it is.
	sdk.catch(() => { sdk = undefined; });
}

function init(Sentry: SentrySDK, panel: Panel, dsn: string, local: boolean) {
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
		// The trail before an error (requests, page changes) keeps addresses
		// without their query or fragment: a search (?q=) may carry an e-mail.
		beforeBreadcrumb(crumb) {
			const data = crumb.data;
			if (!data) return crumb;
			const clean: Record<string, unknown> = { ...data };
			for (const key of ["url", "from", "to"]) {
				const value = clean[key];
				if (typeof value === "string") clean[key] = value.split(/[?#]/)[0];
			}
			return { ...crumb, data: clean };
		},
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
	if (kind === "unknown") void sdk?.then((Sentry) => Sentry.captureException(error), () => {});
};
