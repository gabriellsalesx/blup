import { api } from "#lib/api.ts";

export async function load() {
	return { plans: await api.plans() };
}
