import {attempt} from "../../../lib/load";
import {getSite,type Site} from "../../../lib/site";
// ?kind=whatsapp opens WhatsApp (reconnecting one); ?kind=webhook, a
// webhook; ?kind=site, the deals page (or a link to it, when the account
// has one). Once linked, back to the channels.
const kinds=["telegram","discord","whatsapp","webhook","site"] as const;
export async function load({url,parent}){
 const kind:(typeof kinds)[number]=kinds.find(k=>k===url.searchParams.get("kind"))??"telegram";
 const site=(await parent()).signedIn?await attempt(getSite,null as Site|null):null;
 return {kind,site};
}
