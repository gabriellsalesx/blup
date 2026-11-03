import {errorMessage} from "#lib/i18n/errors.ts";
import {toast} from "#lib/toast.svelte.ts";
// Pages fetch their data in `load`, before they open: hovering a link
// already starts it, and the page shows whole. A failure is a toast and
// an empty page, as before.
export async function attempt<T>(fn:()=>Promise<T>,fallback:T):Promise<T>{try{return await fn();}catch(e){toast(errorMessage(e),{kind:"error"});return fallback;}}
