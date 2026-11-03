import {logoVariant,logoKey} from "./page-look";
// Removing a page's logos from R2, apart from the routes so the clean-up
// can use it without the routes' imports.
export const dropPageAssets=(env:Cloudflare.Env,page:string)=>env.PULSE_ASSETS.delete(logoVariant.options.map(v=>logoKey(page,v)));
