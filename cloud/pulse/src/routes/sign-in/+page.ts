import {redirect} from "@sveltejs/kit";
import {resolve} from "$app/paths";
import {returnPath} from "#lib/auth.ts";
// Already signed in: straight to the page that asked (?next=).
export async function load({parent,url}){
 if((await parent()).signedIn)redirect(307,returnPath(url.search,resolve("/")));
}
// Its own HTML file at build time (still drawn in the browser: ssr is off):
// the file already asks for this page's JavaScript and CSS, so they come
// together with the panel's instead of one after the other.
export const prerender=true;
