import type {Me,PulseUsage} from "./api";
import type {Setup} from "./setup";
export const panel=$state({me:null as Me|null,usage:null as PulseUsage|null,ready:false,error:false,signedOut:false,setup:null as Setup|null});
