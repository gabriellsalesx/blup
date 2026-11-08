import { error } from "@sveltejs/kit";
import { staffApi, StaffError } from "#lib/staff.ts";
import { accountTabs, type AccountTab } from "./tabs";

// The account, loaded again (refreshAll) after each action; the tab to
// open comes from the address (?tab=access).
export async function load({ params, url }) {
	const asked = url.searchParams.get("tab");
	const tab: AccountTab = accountTabs.find((t) => t === asked) ?? "summary";
	try {
		return { tab, detail: await staffApi.user(params.id) };
	} catch (e) {
		// Only a missing account is a 404; a dropped connection or a server
		// error shows as one (the error page offers to try again).
		if (e instanceof StaffError && (e.status === 404 || e.status === 400)) error(404, "Conta não encontrada");
		throw e;
	}
}
