import {language} from "#lib/i18n.svelte.ts";
import type {LogoVariant,MadeLogo} from "#lib/logo-image.ts";
import {pulseApi,parseResponse} from "./api";
// The status page's logos on the server (made small by #lib/logo-image).
// Sends a made logo; its new version comes back.
export async function uploadLogo(page:string,variant:LogoVariant,made:MadeLogo){
 return (await parseResponse(pulseApi.pages[":page"].logo[":variant"].$put({param:{page,variant}},{headers:{"content-type":made.type,"x-blup-language":language()},init:{body:made.blob,credentials:"include"}}))).version;
}
export const removeLogo=(page:string,variant:LogoVariant)=>parseResponse(pulseApi.pages[":page"].logo[":variant"].$delete({param:{page,variant}}));
// The stored logo as an address the panel can show (the page may not be
// public yet), or null without one.
export async function logoPreview(page:string,variant:LogoVariant,version:string|null){
 if(!version)return null;
 const response=await pulseApi.pages[":page"].logo[":variant"].$get({param:{page,variant}});
 return response.ok?URL.createObjectURL(await response.blob()):null;
}
