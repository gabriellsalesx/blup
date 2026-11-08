import {disablePulseDomain} from "./domain-lifecycle";
import {Hono} from "hono";
import {HTTPException} from "hono/http-exception";
import {z} from "zod";
import {DomainError,domainInfo,hostnameSchema,providerCall,providerRecords,provesOwnership,removeDomainProvider,type DomainInfo} from "../domain-provider";
import {claimHostname,releaseHostname} from "../hostname-claim";
import {siteLock,unlockSite} from "../quota";
import {validate} from "../validation";
import {pulseWrite,type PulseApp} from "./access";
import {ownedPage,pageParam} from "./pages";
export type PulseDomain={page_id:string;hostname:string;proof:string;provider_id:string|null;verified_at:number|null;status:string;ssl:string;records:string;checked_at:number|null};
async function read(env:Cloudflare.Env,page:string){return env.DB.prepare("SELECT * FROM pulse_page_domain WHERE page_id=?").bind(page).first<PulseDomain>();}
function display(env:Cloudflare.Env,row:PulseDomain):DomainInfo{
 const records=z.array(z.object({type:z.enum(["TXT","CNAME"]),name:z.string().max(253),value:z.string().max(1024)})).max(8).safeParse(JSON.parse(row.records));
 return domainInfo.parse({hostname:row.hostname,status:row.status,disableAt:null,ssl:row.ssl,records:row.status==="disabled"?[]:[{type:"TXT",name:`_blup.${row.hostname}`,value:row.proof},...(env.SITES_CNAME_TARGET?[{type:"CNAME",name:row.hostname,value:env.SITES_CNAME_TARGET}]:[]),...(records.success?records.data:[])]});
}
async function refresh(env:Cloudflare.Env,input:PulseDomain,reenable=false){
 if(input.checked_at!==null&&Date.now()-input.checked_at<30_000&&!reenable)return display(env,input);
 const lock=await siteLock(env.DB,`pulse-domain:${input.page_id}`,60_000);if(!lock)return display(env,input);
 try{
  let row=await read(env,input.page_id);if(!row)throw new DomainError("DOMAIN_PROVIDER");
  if(row.status==="disabled"){
   if(!reenable)return display(env,row);
   await removeDomainProvider(env,{hostname:row.hostname,providerId:row.provider_id,verifiedAt:row.verified_at});
   const proof=`blup-verification=${crypto.randomUUID().replaceAll("-","")}${crypto.randomUUID().replaceAll("-","")}`;
   await env.DB.batch([releaseHostname(env.DB,row.hostname,"pulse",row.page_id),env.DB.prepare("UPDATE pulse_page_domain SET status='pending_ownership',ssl='pending',proof=?,provider_id=NULL,verified_at=NULL,records='[]',checked_at=NULL WHERE page_id=?").bind(proof,row.page_id)]);return display(env,{...row,proof,status:"pending_ownership",ssl:"pending",provider_id:null,verified_at:null,records:"[]",checked_at:null});
  }
  if(row.verified_at===null&&!await provesOwnership(row.hostname,row.proof)){await env.DB.prepare("UPDATE pulse_page_domain SET checked_at=? WHERE page_id=?").bind(Date.now(),row.page_id).run();return display(env,row);}
  if(!await claimHostname(env.DB,row.hostname,"pulse",row.page_id))throw new DomainError("DOMAIN_TAKEN");
  await env.DB.prepare("UPDATE pulse_page_domain SET verified_at=coalesce(verified_at,?) WHERE page_id=?").bind(Date.now(),row.page_id).run();
  const response=await providerCall(env,row.provider_id?"GET":"POST",row.provider_id??undefined,row.hostname);if(!response)throw new DomainError("DOMAIN_PROVIDER");
  const ssl=response.ssl?.status==="active"?"active":/failed|expired|deleted|timed_out/.test(response.ssl?.status??"")?"error":"pending",status=response.status==="active"&&ssl==="active"?"active":/blocked|deleted|moved/.test(response.status)?"error":"pending";
  row={...row,provider_id:response.id,verified_at:Date.now(),status,ssl,records:JSON.stringify(providerRecords(response)),checked_at:Date.now()};
  await env.DB.prepare("UPDATE pulse_page_domain SET provider_id=?,verified_at=?,status=?,ssl=?,records=?,checked_at=? WHERE page_id=?").bind(row.provider_id,row.verified_at,status,ssl,row.records,row.checked_at,row.page_id).run();return display(env,row);
 }finally{await unlockSite(env.DB,`pulse-domain:${input.page_id}`,lock);}
}
export const pulseDomainsApi=new Hono<PulseApp>()
 .onError((error,c)=>{if(error instanceof HTTPException)return error.getResponse().headers.get("content-type")?.includes("application/json")?error.getResponse():c.json({code:"INVALID"},error.status);const code=error instanceof DomainError?error.code:"DOMAIN_PROVIDER";return c.json({code},code==="DOMAIN_UNAVAILABLE"?503:code==="DOMAIN_TAKEN"?409:502);})
 .get("/:page/domain",validate("param",pageParam),async c=>{
  const page=await ownedPage(c.env,c.get("session").user.id,c.req.valid("param").page);if(!page||!c.get("access").readable)return c.json({code:"NOT_FOUND"},404);
  let row=await read(c.env,page.id);if(!row)return c.json(null);
  if(!c.get("access").writable||!c.get("access").limits?.customDomain||page.planDisabled){await disablePulseDomain(c.env,page.id,false,true);row=await read(c.env,page.id)??{...row,status:"disabled",ssl:"pending"};return c.json(display(c.env,row));}
  return c.json(await refresh(c.env,row));
 })
 .post("/:page/domain",pulseWrite,validate("param",pageParam),validate("json",z.strictObject({hostname:hostnameSchema})),async c=>{
  const page=await ownedPage(c.env,c.get("session").user.id,c.req.valid("param").page);if(!page)return c.json({code:"NOT_FOUND"},404);if(page.planDisabled||!c.get("access").limits?.customDomain)return c.json({code:"PLAN"},403);
  if(!c.env.CF_API_TOKEN||!c.env.CF_ZONE_ID||!c.env.SITES_CNAME_TARGET)return c.json({code:"DOMAIN_UNAVAILABLE"},503);
  const hostname=c.req.valid("json").hostname,current=await read(c.env,page.id);if(current&&current.hostname!==hostname)return c.json({code:"DOMAIN_EXISTS"},409);
  await c.env.DB.prepare("INSERT OR IGNORE INTO pulse_page_domain(page_id,hostname,proof) VALUES(?,?,?)").bind(page.id,hostname,`blup-verification=${crypto.randomUUID().replaceAll("-","")}${crypto.randomUUID().replaceAll("-","")}`).run();
  const row=await read(c.env,page.id);if(!row)throw new DomainError("DOMAIN_PROVIDER");return c.json(await refresh(c.env,row,true));
 })
 .delete("/:page/domain",validate("param",pageParam),async c=>{
  const page=await ownedPage(c.env,c.get("session").user.id,c.req.valid("param").page);if(!page)return c.json({code:"NOT_FOUND"},404);await disablePulseDomain(c.env,page.id,true);return c.json({ok:true});
 });
