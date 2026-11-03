import { parseResponse } from "hono/client";
import { api, rpc } from "#lib/api.ts";
import { loaded } from "#lib/load.ts";

// The plans and, for a paid account, what is going on with its billing
// (a checkout to finish, a change on the way), together.
export async function load({ parent }) {
	// Awaited together, so a signed-out visit (the parent redirects) leaves
	// no request rejected without a handler.
	const [plans, billing, { me }] = await Promise.all([api.plans(), loaded(parseResponse(rpc.billing.state.$get()), parent, null), parent()]);
	return { plans, billing: me.plan.gift ? null : billing };
}
