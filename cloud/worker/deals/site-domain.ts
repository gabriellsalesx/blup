import {z} from "zod";
import {DomainError,domainInfo,providerCall,providerRecords,provesOwnership,removeDomainProvider,type DomainInfo} from "../domain-provider";
import {claimHostname,releaseHostname} from "../hostname-claim";
import {siteLock,unlockSite} from "../quota";

// A deals page's own domain, as a Pulse page's (pulse/domains.ts): a TXT
// proof first, then the Cloudflare custom hostname and its certificate.
export type SiteDomainRow={site_id:string;hostname:string;proof:string;provider_id:string|null;verified_at:number|null;status:string;ssl:string;records:string;checked_at:number|null};
const recordsSchema=z.array(z.object({type:z.enum(["TXT","CNAME"]),name:z.string().max(253),value:z.string().max(1024)})).max(8);

export const readSiteDomain=(env:Cloudflare.Env,site:string)=>env.DB.prepare("SELECT * FROM deals_site_domain WHERE site_id=?").bind(site).first<SiteDomainRow>();
export const newProof=()=>`blup-verification=${crypto.randomUUID().replaceAll("-","")}${crypto.randomUUID().replaceAll("-","")}`;

// What the panel shows: the state and the records to create.
export function showSiteDomain(env:Cloudflare.Env,row:SiteDomainRow):DomainInfo{
 let extra:z.infer<typeof recordsSchema>=[];
 try{const parsed=recordsSchema.safeParse(JSON.parse(row.records));if(parsed.success)extra=parsed.data;}catch{/* none yet */}
 return domainInfo.parse({hostname:row.hostname,status:row.status,disableAt:null,ssl:row.ssl,records:[{type:"TXT",name:`_blup.${row.hostname}`,value:row.proof},...(env.SITES_CNAME_TARGET?[{type:"CNAME",name:row.hostname,value:env.SITES_CNAME_TARGET}]:[]),...extra]});
}

// Checks the domain again (at most every 30 seconds): the proof, then the
// custom hostname and its certificate.
export async function refreshSiteDomain(env:Cloudflare.Env,input:SiteDomainRow,now=Date.now()):Promise<DomainInfo>{
 if(input.checked_at!==null&&now-input.checked_at<30_000)return showSiteDomain(env,input);
 const lock=await siteLock(env.DB,`deals-domain:${input.site_id}`,60_000);if(!lock)return showSiteDomain(env,input);
 try{
  const row=await readSiteDomain(env,input.site_id);if(!row)throw new DomainError("DOMAIN_PROVIDER");
  if(row.verified_at===null&&!await provesOwnership(row.hostname,row.proof)){await env.DB.prepare("UPDATE deals_site_domain SET checked_at=? WHERE site_id=?").bind(now,row.site_id).run();return showSiteDomain(env,{...row,checked_at:now});}
  if(!await claimHostname(env.DB,row.hostname,"deals",row.site_id))throw new DomainError("DOMAIN_TAKEN");
  const response=await providerCall(env,row.provider_id?"GET":"POST",row.provider_id??undefined,row.hostname);if(!response)throw new DomainError("DOMAIN_PROVIDER");
  const ssl=response.ssl?.status==="active"?"active":/failed|expired|deleted|timed_out/.test(response.ssl?.status??"")?"error":"pending";
  const status=response.status==="active"&&ssl==="active"?"active":/blocked|deleted|moved/.test(response.status)?"error":"pending";
  const next:SiteDomainRow={...row,provider_id:response.id,verified_at:row.verified_at??now,status,ssl,records:JSON.stringify(providerRecords(response)),checked_at:now};
  await env.DB.prepare("UPDATE deals_site_domain SET provider_id=?,verified_at=?,status=?,ssl=?,records=?,checked_at=? WHERE site_id=?").bind(next.provider_id,next.verified_at,status,ssl,next.records,now,row.site_id).run();
  return showSiteDomain(env,next);
 }finally{await unlockSite(env.DB,`deals-domain:${input.site_id}`,lock);}
}

// Takes the domain away: the custom hostname, the claim and the row.
export async function removeSiteDomain(env:Cloudflare.Env,site:string){
 const row=await readSiteDomain(env,site);if(!row)return;
 await removeDomainProvider(env,{hostname:row.hostname,providerId:row.provider_id,verifiedAt:row.verified_at});
 await env.DB.batch([releaseHostname(env.DB,row.hostname,"deals",site),env.DB.prepare("DELETE FROM deals_site_domain WHERE site_id=?").bind(site)]);
}
