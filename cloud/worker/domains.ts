import { queueSitePurge } from "./site-cache-purge";
import { Hono } from "hono";
import { z } from "zod";
import { siteLock,unlockSite } from "./quota";
import { disableExpiredDomain } from "./domain-downgrade";
import { accountPlan,observePlan } from "./lifecycle";
import { DomainError,removeDomainProvider,domainInfo,hostnameSchema,providerCall,providerRecords,provesOwnership,type DomainInfo } from "./domain-provider";
import { idParam,validate } from "./validation";
type App={Bindings:Cloudflare.Env;Variables:{userId:string}};
type DomainRow={site_id:string;hostname:string;proof:string;provider_id:string|null;verified_at:number|null;status:string;ssl:string;records:string;checked_at:number|null;disable_at:number|null;disabled_at:number|null};
async function read(env:Cloudflare.Env,id:string){return env.DB.prepare("SELECT * FROM site_domain WHERE site_id=?").bind(id).first<DomainRow>();}
function display(env:Cloudflare.Env,row:DomainRow):DomainInfo {
	const saved=z.array(z.object({type:z.enum(["TXT","CNAME"]),name:z.string(),value:z.string()})).max(8).safeParse(JSON.parse(row.records));
	if(row.status==="disabled")return domainInfo.parse({hostname:row.hostname,status:"disabled",ssl:"pending",disableAt:row.disable_at===null?null:new Date(row.disable_at).toISOString(),records:[]});
	return domainInfo.parse({hostname:row.hostname,status:row.status,ssl:row.ssl,disableAt:row.disable_at===null?null:new Date(row.disable_at).toISOString(),records:[{type:"TXT",name:`_blup.${row.hostname}`,value:row.proof},...(env.SITES_CNAME_TARGET?[{type:"CNAME",name:row.hostname,value:env.SITES_CNAME_TARGET}]:[]),...(saved.success?saved.data:[])]});
}
async function refresh(env:Cloudflare.Env,row:DomainRow,reenable=false):Promise<DomainInfo>{
	if(row.checked_at && Date.now()-row.checked_at<30_000&&!(reenable&&row.status==="disabled"))return display(env,row);
	const lock=await siteLock(env.DB,`domain:${row.site_id}`,60_000);
	if(!lock)return display(env,row);
	try{
		row=await read(env,row.site_id)??row;
		if(row.status==="disabled"){
			if(!reenable)return display(env,row);
			await removeDomainProvider(env,{hostname:row.hostname,providerId:row.provider_id,verifiedAt:row.verified_at});
			await env.DB.prepare("UPDATE site_domain SET status='pending_ownership',ssl='pending',proof=?,verified_at=NULL,provider_id=NULL,records='[]',checked_at=NULL,disable_at=NULL,disabled_at=NULL WHERE site_id=?").bind(`blup-verification=${crypto.randomUUID().replaceAll("-","")}${crypto.randomUUID().replaceAll("-","")}`,row.site_id).run();
			const reset=await read(env,row.site_id);if(!reset)throw new DomainError("DOMAIN_PROVIDER");return display(env,reset);
		}
		if(!row.verified_at && !await provesOwnership(row.hostname,row.proof)){
			await env.DB.prepare("UPDATE site_domain SET checked_at=? WHERE site_id=?").bind(Date.now(),row.site_id).run();
			return display(env,row);
		}
		if(!row.verified_at){
			try{await env.DB.prepare("UPDATE site_domain SET verified_at=? WHERE site_id=? AND verified_at IS NULL").bind(Date.now(),row.site_id).run();}
			catch(e){if(String(e).includes("UNIQUE"))throw new DomainError("DOMAIN_TAKEN");throw e;}
		}
		const response=await providerCall(env,row.provider_id?"GET":"POST",row.provider_id??undefined,row.hostname);
		if(!response)throw new DomainError("DOMAIN_PROVIDER");
		const ssl=response.ssl?.status==="active"?"active":/^(expired|deleted|validation_timed_out|validation_failed|issuance_failed)$/.test(response.ssl?.status??"")?"error":"pending";
		const status=response.status==="active"&&ssl==="active"?"active":/^(blocked|deleted|moved)$/.test(response.status)?"error":"pending";
		await env.DB.prepare("UPDATE site_domain SET verified_at=coalesce(verified_at,?),provider_id=?,status=?,ssl=?,records=?,checked_at=? WHERE site_id=? AND hostname=?").bind(Date.now(),response.id,status,ssl,JSON.stringify(providerRecords(response)),Date.now(),row.site_id,row.hostname).run();
		return display(env,{...row,verified_at:Date.now(),provider_id:response.id,status,ssl,records:JSON.stringify(providerRecords(response))});
	}finally{await unlockSite(env.DB,`domain:${row.site_id}`,lock);}
}
export const domainsApi=new Hono<App>()
	.use(async(c,next)=>{
		const id=idParam.safeParse(c.req.param());if(!id.success)return c.json({code:"INVALID"},400);
		const owned=await c.env.DB.prepare("SELECT id FROM site WHERE id=? AND user_id=?").bind(id.data.id,c.get("userId")).first();
		if(!owned)return c.json({code:"NOT_FOUND"},404);
		await next();
	})
	.get("/",validate("param",idParam),async c=>{
		const id=c.req.valid("param").id;
		try{
			const plan=await accountPlan(c.env,c.get("userId"));
			if(plan)await observePlan(c.env.DB,c.get("userId"),plan.name);
			let row=await read(c.env,id);if(!row)return c.json(null);
			if(row.disable_at!==null&&row.disable_at<=Date.now()){
				try{await disableExpiredDomain(c.env,id);}catch{console.error("[blup] domain_disable_retry");}
				row=await read(c.env,id)??row;
				if(row.disable_at!==null&&row.disable_at<=Date.now())row={...row,status:"disabled"};
			}
			return c.json(plan?.limits.customDomain?await refresh(c.env,row):display(c.env,row));
		}catch(e){if(e instanceof DomainError)return c.json({code:e.code},e.code==="DOMAIN_UNAVAILABLE"?503:e.code==="DOMAIN_TAKEN"?409:502);throw e;}

	})
	.post("/",validate("param",idParam),validate("json",z.object({hostname:hostnameSchema}).strict()),async c=>{
		if(!(await accountPlan(c.env,c.get("userId")))?.limits.customDomain)return c.json({code:"PLAN"},403);
		if(!c.env.CF_API_TOKEN||!c.env.CF_ZONE_ID||!c.env.SITES_CNAME_TARGET)return c.json({code:"DOMAIN_UNAVAILABLE"},503);
		const {id}=c.req.valid("param"),{hostname}=c.req.valid("json");
		const claimed=await c.env.DB.prepare("SELECT site_id FROM site_domain WHERE hostname=? AND verified_at IS NOT NULL AND site_id<>?").bind(hostname,id).first();
		if(claimed)return c.json({code:"DOMAIN_TAKEN"},409);
		const current=await read(c.env,id);
		if(current&&current.hostname!==hostname)return c.json({code:"DOMAIN_EXISTS"},409);
		const added=await c.env.DB.prepare("INSERT OR IGNORE INTO site_domain(site_id,hostname,proof) VALUES (?,?,?) RETURNING site_id").bind(id,hostname,`blup-verification=${crypto.randomUUID().replaceAll("-","")}${crypto.randomUUID().replaceAll("-","")}`).all();
		if(!current&&!added.results.length)return c.json({code:"DOMAIN_TAKEN"},409);
		const row=await read(c.env,id);if(!row)return c.json({code:"DOMAIN_PROVIDER"},502);
		try{return c.json(await refresh(c.env,row,true));}catch(e){if(e instanceof DomainError)return c.json({code:e.code},e.code==="DOMAIN_UNAVAILABLE"?503:e.code==="DOMAIN_TAKEN"?409:502);throw e;}
	})
	.delete("/",validate("param",idParam),async c=>{
		const {id}=c.req.valid("param"),lock=await siteLock(c.env.DB,`domain:${id}`,60_000);
		if(!lock)return c.json({code:"TOO_MANY_REQUESTS"},429);
		try{
			const row=await read(c.env,id);if(row)await removeDomainProvider(c.env,{hostname:row.hostname,providerId:row.provider_id,verifiedAt:row.verified_at});
			await queueSitePurge(c.env.DB,id).run();
			await c.env.DB.prepare("DELETE FROM site_domain WHERE site_id=?").bind(id).run();
			return c.json({ok:true});
		}catch(e){if(e instanceof DomainError)return c.json({code:e.code},e.code==="DOMAIN_UNAVAILABLE"?503:e.code==="DOMAIN_TAKEN"?409:502);throw e;}
		finally{await unlockSite(c.env.DB,`domain:${id}`,lock);}
	});
