// Which old versions of a vault stay (worker/cleanup.ts), apart so it can
// be tested without Cloudflare's runtime.

const HOUR = 3_600_000;
const DAY = 24 * HOUR;

export type Version = { n: number; key: string; size: number; createdAt: Date };

// The versions to keep: the head, and the thinned history.
export function thin(versions: Version[], historyDays: number, now: number): Set<number> {
	const keep = new Set<number>();
	const sorted = [...versions].sort((a, b) => b.n - a.n);
	if (!sorted.length) return keep;
	keep.add(sorted[0].n);
	const seen = new Set<string>();
	for (const v of sorted.slice(1)) {
		const age = now - v.createdAt.getTime();
		if (age > historyDays * DAY) continue;
		if (age <= DAY) {
			keep.add(v.n);
			continue;
		}
		// The newest version of each hour (last week) or day (before).
		const bucket = age <= 7 * DAY ? `h${Math.floor(v.createdAt.getTime() / HOUR)}` : `d${Math.floor(v.createdAt.getTime() / DAY)}`;
		if (seen.has(bucket)) continue;
		seen.add(bucket);
		keep.add(v.n);
	}
	return keep;
}
