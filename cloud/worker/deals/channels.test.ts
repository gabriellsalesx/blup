import {beforeEach,afterEach,it,expect,vi,describe} from "vitest";
import {TestD1,seedUser} from "../security-test-support";
const state=vi.hoisted(()=>({env:{} as Cloudflare.Env}));
vi.mock("cloudflare:workers",()=>({env:state.env,DurableObject:class{}}));
vi.mock("../auth",()=>({auth:{api:{getSession:async({headers}:{headers:Headers})=>{const id=headers.get("authorization")?.replace("Bearer ","");return id?{user:{id}}:null;}}}}));
import {dealsApi} from "./api";
import {upsertOffer,type OfferInput} from "./catalog";
import {enqueue,addChannel} from "./store";
import {dispatchDeals} from "./dispatch";
// The item at i, or the test fails here (the list came out shorter).
const nth=<T>(list:readonly T[]|undefined,i=0):T=>{const v=list?.[i];if(v===undefined)throw new Error(`no item ${i}`);return v;};

// A channel's details: webhooks of the person's own system (signed like
// Pulse's) and the examples sent from the panel.
const KEY=btoa(String.fromCharCode(...new Uint8Array(32).fill(7)));
const HOOK="https://discord.com/api/webhooks/123456789012345678/"+"a".repeat(68);
const OWN="https://hooks.acme.dev/deals";
let db:TestD1,got:Request[],status:number;
const fake=vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{
 const req=new Request(input instanceof URL?input.href:input,init),url=new URL(req.url);
 if(url.hostname==="cloudflare-dns.com")return Response.json({Status:0,Answer:[{type:url.searchParams.get("type")==="A"?1:28,data:url.searchParams.get("type")==="A"?"93.184.216.34":"2606:4700::1"}]});
 got.push(req);
 if(url.hostname==="discord.com")return Response.json({id:"1234567890"});
 if(url.hostname==="api.telegram.org")return Response.json({ok:true,result:{message_id:1}});
 if(url.hostname==="hooks.acme.dev")return new Response(null,{status});
 throw new Error(`unexpected ${req.url}`);
}) as unknown as typeof fetch;
beforeEach(()=>{
 db=new TestD1();seedUser(db,"alice","ultra_deals");got=[];status=200;
 Object.assign(state.env,{DB:db.asD1(),STAFF_OWNER_EMAIL:"",PULSE_CHANNEL_KEY:KEY,DEALS_PUBLIC_HOST:"deals.audit.invalid",DEALS_TELEGRAM_BOT:"BlupDealsBot",DEALS_TELEGRAM_TOKEN:"123456:"+"b".repeat(30),DEALS_TELEGRAM_WEBHOOK_SECRET:"hook-secret"});
 vi.stubGlobal("fetch",fake);
});
afterEach(()=>{db.close();vi.unstubAllGlobals();vi.restoreAllMocks();});
const call=(path:string,init:{method?:string;body?:unknown}={})=>dealsApi.fetch(new Request(`https://cloud.audit.invalid${path}`,{method:init.method??"GET",headers:{authorization:"Bearer alice",...(init.body!==undefined?{"content-type":"application/json"}:{})},body:init.body===undefined?undefined:JSON.stringify(init.body)}),state.env);
const hex=(b:ArrayBuffer)=>[...new Uint8Array(b)].map(x=>x.toString(16).padStart(2,"0")).join("");
async function signed(req:Request,secret:string){
 const body=await req.clone().text(),[t,v1]=(req.headers.get("x-blup-signature")??"").split(",").map(p=>p.split("=")[1]);
 const key=await crypto.subtle.importKey("raw",new TextEncoder().encode(secret),{name:"HMAC",hash:"SHA-256"},false,["sign"]);
 return v1===hex(await crypto.subtle.sign("HMAC",key,new TextEncoder().encode(`${t}.${body}`)));
}
async function addOwn(){const r=await call("/channels",{method:"POST",body:{type:"webhook",url:OWN}});expect(r.status).toBe(201);return await r.json() as {id:string;label:string;secret:string};}

describe("webhooks",()=>{
 it("adds one with a secret shown once, sealed, and refuses private addresses",async()=>{
  const made=await addOwn();
  expect(made.label).toBe("Webhook · hooks.acme.dev");expect(made.secret).toMatch(/^[a-f0-9]{64}$/);
  const row=db.sqlite.prepare("SELECT platform,target FROM deals_channel WHERE id=?").get(made.id) as {platform:string;target:string};
  expect(row.platform).toBe("webhook");expect(row.target).not.toContain(made.secret);expect(row.target).not.toContain("acme");
  expect(await (await call("/channels")).json()).toMatchObject({channels:[{type:"webhook",label:"Webhook · hooks.acme.dev",testAgainAt:0}]});
  for(const url of ["http://hooks.acme.dev/x","https://10.0.0.5/x","https://localhost/x","https://cloud.getblup.io/api/me"])expect((await call("/channels",{method:"POST",body:{type:"webhook",url}})).status).toBe(400);
  // The same address again: the same channel, with a new secret.
  const again=await call("/channels",{method:"POST",body:{type:"webhook",url:OWN}});
  expect(again.status).toBe(200);const second=await again.json() as {id:string;secret:string};
  expect(second.id).toBe(made.id);expect(second.secret).not.toBe(made.secret);
 });
 it("gets the queue's posts as signed JSON, and a new secret stops the old one",async()=>{
  const made=await addOwn();
  const offer:OfferInput={key:"k1",store:"amazon",externalId:null,sourceId:null,title:"Fone JBL",url:"https://www.amazon.com.br/dp/B000000001",imageUrl:"https://m.media-amazon.com/x.jpg",price:19900,originalPrice:24900,currency:"BRL",kind:"product",platforms:[],discount:20,credit:null,expiresAt:null};
  const stored=await upsertOffer(state.env.DB,offer);
  expect(await enqueue(state.env.DB,{owner:"alice",offerId:stored.id,channels:[made.id],link:offer.url,affiliate:true,huntId:null,sendAt:Date.now()})).toBe(1);
  await dispatchDeals(state.env,fetch);
  const req=got.find(r=>r.url===OWN)!;
  expect(req.headers.get("x-blup-event")).toBe("deal");expect(req.headers.get("user-agent")).toContain("BlupDeals");
  expect(await signed(req,made.secret)).toBe(true);
  const body=await req.json() as Record<string,unknown>;
  expect(body).toMatchObject({version:1,source:"blup-deals",event:"deal",test:false,deal:{kind:"product",store:"amazon",price:19900,currency:"BRL",affiliate:true,image:"https://m.media-amazon.com/x.jpg"}});
  expect(String((body.deal as {link:string}).link)).toMatch(/^https:\/\/deals\.audit\.invalid\/r\//);
  expect(db.sqlite.prepare("SELECT state FROM deals_post").get()).toEqual({state:"sent"});
  const rotated=await (await call(`/channels/${made.id}/secret`,{method:"POST",body:{}})).json() as {secret:string};
  expect(rotated.secret).not.toBe(made.secret);
  got=[];expect(await (await call(`/channels/${made.id}/test`,{method:"POST",body:{kinds:["test"]}})).json()).toMatchObject({ok:true,sent:1});
  expect(await signed(nth(got),rotated.secret)).toBe(true);expect(await signed(nth(got),made.secret)).toBe(false);
  expect(await nth(got).json()).toMatchObject({event:"test",test:true,deal:null});
 });
 it("a 410 switches it off",async()=>{
  const made=await addOwn();status=410;
  expect(await (await call(`/channels/${made.id}/test`,{method:"POST",body:{kinds:["test"]}})).json()).toMatchObject({ok:false,code:"DESTINATION_DISABLED"});
  expect(db.sqlite.prepare("SELECT active FROM deals_channel").get()).toEqual({active:0});
 });
 it("a new secret is only for webhooks",async()=>{
  const r=await call("/channels",{method:"POST",body:{type:"discord",webhook:HOOK}});const {id}=await r.json() as {id:string};
  expect((await call(`/channels/${id}/secret`,{method:"POST",body:{}})).status).toBe(404);
 });
});

describe("examples",()=>{
 it("serves the pictures in public, and nothing else",async()=>{
  const r=await dealsApi.fetch(new Request("https://cloud.audit.invalid/samples/store.pt.jpg"),state.env);
  expect(r.status).toBe(200);expect(r.headers.get("content-type")).toBe("image/jpeg");
  expect(new Uint8Array(await r.arrayBuffer()).slice(0,2)).toEqual(new Uint8Array([0xff,0xd8]));
  for(const path of ["store.jpg","outra.pt.jpg","store.fr.jpg"])expect((await dealsApi.fetch(new Request(`https://cloud.audit.invalid/samples/${path}`),state.env)).status).not.toBe(200);
 });
 it("sends the fixed examples, marked, in the account's layout, once a minute",async()=>{
  const {id}=await (await call("/channels",{method:"POST",body:{type:"discord",webhook:HOOK}})).json() as {id:string};
  const r=await call(`/channels/${id}/test`,{method:"POST",body:{kinds:["test","game_free","nuuvem_deal","store_deal","product"]}});expect(r.status).toBe(202);
  const out=await r.json() as {ok:boolean;sent:number;testAgainAt:number};
  expect(out).toMatchObject({ok:true,sent:5});expect(out.testAgainAt).toBeGreaterThan(Date.now()+50_000);
  // The plain test is JSON; each example goes with its picture as a file.
  type Body={content?:string;embeds?:{title:string;description:string;image?:{url:string}}[]};
  const bodies=await Promise.all(got.map(async(g,i)=>{if(!i)return await g.json() as Body;const form=await g.formData(),file=form.get("files[0]") as File;expect(file.type).toBe("image/jpeg");expect(new Uint8Array(await file.arrayBuffer()).slice(0,2)).toEqual(new Uint8Array([0xff,0xd8]));return JSON.parse(String(form.get("payload_json"))) as Body;}));
  for(const b of bodies.slice(1))expect(b.embeds?.[0]?.image?.url).toBe("attachment://exemplo.jpg");
  expect(nth(bodies).content).toContain("Teste do Blup Deals");
  expect(nth(bodies,1).embeds?.[0]?.title).toContain("Aventura Estelar");expect(nth(bodies,1).embeds?.[0]?.description).toContain("Grátis");
  expect(nth(bodies,2).embeds?.[0]?.title).toContain("Promoção na Nuuvem: Cidade Neon");
  expect(nth(bodies,3).embeds?.[0]?.description).toContain("EXEMPLO10");
  for(const b of bodies.slice(1))expect(b.embeds?.[0]?.description).toContain("Exemplo do Blup Deals");
  // Again right away: refused, nothing sent.
  got=[];const again=await call(`/channels/${id}/test`,{method:"POST",body:{kinds:["test"]}});
  expect(again.status).toBe(429);expect(got).toHaveLength(0);
  expect((await (await call("/channels")).json() as {channels:{testAgainAt:number}[]}).channels[0]?.testAgainAt).toBe(out.testAgainAt);
 },10_000);
 it("refuses kinds it does not know and repeated ones",async()=>{
  const {id}=await (await call("/channels",{method:"POST",body:{type:"discord",webhook:HOOK}})).json() as {id:string};
  expect((await call(`/channels/${id}/test`,{method:"POST",body:{kinds:["video"]}})).status).toBe(400);
  expect((await call(`/channels/${id}/test`,{method:"POST",body:{kinds:["test","test"]}})).status).toBe(400);
  expect((await call(`/channels/${id}/test`,{method:"POST",body:{kinds:[]}})).status).toBe(400);
 });
});
it("makes one channel of two requests at once for the same destination",async()=>{
 const [a,b]=await Promise.all([addChannel(state.env,"alice","discord",HOOK,"Discord"),addChannel(state.env,"alice","discord",HOOK,"Discord")]);
 expect(a?.id).toBeDefined();expect(b?.id).toBe(a?.id);
 expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_channel WHERE user_id='alice'").get()).toEqual({n:1});
 // The index itself refuses a second row with the same destination.
 expect(()=>db.sqlite.prepare("INSERT INTO deals_channel(id,user_id,platform,name,target,target_hash,created_at) SELECT 'x',user_id,platform,name,target||'2',target_hash,1 FROM deals_channel WHERE user_id='alice'").run()).toThrow(/UNIQUE/);
});
it("keeps the oldest of channels made twice before the index",async()=>{
 const fresh=new TestD1("0041_indices_oferta_e_componente.sql");seedUser(fresh,"carol","ultra_deals");
 fresh.sqlite.exec("INSERT INTO deals_channel(id,user_id,platform,name,target,target_hash,created_at,active) VALUES('old','carol','discord','A','t1','h',1,1),('new','carol','discord','A','t2','h',2,1),('other','carol','discord','B','t3','h2',3,1)");
 fresh.applyMigration("0042_canal_deals_unico.sql");
 expect(fresh.sqlite.prepare("SELECT id,active,target_hash AS hash FROM deals_channel ORDER BY id").all()).toEqual([{id:"new",active:0,hash:"duplicate:new"},{id:"old",active:1,hash:"h"},{id:"other",active:1,hash:"h2"}]);
 fresh.close();
});
