// Where the panels' errors go (Blup Pulse's ingest), for the CSP's
// connect-src: the production ingest, and the origin of VITE_SENTRY_DSN
// when another one is used (a local test). Read at build time.
type Origin = `${"http" | "https"}://${string}.${string}`;
const buildEnv = (globalThis as { process?: { env: Record<string, string | undefined> } }).process?.env;
export function sentryConnect(dsn = buildEnv?.VITE_SENTRY_DSN): Origin[] {
	const origins = new Set<Origin>(["https://ingest.getblup.io"]);
	try {
		const origin = dsn ? new URL(dsn).origin : "";
		if (/^https?:\/\/[^/]+\.[^/]+$/.test(origin)) origins.add(origin as Origin);
	} catch {
		/* not a DSN: only the production ingest */
	}
	return [...origins];
}

// The visit statistics of the panels (static/stats.js loads Rybbit; the
// staff area has none) and Cloudflare Web Analytics, which the edge adds to
// the pages by itself (it reports to the page's own /cdn-cgi/rum).
export const statsScript = ["https://app.rybbit.io", "https://static.cloudflareinsights.com"] as const;
export const statsConnect = ["https://app.rybbit.io"] as const;
