import type {Me,DealsUsage} from "./api";
// Who is signed in and their Deals plan, loaded once by the layout.
export const panel=$state({me:null as Me|null,usage:null as DealsUsage|null,ready:false,error:false,signedOut:false});
