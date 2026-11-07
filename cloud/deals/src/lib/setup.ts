import {type InferResponseType} from "hono/client";
import {dealsApi,parseResponse} from "./api";
import {panel} from "./panel.svelte";
// The first steps of the account (the guide under the panel's main pages),
// as Pulse's.
export type Setup=InferResponseType<typeof dealsApi.setup.$get,200>;
export type SetupStep=Setup["steps"][number]["id"];
export const setupDone=(setup:Setup|null)=>!!setup&&setup.steps.every(s=>s.done);
// Asked again on each page while some step is left, without holding the
// page; once all are done, no more.
export function refreshSetup(){
 if(!panel.me||setupDone(panel.setup))return;
 parseResponse(dealsApi.setup.$get()).then(setup=>{panel.setup=setup;},()=>{});
}
