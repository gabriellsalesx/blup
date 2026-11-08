import { error } from "@sveltejs/kit";
import { staffApi } from "#lib/staff.ts";
import { loaded } from "#lib/load.ts";

// The staff's log: admins only (the API refuses support too).
export async function load({ parent }) {
	const request = staffApi.log(null);
	const { me } = await parent();
	if (me?.role !== "admin") {
		request.catch(() => {});
		error(403, "Só admins veem o registro.");
	}
	return { first: await loaded(request, parent, null, "Não deu para carregar o registro.") };
}
