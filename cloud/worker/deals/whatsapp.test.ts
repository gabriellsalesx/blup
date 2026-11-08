import {beforeEach,afterEach,it,expect,vi,describe} from "vitest";
import {TestD1,seedUser} from "../security-test-support";
const state=vi.hoisted(()=>({env:{} as Cloudflare.Env}));
vi.mock("cloudflare:workers",()=>({env:state.env,DurableObject:class{}}));
vi.mock("../auth",()=>({auth:{api:{getSession:async({headers}:{headers:Headers})=>{const id=headers.get("authorization")?.replace("Bearer ","");return id?{user:{id}}:null;}}}}));
import {dealsApi} from "./api";
import {upsertOffer,type OfferInput} from "./catalog";
import {enqueue} from "./store";
import {dispatchDeals} from "./dispatch";
import {purgeDealsAccount} from "./purge";
import {signature} from "./bridge";
// The item at i, or the test fails here (the list came out shorter).
const nth=<T>(list:readonly T[]|undefined,i=0):T=>{const v=list?.[i];if(v===undefined)throw new Error(`no item ${i}`);return v;};

// The WhatsApp Canais through the Bridge, with a fake Bridge that checks
// every signature as the real one (cloud/bridge/auth.go) does.
const SECRET="b".repeat(40),CALLBACK="c".repeat(40),KEY=btoa(String.fromCharCode(...new Uint8Array(32).fill(7)));
const CANAL="120363422259812011";
// The fake checks with the same formula, written apart (the signed text is
// fixed by the Bridge; the shared example below ties both).
const hex=(b:ArrayBuffer)=>[...new Uint8Array(b)].map(x=>x.toString(16).padStart(2,"0")).join("");
async function sig(secret:string,at:string,nonce:string,method:string,path:string,body:string){
 const key=await crypto.subtle.importKey("raw",new TextEncoder().encode(secret),{name:"HMAC",hash:"SHA-256"},false,["sign"]);
 const sum=hex(await crypto.subtle.digest("SHA-256",new TextEncoder().encode(body)));
 return hex(await crypto.subtle.sign("HMAC",key,new TextEncoder().encode(`${at}\n${nonce}\n${method}\n${path}\n${sum}`)));
}
const PNG=Uint8Array.from(atob("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="),c=>c.charCodeAt(0));

let db:TestD1;
type Call={method:string;path:string;body:Record<string,unknown>|null};
let calls:Call[],reply:(c:Call)=>Response;
const fake=vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{
 const req=new Request(input instanceof URL?input.href:input,init),url=new URL(req.url);
 if(url.hostname==="cloudflare-dns.com")return Response.json({Status:0,Answer:[{type:url.searchParams.get("type")==="A"?1:28,data:url.searchParams.get("type")==="A"?"93.184.216.34":"2606:4700::1"}]});
 if(url.hostname==="images.lojadeteste.com.br")return new Response(PNG,{headers:{"content-type":"image/png"}});
 if(url.hostname!=="bridge.audit.invalid")throw new Error(`unexpected ${req.url}`);
 const raw=await req.text(),h=req.headers;
 if(h.get("x-bridge-signature")!==await sig(SECRET,h.get("x-bridge-timestamp")??"",h.get("x-bridge-nonce")??"",req.method,url.pathname+url.search,raw))return Response.json({code:"UNAUTHORIZED"},{status:401});
 const c={method:req.method,path:url.pathname,body:raw?JSON.parse(raw) as Record<string,unknown>:null};calls.push(c);return reply(c);
}) as unknown as typeof fetch;
const bridgeOK=(c:Call)=>{
 if(c.method==="POST"&&c.path.endsWith("/posts"))return Response.json({id:"3EB0",at:new Date().toISOString()});
 if(c.path.endsWith("/channels"))return Response.json({channels:[{id:CANAL,name:"BuildFeed",role:"owner"}]});
 if(c.method==="DELETE")return new Response(null,{status:204});
 return Response.json({state:c.method==="POST"?"pairing":"connected",...(c.method==="POST"?{qr:"data:image/png;base64,AAAA"}:{phoneEnd:"5443"})});
};

beforeEach(()=>{
 db=new TestD1();seedUser(db,"alice","ultra_deals");seedUser(db,"plain","ultra");
 calls=[];reply=bridgeOK;
 Object.assign(state.env,{DB:db.asD1(),STAFF_OWNER_EMAIL:"",PULSE_CHANNEL_KEY:KEY,DEALS_PUBLIC_HOST:"deals.audit.invalid",DEALS_BRIDGE_URL:"https://bridge.audit.invalid",DEALS_BRIDGE_SECRET:SECRET,DEALS_BRIDGE_CALLBACK_SECRET:CALLBACK});
 vi.stubGlobal("fetch",fake);
});
afterEach(()=>{db.close();vi.unstubAllGlobals();vi.restoreAllMocks();});
const call=(path:string,init:{method?:string;body?:unknown;user?:string;headers?:Record<string,string>;raw?:string}={})=>dealsApi.fetch(new Request(`https://cloud.audit.invalid${path}`,{method:init.method??"GET",headers:{authorization:`Bearer ${init.user??"alice"}`,...(init.body!==undefined||init.raw!==undefined?{"content-type":"application/json"}:{}),...init.headers},body:init.raw??(init.body===undefined?undefined:JSON.stringify(init.body))}),state.env);
const notice=async(body:object,secret=CALLBACK,at=Math.floor(Date.now()/1000))=>{const raw=JSON.stringify(body),nonce=hex(crypto.getRandomValues(new Uint8Array(16)).buffer);return call("/whatsapp/callback",{method:"POST",raw,headers:{"x-bridge-timestamp":String(at),"x-bridge-nonce":nonce,"x-bridge-signature":await sig(secret,String(at),nonce,"POST","/whatsapp/callback",raw)}});};
const offer=(over:Partial<OfferInput>={}):OfferInput=>({key:"k1",store:"amazon",externalId:null,sourceId:null,title:"Fone JBL Tune 520BT",url:"https://www.amazon.com.br/dp/B000000001",imageUrl:"https://images.lojadeteste.com.br/fone.png",price:19900,originalPrice:24900,currency:"BRL",kind:"product",platforms:[],discount:20,credit:null,expiresAt:null,...over});
const session=()=>db.sqlite.prepare("SELECT id,state,phone_end AS phoneEnd FROM deals_whatsapp WHERE user_id='alice'").get() as {id:string;state:string;phoneEnd:string|null}|undefined;
async function connected(){
 expect((await call("/whatsapp/session",{method:"POST",body:{acceptRisk:true}})).status).toBe(200);
 expect((await call("/whatsapp/session")).status).toBe(200);
 const r=await call("/whatsapp/channels",{method:"POST",body:{channel:CANAL}});expect(r.status).toBe(201);
 return (await r.json() as {id:string}).id;
}

// The same fixed example as the Bridge's own test (cloud/bridge): both
// sides sign alike.
it("signs as the Bridge does",async()=>{
 expect(await signature("s".repeat(40),"1800000000","0".repeat(32),"POST","/v1/sessions/abcdefgh/posts",'{"a":1}')).toBe("d026d2376be2159559d677c68ec1776e3e5cd20a8c343d25082c9b0d3e9701b9");
});

describe("connecting WhatsApp",()=>{
 it("asks for the risk first, then shows the Bridge's QR",async()=>{
  expect((await call("/whatsapp/session")).status).toBe(200);
  expect(await (await call("/whatsapp/session")).json()).toMatchObject({state:"none",ready:true});
  expect((await call("/whatsapp/session",{method:"POST",body:{}})).status).toBe(400);
  expect(calls).toHaveLength(0);
  const r=await call("/whatsapp/session",{method:"POST",body:{acceptRisk:true}});
  expect(await r.json()).toMatchObject({state:"pairing",qr:"data:image/png;base64,AAAA",ready:true});
  const s=session();expect(s?.state).toBe("pairing");
  expect(calls[0]).toMatchObject({method:"POST",path:`/v1/sessions/${s?.id}`});
  // A new start while it waits: a new name at the Bridge (its first QR
  // lasts a minute).
  const first=s?.id;calls=[];
  await call("/whatsapp/session",{method:"POST",body:{acceptRisk:true,fresh:true}});
  expect(session()?.id).not.toBe(first);
  expect(calls.map(c=>`${c.method} ${c.path}`)).toEqual([`DELETE /v1/sessions/${first}`,`POST /v1/sessions/${session()?.id}`]);
  // Read again: the Bridge says it connected; only the number's end is kept.
  expect(await (await call("/whatsapp/session")).json()).toMatchObject({state:"connected",phoneEnd:"5443",qr:null});
  expect(session()).toMatchObject({state:"connected",phoneEnd:"5443"});
 });
 it("says when the Bridge is down, and keeps the last state",async()=>{
  await connected();
  reply=()=>{throw new TypeError("fetch failed");};
  expect(await (await call("/whatsapp/session")).json()).toMatchObject({state:"connected",phoneEnd:"5443",reachable:false});
  const r=await call("/whatsapp/channels");expect(r.status).toBe(503);expect(await r.json()).toEqual({code:"WHATSAPP_UNAVAILABLE"});
  expect((await (await call("/whatsapp/session",{method:"POST",body:{acceptRisk:true}})).json() as {code:string}).code).toBe("WHATSAPP_UNAVAILABLE");
  // An example then: the Bridge is down, the Canal is not to blame.
  const channel=nth((await (await call("/channels")).json() as {channels:{id:string}[]}).channels).id;
  expect(await (await call(`/channels/${channel}/test`,{method:"POST",body:{kinds:["test"]}})).json()).toMatchObject({ok:false,code:"BRIDGE_UNAVAILABLE"});
  expect(await (await call("/channels")).json()).toMatchObject({channels:[{lastError:null,active:true}]});
  reply=bridgeOK;
  expect(await (await call("/whatsapp/session")).json()).toMatchObject({state:"connected",reachable:true});
 });
 it("is not there without the Bridge, nor without Deals",async()=>{
  state.env.DEALS_BRIDGE_SECRET="curta";
  expect(await (await call("/whatsapp/session")).json()).toMatchObject({state:"none",ready:false});
  expect((await call("/whatsapp/session",{method:"POST",body:{acceptRisk:true}})).status).toBe(503);
  state.env.DEALS_BRIDGE_SECRET=SECRET;
  expect((await call("/whatsapp/session",{method:"POST",body:{acceptRisk:true},user:"plain"})).status).toBe(403);
 });
 it("adds only a Canal the number owns or runs, named by WhatsApp",async()=>{
  await call("/whatsapp/session",{method:"POST",body:{acceptRisk:true}});
  expect((await call("/whatsapp/channels")).status).toBe(409);
  await call("/whatsapp/session");
  expect(await (await call("/whatsapp/channels")).json()).toEqual({channels:[{id:CANAL,name:"BuildFeed",role:"owner",added:false}]});
  expect((await call("/whatsapp/channels",{method:"POST",body:{channel:"120363000000000000"}})).status).toBe(404);
  expect((await call("/whatsapp/channels",{method:"POST",body:{channel:CANAL,name:"Outro"}})).status).toBe(400);
  expect((await call("/whatsapp/channels",{method:"POST",body:{channel:CANAL}})).status).toBe(201);
  const row=db.sqlite.prepare("SELECT platform,name,session_id AS sessionId,target FROM deals_channel WHERE user_id='alice'").get() as {platform:string;name:string;sessionId:string;target:string};
  expect(row).toMatchObject({platform:"whatsapp",name:"BuildFeed",sessionId:session()?.id});
  expect(row.target).not.toContain(CANAL);
  expect(await (await call("/whatsapp/channels")).json()).toMatchObject({channels:[{added:true}]});
  expect(await (await call("/channels")).json()).toMatchObject({channels:[{type:"whatsapp",label:"BuildFeed",active:true}]});
 });
});

describe("the Bridge's notices",()=>{
 it("refuses an unsigned, stale or forged notice",async()=>{
  await connected();const id=session()!.id;
  expect((await call("/whatsapp/callback",{method:"POST",raw:JSON.stringify({session:id,state:"banned",at:"x"})})).status).toBe(401);
  expect((await notice({session:id,state:"banned",at:"x"},"x".repeat(40))).status).toBe(401);
  expect((await notice({session:id,state:"banned",at:"x"},CALLBACK,Math.floor(Date.now()/1000)-600)).status).toBe(401);
  expect(session()?.state).toBe("connected");
 });
 it("refuses the same signed notice sent twice (a replay)",async()=>{
  await connected();const id=session()!.id,at=String(Math.floor(Date.now()/1000)),nonce="ab".repeat(16);
  const raw=JSON.stringify({session:id,state:"logged_out",at:new Date().toISOString()});
  const send=async()=>(await call("/whatsapp/callback",{method:"POST",raw,headers:{"x-bridge-timestamp":at,"x-bridge-nonce":nonce,"x-bridge-signature":await sig(CALLBACK,at,nonce,"POST","/whatsapp/callback",raw)}})).status;
  expect(await send()).toBe(200);
  expect(await send()).toBe(401);
 });
 it("does not let a late, older notice undo a newer state",async()=>{
  await connected();const id=session()!.id;
  expect((await notice({session:id,state:"logged_out",at:new Date().toISOString()})).status).toBe(200);
  expect((await notice({session:id,state:"connected",at:new Date(Date.now()-120_000).toISOString()})).status).toBe(200);
  expect(session()?.state).toBe("logged_out");
 });
 it("marks the Canais when the phone disconnects, and clears them on reconnecting",async()=>{
  const channel=await connected();const id=session()!.id;
  expect((await notice({session:id,state:"logged_out",at:new Date().toISOString()})).status).toBe(200);
  expect(session()?.state).toBe("logged_out");
  expect(await (await call("/channels")).json()).toMatchObject({channels:[{id:channel,lastError:"WHATSAPP_LOGGED_OUT"}]});
  // A new QR: a new name at the Bridge (the old one is forgotten there),
  // the same Canais.
  calls=[];
  await call("/whatsapp/session",{method:"POST",body:{acceptRisk:true}});
  const next=session()!.id;
  expect(next).not.toBe(id);
  expect(calls.map(c=>`${c.method} ${c.path}`)).toEqual([`DELETE /v1/sessions/${id}`,`POST /v1/sessions/${next}`]);
  expect(db.sqlite.prepare("SELECT session_id AS s FROM deals_channel WHERE id=?").get(channel)).toEqual({s:next});
  await call("/whatsapp/session");
  expect(await (await call("/channels")).json()).toMatchObject({channels:[{lastError:null}]});
 });
});

describe("posting in a Canal",()=>{
 it("sends the caption and the image itself, then waits 30 minutes",async()=>{
  const channel=await connected(),now=Date.now();
  const a=await upsertOffer(db.asD1(),offer(),now),b=await upsertOffer(db.asD1(),offer({key:"k2",title:"Mouse Logitech"}),now);
  await enqueue(db.asD1(),{owner:"alice",offerId:a.id,channels:[channel],link:"https://x.invalid",affiliate:false,huntId:null,sendAt:now,now});
  await enqueue(db.asD1(),{owner:"alice",offerId:b.id,channels:[channel],link:"https://x.invalid",affiliate:false,huntId:null,sendAt:now,now});
  calls=[];
  await dispatchDeals(state.env,fake,now);
  const posts=calls.filter(c=>c.path.endsWith("/posts"));
  expect(posts).toHaveLength(1);
  const sent=nth(posts).body as {channel:string;key:string;text:string;image:string};
  expect(sent.channel).toBe(CANAL);
  expect(sent.key).toMatch(/^deals:/);
  expect(sent.text.split("\n")[0]).toBe("*Oferta do dia: Fone JBL Tune 520BT*");
  expect(sent.text).toContain("https://deals.audit.invalid/r/");
  expect(sent.image).toBe(btoa(String.fromCharCode(...PNG)));
  expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_post WHERE state='sent'").get()).toEqual({n:1});
  // The second waits for the Canal's next turn, 30 minutes on.
  await dispatchDeals(state.env,fake,now+10*60_000);
  expect(calls.filter(c=>c.path.endsWith("/posts"))).toHaveLength(1);
  await dispatchDeals(state.env,fake,now+31*60_000);
  expect(calls.filter(c=>c.path.endsWith("/posts"))).toHaveLength(2);
 });
 it("takes the Bridge's 'too soon' as a turn, not a failure",async()=>{
  const channel=await connected(),now=Date.now(),o=await upsertOffer(db.asD1(),offer(),now);
  await enqueue(db.asD1(),{owner:"alice",offerId:o.id,channels:[channel],link:"https://x.invalid",affiliate:false,huntId:null,sendAt:now,now});
  reply=c=>c.path.endsWith("/posts")?Response.json({code:"TOO_SOON",retryAfter:120},{status:429}):bridgeOK(c);
  await dispatchDeals(state.env,fake,now);
  const post=db.sqlite.prepare("SELECT state,attempts,error,next_at AS nextAt FROM deals_post").get() as {state:string;attempts:number;error:string|null;nextAt:number};
  expect(post).toMatchObject({state:"queued",attempts:0,error:null});
  expect(post.nextAt).toBeGreaterThanOrEqual(now+120_000);
  expect(db.sqlite.prepare("SELECT last_error AS e FROM deals_channel WHERE id=?").get(channel)).toEqual({e:null});
 });
 it("holds the queue while the WhatsApp is not connected",async()=>{
  const channel=await connected(),now=Date.now(),o=await upsertOffer(db.asD1(),offer(),now);
  await notice({session:session()!.id,state:"logged_out",at:new Date().toISOString()});
  await enqueue(db.asD1(),{owner:"alice",offerId:o.id,channels:[channel],link:"https://x.invalid",affiliate:false,huntId:null,sendAt:now,now});
  calls=[];
  await dispatchDeals(state.env,fake,now);
  expect(calls).toHaveLength(0);
  expect(db.sqlite.prepare("SELECT state FROM deals_post").get()).toEqual({state:"queued"});
 });
 it("turns a Canal off when the number is no longer its admin",async()=>{
  const channel=await connected(),now=Date.now(),o=await upsertOffer(db.asD1(),offer(),now);
  await enqueue(db.asD1(),{owner:"alice",offerId:o.id,channels:[channel],link:"https://x.invalid",affiliate:false,huntId:null,sendAt:now,now});
  reply=c=>c.path.endsWith("/posts")?Response.json({code:"NOT_ADMIN"},{status:403}):bridgeOK(c);
  await dispatchDeals(state.env,fake,now);
  expect(db.sqlite.prepare("SELECT active,last_error AS e FROM deals_channel WHERE id=?").get(channel)).toEqual({active:0,e:"DESTINATION_DISABLED"});
 });
});

it("logs the WhatsApp out at the Bridge when the account is purged",async()=>{
 await connected();const id=session()!.id;calls=[];
 await purgeDealsAccount(state.env,"alice");
 expect(calls).toEqual([{method:"DELETE",path:`/v1/sessions/${id}`,body:null}]);
 expect(session()).toBeUndefined();
 expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_channel").get()).toEqual({n:0});
});
