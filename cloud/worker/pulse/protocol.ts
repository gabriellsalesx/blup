/// <reference path="./node-zlib.d.ts" />
import {z} from "zod";
import {boundedBody,BodyError} from "../validation";
import {projectId} from "./limits";
import {brotliDecompressSync} from "node:zlib";

export class IngestError extends Error{constructor(readonly status:400|403|413|415|429|503,readonly code:string,readonly wait=0){super(code);}}
const jsonObject=z.record(z.string(),z.unknown());
const eventId=z.string().regex(/^(?:[a-fA-F0-9]{32}|[a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})$/).transform(v=>v.replaceAll("-","").toLowerCase());
const envelopeHeader=z.object({event_id:eventId.optional(),dsn:z.string().max(2048).optional()});
const itemHeader=z.object({type:z.string().min(1).max(80),length:z.number().int().min(0).max(1024*1024).optional(),attachment_type:z.string().max(80).optional()});
const decoder=new TextDecoder("utf-8",{fatal:true,ignoreBOM:false});
function parseJson(bytes:Uint8Array){try{return JSON.parse(decoder.decode(bytes)) as unknown;}catch{throw new IngestError(400,"INVALID");}}
export function checkDepth(value:unknown,depth=0,budget={left:20_000}):void{
 if(depth>32||--budget.left<0)throw new IngestError(400,"STRUCTURE");
 if(Array.isArray(value)){if(value.length>2048)throw new IngestError(400,"STRUCTURE");for(const v of value)checkDepth(v,depth+1,budget);}
 else if(value!==null&&typeof value==="object"){const pairs=Object.entries(value);if(pairs.length>512)throw new IngestError(400,"STRUCTURE");for(const [k,v] of pairs){if(k.length>256)throw new IngestError(400,"STRUCTURE");checkDepth(v,depth+1,budget);}}
}
function parsedEvent(bytes:Uint8Array){if(bytes.byteLength>204_800)throw new IngestError(413,"EVENT_SIZE");const value=parseJson(bytes);checkDepth(value);const p=jsonObject.safeParse(value);if(!p.success)throw new IngestError(400,"INVALID");return p.data;}
export async function ingestBytes(req:Request):Promise<Uint8Array<ArrayBuffer>>{
 let raw:Uint8Array<ArrayBuffer>;try{raw=await boundedBody(req,1024*1024);}catch{throw new IngestError(413,"ENVELOPE_SIZE");}
 const encoding=req.headers.get("content-encoding")?.toLowerCase()??"identity";
 if(encoding==="identity")return raw;
 // sentry-python sends "br" when the brotli module is installed; Relay
 // takes gzip, deflate and br (develop.sentry.dev, envelopes). Workers have
 // no "br" DecompressionStream, so node:zlib, capped at the same 1 MiB.
 if(encoding==="br"){
  try{return new Uint8Array(brotliDecompressSync(raw,{maxOutputLength:1024*1024}));}
  catch(e){throw e instanceof RangeError?new IngestError(413,"ENVELOPE_SIZE"):new IngestError(400,"COMPRESSION");}
 }
 if(encoding!=="gzip"&&encoding!=="deflate")throw new IngestError(415,"ENCODING");
 try{
  const stream=new Blob([raw]).stream().pipeThrough(new DecompressionStream(encoding));
  const init={method:"POST",body:stream,duplex:"half"};return await boundedBody(new Request("https://decompress.invalid",init),1024*1024);
 }catch(e){if(e instanceof BodyError)throw new IngestError(e.status===413?413:400,e.status===413?"ENVELOPE_SIZE":"INVALID");throw new IngestError(400,"COMPRESSION");}
}
export function parseEnvelope(bytes:Uint8Array,store=false){
 if(store)return {event:parsedEvent(bytes),id:undefined as string|undefined,dsn:undefined as string|undefined,minidump:false};
 let offset=0;
 const line=()=>{const start=offset;while(offset<bytes.length&&bytes[offset]!==10)offset++;if(offset-start>8192)throw new IngestError(400,"HEADER_SIZE");const result=bytes.subarray(start,offset);if(offset<bytes.length)offset++;return result;};
 const header=envelopeHeader.safeParse(parseJson(line()));if(!header.success)throw new IngestError(400,"INVALID");
 // A minidump (Electron and native SDKs) is never kept: it only marks the
 // event as a native crash.
 let event:Record<string,unknown>|undefined,count=0,minidump=false;
 while(offset<bytes.length){
  if(++count>100)throw new IngestError(400,"ITEMS");
  const parsed=itemHeader.safeParse(parseJson(line()));if(!parsed.success)throw new IngestError(400,"INVALID");
  const item=parsed.data;let payload:Uint8Array;
  if(item.length!==undefined){if(offset+item.length>bytes.length)throw new IngestError(400,"LENGTH");payload=bytes.subarray(offset,offset+item.length);offset+=item.length;if(offset<bytes.length){if(bytes[offset]!==10)throw new IngestError(400,"LENGTH");offset++;}}
  else {const start=offset;while(offset<bytes.length&&bytes[offset]!==10)offset++;payload=bytes.subarray(start,offset);if(offset<bytes.length)offset++;}
  if(item.type==="event"){if(event)throw new IngestError(400,"MULTIPLE_EVENTS");event=parsedEvent(payload);}
  else if(item.type==="attachment"&&item.attachment_type==="event.minidump")minidump=true;
 }
 return {event,id:header.data.event_id,dsn:header.data.dsn,minidump};
}
export function normalizeEventId(event:Record<string,unknown>,headerId?:string){
 let id:string|undefined;
 if(event.event_id!==undefined){const p=eventId.safeParse(event.event_id);if(!p.success)throw new IngestError(400,"EVENT_ID");id=p.data;}
 if(id&&headerId&&id!==headerId)throw new IngestError(400,"EVENT_ID");
 return id??headerId??crypto.randomUUID().replaceAll("-","");
}
// The protocol version an SDK states: a whole number, as Sentry accepts it
// (most SDKs send 7; sentry-elixir still sends 5).
const sentryVersion=(v:string|null|undefined)=>!!v&&/^[1-9][0-9]?$/.test(v);
export function authKey(req:Request,project:string,dsn?:string){
 if(!projectId.safeParse(project).success)throw new IngestError(403,"KEY");
 const keys:string[]=[];const url=new URL(req.url);
 for(const [name,value] of url.searchParams){if(!["sentry_key","sentry_version","sentry_client","sentry_secret"].includes(name)||value.length>512)throw new IngestError(400,"AUTH");if(url.searchParams.getAll(name).length!==1)throw new IngestError(400,"AUTH");}
 if(url.searchParams.has("sentry_key")){if(!sentryVersion(url.searchParams.get("sentry_version")))throw new IngestError(400,"AUTH");keys.push(url.searchParams.get("sentry_key")??"");}
 const auth=req.headers.get("x-sentry-auth");
 if(auth){if(auth.length>4096||!auth.startsWith("Sentry "))throw new IngestError(400,"AUTH");const pairs=new Map<string,string>();for(const part of auth.slice(7).split(",")){const [,name,value]=/^\s*(sentry_\w+)=(.*?)\s*$/.exec(part)??[];if(name===undefined||value===undefined||pairs.has(name))throw new IngestError(400,"AUTH");pairs.set(name,value);}if(!sentryVersion(pairs.get("sentry_version")))throw new IngestError(400,"AUTH");keys.push(pairs.get("sentry_key")??"");}
 if(dsn){let d:URL;try{d=new URL(dsn);}catch{throw new IngestError(400,"AUTH");}if(d.protocol!==url.protocol||d.host!==url.host||d.password||d.search||d.hash||d.pathname!==`/${project}`)throw new IngestError(403,"KEY");keys.push(d.username);}
 const [key]=keys;if(key===undefined||keys.some(k=>!/^([a-f0-9]{48})$/.test(k)||k!==key))throw new IngestError(403,"KEY");return key;
}
