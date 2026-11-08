import {beforeEach,afterEach,it,expect,vi,describe} from "vitest";
import {TestD1,TestBucket,seedUser} from "../security-test-support";
const state=vi.hoisted(()=>({env:{} as Cloudflare.Env}));
vi.mock("cloudflare:workers",()=>({env:state.env,DurableObject:class{}}));
vi.mock("../auth",()=>({auth:{api:{getSession:async({headers}:{headers:Headers})=>{const id=headers.get("authorization")?.replace("Bearer ","");return id?{user:{id}}:null;}}}}));
import {dealsApi} from "./api";
import {upsertOffer,type OfferInput} from "./catalog";
import {enqueue} from "./store";
import {dispatchDeals} from "./dispatch";
import {serveDealsSite} from "./site-public";
import {priceStats} from "./site-detail";

// The public deals page: made from the panel (with its "site" channel),
// fed by the send queue like any channel, served light from the server.
const KEY=btoa(String.fromCharCode(...new Uint8Array(32).fill(7)));
const HOST="deals.audit.invalid";
let db:TestD1,assets:TestBucket,limited:boolean;
beforeEach(()=>{
 db=new TestD1();assets=new TestBucket();seedUser(db,"alice","ultra_deals");seedUser(db,"bob","ultra_deals");limited=false;
 Object.assign(state.env,{DB:db.asD1(),DEALS_ASSETS:assets.asR2(),STAFF_OWNER_EMAIL:"",PULSE_CHANNEL_KEY:KEY,DEALS_PUBLIC_HOST:HOST,DEALS_PUBLIC_LIMIT:{limit:async()=>({success:!limited})}});
 vi.stubGlobal("fetch",vi.fn(async()=>{throw new Error("no network in these tests");}));
});
afterEach(()=>{db.close();vi.unstubAllGlobals();vi.restoreAllMocks();});
const call=(path:string,init:{method?:string;body?:unknown;user?:string}={})=>dealsApi.fetch(new Request(`https://cloud.audit.invalid${path}`,{method:init.method??"GET",headers:{authorization:`Bearer ${init.user??"alice"}`,...(init.body!==undefined?{"content-type":"application/json"}:{})},body:init.body===undefined?undefined:JSON.stringify(init.body)}),state.env);
const visit=(path:string,headers:Record<string,string>={})=>serveDealsSite(new Request(`https://${HOST}${path}`,{headers}),state.env);
type Site={id:string;channelId:string;slug:string;url:string};
async function makeSite(){
 const res=await call("/site",{method:"POST",body:{title:"Ofertas da <b>Ana</b>",slug:"ofertas-ana",description:"Só as boas & baratas",language:"pt"}});
 expect(res.status).toBe(201);return (await res.json() as {site:Site}).site;
}
const offer=(key:string,title:string,price:number):OfferInput=>({key,store:"amazon",externalId:null,sourceId:null,title,url:`https://www.amazon.com.br/dp/${key}`,imageUrl:"https://m.media-amazon.com/x.jpg",price,originalPrice:price+5000,currency:"BRL",kind:"product",platforms:[],discount:20,credit:null,expiresAt:null});
async function send(site:Site,input:OfferInput){
 const stored=await upsertOffer(state.env.DB,input);
 expect(await enqueue(state.env.DB,{owner:"alice",offerId:stored.id,channels:[site.channelId],link:input.url,affiliate:true,huntId:null,sendAt:Date.now()})).toBe(1);
 return stored.id;
}

describe("deals page",()=>{
 it("is made with its own channel, at an address only it has",async()=>{
  const site=await makeSite();
  expect(site).toMatchObject({slug:"ofertas-ana",url:`https://${HOST}/ofertas-ana/`});
  expect(db.sqlite.prepare("SELECT platform,name FROM deals_channel WHERE id=?").get(site.channelId)).toEqual({platform:"site",name:"Ofertas da <b>Ana</b>"});
  const slug=async(value:string,user="alice")=>(await (await call(`/site/slug?slug=${value}`,{user})).json() as {state:string}).state;
  expect(await slug("ofertas-ana")).toBe("yours");expect(await slug("ofertas-ana","bob")).toBe("taken");
  expect(await slug("r")).toBe("reserved");expect(await slug("Com Espaço")).toBe("invalid");expect(await slug("livre")).toBe("available");
  // One page per account, and an address nobody else can take.
  expect((await call("/site",{method:"POST",body:{title:"Outra",slug:"outra",language:"pt"}})).status).toBe(409);
  expect((await call("/site",{method:"POST",user:"bob",body:{title:"Bob",slug:"ofertas-ana",language:"pt"}})).status).toBe(409);
  // A page has nothing to test: its posts just show on it.
  expect((await call(`/channels/${site.channelId}/test`,{method:"POST",body:{kinds:["test"]}})).status).toBe(400);
 });
 it("shows the deals sent to it, escaped, with search, 304 and the click counter",async()=>{
  const site=await makeSite();
  await send(site,offer("B01","Fone JBL <script>",19900));
  const games=await send(site,offer("B02","Controle Xbox",29900));
  db.sqlite.prepare("UPDATE deals_offer SET category='games' WHERE id=?").run(games);
  await dispatchDeals(state.env,fetch);
  expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_post WHERE state='sent'").get()).toEqual({n:2});
  // The address without the last slash goes to the one with it.
  const bare=await visit("/ofertas-ana");expect(bare?.status).toBe(301);expect(bare?.headers.get("location")).toBe(`https://${HOST}/ofertas-ana/`);
  const page=await visit("/ofertas-ana/");expect(page?.status).toBe(200);
  const html=await page!.text();
  expect(html.startsWith("<!doctype html>")).toBe(true);
  expect(html).toContain("Ofertas da &lt;b&gt;Ana&lt;/b&gt;");expect(html).not.toContain("<b>Ana</b>");
  expect(html).toContain("Fone JBL &lt;script&gt;");expect(html).not.toContain("JBL <script>");
  expect(html).toMatch(new RegExp(`href="https://${HOST}/r/[A-Za-z0-9]{8}"`));
  expect(html).toContain("R$");expect(html).toContain('lang="pt-BR"');
  expect(page!.headers.get("content-security-policy")).toContain("script-src 'sha256-");
  // The same version again: 304, nothing rendered.
  const again=await visit("/ofertas-ana/",{"if-none-match":page!.headers.get("etag")!});expect(again?.status).toBe(304);
  // Search and category.
  const found=await (await visit("/ofertas-ana/?q=jbl"))!.text();expect(found).toContain("Fone JBL");expect(found).not.toContain("Controle Xbox");
  const none=await (await visit("/ofertas-ana/?q=geladeira"))!.text();expect(none).toContain("Nada encontrado para");
  const cat=await (await visit("/ofertas-ana/?category=games"))!.text();expect(cat).toContain("Controle Xbox");expect(cat).not.toContain("Fone JBL");
  // The search dialog's results, as the person types.
  const results=await (await visit("/ofertas-ana/search?q=jbl"))!.json() as {results:{url:string;title:string;detail:string}[]};
  expect(results.results).toHaveLength(1);expect(results.results[0]?.title).toBe("Fone JBL <script>");expect(results.results[0]?.detail).toMatch(/^Amazon · R\$\s199,00$/);expect(results.results[0]!.url).toMatch(/^\/ofertas-ana\/o\/[A-Za-z0-9]{8}\/$/);
  expect(html).toContain('<dialog id="search"');expect(page!.headers.get("content-security-policy")).toContain("connect-src 'self'");
  // Another language: the page's own words change.
  expect(await (await visit("/ofertas-ana/?lang=en"))!.text()).toContain("See deal");
  // Not a page, other paths and other methods.
  expect((await visit("/nao-existe/"))?.status).toBe(404);
  expect(await visit("/ofertas-ana/outra-coisa")).toBeNull();
  expect((await serveDealsSite(new Request(`https://${HOST}/ofertas-ana/`,{method:"POST"}),state.env))?.status).toBe(405);
  limited=true;expect((await visit("/ofertas-ana/"))?.status).toBe(429);
 });
 it("stops without the plan, and goes with its channel",async()=>{
  const site=await makeSite();
  expect((await visit("/ofertas-ana/"))?.status).toBe(200);
  db.sqlite.prepare("UPDATE subscription SET plan='max' WHERE reference_id='alice'").run();
  expect((await visit("/ofertas-ana/?q=x"))?.status).toBe(404);
  db.sqlite.prepare("UPDATE subscription SET plan='ultra_deals' WHERE reference_id='alice'").run();
  // A banned account's page stops at once, even for a visit that has its version.
  const seen=(await visit("/ofertas-ana/"))!;expect(seen.status).toBe(200);
  // A lost plan too: no 304 from the version the visitor already has.
  db.sqlite.prepare("UPDATE subscription SET plan='max' WHERE reference_id='alice'").run();
  expect((await serveDealsSite(new Request(`https://${HOST}/ofertas-ana/`,{headers:{"if-none-match":seen.headers.get("etag")??""}}),state.env))?.status).toBe(404);
  db.sqlite.prepare("UPDATE subscription SET plan='ultra_deals' WHERE reference_id='alice'").run();
  db.sqlite.prepare("UPDATE user SET banned=1 WHERE id='alice'").run();
  expect((await serveDealsSite(new Request(`https://${HOST}/ofertas-ana/`,{headers:{"if-none-match":seen.headers.get("etag")??""}}),state.env))?.status).toBe(404);
  db.sqlite.prepare("UPDATE user SET banned=0 WHERE id='alice'").run();
  expect((await call(`/channels/${site.channelId}`,{method:"DELETE"})).status).toBe(200);
  expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_site").get()).toEqual({n:0});
  expect((await visit("/ofertas-ana/"))?.status).toBe(404);
 });
 it("opens each deal on its own page, with its price history",async()=>{
  const site=await makeSite();
  const id=await send(site,offer("B03","Kindle <b>11</b>",49900));
  await dispatchDeals(state.env,fetch);
  const code=(db.sqlite.prepare("SELECT code FROM deals_post WHERE offer_id=?").get(id) as {code:string}).code;
  // The card opens the deal's page; its button still goes to the store.
  const list=await (await visit("/ofertas-ana/"))!.text();
  expect(list).toContain(`href="/ofertas-ana/o/${code}/"`);expect(list).toContain(`href="https://${HOST}/r/${code}"`);
  const bare=await visit(`/ofertas-ana/o/${code}`);expect(bare?.status).toBe(301);expect(bare?.headers.get("location")).toBe(`https://${HOST}/ofertas-ana/o/${code}/`);
  const page=await visit(`/ofertas-ana/o/${code}/`);expect(page?.status).toBe(200);
  let html=await page!.text();
  expect(html).toContain("Kindle &lt;b&gt;11&lt;/b&gt;");expect(html).not.toContain("<b>11</b>");
  expect(html).toContain("Acessar oferta");expect(html).toContain('"@type":"Product"');expect(html).toContain("data-share");
  // One price only: no chart yet.
  expect(html).not.toContain("Histórico de preço");
  // A lower price seen later: the chart and how it compares.
  const day=86_400_000;
  db.sqlite.prepare("UPDATE deals_price SET at=? WHERE offer_id=?").run(Date.now()-20*day,id);
  db.sqlite.prepare("INSERT INTO deals_price(offer_id,at,price,currency) VALUES(?,?,?,?)").run(id,Date.now()-day,39900,"BRL");
  db.sqlite.prepare("UPDATE deals_offer SET price=39900 WHERE id=?").run(id);
  html=await (await visit(`/ofertas-ana/o/${code}/`))!.text();
  expect(html).toContain("Histórico de preço");expect(html).toMatch(/abaixo da média dos últimos 30 dias/);expect(html).toContain("<svg viewBox=\"0 0 100 100\"");
  // Someone else's code, or none: not here.
  expect((await visit("/ofertas-ana/o/ZZZZZZZZ/"))?.status).toBe(404);
  expect(await visit("/ofertas-ana/o/curto/")).toBeNull();
 });
 it("is one of the plan's sites, with the ones Blup publishes",async()=>{
  // Ultra has 10 sites: with 10 published, no room for the page.
  for(let i=0;i<10;i++)db.sqlite.prepare("INSERT INTO site(id,user_id,slug,title) VALUES(?,?,?,?)").run(`s${i}`,"alice",`site-${i}`,`Site ${i}`);
  const full=await call("/site",{method:"POST",body:{title:"Ofertas",slug:"ofertas-ana",language:"pt"}});
  expect(full.status).toBe(403);expect(await full.json()).toMatchObject({code:"SITES_LIMIT",limit:10});
  expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_channel").get()).toEqual({n:0});
  // One site less: the page fits, and then a new Blup site does not.
  db.sqlite.prepare("DELETE FROM site WHERE id='s9'").run();
  await makeSite();
  const {reserve}=await import("../quota");
  expect(await reserve(state.env.DB,"alice","sites",1,10)).toBeNull();
  // An account that goes takes its page along.
  const {purgeDealsAccount}=await import("./purge");
  await purgeDealsAccount(state.env,"alice");
  expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_site").get()).toEqual({n:0});
 });
 it("shows a free game by its platform, until when it is free",async()=>{
  const site=await makeSite();
  const game:OfferInput={key:"gp:1",store:"gamerpower",externalId:"1",sourceId:"gamerpower",title:"Hogwarts Legacy (Epic Games) Giveaway",url:"https://www.gamerpower.com/open/hogwarts",imageUrl:"https://img.example/cover.jpg",price:0,originalPrice:4999,currency:"USD",kind:"game_free",platforms:["epic","pc"],discount:100,credit:"gamerpower",expiresAt:Date.now()+5*86_400_000};
  await send(site,game);await dispatchDeals(state.env,fetch);
  const code=(db.sqlite.prepare("SELECT code FROM deals_post").get() as {code:string}).code;
  for(const html of [await (await visit("/ofertas-ana/"))!.text(),await (await visit(`/ofertas-ana/o/${code}/`))!.text()]){
   expect(html).toContain("Hogwarts Legacy");expect(html).not.toContain("Giveaway");
   expect(html).toContain("Epic Games");expect(html).not.toMatch(/GamerPower/i);
   expect(html).toContain("Grátis até");expect(html).toMatch(/Valor normal: US\$\s49,99/);expect(html).not.toContain("<s>US$");
  }
 });
 it("keeps each deal on the page for 30 days",async()=>{
  const site=await makeSite();
  const id=await send(site,offer("B09","Mouse gamer",9900));await dispatchDeals(state.env,fetch);
  const code=(db.sqlite.prepare("SELECT code FROM deals_post WHERE offer_id=?").get(id) as {code:string}).code;
  expect(await (await visit("/ofertas-ana/"))!.text()).toContain("Mouse gamer");
  db.sqlite.prepare("UPDATE deals_post SET sent_at=? WHERE offer_id=?").run(Date.now()-31*86_400_000,id);
  expect(await (await visit("/ofertas-ana/?x=1"))!.text()).not.toContain("Mouse gamer");
  expect((await visit(`/ofertas-ana/o/${code}/`))?.status).toBe(404);
 });
 it("weighs each price by how long it lasted",()=>{
  const day=86_400_000,now=100*day;
  // 100 for 20 days, then 80 for 10: average 93.3, now 14% below.
  const stats=priceStats([{at:now-40*day,price:100},{at:now-10*day,price:80}],80,now);
  expect(stats).toMatchObject({mood:"below",percent:14,low:80,high:100,start:now-30*day});
  expect(Math.round(stats!.average)).toBe(93);
  // One price, or the same price twice: nothing to compare.
  expect(priceStats([{at:now-day,price:80}],80,now)).toBeNull();
  expect(priceStats([{at:now-5*day,price:80},{at:now-day,price:80}],80,now)).toBeNull();
 });
 it("shows the header as the panel sets it: the name or not, and its links",async()=>{
  await makeSite();
  let html=await (await visit("/ofertas-ana/"))!.text();
  expect(html).toContain('<a class="brand" href="/ofertas-ana/"><span>Ofertas da &lt;b&gt;Ana&lt;/b&gt;</span>');expect(html).not.toContain('id="mobile-menu"');
  const patch=await call("/site",{method:"PATCH",body:{title:"Ofertas da Ana",description:"",language:"pt",showTitle:false,links:[{label:"Instagram",url:"https://instagram.com/ana"}],socials:[]}});
  expect(patch.status).toBe(200);
  expect((await call("/site",{method:"PATCH",body:{title:"X",description:"",language:"pt",showTitle:true,links:[{label:"Ruim",url:"http://inseguro.com"}],socials:[]}})).status).toBe(400);
  const site=(await (await call("/site")).json() as {site:{showTitle:boolean;links:unknown[]}}).site;
  expect(site).toMatchObject({showTitle:false,links:[{label:"Instagram",url:"https://instagram.com/ana"}],socials:[]});
  html=await (await visit("/ofertas-ana/"))!.text();
  expect(html).toContain('href="https://instagram.com/ana" rel="noopener nofollow">Instagram</a>');expect(html).toContain('id="mobile-menu"');
  // Without a logo the name still shows (something has to).
  expect(html).toContain("<span>Ofertas da Ana</span>");
 });
 it("shows the page's social networks, one of each, each on its own site",async()=>{
  await makeSite();
  const save=(socials:unknown)=>call("/site",{method:"PATCH",body:{title:"Ofertas da Ana",description:"",language:"pt",showTitle:true,links:[],socials}});
  expect((await save([{network:"instagram",url:"https://evil.example/ana"}])).status).toBe(400);
  expect((await save([{network:"telegram",url:"https://t.me/a"},{network:"telegram",url:"https://t.me/b"}])).status).toBe(400);
  expect((await save([{network:"youtube",url:"https://youtube.com/@ana"}])).status).toBe(400);
  expect((await save([{network:"instagram",url:"https://instagram.com/ana"},{network:"whatsapp",url:"https://whatsapp.com/channel/abc"}])).status).toBe(200);
  const html=await (await visit("/ofertas-ana/"))!.text();
  expect(html).toContain('class="socials"');expect(html).toContain('href="https://instagram.com/ana"');expect(html).toContain(">Instagram</a>");expect(html).toContain("Siga também");
 });
 it("keeps logos checked and versioned",async()=>{
  await makeSite();
  // A 1×1 PNG.
  const png=Uint8Array.from(atob("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII="),c=>c.charCodeAt(0));
  const up=await dealsApi.fetch(new Request("https://cloud.audit.invalid/site/logo/light",{method:"PUT",headers:{authorization:"Bearer alice","content-type":"image/png"},body:png}),state.env);
  expect(up.status).toBe(201);const {version}=await up.json() as {version:string};
  expect((await dealsApi.fetch(new Request("https://cloud.audit.invalid/site/logo/light",{method:"PUT",headers:{authorization:"Bearer alice","content-type":"image/png"},body:new Uint8Array([1,2,3])}),state.env)).status).toBe(400);
  const html=await (await visit("/ofertas-ana/"))!.text();expect(html).toContain(`/ofertas-ana/logo-light?v=${version}`);
  const logo=await visit(`/ofertas-ana/logo-light?v=${version}`);expect(logo?.status).toBe(200);expect(logo?.headers.get("cache-control")).toContain("immutable");
 });
});
