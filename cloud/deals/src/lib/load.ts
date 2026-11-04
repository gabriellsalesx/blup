import {toast} from "#lib/toast.svelte.ts";
import {dealsError} from "./errors";
// Pages fetch their data in `load`, before they open: hovering a link
// already starts it, and the page shows whole. A failure is a toast and
// an empty page.
export async function attempt<T>(fn:()=>Promise<T>,fallback:T):Promise<T>{try{return await fn();}catch(e){toast(dealsError(e),{kind:"error"});return fallback;}}
