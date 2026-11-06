import {z} from "zod";
import {sha256} from "./limits";
import {IngestError} from "./protocol";

// Intentionally closed projection: users, cookies, IP, headers, bodies,
// query strings, extra, tags, local variables and attachments never reach
// storage. Kept, cleaned: the app's own code around each of its frames, the
// route (no query, ids as :id), browser/OS/runtime name with the major
// version, and the trail of the last actions (kind and cleaned message).
export function cleanText(value:string,max=256){return value
 .replace(/(?:https?:\/\/)[^\s"<>]+/gi,v=>{try{const u=new URL(v);return `${u.protocol}//${u.hostname}${u.pathname.replace(/\b[a-f0-9]{16,}\b/gi,"[id]")}`;}catch{return "[url]";}})
 .replace(/\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b/gi,"[email]")
 .replace(/(?:[A-Z]:\\|\/(?:home|Users|root)\/)[^\s"<>]+/g,"[path]")
 .replace(/\b(?:bearer\s+\S+|(?:sk|pk)_(?:live|test)_\S+|eyJ[\w-]+\.[\w-]+\.[\w-]+)\b/gi,"[secret]")
 .replace(/\b(password|passwd|secret|token|authorization|api[_-]?key)\s*[:=]\s*[^\s,;]+/gi,"$1=[secret]")
 .replace(/[\u0000-\u001f\u007f]/g," ").slice(0,max);}
const codeLines=z.array(z.string().max(8192)).max(64);
const frame=z.object({filename:z.string().max(8192).optional(),abs_path:z.string().max(8192).optional(),function:z.string().max(8192).optional(),module:z.string().max(8192).optional(),lineno:z.number().int().min(0).max(10_000_000).optional(),colno:z.number().int().min(0).max(10_000_000).optional(),in_app:z.boolean().optional(),pre_context:codeLines.optional().catch(undefined),context_line:z.string().max(8192).optional().catch(undefined),post_context:codeLines.optional().catch(undefined)});
const error=z.object({type:z.string().max(8192).optional(),value:z.string().max(204800).optional(),stacktrace:z.object({frames:z.array(frame).max(512)}).optional()});
const schema=z.object({type:z.enum(["default","error","event"]).optional(),platform:z.string().max(80).optional(),level:z.enum(["fatal","error","warning","info","debug"]).optional(),message:z.union([z.string().max(204800),z.object({formatted:z.string().max(204800).optional(),message:z.string().max(204800).optional()})]).optional(),logentry:z.object({formatted:z.string().max(204800).optional(),message:z.string().max(204800).optional()}).optional(),exception:z.union([z.object({values:z.array(error).max(32)}),z.array(error).max(32)]).transform(v=>Array.isArray(v)?{values:v}:v).optional(),fingerprint:z.array(z.string().max(256)).max(32).optional(),environment:z.string().max(256).optional(),release:z.string().max(256).optional()});
// The stored projection, checked again when read back from the project's SQLite.
const storedFrame=z.object({filename:z.string(),function:z.string(),module:z.string(),line:z.number().int().nullable(),column:z.number().int().nullable(),inApp:z.boolean(),context:z.object({pre:z.array(z.string()).max(5),line:z.string(),post:z.array(z.string()).max(5)}).strict().optional()}).strict();
const storedContext=z.object({method:z.string().optional(),route:z.string().optional(),browser:z.string().optional(),os:z.string().optional(),runtime:z.string().optional()}).strict();
const storedCrumb=z.object({at:z.number().int().nullable(),type:z.string(),category:z.string(),level:z.string(),message:z.string()}).strict();
export const storedEvent=z.object({id:z.string(),group:z.string(),title:z.string(),level:schema.shape.level.unwrap(),platform:z.string(),environment:z.string(),release:z.string(),message:z.string(),exceptions:z.array(z.object({type:z.string(),value:z.string(),frames:z.array(storedFrame)}).strict()),context:storedContext.optional(),breadcrumbs:z.array(storedCrumb).max(30).optional()}).strict();
export type SafeEvent=z.infer<typeof storedEvent>;
// The route of a URL: its path only, with numbers, UUIDs and long hex ids
// as ":id" (they name records, sometimes people).
export function routeOf(value:string){try{const url=new URL(value,"https://route.invalid");return url.pathname.split("/").map(part=>/^(?:\d+|[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}|[0-9a-f]{16,})$/i.test(part)?":id":part).join("/").slice(0,200);}catch{return "";}}
// A name and its major version ("Chrome 131"), never the full build.
const named=z.object({name:z.string().max(256).optional(),version:z.string().max(256).optional()});
function nameOf(value:unknown){const parsed=named.safeParse(value);if(!parsed.success||!parsed.data.name)return undefined;const major=parsed.data.version?.match(/^v?(\d+)/i)?.[1];return cleanText(`${parsed.data.name}${major?` ${major}`:""}`,64);}
const request=z.object({url:z.string().max(8192).optional(),method:z.string().max(16).optional()});
const crumb=z.object({timestamp:z.union([z.number(),z.string().max(64)]).optional(),type:z.string().max(64).optional(),category:z.string().max(128).optional(),level:z.string().max(16).optional(),message:z.string().max(8192).optional(),data:z.record(z.string(),z.unknown()).optional()});
const crumbs=z.union([z.object({values:z.array(z.unknown()).max(512)}),z.array(z.unknown()).max(512)]);
// The code around a frame of the app: five lines each side at most, each
// cleaned like any text and cut at 200 characters.
function codeOf(f:z.infer<typeof frame>){if(!f.in_app||f.context_line===undefined)return undefined;const line=(v:string)=>cleanText(v,200);return {pre:(f.pre_context??[]).slice(-5).map(line),line:line(f.context_line),post:(f.post_context??[]).slice(0,5).map(line)};}
function contextOf(raw:Record<string,unknown>){
 const req=request.safeParse(raw.request),contexts=z.record(z.string(),z.unknown()).safeParse(raw.contexts),c=contexts.success?contexts.data:{};
 const method=req.success&&req.data.method&&/^[A-Z]{3,10}$/i.test(req.data.method)?req.data.method.toUpperCase():undefined,route=req.success&&req.data.url?routeOf(req.data.url):undefined;
 const context={...(method?{method}:{}),...(route?{route}:{}),...(nameOf(c.browser)?{browser:nameOf(c.browser)}:{}),...(nameOf(c.os)?{os:nameOf(c.os)}:{}),...(nameOf(c.runtime)?{runtime:nameOf(c.runtime)}:{})};
 return Object.keys(context).length?context:undefined;
}
// What happened before the error: the last 30 actions, with their kind and
// a cleaned message; of an HTTP call or a page change, only the method,
// route and status (never its query, headers or body).
function breadcrumbsOf(raw:Record<string,unknown>){
 const list=crumbs.safeParse(raw.breadcrumbs);if(!list.success)return undefined;
 const values=(Array.isArray(list.data)?list.data:list.data.values).slice(-30).flatMap(item=>{const c=crumb.safeParse(item);if(!c.success)return [];const d=c.data.data??{},text=(v:unknown)=>typeof v==="string"?v:"";
  const at=typeof c.data.timestamp==="number"?Math.round(c.data.timestamp<1e11?c.data.timestamp*1000:c.data.timestamp):c.data.timestamp?Date.parse(c.data.timestamp)||null:null;
  const http=c.data.type==="http"||["fetch","xhr","http"].includes(c.data.category??""),nav=c.data.category==="navigation"||c.data.type==="navigation";
  const message=http?[text(d.method).toUpperCase().slice(0,10),routeOf(text(d.url)),typeof d.status_code==="number"?String(d.status_code):""].filter(Boolean).join(" "):nav?[routeOf(text(d.from)),routeOf(text(d.to))].filter(Boolean).join(" → "):cleanText(c.data.message??"",200);
  return [{at:Number.isFinite(at)?at:null,type:cleanText(c.data.type??"default",32),category:cleanText(c.data.category??"",64),level:cleanText(c.data.level??"info",16),message}];});
 return values.length?values:undefined;
}
// How a stack groups (version 2, close to Sentry's): the two app frames
// nearest the error, where it was thrown and its caller (all frames' when
// none is the app's). A named frame counts by file and function, never the
// line, so a release that moves the code keeps the issue; an anonymous one
// by its source line (or the line number when the SDK sends no code). The
// frames further out, often a framework's own bundled code that varies
// between calls, do not split an issue.
type GroupFrame={filename:string;function:string;line:number|null;inApp:boolean;context?:{line:string}};
const namedFrame=(f:GroupFrame)=>!!f.function&&!/^(?:\?|<anonymous>|anonymous|Object\.\?)$/.test(f.function);
function groupFrames(frames:GroupFrame[]){
 const app=frames.filter(f=>f.inApp);
 return (app.length?app:frames).slice(-2).map(f=>namedFrame(f)?[f.filename,f.function]:[f.filename,f.context?.line??f.line]);
}
// Some SDKs (sentry-elixir) send absent fields as JSON null: they count as
// missing. Only object keys, a bounded depth down (deeper is kept as it is).
function withoutNulls(value:unknown,depth=0):unknown{
 if(depth>24||value===null||typeof value!=="object")return value;
 if(Array.isArray(value))return value.map(v=>withoutNulls(v,depth+1));
 return Object.fromEntries(Object.entries(value).filter(([,v])=>v!==null).map(([k,v])=>[k,withoutNulls(v,depth+1)]));
}
export async function safeEvent(input:Record<string,unknown>,id:string,minidump=false):Promise<SafeEvent>{
 const raw=withoutNulls(input) as Record<string,unknown>;
 const parsed=schema.safeParse(raw);if(!parsed.success)throw new IngestError(400,"EVENT");const value=parsed.data;
 const exceptions=value.exception?.values.map(e=>({type:cleanText(e.type??"Error",128),value:cleanText(e.value??"",1024),frames:e.stacktrace?.frames.slice(-50).map(f=>{const code=codeOf(f);return {filename:cleanText((f.filename??f.abs_path??"").split(/[\\/]/).pop()?.split(/[?#]/)[0]??"",160),function:cleanText(f.function??"",128),module:cleanText(f.module??"",128),line:f.lineno??null,column:f.colno??null,inApp:f.in_app??false,...(code?{context:code}:{})};})??[]}))??[];
 // Code only around the ten app frames nearest the error, in each exception.
 for(const e of exceptions){let kept=0;for(let i=e.frames.length-1;i>=0;i--){const f=e.frames[i];if(f.context&&++kept>10)delete f.context;}}
 const context=contextOf(raw),breadcrumbs=breadcrumbsOf(raw);
 const message=typeof value.message==="string"?value.message:value.message?.formatted??value.message?.message??value.logentry?.formatted??value.logentry?.message??"";
 // The title: the error's first line, without a trailing " (Type)" that
 // only repeats it (sentry-ruby sends the message, its class and a
 // highlight of the code); the whole message stays in the details.
 const lastRaw=value.exception?.values.at(-1);
 const headline=(()=>{const first=(lastRaw?.value??"").split(/\r?\n/)[0]??"",suffix=lastRaw?.type?` (${lastRaw.type})`:"";return suffix&&first.endsWith(suffix)?first.slice(0,-suffix.length):first;})();
 // A native crash comes with a minidump and no exception: named after the
 // process that died (Electron's "event.process" tag: browser, renderer, gpu).
 const tag=minidump&&raw.tags&&typeof raw.tags==="object"&&!Array.isArray(raw.tags)?(raw.tags as Record<string,unknown>)["event.process"]:undefined;
 const crashed=typeof tag==="string"&&/^[a-z-]{1,24}$/.test(tag)?` (${tag})`:"";
 const title=cleanText(headline||message||(minidump?`Native crash${crashed}`:"")||exceptions.at(-1)?.type||"Error",256);
 const base=JSON.stringify([2,cleanText(value.platform??"other",80),exceptions.map(e=>[e.type,groupFrames(e.frames)]),exceptions.some(e=>e.frames.length)?"":title.replace(/\b(?:\d+|[a-f0-9]{8,})\b/gi,"#")]);
 // An empty fingerprint (the Java SDKs send one on every event) is the default, as in Sentry.
 const identity=value.fingerprint?.length?value.fingerprint.map(s=>s==="{{ default }}"?base:cleanText(s)):[base];
 return {id,group:await sha256(JSON.stringify([1,identity])),title,level:value.level??"error",platform:cleanText(value.platform??"other",80),environment:cleanText(value.environment??"",80),release:cleanText(value.release??"",128),message:cleanText(message,1024),exceptions,...(context?{context}:{}),...(breadcrumbs?{breadcrumbs}:{})};
}
