import { redirect } from "@sveltejs/kit";
import { returnPath, authClient } from "#lib/auth.ts";
import { cloudPaths } from "../../../../shared/panel-paths";

// Already signed in (the site's "Sign in", a bookmark): straight to the
// account or the page that asked, before the sign-in draws anything.
export async function load({ url }) {
	const { data } = await authClient.getSession().catch(() => ({ data: null }));
	if (data) redirect(307, returnPath(url.search, cloudPaths.account));
}
