import { staffApi, StaffError } from "#lib/staff.ts";

// Who is in the staff area (or why not).
export async function load() {
	try {
		return { me: await staffApi.me(), denied: false };
	} catch (e) {
		if (e instanceof StaffError && e.status === 403) return { me: null, denied: true };
		throw e;
	}
}
