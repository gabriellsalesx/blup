import {toast} from "#lib/toast.svelte.ts";
import {dealsError} from "./errors";
// Pages fetch their data in `load`, before they open: hovering a link
// already starts it, and the page shows whole. A failure is a toast and
// an empty page.
export async function attempt<T>(fn:()=>Promise<T>,fallback:T):Promise<T>{try{return await fn();}catch(e){toast(dealsError(e),{kind:"error"});return fallback;}}

// A page's request leaves at once, alongside the layout's (`parent`: who
// is signed in), not after it; it is read only once someone is signed in,
// else `fallback` (and nothing is shown).
export async function early<T>(fn:()=>Promise<T>,parent:()=>Promise<{signedIn:boolean}>,fallback:T):Promise<T>{
 const settled=new Promise<T>(ok=>ok(fn())).then(value=>({ok:true as const,value}),(error:unknown)=>({ok:false as const,error}));
 if(!(await parent()).signedIn)return fallback;
 const result=await settled;if(result.ok)return result.value;
 toast(dealsError(result.error),{kind:"error"});return fallback;
}
