import { i18n, loadLanguage } from "#lib/i18n.svelte.ts";

// A single-page app: everything renders in the browser.
export const ssr = false;

// The texts of the language in use before the first page (the staff area
// is always in Portuguese).
export async function load({ url }) {
	const staff = url.hostname.startsWith("staff.") || url.pathname === "/staff" || url.pathname.startsWith("/staff/");
	await loadLanguage(staff ? "pt" : i18n.locale);
}
