// How an account's posts go out and read, shared by the server (which
// writes them) and the panel (which previews them): the pace of the queue,
// the opening line of each kind of offer, the extra text and the social
// networks under it.

// Minutes between two posts from the queue (0: as soon as found).
export const dealsPaces=[0,30,60,180,300,480] as const;
export type DealsPace=(typeof dealsPaces)[number];

// The kinds of offer with an opening line of their own (a video keeps its
// "New video" line).
export const headlineKinds=["game_free","nuuvem_deal","store_deal","product"] as const;
export type HeadlineKind=(typeof headlineKinds)[number];
export type PostLanguage="pt"|"en"|"es";

// The words a line can carry, in each language's spelling (any of them
// works in any language).
export const headlineTokens={name:["nome","name","nombre"],discount:["desconto","discount","descuento"],store:["loja","store","tienda"]} as const;
export type HeadlineToken=keyof typeof headlineTokens;
const tokenIndex={pt:0,en:1,es:2} as const;
export const tokenText=(token:HeadlineToken,language:PostLanguage)=>`{${headlineTokens[token][tokenIndex[language]]}}`;

export const defaultHeadlines:Record<PostLanguage,Record<HeadlineKind,string>>={
 pt:{game_free:"Atenção: {nome} está grátis! Aproveite antes que acabe.",nuuvem_deal:"Promoção na Nuuvem: {nome} com {desconto} de desconto. Aproveite!",store_deal:"Achado do dia: {nome} com {desconto} de desconto.",product:"Oferta do dia: {nome}"},
 en:{game_free:"Heads up: {name} is free! Grab it before it is gone.",nuuvem_deal:"Nuuvem sale: {name} at {discount} off. Do not miss it!",store_deal:"Find of the day: {name} at {discount} off.",product:"Deal of the day: {name}"},
 es:{game_free:"Atención: ¡{nombre} está gratis! Aprovéchalo antes de que se acabe.",nuuvem_deal:"Oferta en Nuuvem: {nombre} con {descuento} de descuento. ¡Aprovecha!",store_deal:"Hallazgo del día: {nombre} con {descuento} de descuento.",product:"Oferta del día: {nombre}"},
};
export const HEADLINE_MAX=200;
export const EXTRA_MAX=200;
// Text the person writes carries no address: links come only from the
// networks, checked one by one.
export const hasAddress=(text:string)=>/https?:|www\.|:\/\/|[a-z0-9-]\.(?:com|net|org|io|me|gg|br|ly|link|site|app|tv|co)(?:\/|\b)/i.test(text);

// The opening line with the offer's words in it. A word the offer does not
// have (a pasted link has no discount) goes, with the space around it.
export function headline(template:string,values:{name:string;discount:number|null;store:string}){
 const value={name:values.name,discount:values.discount&&values.discount>0?`${values.discount}%`:"",store:values.store};
 let text=template;
 for(const token of Object.keys(headlineTokens) as HeadlineToken[])for(const spelling of headlineTokens[token])text=text.split(`{${spelling}}`).join(value[token]);
 return text.replace(/\s+/g," ").replace(/\s+([.,!?;:])/g,"$1").trim();
}

// The networks a person can show under each post, one of each: the
// address must be on the network's own site (the person's own site can be
// anywhere, by name, not by IP).
export const socialNetworks=[
 {id:"site",name:"Site",prefix:"https://",hosts:[]},
 {id:"instagram",name:"Instagram",prefix:"https://instagram.com/",hosts:["instagram.com"]},
 {id:"tiktok",name:"TikTok",prefix:"https://tiktok.com/@",hosts:["tiktok.com"]},
 {id:"youtube",name:"YouTube",prefix:"https://youtube.com/@",hosts:["youtube.com","youtu.be"]},
 {id:"x",name:"X",prefix:"https://x.com/",hosts:["x.com","twitter.com"]},
 {id:"facebook",name:"Facebook",prefix:"https://facebook.com/",hosts:["facebook.com","fb.com"]},
 {id:"threads",name:"Threads",prefix:"https://threads.net/@",hosts:["threads.net","threads.com"]},
 {id:"whatsapp",name:"WhatsApp",prefix:"https://whatsapp.com/channel/",hosts:["whatsapp.com","wa.me"]},
 {id:"telegram",name:"Telegram",prefix:"https://t.me/",hosts:["t.me","telegram.me"]},
 {id:"discord",name:"Discord",prefix:"https://discord.gg/",hosts:["discord.gg","discord.com"]},
 {id:"twitch",name:"Twitch",prefix:"https://twitch.tv/",hosts:["twitch.tv"]},
 {id:"kick",name:"Kick",prefix:"https://kick.com/",hosts:["kick.com"]},
] as const satisfies readonly {id:string;name:string;prefix:string;hosts:readonly string[]}[];
export type SocialNetwork=(typeof socialNetworks)[number]["id"];
export const socialIds=socialNetworks.map(n=>n.id) as [SocialNetwork,...SocialNetwork[]];
export const networkOf=(id:string)=>socialNetworks.find(n=>n.id===id);
// The address, cleaned, when it is a profile on that network's site.
export function socialURL(network:SocialNetwork,raw:string):string|null{
 try{
  const url=new URL(raw.trim()),net=networkOf(network);
  if(!net||url.protocol!=="https:"||url.username||url.password||url.port)return null;
  const host=url.hostname.toLowerCase().replace(/\.$/,"");
  if(raw.length>300)return null;
  if(!net.hosts.length)return /^(?=.{4,253}$)(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$/.test(host)?(url.hash="",url.href):null;
  if(!net.hosts.some(h=>host===h||host.endsWith(`.${h}`)))return null;
  if(url.pathname.replace(/\/+$/,"").length<2)return null;
  url.hash="";return url.href;
 }catch{return null;}
}

// The names {store} gives each store.
export const storeTitles:Record<string,string>={amazon:"Amazon",mercadolivre:"Mercado Livre",shopee:"Shopee",aliexpress:"AliExpress",nuuvem:"Nuuvem",magalu:"Magalu",terabyte:"Terabyte",gamerpower:"GamerPower"};
