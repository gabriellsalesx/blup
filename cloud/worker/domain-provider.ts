import { z } from "zod";

export const dnsRecord=z.object({type:z.enum(["TXT","CNAME"]),name:z.string().min(1).max(253),value:z.string().min(1).max(1024)});
export const domainInfo=z.object({hostname:z.string(),status:z.enum(["pending_ownership","pending","active","error","disabled"]),disableAt:z.iso.datetime().nullable(),ssl:z.enum(["pending","active","error"]),records:z.array(dnsRecord).max(10)});
export type DomainInfo=z.infer<typeof domainInfo>;
export const hostnameSchema=z.string().trim().min(3).max(253).transform((raw,ctx)=>{
	let host="";
	try{
		if(/[\s/:@?#\\%\u0000-\u001f\u007f]/.test(raw))throw new Error();
		host=new URL(`https://${raw}`).hostname.toLowerCase();
		if(!host.includes(".") || host.endsWith(".") || host.length>253 || host.split(".").some(l=>!/^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/.test(l)))throw new Error();
		if(/^[0-9.]+$/.test(host) || ["getblup.io","getblup.site","localhost","local","internal"].some(d=>host===d||host.endsWith(`.${d}`)))throw new Error();
	}catch{ctx.addIssue({code:"custom",message:"hostname"});return z.NEVER;}
	return host;
});
const providerHostname=z.object({
	id:z.string().regex(/^[A-Za-z0-9-]{1,128}$/),hostname:hostnameSchema,status:z.string().max(64),
	ssl:z.object({status:z.string().max(64),validation_records:z.array(z.object({txt_name:z.string().max(253).optional(),txt_value:z.string().max(1024).optional(),cname_name:z.string().max(253).optional(),cname_target:z.string().max(253).optional()})).max(20).optional()}).optional(),
	ownership_verification:z.object({type:z.literal("txt"),name:z.string().max(253),value:z.string().max(1024)}).optional(),
});
export type ProviderHostname=z.infer<typeof providerHostname>;
export class DomainError extends Error {
	constructor(readonly code:"DOMAIN_PROVIDER"|"DOMAIN_UNAVAILABLE"|"DOMAIN_TAKEN"){super(code);}
}
export async function providerCall(env:Cloudflare.Env,method:"GET"|"POST"|"DELETE",id?:string,hostname?:string):Promise<ProviderHostname|null> {
	const zone=z.string().regex(/^[a-f0-9]{32}$/i).safeParse(env.CF_ZONE_ID);
	if(!zone.success||!env.CF_API_TOKEN||method!=="DELETE"&&!env.SITES_CNAME_TARGET)throw new DomainError("DOMAIN_UNAVAILABLE");
	const path=`https://api.cloudflare.com/client/v4/zones/${zone.data}/custom_hostnames${id?`/${encodeURIComponent(id)}`:""}`;
	try{
		const res=await fetch(path,{method,redirect:"error",signal:AbortSignal.timeout(15_000),headers:{Authorization:`Bearer ${env.CF_API_TOKEN}`,"content-type":"application/json"},...(method==="POST"?{body:JSON.stringify({hostname,ssl:{method:"txt",type:"dv",settings:{min_tls_version:"1.2",early_hints:"on"}}})}:{})});
		if(method==="DELETE"&&res.status===404)return null;
		if(!res.ok){
   // Recover a hostname created before a previous request lost its D1
   // acknowledgement. This path runs only after our TXT proof succeeded.
   if(method==="POST"&&hostname){
    const lookup=new URL(path);lookup.searchParams.set("hostname.exact",hostname);lookup.searchParams.set("per_page","1");
    const found=await fetch(lookup,{headers:{Authorization:`Bearer ${env.CF_API_TOKEN}`},redirect:"error",signal:AbortSignal.timeout(15_000)});
    const existing=z.object({success:z.literal(true),result:z.array(providerHostname).max(1)}).safeParse(found.ok?await found.json():null);
    const match=existing.success?existing.data.result[0]:undefined;
    if(match?.hostname===hostname)return match;
   }
   throw new DomainError("DOMAIN_PROVIDER");
  }
		const value:unknown=await res.json();
		if(method==="DELETE"){
			if(!z.object({success:z.literal(true)}).safeParse(value).success)throw new DomainError("DOMAIN_PROVIDER");
			return null;
		}
		const parsed=z.object({success:z.literal(true),result:providerHostname}).safeParse(value);
		if(!parsed.success || hostname&&parsed.data.result.hostname!==hostname)throw new DomainError("DOMAIN_PROVIDER");
		return parsed.data.result;
	}catch(e){if(e instanceof DomainError)throw e;throw new DomainError("DOMAIN_PROVIDER");}
}
export function providerRecords(info:ProviderHostname):DomainInfo["records"] {
	const records:DomainInfo["records"]=[];
	if(info.ownership_verification){const r=info.ownership_verification;records.push({type:"TXT",name:r.name,value:r.value});}
	for(const r of info.ssl?.validation_records??[]){
		if(r.txt_name&&r.txt_value)records.push({type:"TXT",name:r.txt_name,value:r.txt_value});
		if(r.cname_name&&r.cname_target)records.push({type:"CNAME",name:r.cname_name,value:r.cname_target});
	}
	return records.slice(0,8);
}
export async function provesOwnership(hostname:string,proof:string) {
	try{
		const name=`_blup.${hostname}`;
		const url=new URL("https://cloudflare-dns.com/dns-query");url.searchParams.set("name",name);url.searchParams.set("type","TXT");
		const res=await fetch(url,{headers:{accept:"application/dns-json"},redirect:"error",signal:AbortSignal.timeout(5000)});
		if(!res.ok)return false;
		const dns=z.object({Status:z.literal(0),Answer:z.array(z.object({name:z.string().max(254),type:z.number().int(),data:z.string().max(4096)})).max(100).optional()}).safeParse(await res.json());
		if(!dns.success)return false;
		return dns.data.Answer?.some(r=>r.type===16&&r.name.toLowerCase().replace(/\.$/,"")===name&&r.data.replace(/^"|"$/g,"").replace(/"\s*"/g,"")===proof)??false;
	}catch{return false;}
}

// Provisioning can succeed remotely before its D1 acknowledgement. Keep
// deletion recoverable even when that request lost the provider ID.
export async function removeDomainProvider(env:Cloudflare.Env,row:{hostname:string;providerId:string|null;verifiedAt:number|null}){
	let id=row.providerId;
	if(!id&&row.verifiedAt!==null){
		const zone=z.string().regex(/^[a-f0-9]{32}$/i).safeParse(env.CF_ZONE_ID);
		if(!zone.success||!env.CF_API_TOKEN)throw new DomainError("DOMAIN_UNAVAILABLE");
		const url=new URL(`https://api.cloudflare.com/client/v4/zones/${zone.data}/custom_hostnames`);
		url.searchParams.set("hostname.exact",row.hostname);url.searchParams.set("per_page","1");
		try{
			const res=await fetch(url,{headers:{Authorization:`Bearer ${env.CF_API_TOKEN}`},redirect:"error",signal:AbortSignal.timeout(15_000)});
			const parsed=z.object({success:z.literal(true),result:z.array(providerHostname).max(1)}).safeParse(res.ok?await res.json():null);
			if(!parsed.success)throw new DomainError("DOMAIN_PROVIDER");
			const match=parsed.data.result[0];
			if(match&&match.hostname!==row.hostname)throw new DomainError("DOMAIN_PROVIDER");
			id=match?.id??null;
		}catch(e){if(e instanceof DomainError)throw e;throw new DomainError("DOMAIN_PROVIDER");}
	}
	if(id)await providerCall(env,"DELETE",id);
}
