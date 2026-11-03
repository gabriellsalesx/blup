import type {Me,PulseUsage} from "./api";
export const panel=$state({me:null as Me|null,usage:null as PulseUsage|null,ready:false,error:false,signedOut:false});
