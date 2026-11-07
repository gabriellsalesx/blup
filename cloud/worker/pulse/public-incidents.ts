import {z} from "zod";
import {pulseAccess} from "./limits";
import {pageSlug,serviceStatus} from "./page-rules";
import type {PublicIncident} from "./public-html";
// The incidents of a published page beyond the few its main page carries:
// those of a window of months (the "previous incidents" page) and one by
// its id (its own page). The same closed projection as the main page: the
// public names of the services, never their monitors.
const affectedStatus=serviceStatus.exclude(["none"]);
export const MONTH=/^(\d{4})-(0[1-9]|1[0-2])$/;
export const monthParam=z.string().regex(MONTH);

// The page behind a public address, with how far back its plan keeps.
export async function publishedPage(env:Cloudflare.Env,slug:string){
 const parsed=pageSlug.safeParse(slug);if(!parsed.success)return null;
 const page=await env.DB.prepare("SELECT id,user_id AS userId FROM pulse_page WHERE slug=? AND published=1 AND plan_disabled=0").bind(parsed.data).first<{id:string;userId:string}>();if(!page)return null;
 const access=await pulseAccess(env,page.userId);if(!access.writable)return null;
 return {...page,cutoff:Date.now()-(access.limits?.retentionDays??30)*86_400_000};
}
type Page=NonNullable<Awaited<ReturnType<typeof publishedPage>>>;
type Row={id:string;title:string;state:string;impact:string;createdAt:number;updatedAt:number};

async function project(env:Cloudflare.Env,page:Page,rows:Row[]):Promise<PublicIncident[]>{
 const out:PublicIncident[]=[];
 // Two reads per incident, sent 25 incidents (50 statements) per batch: a
 // public page never makes one round trip per incident.
 const results:D1Result[]=[];
 for(let start=0;start<rows.length;start+=25)results.push(...await env.DB.batch(rows.slice(start,start+25).flatMap(item=>[
  env.DB.prepare("SELECT body,state,created_at AS createdAt FROM pulse_incident_update WHERE incident_id=? AND created_at>? ORDER BY created_at DESC LIMIT 50").bind(item.id,page.cutoff),
  env.DB.prepare("SELECT c.name,ic.status FROM pulse_incident_component ic JOIN pulse_component c ON c.page_id=? AND c.monitor_id=ic.monitor_id WHERE ic.incident_id=? ORDER BY c.position").bind(page.id,item.id),
 ])));
 for(const [i,item] of rows.entries()){
  const updates=results[i*2] as D1Result<{body:string;state:string;createdAt:number}>,affected=results[i*2+1] as D1Result<{name:string;status:string}>;
  out.push({id:item.id,title:item.title,state:item.state,impact:item.impact,createdAt:new Date(item.createdAt).toISOString(),updatedAt:new Date(item.updatedAt).toISOString(),
   services:affected.results.flatMap(r=>{const status=affectedStatus.safeParse(r.status);return status.success?[{name:r.name,status:status.data}]:[];}),
   updates:updates.results.map(u=>({...u,createdAt:new Date(u.createdAt).toISOString()}))});
 }
 return out;
}
const columns="id,title,state,impact,created_at AS createdAt,updated_at AS updatedAt";

// The incidents that started in [from, to), newest first (at most 200),
// within what the plan keeps.
export async function incidentsBetween(env:Cloudflare.Env,page:Page,from:number,to:number){
 const rows=await env.DB.prepare(`SELECT ${columns} FROM pulse_incident WHERE page_id=? AND updated_at>? AND created_at>=? AND created_at<? ORDER BY created_at DESC LIMIT 200`).bind(page.id,page.cutoff,from,to).all<Row>();
 return project(env,page,rows.results);
}
export async function incidentById(env:Cloudflare.Env,page:Page,id:string){
 const row=await env.DB.prepare(`SELECT ${columns} FROM pulse_incident WHERE id=? AND page_id=? AND updated_at>?`).bind(id,page.id,page.cutoff).first<Row>();
 return row?(await project(env,page,[row]))[0]??null:null;
}

// A window of three months ending at `to` ("2026-11"), as UTC times, with
// the windows before and after (null past what is kept or after now).
export function monthWindow(to:string|null,cutoff:number,now=Date.now()){
 const current=new Date(now),match=to?MONTH.exec(to):null;
 let year=match?Number(match[1]):current.getUTCFullYear(),month=match?Number(match[2])-1:current.getUTCMonth();
 const latest=current.getUTCFullYear()*12+current.getUTCMonth();if(year*12+month>latest){year=current.getUTCFullYear();month=current.getUTCMonth();}
 const key=(y:number,m:number)=>{const d=new Date(Date.UTC(y,m,1));return `${d.getUTCFullYear()}-${String(d.getUTCMonth()+1).padStart(2,"0")}`;};
 const months=[0,1,2].map(i=>{const start=Date.UTC(year,month-i,1),end=Date.UTC(year,month-i+1,1);return {key:key(year,month-i),start,end};});
 const from=months[2].start,until=months[0].end;
 return {months,from,to:until,older:from>cutoff?key(year,month-3):null,newer:year*12+month<latest?key(year,month+3):null};
}
