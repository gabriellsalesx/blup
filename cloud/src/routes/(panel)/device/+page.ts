import { authClient } from "#lib/auth.ts";

// Opened from the Studio, the code comes in the address and is checked
// before the page shows, so it opens straight on the confirmation. The
// check claims the code for this session: it waits for the sign-in check.
export async function load({ url, parent }) {
	const code = url.searchParams.get("user_code") ?? "";
	const clean = code.replace(/[^0-9a-z]/gi, "").toUpperCase();
	await parent();
	if (!clean) return { code, checked: null };
	const { error } = await authClient.device({ query: { user_code: clean } });
	return { code, checked: !error };
}
