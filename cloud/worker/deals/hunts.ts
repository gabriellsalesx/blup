import {Hono} from "hono";
import {z} from "zod";
import {validate,opaqueId} from "../validation";
import {dealsLimits} from "../../shared/pulse-plans";
import {dealsWrite,type DealsApp} from "./access";
import {huntFilters,huntKinds,huntSource,feedHunt,huntRow,type HuntKind,type HuntFilters} from "./catalog";
import {usableChannels,affiliateOf,DAY} from "./store";

const huntParam=z.object({hunt:opaqueId});
const channelIds=z.array(opaqueId).min(1).max(dealsLimits.channels);
// Giveaways take platforms; Nuuvem's sales a country instead, and a price
// limit only in Brazil (the catalog's reais).
const fits=(kind:HuntKind,filters:HuntFilters)=>kind==="games_free"?filters.country===undefined&&filters.gamesOnly===undefined&&filters.maxPrice===undefined:!filters.platforms?.length&&(filters.maxPrice===undefined||(filters.country??"br")==="br");
const names={games_free:"Jogos grátis",nuuvem_deals:"Promoções da Nuuvem"} as const;
const kinds=huntKinds.map(k=>`'${k}'`).join(",");
type Row={id:string;kind:string;active:number;channels:string;filters:string;createdAt:number};
const view=(r:Row)=>({id:r.id,kind:z.enum(huntKinds).parse(r.kind),active:!!r.active,channelIds:z.array(z.string()).catch([]).parse(JSON.parse(r.channels)),filters:huntFilters.catch({}).parse(JSON.parse(r.filters)),createdAt:new Date(r.createdAt).toISOString()});
// A new or changed hunt gets the recent matching offers (last 3 days) at
// once, then whatever new comes in each round.
async function feedNow(env:Cloudflare.Env,id:string){
 const raw=await env.DB.prepare("SELECT id,user_id AS userId,kind,filters,channels FROM deals_hunt WHERE id=? AND active=1").bind(id).first();
 const hunt=huntRow.safeParse(raw);if(hunt.success)await feedHunt(env,hunt.data,Date.now()-3*DAY);
}

export const huntsApi=new Hono<DealsApp>()
 .get("/",async c=>{
  const rows=await c.env.DB.prepare(`SELECT id,kind,active,channels,filters,created_at AS createdAt FROM deals_hunt WHERE user_id=? AND kind IN(${kinds}) AND (retire_at IS NULL OR retire_at>?) ORDER BY created_at,id LIMIT 50`).bind(c.get("session").user.id,Date.now()).all<Row>();
  return c.json({hunts:rows.results.map(view)});
 })
 .post("/",dealsWrite,validate("json",z.object({kind:z.enum(huntKinds),channelIds,filters:huntFilters}).strict()),async c=>{
  const owner=c.get("session").user.id,input=c.req.valid("json"),now=Date.now();
  if(!fits(input.kind,input.filters))return c.json({code:"INVALID"},400);
  // Nuuvem's sales earn only with the person's Rakuten ID.
  if(input.kind==="nuuvem_deals"&&!await affiliateOf(c.env,owner,"nuuvem"))return c.json({code:"NUUVEM_NOT_CONFIGURED"},409);
  if((await usableChannels(c.env.DB,owner,input.channelIds)).length!==new Set(input.channelIds).size)return c.json({code:"NOT_FOUND"},404);
  const id=crypto.randomUUID();
  const row=await c.env.DB.prepare("INSERT INTO deals_hunt(id,user_id,name,kind,source_id,filters,channels,created_at) SELECT ?,?,?,?,?,?,?,? WHERE (SELECT count(*) FROM deals_hunt WHERE user_id=? AND kind<>'mirror' AND (retire_at IS NULL OR retire_at>?))<? RETURNING id").bind(id,owner,names[input.kind],input.kind,huntSource[input.kind],JSON.stringify(input.filters),JSON.stringify([...new Set(input.channelIds)]),now,owner,now,dealsLimits.hunts).first();
  if(!row)return c.json({code:"QUOTA"},409);
  await feedNow(c.env,id);
  return c.json({id},201);
 })
 .patch("/:hunt",dealsWrite,validate("param",huntParam),validate("json",z.object({active:z.boolean().optional(),channelIds:channelIds.optional(),filters:huntFilters.optional()}).strict()),async c=>{
  const owner=c.get("session").user.id,id=c.req.valid("param").hunt,input=c.req.valid("json");
  const current=await c.env.DB.prepare(`SELECT kind FROM deals_hunt WHERE id=? AND user_id=? AND kind IN(${kinds})`).bind(id,owner).first<{kind:HuntKind}>();
  if(!current)return c.json({code:"NOT_FOUND"},404);
  if(input.filters&&!fits(current.kind,input.filters))return c.json({code:"INVALID"},400);
  if(input.channelIds&&(await usableChannels(c.env.DB,owner,input.channelIds)).length!==new Set(input.channelIds).size)return c.json({code:"NOT_FOUND"},404);
  await c.env.DB.prepare("UPDATE deals_hunt SET active=coalesce(?,active),channels=coalesce(?,channels),filters=coalesce(?,filters) WHERE id=? AND user_id=?").bind(input.active===undefined?null:input.active?1:0,input.channelIds?JSON.stringify([...new Set(input.channelIds)]):null,input.filters?JSON.stringify(input.filters):null,id,owner).run();
  if(input.active!==false)await feedNow(c.env,id);
  return c.json({ok:true});
 })
 .delete("/:hunt",validate("param",huntParam),async c=>{
  const owner=c.get("session").user.id,id=c.req.valid("param").hunt;
  const [removed]=await c.env.DB.batch([
   c.env.DB.prepare("DELETE FROM deals_hunt WHERE id=? AND user_id=? RETURNING id").bind(id,owner),
   c.env.DB.prepare("DELETE FROM deals_post WHERE hunt_id=? AND user_id=? AND state='queued'").bind(id,owner),
  ]);
  return removed.results.length?c.json({ok:true}):c.json({code:"NOT_FOUND"},404);
 });

export const statsApi=new Hono<DealsApp>()
 .get("/",validate("query",z.object({days:z.enum(["7","30","90"]).default("7")})),async c=>{
  const owner=c.get("session").user.id,count=Number(c.req.valid("query").days),now=Date.now();
  const first=new Date(Math.floor(now/DAY)*DAY-(count-1)*DAY),from=first.toISOString().slice(0,10);
  const clicks=await c.env.DB.prepare("SELECT day,sum(clicks) AS n FROM deals_click_day WHERE user_id=? AND day>=? GROUP BY day").bind(owner,from).all<{day:string;n:number}>();
  const posts=await c.env.DB.prepare("SELECT strftime('%Y-%m-%d',sent_at/1000,'unixepoch') AS day,count(*) AS n FROM deals_post WHERE user_id=? AND state='sent' AND sent_at>=? GROUP BY day").bind(owner,first.getTime()).all<{day:string;n:number}>();
  const top=await c.env.DB.prepare("SELECT o.title,sum(k.clicks) AS clicks FROM deals_click_day k JOIN deals_post p ON p.id=k.post_id JOIN deals_offer o ON o.id=p.offer_id WHERE k.user_id=? AND k.day>=? GROUP BY o.id ORDER BY clicks DESC,o.title LIMIT 5").bind(owner,from).all<{title:string;clicks:number}>();
  const byDay=(rows:{day:string;n:number}[])=>new Map(rows.map(r=>[r.day,r.n]));
  const c1=byDay(clicks.results),p1=byDay(posts.results);
  const days=Array.from({length:count},(_,i)=>{const day=new Date(first.getTime()+i*DAY).toISOString().slice(0,10);return {day,clicks:c1.get(day)??0,posts:p1.get(day)??0};});
  return c.json({days,top:top.results.map(t=>({title:t.title,clicks:t.clicks}))});
 });
