import {z} from "zod";
import {sha256} from "./limits";
import {IngestError} from "./protocol";

// Intentionally closed projection: requests, users, contexts, extra,
// breadcrumbs, local variables and attachments never reach storage.
export function cleanText(value:string,max=256){return value
 .replace(/(?:https?:\/\/)[^\s"<>]+/gi,v=>{try{const u=new URL(v);return `${u.protocol}//${u.hostname}${u.pathname.replace(/\b[a-f0-9]{16,}\b/gi,"[id]")}`;}catch{return "[url]";}})
 .replace(/\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b/gi,"[email]")
 .replace(/(?:[A-Z]:\\|\/(?:home|Users|root)\/)[^\s"<>]+/g,"[path]")
 .replace(/\b(?:bearer\s+\S+|(?:sk|pk)_(?:live|test)_\S+|eyJ[\w-]+\.[\w-]+\.[\w-]+)\b/gi,"[secret]")
 .replace(/\b(password|passwd|secret|token|authorization|api[_-]?key)\s*[:=]\s*[^\s,;]+/gi,"$1=[secret]")
 .replace(/[\u0000-\u001f\u007f]/g," ").slice(0,max);}
const frame=z.object({filename:z.string().max(8192).optional(),abs_path:z.string().max(8192).optional(),function:z.string().max(8192).optional(),module:z.string().max(8192).optional(),lineno:z.number().int().min(0).max(10_000_000).optional(),colno:z.number().int().min(0).max(10_000_000).optional(),in_app:z.boolean().optional()});
const error=z.object({type:z.string().max(8192).optional(),value:z.string().max(204800).optional(),stacktrace:z.object({frames:z.array(frame).max(512)}).optional()});
const schema=z.object({type:z.enum(["default","error"]).optional(),platform:z.string().max(80).optional(),level:z.enum(["fatal","error","warning","info","debug"]).optional(),message:z.union([z.string().max(204800),z.object({formatted:z.string().max(204800).optional(),message:z.string().max(204800).optional()})]).optional(),logentry:z.object({formatted:z.string().max(204800).optional(),message:z.string().max(204800).optional()}).optional(),exception:z.union([z.object({values:z.array(error).max(32)}),z.array(error).max(32)]).transform(v=>Array.isArray(v)?{values:v}:v).optional(),fingerprint:z.array(z.string().max(256)).min(1).max(32).optional(),environment:z.string().max(256).optional(),release:z.string().max(256).optional()});
// The stored projection, checked again when read back from the project's SQLite.
const storedFrame=z.object({filename:z.string(),function:z.string(),module:z.string(),line:z.number().int().nullable(),column:z.number().int().nullable(),inApp:z.boolean()}).strict();
export const storedEvent=z.object({id:z.string(),group:z.string(),title:z.string(),level:schema.shape.level.unwrap(),platform:z.string(),environment:z.string(),release:z.string(),message:z.string(),exceptions:z.array(z.object({type:z.string(),value:z.string(),frames:z.array(storedFrame)}).strict())}).strict();
export type SafeEvent=z.infer<typeof storedEvent>;
export async function safeEvent(raw:Record<string,unknown>,id:string):Promise<SafeEvent>{
 const parsed=schema.safeParse(raw);if(!parsed.success)throw new IngestError(400,"EVENT");const value=parsed.data;
 const exceptions=value.exception?.values.map(e=>({type:cleanText(e.type??"Error",128),value:cleanText(e.value??"",1024),frames:e.stacktrace?.frames.slice(-50).map(f=>({filename:cleanText((f.filename??f.abs_path??"").split(/[\\/]/).pop()?.split(/[?#]/)[0]??"",160),function:cleanText(f.function??"",128),module:cleanText(f.module??"",128),line:f.lineno??null,column:f.colno??null,inApp:f.in_app??false}))??[]}))??[];
 const message=typeof value.message==="string"?value.message:value.message?.formatted??value.message?.message??value.logentry?.formatted??value.logentry?.message??"";
 const title=cleanText(exceptions.at(-1)?.value||message||exceptions.at(-1)?.type||"Error",256);
 const base=JSON.stringify([1,cleanText(value.platform??"other",80),exceptions.map(e=>[e.type,e.frames.filter(f=>f.inApp).length?e.frames.filter(f=>f.inApp).map(f=>[f.filename,f.function,f.line]):e.frames.slice(-5).map(f=>[f.filename,f.function,f.line])]),exceptions.some(e=>e.frames.length)?"":title.replace(/\b(?:\d+|[a-f0-9]{8,})\b/gi,"#")]);
 const identity=value.fingerprint?.map(s=>s==="{{ default }}"?base:cleanText(s))??[base];
 return {id,group:await sha256(JSON.stringify([1,identity])),title,level:value.level??"error",platform:cleanText(value.platform??"other",80),environment:cleanText(value.environment??"",80),release:cleanText(value.release??"",128),message:cleanText(message,1024),exceptions};
}
