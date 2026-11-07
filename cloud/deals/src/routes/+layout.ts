import {redirect} from "@sveltejs/kit";
import {resolve} from "$app/paths";
import {DetailedError} from "hono/client";
import {nextParam} from "../../../shared/panel-paths";
import {rpc,dealsApi,parseResponse} from "../lib/api";
import {panel} from "../lib/panel.svelte";
import {refreshSetup} from "../lib/setup";
export const ssr=false;
// Who is signed in and their Deals plan, once, before any page loads; the
// pages wait for it (`parent`). Nobody signed in goes to the sign-in, which
// comes back to the same page.
export async function load({url,route}){
 if(!panel.ready){try{const [me,usage]=await Promise.all([parseResponse(rpc.me.$get()),parseResponse(dealsApi.usage.$get())]);panel.me=me;panel.usage=usage;panel.signedOut=false;}catch(e){panel.error=true;panel.signedOut=e instanceof DetailedError&&e.statusCode===401;}panel.ready=true;}
 if(panel.signedOut&&route.id!=="/sign-in")redirect(307,`${resolve("/sign-in")}?${new URLSearchParams({[nextParam]:url.pathname+url.search})}`);
 refreshSetup();
 return {signedIn:!!panel.me};
}
