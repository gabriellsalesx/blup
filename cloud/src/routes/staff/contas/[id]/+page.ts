import { error } from "@sveltejs/kit";
import { staffApi } from "#lib/staff.ts";

// The account, loaded again (invalidateAll) after each action.
export async function load({ params }) {
	try {
		return { detail: await staffApi.user(params.id) };
	} catch {
		error(404, "Conta não encontrada");
	}
}
