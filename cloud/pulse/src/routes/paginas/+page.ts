import {type InferResponseType} from "hono/client";
import {pulseApi,parseResponse} from "../../lib/api";
import {attempt} from "../../lib/load";
type Page=InferResponseType<typeof pulseApi.pages.$get,200>["pages"][number];
type Monitor=InferResponseType<typeof pulseApi.monitors.$get,200>["monitors"][number];
export async function load({parent}){
 const empty={pages:[] as Page[],monitors:[] as Monitor[]};
 if(!(await parent()).signedIn)return empty;
 return attempt(async()=>{const [pages,monitors]=await Promise.all([parseResponse(pulseApi.pages.$get()),parseResponse(pulseApi.monitors.$get())]);return {pages:pages.pages,monitors:monitors.monitors};},empty);
}
