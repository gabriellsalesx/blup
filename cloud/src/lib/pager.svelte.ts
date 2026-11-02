// Pages by cursor, for any list the API serves as { items, nextCursor }:
// next() continues after the last item, prev() goes back to the cursor
// the earlier page started at. reset() starts over (a new search).

type Page<T> = { items: T[]; nextCursor: string | null };

export function createPager<T>(load: (cursor: string | null) => Promise<Page<T>>) {
	let items = $state<T[]>([]);
	let next = $state<string | null>(null);
	let busy = $state(false);
	let failed = $state(false);
	// The cursor each page started at (null: the first page).
	let starts = $state<(string | null)[]>([null]);

	async function show(cursor: string | null) {
		busy = true;
		failed = false;
		try {
			const page = await load(cursor);
			items = page.items;
			next = page.nextCursor;
		} catch {
			failed = true;
		} finally {
			busy = false;
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
