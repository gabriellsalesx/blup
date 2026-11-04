import gamesFreeArt from "./assets/hunt-games-free.svg";
import gamesDealsArt from "./assets/hunt-games-deals.svg";
import storesArt from "./assets/hunt-stores.svg";
import mirrorArt from "./assets/hunt-mirror.svg";
import type {PlainMessageKey} from "#lib/i18n/catalog.ts";
import {t} from "#lib/i18n.svelte.ts";
import type {Hunt,HuntKind,Store as StoreId} from "./api";
import {aliexpressCurrency,shopeeCurrency,type ShopeeCountry} from "../../../shared/deals-markets";
export {aliexpressCountries,shopeeCountries,currencyDigits,type AliexpressCountry,type ShopeeCountry} from "../../../shared/deals-markets";
// What the Deals panel knows about stores, hunts and platforms (names,
// help, which fields a store asks for). Brand names stay as they are.
export const storeName:Record<StoreId,string>={amazon:"Amazon",mercadolivre:"Mercado Livre",shopee:"Shopee",aliexpress:"AliExpress",nuuvem:"Nuuvem",magalu:"Magalu",terabyte:"Terabyte"};
// The names of every source a post can come from: the affiliate stores,
// the giveaways and TikTok (a video's news). Only "other" (or one the panel doesn't know yet) is
// named in the panel's language.
const sourceName:Record<string,string>={...storeName,gamerpower:"GamerPower",tiktok:"TikTok"};
export const storeLabel=(store:string)=>Object.hasOwn(sourceName,store)?sourceName[store]:t("dealsOtherStore");
export type StoreField={id:string;label:PlainMessageKey|null;text?:string;secret:boolean;optional?:boolean};
// `about` says what connecting does; `keys` marks the stores that take
// API keys (and an approved affiliate account) rather than one ID; `needs`
// names that one ID when it is not a tag.
export const stores:{id:StoreId;site:string;about:PlainMessageKey;keys:boolean;needs?:PlainMessageKey;steps:PlainMessageKey[];fields:StoreField[]}[]=[
 {id:"amazon",site:"https://associados.amazon.com.br/",about:"dealsAmazonAbout",keys:false,steps:["dealsAmazonStep1","dealsAmazonStep2","dealsAmazonStep3"],fields:[{id:"tag",label:"dealsAmazonTag",secret:false}]},
 {id:"mercadolivre",site:"https://www.mercadolivre.com.br/afiliados",about:"dealsMlAbout",keys:false,steps:["dealsMlStep1","dealsMlStep2","dealsMlStep3"],fields:[{id:"word",label:"dealsMlWord",secret:false},{id:"tool",label:"dealsMlTool",secret:false,optional:true}]},
 {id:"shopee",site:"https://affiliate.shopee.com.br/",about:"dealsShopeeAbout",keys:true,steps:["dealsShopeeStep1","dealsShopeeStep2","dealsShopeeStep3"],fields:[{id:"appId",label:null,text:"App ID",secret:false},{id:"secret",label:null,text:"Secret",secret:true}]},
 {id:"aliexpress",site:"https://portals.aliexpress.com/",about:"dealsAliAbout",keys:true,steps:["dealsAliStep1","dealsAliStep2","dealsAliStep3"],fields:[{id:"appKey",label:null,text:"App Key",secret:false},{id:"appSecret",label:null,text:"App Secret",secret:true},{id:"trackingId",label:null,text:"Tracking ID",secret:false}]},
 {id:"nuuvem",site:"https://secure.nuuvem.com/lp/pt/nuuvem-coop",about:"dealsNuuvemAbout",keys:false,needs:"dealsNeedsId",steps:["dealsNuuvemStep1","dealsNuuvemStep2","dealsNuuvemStep3"],fields:[{id:"siteId",label:"dealsNuuvemSiteId",secret:false}]},
 {id:"magalu",site:"https://www.magazinevoce.com.br/",about:"dealsMagaluAbout",keys:false,needs:"dealsNeedsCode",steps:["dealsMagaluStep1","dealsMagaluStep2","dealsMagaluStep3"],fields:[{id:"promoterId",label:"dealsMagaluPromoter",secret:false},{id:"showcase",label:"dealsMagaluShowcase",secret:false,optional:true}]},
 {id:"terabyte",site:"https://www.terabyteshop.com.br/site/regras-programa-de-parceiros",about:"dealsTerabyteAbout",keys:false,needs:"dealsNeedsCode",steps:["dealsTerabyteStep1","dealsTerabyteStep2","dealsTerabyteStep3"],fields:[{id:"partner",label:"dealsTerabytePartner",secret:false},{id:"utmMedium",label:"dealsTerabyteMedium",secret:false,optional:true},{id:"utmCampaign",label:"dealsTerabyteCampaign",secret:false,optional:true}]},
];
// What one of the person's own Magalu links says about them: the
// promoter_id and the Magazine Você showcase (the first part of its path,
// or the showcase parameter). Short links can't be read here.
export function magaluFromLink(raw:string):{promoterId:string;showcase:string|null}|null{
 let url:URL;try{url=new URL(raw.trim());}catch{return null;}
 const promoterId=url.searchParams.get("promoter_id")?.trim();if(!promoterId||!/^[A-Za-z0-9_-]{2,64}$/.test(promoterId))return null;
 const segment=/(?:^|\.)magazinevoce\.com\.br$/.test(url.hostname)?url.pathname.split("/")[1]:null,showcase=url.searchParams.get("showcase")??segment??null;
 return {promoterId,showcase:showcase&&/^magazine[a-z0-9_-]{1,60}$/i.test(showcase)?showcase:null};
}
export type HuntOption={id:HuntKind;title:PlainMessageKey;help:PlainMessageKey;art:string;soon?:boolean};
// Each kind has its own drawing of the coral Blup (by Aquarela); the kind's
// name always sits beside it, so the image is decorative (alt="").
export const huntKinds:HuntOption[]=[
 {id:"games_free",title:"dealsGamesFree",help:"dealsGamesFreeHelp",art:gamesFreeArt},
 // Nuuvem's sales use the game-sales drawing.
 {id:"nuuvem_deals",title:"dealsNuuvemDeals",help:"dealsNuuvemDealsHelp",art:gamesDealsArt},
 {id:"store_deals",title:"dealsStoresKind",help:"dealsStoresKindHelp",art:storesArt},
 {id:"mirror",title:"dealsMirrorKind",help:"dealsMirrorKindHelp",art:mirrorArt},
];
const huntNames={games_free:"dealsGamesFree",nuuvem_deals:"dealsNuuvemDeals",store_deals:"dealsStoresKind",mirror:"dealsMirrorKind"} as const satisfies Record<HuntKind,PlainMessageKey>;
export const huntName=(kind:HuntKind)=>huntNames[kind];
// Nuuvem's stores by country, named in the panel's language. Its catalog
// is Brazil's (in reais): elsewhere the post gives the discount and the
// link opens that country's page, with its own price.
export type NuuvemCountry=NonNullable<Hunt["filters"]["country"]>;
export const nuuvemCountries=["br","ar","bo","ca","cl","co","ec","mx","pe","us","uy"] as const satisfies readonly NuuvemCountry[];
// Each country's flag (Nuuvem's, Shopee's and AliExpress's countries),
// from flag-icons 7.6.0 (MIT, assets/flags/LICENSE), unchanged.
const flagFiles=import.meta.glob<string>("./assets/flags/*.svg",{eager:true,import:"default"});
export const countryFlag=(country:string):string|undefined=>flagFiles[`./assets/flags/${country}.svg`];
// Countries for a Select: named in the panel's language, with flags, in
// alphabetical order with `first` on top.
export const countryOptions=<T extends string>(list:readonly T[],locale:string,first?:T)=>list.map(value=>({value,label:countryName(value,locale),image:countryFlag(value)})).sort((a,b)=>a.value===first?-1:b.value===first?1:a.label.localeCompare(b.label,locale));
// A currency's name inside a sentence: "iene japonês" in Portuguese and
// Spanish, which write it in lower case; English keeps its capitals.
export function currencyName(currency:string,locale:string){
 let name=currency;try{name=new Intl.DisplayNames([locale],{type:"currency"}).of(currency)??currency;}catch{/* the code itself */}
 return /^(?:pt|es)\b/.test(locale)&&name!==currency?name.charAt(0).toLocaleLowerCase(locale)+name.slice(1):name;
}
// The money a store hunt's prices come in: Shopee's account country's,
// AliExpress's country's (its market).
export const shopCurrency=(store:"shopee"|"aliexpress",market:Parameters<typeof aliexpressCurrency>[0],account:ShopeeCountry)=>store==="shopee"?shopeeCurrency[account]:aliexpressCurrency(market);
export function countryName(country:string,locale:string){try{return new Intl.DisplayNames([locale],{type:"region"}).of(country.toUpperCase())??country;}catch{return country.toUpperCase();}}
// The platforms the server accepts (giveaways); Nuuvem's sales take none.
export const platforms=[
 {id:"pc",label:"PC"},{id:"steam",label:"Steam"},{id:"epic",label:"Epic Games"},{id:"gog",label:"GOG"},{id:"ubisoft",label:"Ubisoft"},
 {id:"itchio",label:"itch.io"},{id:"playstation",label:"PlayStation"},{id:"xbox",label:"Xbox"},{id:"switch",label:"Nintendo Switch"},{id:"android",label:"Android"},{id:"ios",label:"iOS"},
] as const satisfies readonly {id:Platform;label:string}[];
export const platformsFor=(kind:HuntKind)=>kind==="games_free"?platforms:[];
export const platformLabel=(id:string)=>platforms.find(p=>p.id===id)?.label??id;
// The stores a store hunt searches, with the person's own API keys.
export const shopStores=["shopee","aliexpress"] as const satisfies readonly NonNullable<Hunt["filters"]["store"]>[];
// What the hunt form gives back (POST /hunts, and PATCH without the kind).
export type Platform=NonNullable<Hunt["filters"]["platforms"]>[number];
// mirrors: the channels a new mirror reads (one mirror each).
export type HuntInput={kind:HuntKind;channelIds:string[];filters:Hunt["filters"];mirrors?:string[]};
// The AI's categories, in the panel's words.
export {dealCategories,type DealCategory} from "../../../shared/deals-post";
export const categoryLabel:Record<import("../../../shared/deals-post").DealCategory,PlainMessageKey>={electronics:"dealsCatElectronics",computers:"dealsCatComputers",phones:"dealsCatPhones",games:"dealsCatGames",home:"dealsCatHome",kitchen:"dealsCatKitchen",fashion:"dealsCatFashion",beauty:"dealsCatBeauty",sports:"dealsCatSports",kids:"dealsCatKids",pets:"dealsCatPets",books:"dealsCatBooks",tools:"dealsCatTools",auto:"dealsCatAuto",food:"dealsCatFood",other:"dealsCatOther"};
