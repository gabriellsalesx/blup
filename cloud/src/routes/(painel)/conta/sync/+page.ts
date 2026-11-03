import { api, type Device } from "#lib/api.ts";
import { loaded } from "#lib/load.ts";

export async function load({ parent }) {
	return { devices: await loaded(api.devices().then((r) => r.devices), parent, [] as Device[]) };
}
