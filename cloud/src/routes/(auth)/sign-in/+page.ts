import { redirect } from "@sveltejs/kit";
import { returnPath, authClient } from "#lib/auth.ts";
import { cloudPaths } from "../../../../shared/panel-paths";

// Already signed in (the site's "Sign in", a bookmark): straight to the
// account or the page that asked, before the sign-in draws anything.
export async function load({ url }) {
	const { data } = await authClient.getSession().catch(() => ({ data: null }));
	if (data) redirect(307, returnPath(url.search, cloudPaths.account));
}

// Its own HTML file at build time (still drawn in the browser: ssr is off):
// the file already asks for this page's JavaScript and CSS, so they come
// together with the panel's instead of one after the other.
export const prerender = true;
