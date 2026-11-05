import {beforeEach,afterEach,it,expect,vi,describe} from "vitest";
import {TestD1,seedUser} from "../security-test-support";
const state=vi.hoisted(()=>({env:{} as Cloudflare.Env}));
vi.mock("cloudflare:workers",()=>({env:state.env,DurableObject:class{}}));
vi.mock("../auth",()=>({auth:{api:{getSession:async({headers}:{headers:Headers})=>{const id=headers.get("authorization")?.replace("Bearer ","");return id?{user:{id}}:null;}}}}));
import {dealsApi} from "./api";
import {upsertOffer,type OfferInput} from "./catalog";
import {enqueue} from "./store";
import {dispatchDeals} from "./dispatch";

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
  expect(await signed(got[0],rotated.secret)).toBe(true);expect(await signed(got[0],made.secret)).toBe(false);
  expect(await got[0].json()).toMatchObject({event:"test",test:true,deal:null});
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
 it("sends the fixed examples, marked, in the account's layout, once a minute",async()=>{
  const {id}=await (await call("/channels",{method:"POST",body:{type:"discord",webhook:HOOK}})).json() as {id:string};
  const r=await call(`/channels/${id}/test`,{method:"POST",body:{kinds:["test","game_free","nuuvem_deal","store_deal","product"]}});expect(r.status).toBe(202);
  const out=await r.json() as {ok:boolean;sent:number;testAgainAt:number};
  expect(out).toMatchObject({ok:true,sent:5});expect(out.testAgainAt).toBeGreaterThan(Date.now()+50_000);
  const bodies=await Promise.all(got.map(g=>g.json() as Promise<{content?:string;embeds?:{title:string;description:string}[]}>));
  expect(bodies[0].content).toContain("Teste do Blup Deals");
  expect(bodies[1].embeds?.[0].title).toContain("Aventura Estelar");expect(bodies[1].embeds?.[0].description).toContain("Grátis");
  expect(bodies[2].embeds?.[0].title).toContain("Promoção na Nuuvem: Cidade Neon");
  expect(bodies[3].embeds?.[0].description).toContain("EXEMPLO10");
  for(const b of bodies.slice(1))expect(b.embeds?.[0].description).toContain("Exemplo do Blup Deals");
  // Again right away: refused, nothing sent.
  got=[];const again=await call(`/channels/${id}/test`,{method:"POST",body:{kinds:["test"]}});
  expect(again.status).toBe(429);expect(got).toHaveLength(0);
  expect((await (await call("/channels")).json() as {channels:{testAgainAt:number}[]}).channels[0].testAgainAt).toBe(out.testAgainAt);
 },10_000);
 it("refuses kinds it does not know and repeated ones",async()=>{
  const {id}=await (await call("/channels",{method:"POST",body:{type:"discord",webhook:HOOK}})).json() as {id:string};
  expect((await call(`/channels/${id}/test`,{method:"POST",body:{kinds:["video"]}})).status).toBe(400);
  expect((await call(`/channels/${id}/test`,{method:"POST",body:{kinds:["test","test"]}})).status).toBe(400);
  expect((await call(`/channels/${id}/test`,{method:"POST",body:{kinds:[]}})).status).toBe(400);
 });
});
