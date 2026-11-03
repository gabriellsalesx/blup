import { authClient } from "#lib/auth.ts";

// The account's passkeys, ready with the page; they load again after
// one is added or removed (`depends`).
export async function load({ parent, depends }) {
	depends("app:passkeys");
	const passkeys = authClient.passkey.listUserPasskeys();
	await parent();
	return { passkeys: (await passkeys).data ?? [] };
}
