// Pages by cursor, for any list the API serves as { items, nextCursor }:
// next() continues after the last item, prev() goes back to the cursor
// the earlier page started at. reset() starts over (a new search).
// `first` is the first page already loaded by the route's `load` (null
// when that failed), so the list shows with the page.

export type Page<T> = { items: T[]; nextCursor: string | null };

export function createPager<T>(load: (cursor: string | null) => Promise<Page<T>>, first?: Page<T> | null) {
	let items = $state<T[]>(first?.items ?? []);
	let next = $state<string | null>(first?.nextCursor ?? null);
	let busy = $state(false);
	let failed = $state(first === null);
	// The cursor each page started at (null: the first page).
	let starts = $state<(string | null)[]>([null]);

	// Only the newest request shows: an older answer that arrives late (a
	// search typed further meanwhile) is dropped.
	let seq = 0;
	async function show(cursor: string | null) {
		const mine = ++seq;
		busy = true;
		failed = false;
		try {
			const page = await load(cursor);
			if (mine !== seq) return;
			items = page.items;
			next = page.nextCursor;
		} catch {
			if (mine === seq) failed = true;
		} finally {
			if (mine === seq) busy = false;
		}
	}

	return {
		get items() { return items; },
		get busy() { return busy; },
		get failed() { return failed; },
		get page() { return starts.length; },
		get hasNext() { return next !== null; },
		get hasPrev() { return starts.length > 1; },
		reset() { starts = [null]; return show(null); },
		async next() { if (next === null) return; starts = [...starts, next]; await show(next); },
		async prev() { if (starts.length < 2) return; starts = starts.slice(0, -1); await show(starts[starts.length - 1]); },
	};
}
