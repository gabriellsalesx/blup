import {redirect} from "@sveltejs/kit";
import {resolve} from "$app/paths";
import {returnPath} from "#lib/auth.ts";
// Already signed in: straight to the page that asked (?next=).
export async function load({parent,url}){
 if((await parent()).signedIn)redirect(307,returnPath(url.search,resolve("/")));
}
