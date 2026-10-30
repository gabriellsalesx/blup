import { error } from "@sveltejs/kit";
import { staffApi, StaffError } from "#lib/staff.ts";

// Who is in the staff area; anyone else gets the 403 page.
export async function load() {
	try {
		return { me: await staffApi.me() };
	} catch (e) {
		if (e instanceof StaffError && e.status === 403) error(403, "Esta área é só da equipe do Blup.");
		throw e;
	}
}
