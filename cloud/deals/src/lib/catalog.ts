import gamesFreeArt from "./assets/hunt-games-free.svg";
import gamesDealsArt from "./assets/hunt-games-deals.svg";
import storesArt from "./assets/hunt-stores.svg";
import mirrorArt from "./assets/hunt-mirror.svg";
import type {PlainMessageKey} from "#lib/i18n/catalog.ts";
import {t} from "#lib/i18n.svelte.ts";
import type {Hunt,HuntKind,Store as StoreId} from "./api";
// What the Deals panel knows about stores, hunts and platforms (names,
// help, which fields a store asks for). Brand names stay as they are.
export const storeName:Record<StoreId,string>={amazon:"Amazon",mercadolivre:"Mercado Livre",shopee:"Shopee",aliexpress:"AliExpress"};
// The names of every source a post can come from: the affiliate stores
// and the game feeds. Only "other" (or one the panel doesn't know yet) is
// named in the panel's language.
const sourceName:Record<string,string>={...storeName,gamerpower:"GamerPower",cheapshark:"CheapShark"};
export const storeLabel=(store:string)=>Object.hasOwn(sourceName,store)?sourceName[store]:t("dealsOtherStore");
export type StoreField={id:string;label:PlainMessageKey|null;text?:string;secret:boolean;optional?:boolean};
// `about` says what connecting does; `keys` marks the stores that take
// API keys (and an approved affiliate account) rather than one ID.
export const stores:{id:StoreId;site:string;about:PlainMessageKey;keys:boolean;steps:PlainMessageKey[];fields:StoreField[]}[]=[
 {id:"amazon",site:"https://associados.amazon.com.br/",about:"dealsAmazonAbout",keys:false,steps:["dealsAmazonStep1","dealsAmazonStep2","dealsAmazonStep3"],fields:[{id:"tag",label:"dealsAmazonTag",secret:false}]},
 {id:"mercadolivre",site:"https://www.mercadolivre.com.br/afiliados",about:"dealsMlAbout",keys:false,steps:["dealsMlStep1","dealsMlStep2","dealsMlStep3"],fields:[{id:"word",label:"dealsMlWord",secret:false},{id:"tool",label:"dealsMlTool",secret:false,optional:true}]},
 {id:"shopee",site:"https://affiliate.shopee.com.br/",about:"dealsShopeeAbout",keys:true,steps:["dealsShopeeStep1","dealsShopeeStep2","dealsShopeeStep3"],fields:[{id:"appId",label:null,text:"App ID",secret:false},{id:"secret",label:null,text:"Secret",secret:true}]},
 {id:"aliexpress",site:"https://portals.aliexpress.com/",about:"dealsAliAbout",keys:true,steps:["dealsAliStep1","dealsAliStep2","dealsAliStep3"],fields:[{id:"appKey",label:null,text:"App Key",secret:false},{id:"appSecret",label:null,text:"App Secret",secret:true},{id:"trackingId",label:null,text:"Tracking ID",secret:false}]},
];
export type HuntOption={id:HuntKind|"stores"|"mirror";title:PlainMessageKey;help:PlainMessageKey;art:string;soon?:boolean};
// Each kind has its own drawing of the coral Blup (by Aquarela); the kind's
// name always sits beside it, so the image is decorative (alt="").
export const huntKinds:HuntOption[]=[
 {id:"games_free",title:"dealsGamesFree",help:"dealsGamesFreeHelp",art:gamesFreeArt},
 {id:"games_deals",title:"dealsGamesDeals",help:"dealsGamesDealsHelp",art:gamesDealsArt},
 {id:"stores",title:"dealsStoresKind",help:"dealsStoresKindHelp",art:storesArt,soon:true},
 {id:"mirror",title:"dealsMirrorKind",help:"dealsMirrorKindHelp",art:mirrorArt,soon:true},
];
export const huntName=(kind:HuntKind)=>kind==="games_free"?"dealsGamesFree" as const:"dealsGamesDeals" as const;
// The platforms the server accepts; game sales (CheapShark) only the PC ones.
export const platforms=[
 {id:"pc",label:"PC",pc:true},{id:"steam",label:"Steam",pc:true},{id:"epic",label:"Epic Games",pc:true},{id:"gog",label:"GOG",pc:true},{id:"ubisoft",label:"Ubisoft",pc:true},
 {id:"itchio",label:"itch.io",pc:false},{id:"playstation",label:"PlayStation",pc:false},{id:"xbox",label:"Xbox",pc:false},{id:"switch",label:"Nintendo Switch",pc:false},{id:"android",label:"Android",pc:false},{id:"ios",label:"iOS",pc:false},
] as const satisfies readonly {id:Platform;label:string;pc:boolean}[];
export const platformsFor=(kind:HuntKind)=>kind==="games_deals"?platforms.filter(p=>p.pc):platforms;
export const platformLabel=(id:string)=>platforms.find(p=>p.id===id)?.label??id;
// What the hunt form gives back (POST /hunts, and PATCH without the kind).
export type Platform=NonNullable<Hunt["filters"]["platforms"]>[number];
export type HuntInput={kind:HuntKind;channelIds:string[];filters:Hunt["filters"]};
