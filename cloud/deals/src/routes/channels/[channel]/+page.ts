import {error} from "@sveltejs/kit";
import {dealsApi,parseResponse,type Channel} from "../../../lib/api";
import {attempt} from "../../../lib/load";
import {getSite,siteLogoPreview,type Site} from "../../../lib/site";
// One channel of the account, by its id: its details and examples; for
// the deals page, the page too, with its logos as the panel can show them.
type Page={site:Site;logos:{light:string|null;dark:string|null}};
export async function load({parent,params,url}){
 if(!/^[0-9a-f-]{36}$/.test(params.channel))error(404);
 const tab=url.searchParams.get("tab")==="domain"?"domain" as const:"settings" as const;
 if(!(await parent()).signedIn)return {channel:null as Channel|null,page:null as Page|null,tab};
 const found=await attempt(async()=>({channel:(await parseResponse(dealsApi.channels.$get())).channels.find(c=>c.id===params.channel)??null,loaded:true}),{channel:null as Channel|null,loaded:false});
 if(found.loaded&&!found.channel)error(404);
 let page:Page|null=null;
 if(found.channel?.type==="site")page=await attempt(async()=>{
  const site=await getSite();if(!site)return null;
  const [light,dark]=await Promise.all([siteLogoPreview("light",site.logos.light).catch(()=>null),siteLogoPreview("dark",site.logos.dark).catch(()=>null)]);
  return {site,logos:{light,dark}};
 },null);
 return {channel:found.channel,page,tab};
}
