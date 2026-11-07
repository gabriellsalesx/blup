import type {Me,DealsUsage} from "./api";
import type {Setup} from "./setup";
import {dealsApi,parseResponse} from "./api";
// Who is signed in and their Deals plan, loaded once by the layout.
export const panel=$state({me:null as Me|null,usage:null as DealsUsage|null,ready:false,error:false,signedOut:false,setup:null as Setup|null});
// After a change that counts toward a limit (a channel, a hunt, a post),
// the usage is asked again; a failure keeps the old one.
export async function refreshUsage(){try{panel.usage=await parseResponse(dealsApi.usage.$get());}catch{/* the old numbers stay */}}
