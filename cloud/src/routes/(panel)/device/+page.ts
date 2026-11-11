import { authClient } from "#lib/auth.ts";

// Opened from the Studio, the code comes in the address and is checked
// before the page shows, so it opens straight on the confirmation. The
// check claims the code for this session: it waits for the sign-in check.
export async function load({ url, parent }) {
	const code = url.searchParams.get("user_code") ?? "";
	// Opened by Blup Notes (the phone): once approved, the page brings the
	// app back on screen.
	const notes = url.searchParams.get("app") === "notes";
	const clean = code.replace(/[^0-9a-z]/gi, "").toUpperCase();
	await parent();
	if (!clean) return { code, notes, checked: null };
	const { error } = await authClient.device({ query: { user_code: clean } });
	return { code, notes, checked: !error };
}
