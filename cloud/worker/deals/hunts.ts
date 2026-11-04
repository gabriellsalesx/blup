import {Hono} from "hono";
import {z} from "zod";
import {validate,opaqueId} from "../validation";
import {dealsLimits} from "../../shared/pulse-plans";
import {dealsWrite,type DealsApp} from "./access";
import {huntFilters,huntKinds,huntSource,feedHunt,huntRow,type HuntFilters} from "./catalog";
import {usableChannels,affiliateOf,DAY} from "./store";
import {channelName,mirrorSourceId,fetchChannel} from "./mirror";
import {shopSourceId} from "./shops";
import {FetchError} from "./fetch";

const huntParam=z.object({hunt:opaqueId});
const channelIds=z.array(opaqueId).min(1).max(dealsLimits.channels);
// The kinds a person can make: the feeds and mirroring (a public Telegram
// channel). Giveaways take platforms; Nuuvem's sales a country instead,
// and a price limit only in Brazil (the catalog's reais); a mirror only
// the stores it keeps; a store hunt (Shopee or AliExpress, with the
// person's API keys) its store, its words, a discount and a price limit.
const allKinds=[...huntKinds,"store_deals","mirror"] as const;
type Kind=(typeof allKinds)[number];
const fits=(kind:Kind,filters:HuntFilters)=>{
 if(kind==="mirror")return Object.keys(filters).every(k=>k==="stores");
 if(kind==="store_deals")return !!filters.store&&!!filters.keywords?.length&&Object.keys(filters).every(k=>["store","keywords","minDiscount","maxPrice"].includes(k));
 if(filters.stores!==undefined||filters.store!==undefined||filters.keywords!==undefined)return false;
 return kind==="games_free"?filters.country===undefined&&filters.gamesOnly===undefined&&filters.maxPrice===undefined:!filters.platforms?.length&&(filters.maxPrice===undefined||(filters.country??"br")==="br");
};
const names={games_free:"Jogos grátis",nuuvem_deals:"Promoções da Nuuvem",store_deals:"Promoções de lojas",mirror:"Espelhamento"} as const;
const kinds=allKinds.map(k=>`'${k}'`).join(",");
type Row={id:string;kind:string;active:number;channels:string;filters:string;createdAt:number;sourceId:string|null;sourceKey:string|null;sourceConfig:string|null;sourceError:string|null};
// A mirror shows its channel (name, title) and, when the channel can't be
// read, why; a store hunt why its store can't be searched.
const view=(r:Row)=>({id:r.id,kind:z.enum(allKinds).parse(r.kind),active:!!r.active,channelIds:z.array(z.string()).catch([]).parse(JSON.parse(r.channels)),filters:huntFilters.catch({}).parse(JSON.parse(r.filters)),createdAt:new Date(r.createdAt).toISOString(),
 mirror:r.kind==="mirror"&&r.sourceKey?{channel:r.sourceKey.replace(/^telegram:/,""),title:z.object({title:z.string().max(120).nullable()}).catch({title:null}).parse(JSON.parse(r.sourceConfig??"{}")).title,error:r.sourceError}:null,
 shop:r.kind==="store_deals"?{error:r.sourceError}:null});
const select=`SELECT h.id,h.kind,h.active,h.channels,h.filters,h.created_at AS createdAt,h.source_id AS sourceId,s.key AS sourceKey,s.config AS sourceConfig,s.error AS sourceError FROM deals_hunt h LEFT JOIN deals_source s ON s.id=h.source_id`;
// Whether a public channel can be mirrored: its preview opens and shows
// posts. The answer says why not.
async function checkChannel(raw:string){
 const name=channelName.safeParse(raw);if(!name.success)return {code:"CHANNEL_INVALID" as const,status:400 as const};
 try{
  const read=await fetchChannel(name.data);
  if(!read.page)return read.status===404?{code:"CHANNEL_GONE" as const,status:404 as const}:{code:"CHANNEL_REFUSED" as const,status:502 as const};
  return {name:name.data,title:read.page.title,posts:read.page.posts.length};
 }catch(e){if(e instanceof FetchError)return {code:"CHANNEL_REFUSED" as const,status:502 as const};throw e;}
}
// A new or changed hunt gets the recent matching offers (last 3 days) at
// once, then whatever new comes in each round (a mirror, from the
// channel's next posts).
async function feedNow(env:Cloudflare.Env,id:string){
 const raw=await env.DB.prepare("SELECT id,user_id AS userId,kind,filters,channels FROM deals_hunt WHERE id=? AND active=1").bind(id).first();
 const hunt=huntRow.safeParse(raw);if(hunt.success)await feedHunt(env,hunt.data,Date.now()-3*DAY);
}

export const huntsApi=new Hono<DealsApp>()
 .get("/",async c=>{
  const rows=await c.env.DB.prepare(`${select} WHERE h.user_id=? AND h.kind IN(${kinds}) AND (h.retire_at IS NULL OR h.retire_at>?) ORDER BY h.created_at,h.id LIMIT 50`).bind(c.get("session").user.id,Date.now()).all<Row>();
  return c.json({hunts:rows.results.map(view)});
 })
 .post("/mirror/check",dealsWrite,validate("json",z.object({channel:z.string().max(200)}).strict()),async c=>{
  const checked=await checkChannel(c.req.valid("json").channel);
  return "code" in checked?c.json({code:checked.code},checked.status):c.json(checked);
 })
 .post("/",dealsWrite,validate("json",z.object({kind:z.enum(allKinds),channelIds,filters:huntFilters,channel:z.string().max(200).optional()}).strict()),async c=>{
  const owner=c.get("session").user.id,input=c.req.valid("json"),now=Date.now();
  if(!fits(input.kind,input.filters)||(input.kind==="mirror")!==(input.channel!==undefined))return c.json({code:"INVALID"},400);
  // Nuuvem's sales earn only with the person's Rakuten ID; a store hunt
  // searches with the person's own keys of that store.
  if(input.kind==="nuuvem_deals"&&!await affiliateOf(c.env,owner,"nuuvem"))return c.json({code:"NUUVEM_NOT_CONFIGURED"},409);
  if(input.kind==="store_deals"&&input.filters.store&&!await affiliateOf(c.env,owner,input.filters.store))return c.json({code:"STORE_NOT_CONFIGURED"},409);
  if((await usableChannels(c.env.DB,owner,input.channelIds)).length!==new Set(input.channelIds).size)return c.json({code:"NOT_FOUND"},404);
  const id=crypto.randomUUID(),mirror=input.kind==="mirror";
  let source:string=input.kind==="mirror"?"":input.kind==="store_deals"?shopSourceId(id):huntSource[input.kind];
  if(input.kind==="mirror"){
   // One read for everyone who mirrors the same channel.
   const checked=await checkChannel(input.channel??"");if("code" in checked)return c.json({code:checked.code},checked.status);
   source=mirrorSourceId(checked.name);
   await c.env.DB.prepare("INSERT INTO deals_source(id,kind,key,config,next_at,created_at) VALUES(?,'telegram',?,?,0,?) ON CONFLICT(id) DO UPDATE SET config=excluded.config").bind(source,source,JSON.stringify({title:checked.title}),now).run();
  }
  const row=await c.env.DB.prepare(`INSERT INTO deals_hunt(id,user_id,name,kind,source_id,filters,channels,created_at) SELECT ?,?,?,?,?,?,?,? WHERE (SELECT count(*) FROM deals_hunt WHERE user_id=? AND kind${mirror?"=":"<>"}'mirror' AND (retire_at IS NULL OR retire_at>?))<? AND NOT EXISTS(SELECT 1 FROM deals_hunt WHERE user_id=? AND kind='mirror' AND source_id=? AND (retire_at IS NULL OR retire_at>?)) RETURNING id`).bind(id,owner,names[input.kind],input.kind,source,JSON.stringify(input.filters),JSON.stringify([...new Set(input.channelIds)]),now,owner,now,mirror?dealsLimits.mirrors:dealsLimits.hunts,owner,mirror?source:"",now).first();
  if(!row)return mirror&&await c.env.DB.prepare("SELECT 1 FROM deals_hunt WHERE user_id=? AND kind='mirror' AND source_id=? AND (retire_at IS NULL OR retire_at>?)").bind(owner,source,now).first()?c.json({code:"MIRROR_EXISTS"},409):c.json({code:"QUOTA"},409);
  // A store hunt reads on the next minute's run, then every 30 minutes.
  if(input.kind==="store_deals")await c.env.DB.prepare("INSERT INTO deals_source(id,kind,key,next_at,created_at) VALUES(?,?,?,0,?)").bind(source,input.filters.store??"",source,now).run();
  else if(!mirror)await feedNow(c.env,id);
  return c.json({id},201);
 })
 .patch("/:hunt",dealsWrite,validate("param",huntParam),validate("json",z.object({active:z.boolean().optional(),channelIds:channelIds.optional(),filters:huntFilters.optional()}).strict()),async c=>{
  const owner=c.get("session").user.id,id=c.req.valid("param").hunt,input=c.req.valid("json");
  const current=await c.env.DB.prepare(`SELECT kind FROM deals_hunt WHERE id=? AND user_id=? AND kind IN(${kinds})`).bind(id,owner).first<{kind:Kind}>();
  if(!current)return c.json({code:"NOT_FOUND"},404);
  if(input.filters&&!fits(current.kind,input.filters))return c.json({code:"INVALID"},400);
  if(input.channelIds&&(await usableChannels(c.env.DB,owner,input.channelIds)).length!==new Set(input.channelIds).size)return c.json({code:"NOT_FOUND"},404);
  if(current.kind==="store_deals"&&input.filters?.store&&!await affiliateOf(c.env,owner,input.filters.store))return c.json({code:"STORE_NOT_CONFIGURED"},409);
  await c.env.DB.prepare("UPDATE deals_hunt SET active=coalesce(?,active),channels=coalesce(?,channels),filters=coalesce(?,filters) WHERE id=? AND user_id=?").bind(input.active===undefined?null:input.active?1:0,input.channelIds?JSON.stringify([...new Set(input.channelIds)]):null,input.filters?JSON.stringify(input.filters):null,id,owner).run();
  // A changed store hunt searches again on the next run, its store as set.
  if(current.kind==="store_deals")await c.env.DB.prepare("UPDATE deals_source SET kind=coalesce(?,kind),next_at=0,error=NULL,failures=0 WHERE id=?").bind(input.filters?.store??null,shopSourceId(id)).run();
  else if(input.active!==false&&current.kind!=="mirror")await feedNow(c.env,id);
  return c.json({ok:true});
 })
 .delete("/:hunt",validate("param",huntParam),async c=>{
  const owner=c.get("session").user.id,id=c.req.valid("param").hunt;
  const [removed]=await c.env.DB.batch([
   c.env.DB.prepare("DELETE FROM deals_hunt WHERE id=? AND user_id=? RETURNING id").bind(id,owner),
   c.env.DB.prepare("DELETE FROM deals_post WHERE hunt_id=? AND user_id=? AND state='queued'").bind(id,owner),
   c.env.DB.prepare("DELETE FROM deals_source WHERE id=? AND kind IN('shopee','aliexpress') AND NOT EXISTS(SELECT 1 FROM deals_hunt WHERE id=?)").bind(shopSourceId(id),id),
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
