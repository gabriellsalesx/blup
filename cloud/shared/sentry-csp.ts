// Where the panels' errors go (Blup Pulse's ingest), for the CSP's
// connect-src: the production ingest, and the origin of VITE_SENTRY_DSN
// when another one is used (a local test).
export function sentryConnect(dsn = process.env.VITE_SENTRY_DSN): string[] {
	const origins = new Set(["https://ingest.getblup.io"]);
	try {
		if (dsn) origins.add(new URL(dsn).origin);
	} catch {
		/* not a DSN: only the production ingest */
	}
	return [...origins];
}
