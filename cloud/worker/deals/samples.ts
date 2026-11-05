import type {Language} from "../../shared/language";
import {openingLine} from "./dispatch";
import {settingsOf,headlineOf,socialLinks} from "./store";
import type {Post} from "./message";

// The examples the panel sends to a channel, so the person sees how each
// kind of post arrives: the plain test, a free game, a Nuuvem sale, a store
// find (AliExpress) and a pasted link. Each goes in the account's own
// layout, opening lines, extra text and networks, marked as an example.
export const sampleKinds=["test","game_free","nuuvem_deal","store_deal","product"] as const;
export type SampleKind=(typeof sampleKinds)[number];

// All made up and fixed: the same examples every time, never a real offer
// (nor another account's).
type Made={title:string;store:string;link:string;price:number;originalPrice:number;currency:string;discount:number;description:string|null;coupon:string|null};
const made:Record<"store_deal"|"product"|"game_free"|"nuuvem_deal",Record<Language,Made>>={
 store_deal:{
  pt:{title:"Fone de ouvido Bluetooth sem fio",store:"aliexpress",link:"https://pt.aliexpress.com/",price:8990,originalPrice:15990,currency:"BRL",discount:44,description:"Bateria de até 30 horas e estojo de recarga.",coupon:"EXEMPLO10"},
  en:{title:"Wireless Bluetooth earbuds",store:"aliexpress",link:"https://www.aliexpress.com/",price:1799,originalPrice:3199,currency:"USD",discount:44,description:"Up to 30 hours of battery and a charging case.",coupon:"SAMPLE10"},
  es:{title:"Auriculares Bluetooth inalámbricos",store:"aliexpress",link:"https://es.aliexpress.com/",price:1799,originalPrice:3199,currency:"USD",discount:44,description:"Hasta 30 horas de batería y estuche de carga.",coupon:"EJEMPLO10"},
 },
 product:{
  pt:{title:"Smartwatch com GPS e monitor cardíaco",store:"amazon",link:"https://www.amazon.com.br/",price:29900,originalPrice:39900,currency:"BRL",discount:25,description:null,coupon:null},
  en:{title:"Smartwatch with GPS and heart rate monitor",store:"amazon",link:"https://www.amazon.com/",price:5999,originalPrice:7999,currency:"USD",discount:25,description:null,coupon:null},
  es:{title:"Smartwatch con GPS y monitor cardíaco",store:"amazon",link:"https://www.amazon.com/",price:5999,originalPrice:7999,currency:"USD",discount:25,description:null,coupon:null},
 },
 game_free:{
  pt:{title:"Aventura Estelar",store:"gamerpower",link:"https://www.gamerpower.com/",price:0,originalPrice:0,currency:"USD",discount:0,description:null,coupon:null},
  en:{title:"Star Adventure",store:"gamerpower",link:"https://www.gamerpower.com/",price:0,originalPrice:0,currency:"USD",discount:0,description:null,coupon:null},
  es:{title:"Aventura Estelar",store:"gamerpower",link:"https://www.gamerpower.com/",price:0,originalPrice:0,currency:"USD",discount:0,description:null,coupon:null},
 },
 nuuvem_deal:{
  pt:{title:"Cidade Neon",store:"nuuvem",link:"https://www.nuuvem.com/",price:4999,originalPrice:9999,currency:"BRL",discount:50,description:null,coupon:null},
  en:{title:"Neon City",store:"nuuvem",link:"https://www.nuuvem.com/",price:4999,originalPrice:9999,currency:"BRL",discount:50,description:null,coupon:null},
  es:{title:"Ciudad Neón",store:"nuuvem",link:"https://www.nuuvem.com/",price:4999,originalPrice:9999,currency:"BRL",discount:50,description:null,coupon:null},
 },
};

// One example post, in the channel's language (lang) for an account whose
// own language is account. A custom opening line is written in the
// account's language and stays so (an example is not translated).
export async function samplePost(db:D1Database,owner:string,kind:Exclude<SampleKind,"test">,lang:Language,account:Language):Promise<Post>{
 const own=await settingsOf(db,owner),fake=made[kind][lang],free=kind==="game_free";
 const custom=headlineOf(own,kind)!==null;
 const title=openingLine({title:fake.title,kind,discount:free?null:fake.discount,store:fake.store,language:custom?account:lang},own);
 return {sample:true,store:fake.store,title,extra:own.extra.on&&own.extra.text?own.extra.text:null,socials:socialLinks(own),style:own.style,description:fake.description,coupon:fake.coupon,image:null,price:free?null:fake.price,originalPrice:free?null:fake.originalPrice,currency:free?null:fake.currency,discount:free?null:fake.discount,link:fake.link,affiliate:false,kind,credit:free?"gamerpower":null};
}
