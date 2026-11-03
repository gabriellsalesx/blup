import { parseResponse } from "hono/client";
import { api, rpc } from "#lib/api.ts";
import { loaded } from "#lib/load.ts";

// The plans and, for a paid account, what is going on with its billing
// (a checkout to finish, a change on the way), together.
export async function load({ parent }) {
	const plans = api.plans();
	const billing = loaded(parseResponse(rpc.billing.state.$get()), parent, null);
	const { me } = await parent();
	return { plans: await plans, billing: me.plan.gift ? null : await billing };
}
