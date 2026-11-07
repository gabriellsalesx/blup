import {z} from "zod";

// Threat model: global_fetch_strictly_public routes like a public client,
// never to the zone's private origin. Network address restrictions are a
// Cloudflare production boundary, not reproduced by local workerd.
// https://developers.cloudflare.com/workers/configuration/compatibility-flags/#global-fetch-strictly-public
// https://developers.cloudflare.com/support/troubleshooting/http-status-codes/cloudflare-1xxx-errors/error-1002/
// Application checks are defense in depth; DNS preflight alone does not
// pin a later DNS lookup. Do not attach VPC/private networking bindings.
export function publicIPv4(ip:string){
 if(!z.ipv4().safeParse(ip).success)return false;
 const numeric=(v:string)=>v.split(".").reduce((n,b)=>n*256+Number(b),0),address=numeric(ip);
 const ranges=[["0.0.0.0",8],["10.0.0.0",8],["100.64.0.0",10],["127.0.0.0",8],["169.254.0.0",16],["172.16.0.0",12],["192.0.0.0",24],["192.0.2.0",24],["192.88.99.0",24],["192.168.0.0",16],["198.18.0.0",15],["198.51.100.0",24],["203.0.113.0",24],["224.0.0.0",4],["240.0.0.0",4]] as const;
 return !ranges.some(([start,bits])=>Math.floor(address/2**(32-bits))===Math.floor(numeric(start)/2**(32-bits)));
}
export function publicIPv6(ip:string){
 if(!z.ipv6().safeParse(ip).success||ip.includes("."))return false;
 const value=(v:string)=>{const [left,right]=v.toLowerCase().split("::"),a=left?left.split(":"):[],b=right?right.split(":"):[];return [...a,...Array<string>(8-a.length-b.length).fill("0"),...b].reduce((n,h)=>n*65536n+BigInt(`0x${h}`),0n);};
 const address=value(ip),inRange=(start:string,bits:number)=>address>>BigInt(128-bits)===value(start)>>BigInt(128-bits);
 const reserved=[["2001::",23],["2001:db8::",32],["2002::",16],["3ffe::",16],["3fff::",20]] as const;
 return inRange("2000::",3)&&!reserved.some(([start,bits])=>inRange(start,bits));
}
export function monitorURL(raw:string,allowBlup=false):URL{
 if(raw.length>2048||/[\u0000-\u0020\u007f\\]/.test(raw))throw new Error("INVALID_TARGET");
 const url=new URL(raw),host=url.hostname.toLowerCase().replace(/\.$/,"");
 if(!["https:","http:"].includes(url.protocol)||url.username||url.password||url.hash||url.port||!host)throw new Error("INVALID_TARGET");
 const ip=host.replace(/^\[|\]$/g,"");
 if(z.ipv4().safeParse(ip).success){if(!publicIPv4(ip))throw new Error("INVALID_TARGET");}
 else if(host.startsWith("[")){if(!publicIPv6(ip))throw new Error("INVALID_TARGET");}
 else if(!host.includes(".")||["localhost","local","internal","invalid","test","example","home","lan","onion","arpa"].some(d=>host===d||host.endsWith(`.${d}`)))throw new Error("INVALID_TARGET");
 const blup=["getblup.io","getblup.site","workers.dev"].some(d=>host===d||host.endsWith(`.${d}`));
 // Owner operational checks can reach public Blup pages, never ingestion
 // or private APIs. Customer monitors cannot loop into our own services.
 if(blup&&(!allowBlup||host.endsWith("workers.dev")||host==="ingest.getblup.io"||/^\/api(?:\/|$)/.test(url.pathname)))throw new Error("INVALID_TARGET");
 url.hostname=host;return url;
}
export const monitorURLSchema=z.string().max(2048);
const dnsSchema=z.object({Status:z.literal(0),Answer:z.array(z.object({type:z.number().int(),data:z.string().max(1024)})).max(100).optional()});
export async function publicDNS(host:string,signal:AbortSignal,send:typeof fetch=fetch){
 const ip=host.replace(/^\[|\]$/g,"");if(z.ipv4().safeParse(ip).success||z.ipv6().safeParse(ip).success)return;
 let found=false;
 for(const type of ["A","AAAA"]){const url=new URL("https://cloudflare-dns.com/dns-query");url.searchParams.set("name",host);url.searchParams.set("type",type);const res=await send(url,{headers:{accept:"application/dns-json"},redirect:"manual",signal});if(!res.ok)throw new Error("DNS");const parsed=dnsSchema.safeParse(await res.json());if(!parsed.success)throw new Error("DNS");for(const answer of parsed.data.Answer??[]){if(answer.type===1||answer.type===28){if(!(answer.type===1?publicIPv4(answer.data):publicIPv6(answer.data)))throw new Error("INVALID_TARGET");found=true;}}}
 if(!found)throw new Error("DNS");
}
export type CheckResult={outcome:"up"|"down"|"unknown";status:number|null;durationMs:number;reason:"http"|"timeout"|"network"|"blocked"|"gap"|"keyword"};
// How a monitor asks and what counts as up: the method, the statuses that
// count (empty = any 2xx), a keyword that must (or must not) be in the page
// and the time to wait. Request headers go apart (sealed at rest).
export const monitorOptions=z.object({
 method:z.enum(["GET","HEAD"]),
 expectedStatus:z.array(z.number().int().min(100).max(599)).max(10).refine(v=>new Set(v).size===v.length),
 keyword:z.object({text:z.string().trim().min(1).max(200),mode:z.enum(["contains","absent"])}).strict().nullable(),
 timeoutSeconds:z.number().int().min(1).max(30),
}).strict().refine(o=>!(o.keyword&&o.method==="HEAD"),{message:"A keyword needs the page body (GET)."});
export type MonitorOptions=z.infer<typeof monitorOptions>;
export const defaultMonitorOptions:MonitorOptions={method:"GET",expectedStatus:[],keyword:null,timeoutSeconds:10};
// Headers the runtime or the connection own, or that would say who we are.
const reservedHeaders=new Set(["host","content-length","transfer-encoding","connection","upgrade","keep-alive","te","trailer","proxy-authorization","proxy-connection","user-agent","accept-encoding","expect","forwarded","x-forwarded-for","x-real-ip","cf-connecting-ip"]);
export const headerName=z.string().regex(/^[A-Za-z0-9!#$%&'*+.^_`|~-]{1,64}$/).refine(n=>!reservedHeaders.has(n.toLowerCase()));
export const monitorHeader=z.object({name:headerName,value:z.string().max(1024).regex(/^[^\r\n\0]*$/)}).strict();
export type MonitorHeader=z.infer<typeof monitorHeader>;
const BODY_LIMIT=256*1024;
// Reads at most the first 256 KB of a page, as text, then lets the rest go.
async function pageText(res:Response){
 const reader=res.body?.getReader();if(!reader)return "";const chunks:Uint8Array[]=[];let size=0;
 try{while(size<BODY_LIMIT){const {done,value}=await reader.read();if(done)break;chunks.push(value);size+=value.byteLength;}}finally{await reader.cancel().catch(()=>{});}
 const bytes=new Uint8Array(Math.min(size,BODY_LIMIT));let at=0;for(const c of chunks){const part=c.subarray(0,Math.max(0,bytes.length-at));bytes.set(part,at);at+=part.byteLength;if(at>=bytes.length)break;}
 return new TextDecoder().decode(bytes);
}
export async function checkHTTP(raw:string,allowBlup=false,send:typeof fetch=fetch,options:MonitorOptions=defaultMonitorOptions,headers:MonitorHeader[]=[]):Promise<CheckResult>{
 const start=Date.now(),signal=AbortSignal.timeout(options.timeoutSeconds*1000);let url:URL;
 try{url=monitorURL(raw,allowBlup);}catch{return {outcome:"unknown",status:null,durationMs:0,reason:"blocked"};}
 const origin=url.origin,accepted=(status:number)=>options.expectedStatus.length?options.expectedStatus.includes(status):status>=200&&status<300;
 try{
  for(let hop=0;hop<=3;hop++){
   await publicDNS(url.hostname,signal,send);
   // The monitor's own headers only go to its own origin, never along a
   // redirect to another one.
   const sent=new Headers({"User-Agent":"BlupPulse/1.0 (+https://getblup.io)",Accept:"*/*"});if(url.origin===origin)for(const h of headers)sent.set(h.name,h.value);
   const res=await send(url,{method:options.method,redirect:"manual",signal,cache:"no-store",headers:sent});
   if([301,302,303,307,308].includes(res.status)&&!accepted(res.status)){
    await res.body?.cancel();
    const location=res.headers.get("location");if(!location||hop===3)return {outcome:"down",status:res.status,durationMs:Date.now()-start,reason:"http"};
    url=monitorURL(new URL(location,url).href,allowBlup);continue;
   }
   if(!accepted(res.status)){await res.body?.cancel();return {outcome:"down",status:res.status,durationMs:Date.now()-start,reason:"http"};}
   // The page is read only for a keyword, and never kept.
   if(options.keyword){const found=(await pageText(res)).toLowerCase().includes(options.keyword.text.toLowerCase());if(found!==(options.keyword.mode==="contains"))return {outcome:"down",status:res.status,durationMs:Date.now()-start,reason:"keyword"};}
   else await res.body?.cancel();
   return {outcome:"up",status:res.status,durationMs:Date.now()-start,reason:"http"};
  }
 }catch(e){return {outcome:e instanceof Error&&e.message==="INVALID_TARGET"?"unknown":"down",status:null,durationMs:Date.now()-start,reason:e instanceof Error&&e.message==="INVALID_TARGET"?"blocked":signal.aborted?"timeout":"network"};}
 return {outcome:"unknown",status:null,durationMs:Date.now()-start,reason:"gap"};
}
