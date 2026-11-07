import {language} from "#lib/i18n.svelte.ts";
import type {LogoVariant,MadeLogo} from "#lib/logo-image.ts";
import type {InferResponseType} from "hono/client";
import {dealsApi,parseResponse} from "./api";
// The account's public deals page (its channel is of type "site"): read,
// its logos sent, removed and shown in the panel.
export type Site=NonNullable<InferResponseType<typeof dealsApi.site.$get,200>["site"]>;
export type SiteLanguage=Site["language"];
export const SITE_HOST="deals.getblup.site/";
export const getSite=async()=>(await parseResponse(dealsApi.site.$get())).site;
export async function uploadSiteLogo(variant:LogoVariant,made:MadeLogo){
 return (await parseResponse(dealsApi.site.logo[":variant"].$put({param:{variant}},{headers:{"content-type":made.type,"x-blup-language":language()},init:{body:made.blob,credentials:"include"}}))).version;
}
export const removeSiteLogo=(variant:LogoVariant)=>parseResponse(dealsApi.site.logo[":variant"].$delete({param:{variant}}));
// The stored logo as an address the panel can show, or null without one.
export async function siteLogoPreview(variant:LogoVariant,version:string|null){
 if(!version)return null;
 const response=await dealsApi.site.logo[":variant"].$get({param:{variant}});
 return response.ok?URL.createObjectURL(await response.blob()):null;
}
// Whether an address can be had: "yours" is the page's own.
export async function slugState(slug:string){
 const state=(await parseResponse(dealsApi.site.slug.$get({query:{slug}}))).state;
 return state==="yours"?"taken" as const:state;
}
