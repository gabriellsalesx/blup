// AppSumo's files, read in the browser before anything is sent (the staff's
// AppSumo page). The server normalizes and checks each code; here they are
// only found, trimmed and taken once.

/** The cells of a CSV text (RFC 4180: quotes, "" inside quotes, CRLF). */
export function csvRows(text: string): string[][] {
	const rows: string[][] = [];
	let row: string[] = [], cell = "", quoted = false;
	for (let i = 0; i < text.length; i++) {
		const ch = text[i];
		if (quoted) {
			if (ch === '"' && text[i + 1] === '"') { cell += '"'; i++; }
			else if (ch === '"') quoted = false;
			else cell += ch;
		} else if (ch === '"') quoted = true;
		else if (ch === ",") { row.push(cell); cell = ""; }
		else if (ch === "\n" || ch === "\r") {
			if (ch === "\r" && text[i + 1] === "\n") i++;
			row.push(cell); rows.push(row); row = []; cell = "";
		} else cell += ch;
	}
	if (cell || row.length) { row.push(cell); rows.push(row); }
	return rows;
}

// A UTF-8 byte order mark at the start of a file (Excel writes one).
const BOM = /^\uFEFF/;
const once = (codes: Iterable<string>) => [...new Set([...codes].map((c) => c.trim()).filter(Boolean))];

/** The codes to import: one per line, no header. */
export function codesFromList(text: string): string[] {
	return once(text.replace(BOM, "").split(/\r?\n/).map((line) => line.trim().replace(/^"(.*)"$/, "$1")));
}

/** Our codes, anywhere in a text: they start with BLUP, maybe hyphenated (the server normalizes). */
const ours = /\bBLUP[A-Z0-9-]{4,80}\b/gi;

/**
 * The codes of AppSumo's refunds CSV: the code column (a header that is
 * just "code", "codes", "redemption code" or "license code" first, else
 * the first header with "code" in it, so "Country Code" loses to
 * "Redemption Code"); without one, every token that looks like our codes.
 */
export function codesFromRefunds(text: string): string[] {
	const [header, ...rows] = csvRows(text.replace(BOM, ""));
	const exact = header?.findIndex((name) => /^(redemption |license )?codes?$/i.test(name.trim())) ?? -1;
	const column = exact >= 0 ? exact : header?.findIndex((name) => /code/i.test(name)) ?? -1;
	if (column >= 0) return once(rows.map((r) => r[column] ?? ""));
	return once(text.match(ours) ?? []);
}

/** A list in parts of at most `size`, in order. */
export function chunks<T>(list: readonly T[], size: number): T[][] {
	const parts: T[][] = [];
	for (let i = 0; i < list.length; i += size) parts.push(list.slice(i, i + size));
	return parts;
}
