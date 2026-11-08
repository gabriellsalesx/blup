import { error } from "@sveltejs/kit";
import { staffApi } from "#lib/staff.ts";
import { loaded } from "#lib/load.ts";

// The team: admins only (the API refuses support too).
export async function load({ parent }) {
	const request = staffApi.team();
	const { me } = await parent();
	if (me?.role !== "admin") {
		request.catch(() => {});
		error(403, "Só admins cuidam da equipe.");
	}
	return { team: await loaded(request, parent, null, "Não deu para carregar a equipe.") };
}
