// ?kind=whatsapp opens WhatsApp (reconnecting one); ?kind=webhook, a
// webhook. Once linked, back to the channels.
const kinds=["telegram","discord","whatsapp","webhook"] as const;
export function load({url}){
 const kind:(typeof kinds)[number]=kinds.find(k=>k===url.searchParams.get("kind"))??"telegram";
 return {kind};
}
