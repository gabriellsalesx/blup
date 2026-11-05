// Where to go once the channel is linked: the first steps when they sent
// here (?next=/start...), else the channels. ?kind=whatsapp opens
// WhatsApp (reconnecting one); ?kind=webhook, a webhook.
const kinds=["telegram","discord","whatsapp","webhook"] as const;
export function load({url}){
 const next=url.searchParams.get("next"),kind:(typeof kinds)[number]=kinds.find(k=>k===url.searchParams.get("kind"))??"telegram";
 return {next:next&&/^\/start(\?[\w=&-]*)?$/.test(next)?next:null,kind};
}
