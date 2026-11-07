import { error } from "@sveltejs/kit";
import { staffApi } from "#lib/staff.ts";
import { accountTabs, type AccountTab } from "./tabs";

// The account, loaded again (refreshAll) after each action; the tab to
// open comes from the address (?tab=access).
export async function load({ params, url }) {
	const asked = url.searchParams.get("tab");
	const tab: AccountTab = accountTabs.find((t) => t === asked) ?? "summary";
	try {
		return { tab, detail: await staffApi.user(params.id) };
	} catch {
		error(404, "Conta não encontrada");
	}
}
