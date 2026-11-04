import {beforeEach,afterEach,it,expect,vi,describe} from "vitest";
import {TestD1,TestBucket,seedUser} from "../security-test-support";
const state=vi.hoisted(()=>({env:{} as Cloudflare.Env}));
vi.mock("cloudflare:workers",()=>({env:state.env,DurableObject:class{}}));
vi.mock("../auth",()=>({auth:{api:{getSession:async({headers}:{headers:Headers})=>{const id=headers.get("authorization")?.replace("Bearer ","");return id?{user:{id}}:null;}}}}));
import {dealsApi} from "./api";
import {safeFetch,FetchError} from "./fetch";
import {magaluLink,terabyteLink,nuuvemLink,rakutenTarget,amazonLink,mercadoLivreLink,shopeeLink,aliexpressLink,aliexpressSign,aliexpressResponse,aliexpressHost,convertLink,storeOf} from "./links";
import {pageInfo,priceCents} from "./page";
import {upsertOffer,collectDeals,dealsCleanup,gamerPowerOffers,nuuvemPage,type OfferInput} from "./catalog";
import {enqueue,dayOf,seal} from "./store";
import {dispatchDeals,DEALS_BUDGET} from "./dispatch";
import {telegramText,discordPayload,sendTelegram} from "./message";
import {videoOf} from "./video";
import {readChannel,textPrices,textTitle,channelName,collectMirrors} from "./mirror";
import {nuuvemOffers,isCredit,NUUVEM_PAGES_PER_RUN} from "./nuuvem";
import {shopeeOffers,aliexpressOffers,collectShops,ShopRefused} from "./shops";
import {serveDealsLink} from "./redirect";
import {panelCorsOrigin} from "../security";
import {softDelete,purgeAccounts,DAY as LIFE_DAY} from "../lifecycle";
import {dealsLifecycle} from "./purge";
import {dealsAccess} from "./access";

let db:TestD1;
const KEY=btoa(String.fromCharCode(...new Uint8Array(32).fill(7)));
const HOOK="https://discord.com/api/webhooks/123456789012345678/"+"a".repeat(68);
beforeEach(()=>{
 db=new TestD1();seedUser(db,"alice","ultra_deals");seedUser(db,"bob","ultra_pulse_deals");seedUser(db,"plain","ultra");
 Object.assign(state.env,{DB:db.asD1(),STAFF_OWNER_EMAIL:"",PULSE_CHANNEL_KEY:KEY,DEALS_PUBLIC_HOST:"deals.audit.invalid",DEALS_TELEGRAM_BOT:"BlupDealsBot",DEALS_TELEGRAM_TOKEN:"123456:"+"b".repeat(30),DEALS_TELEGRAM_WEBHOOK_SECRET:"hook-secret"});
});
afterEach(()=>{db.close();vi.unstubAllGlobals();vi.restoreAllMocks();});
const call=(path:string,init:{method?:string;body?:unknown;user?:string|null;headers?:Record<string,string>}={})=>dealsApi.fetch(new Request(`https://cloud.audit.invalid${path}`,{method:init.method??"GET",headers:{...(init.user===null?{}:{authorization:`Bearer ${init.user??"alice"}`}),...(init.body!==undefined?{"content-type":"application/json"}:{}),...init.headers},body:init.body===undefined?undefined:JSON.stringify(init.body)}),state.env);
// A fake Internet: public DNS answers, then the page or API by address.
function net(routes:Record<string,(req:Request)=>Response|Promise<Response>>,ip="93.184.216.34"){
 const calls:Request[]=[];
 const send=vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{
  const req=new Request(input instanceof URL?input.href:input,init);calls.push(req);const url=new URL(req.url);
  if(url.hostname==="cloudflare-dns.com")return Response.json({Status:0,Answer:[{type:url.searchParams.get("type")==="A"?1:28,data:url.searchParams.get("type")==="A"?ip:"2606:4700::1"}]});
  const route=routes[`${url.hostname}${url.pathname}`]??routes[url.hostname];
  if(!route)throw new Error(`unexpected ${req.url}`);
  return route(req);
 }) as unknown as typeof fetch;
 return {send,calls};
}
const html=(body:string)=>new Response(body,{headers:{"content-type":"text/html; charset=utf-8"}});
async function addDiscord(user="alice"){const r=await call("/channels",{method:"POST",user,body:{type:"discord",webhook:HOOK}});return (await r.json() as {id:string}).id;}
const offer=(over:Partial<OfferInput>={}):OfferInput=>({key:"k1",store:"amazon",externalId:null,sourceId:null,title:"Fone",url:"https://www.amazon.com.br/dp/B000000001",imageUrl:null,price:10000,originalPrice:null,currency:"BRL",kind:"product",platforms:[],discount:null,credit:null,expiresAt:null,...over});

describe("link conversion by store",()=>{
 it("knows the stores by address",()=>{
  expect(storeOf(new URL("https://www.amazon.com.br/x"))).toBe("amazon");expect(storeOf(new URL("https://produto.mercadolivre.com.br/MLB-1"))).toBe("mercadolivre");
  expect(storeOf(new URL("https://shopee.com.br/p-i.1.2"))).toBe("shopee");expect(storeOf(new URL("https://pt.aliexpress.com/item/1005001.html"))).toBe("aliexpress");
  expect(storeOf(new URL("https://amazon.com.br.evil.invalid/x"))).toBe("other");expect(storeOf(new URL("https://notmercadolivre.com.br/x"))).toBe("other");
 });
 it("puts the Amazon tag on the product and drops someone else's",()=>{
  expect(amazonLink(new URL("https://www.amazon.com.br/Fone-X/dp/B0ABCDEF12/ref=sr_1_1?tag=outro-20&keywords=fone"),"meu-20")).toBe("https://www.amazon.com.br/dp/B0ABCDEF12?tag=meu-20");
  expect(amazonLink(new URL("https://www.amazon.com.br/s?k=fone&tag=outro-20&ascsubtag=x"),"meu-20")).toBe("https://www.amazon.com.br/s?k=fone&tag=meu-20");
 });
 it("puts the Mercado Livre word and tool, dropping old ones",()=>{
  const out=new URL(mercadoLivreLink(new URL("https://produto.mercadolivre.com.br/MLB-123-fone?matt_word=outro&matt_tool=1&color=azul"),"minhaloja","98765"));
  expect(out.searchParams.get("matt_word")).toBe("minhaloja");expect(out.searchParams.get("matt_tool")).toBe("98765");expect(out.searchParams.get("color")).toBe("azul");expect(out.searchParams.getAll("matt_word")).toHaveLength(1);
 });
 it("asks Shopee for the short link with the signed request",async()=>{
  const {send,calls}=net({"open-api.affiliate.shopee.com.br":()=>Response.json({data:{generateShortLink:{shortLink:"https://s.shopee.com.br/abc"}}})});
  expect(await shopeeLink(new URL("https://shopee.com.br/Fone-i.1.2?sp_atk=x"),"12345","s".repeat(32),send,1_800_000_000_000)).toBe("https://s.shopee.com.br/abc");
  const auth=calls[0].headers.get("authorization")??"",body=await calls[0].text();
  const expected=Array.from(new Uint8Array(await crypto.subtle.digest("SHA-256",new TextEncoder().encode(`12345${1_800_000_000}${body}${"s".repeat(32)}`))),b=>b.toString(16).padStart(2,"0")).join("");
  expect(auth).toBe(`SHA256 Credential=12345, Timestamp=1800000000, Signature=${expected}`);expect(body).toContain("https://shopee.com.br/Fone-i.1.2");expect(body).not.toContain("sp_atk");
 });
 it("asks AliExpress with the HMAC signature and takes only its own links",async()=>{
  expect(await aliexpressSign({b:"2",a:"1"},"secret")).toMatch(/^[0-9A-F]{64}$/);
  const ok=net({"api-sg.aliexpress.com":()=>Response.json({aliexpress_affiliate_link_generate_response:{resp_result:{resp_code:200,result:{promotion_links:{promotion_link:[{promotion_link:"https://s.click.aliexpress.com/e/_abc"}]}}}}})});
  expect(await aliexpressLink(new URL("https://pt.aliexpress.com/item/1005001234567.html?spm=x"),"5000","k".repeat(32),"meu",ok.send,1)).toBe("https://s.click.aliexpress.com/e/_abc");
  const query=new URL(ok.calls[0].url).searchParams;expect(query.get("source_values")).toBe("https://www.aliexpress.com/item/1005001234567.html");expect(query.get("tracking_id")).toBe("meu");
  const {sign,...params}=Object.fromEntries(query);expect(sign).toBe(await aliexpressSign(params,"k".repeat(32)));
  const evil=net({"api-sg.aliexpress.com":()=>Response.json({aliexpress_affiliate_link_generate_response:{resp_result:{resp_code:200,result:{promotion_links:{promotion_link:[{promotion_link:"https://evil.invalid/x"}]}}}}})});
  expect(await convertLink(new URL("https://pt.aliexpress.com/item/1005001234567.html"),{store:"aliexpress",appKey:"5000",appSecret:"k".repeat(32),trackingId:"meu"},evil.send)).toMatchObject({affiliate:false,reason:"conversion_failed"});
 });
 it("takes AliExpress's documented answer and only its own hosts",async()=>{
  // The shape of the official example (resp_code "200", promotion_links as a list).
  const official={aliexpress_affiliate_link_generate_response:{resp_result:{resp_code:"200",resp_msg:"success",result:{promotion_link_type:0,total_result_count:1,tracking_id:"meu",promotion_links:[{source_value:"https://www.aliexpress.com/item/1005001234567.html",promotion_link:"https://s.click.aliexpress.com/e/_official"}]}},request_id:"x"}};
  const ok=net({"api-sg.aliexpress.com":()=>Response.json(official)});
  expect(await aliexpressLink(new URL("https://pt.aliexpress.com/item/1005001234567.html"),"5000","k".repeat(32),"meu",ok.send,1)).toBe("https://s.click.aliexpress.com/e/_official");
  for(const bad of [{...official,aliexpress_affiliate_link_generate_response:{resp_result:{resp_code:"2000",result:official.aliexpress_affiliate_link_generate_response.resp_result.result}}},{aliexpress_affiliate_link_generate_response:{resp_result:{resp_code:"200",result:{promotion_links:[]}}}},{aliexpress_affiliate_link_generate_response:{resp_result:{resp_code:"200",result:{promotion_links:Array.from({length:51},()=>({promotion_link:"https://s.click.aliexpress.com/e/_x"}))}}}}])expect(aliexpressResponse.safeParse(bad).success).toBe(false);
  for(const host of ["https://evilaliexpress.com/e/_x","https://aliexpress.com.evil.invalid/e/_x","http://s.click.aliexpress.com/e/_x","https://user@s.click.aliexpress.com/e/_x"]){
   expect(aliexpressHost(host)).toBe(false);
   const lookalike=net({"api-sg.aliexpress.com":()=>Response.json({...official,aliexpress_affiliate_link_generate_response:{resp_result:{resp_code:"200",result:{promotion_links:[{promotion_link:host}]}}}})});
   await expect(aliexpressLink(new URL("https://pt.aliexpress.com/item/1005001234567.html"),"5000","k".repeat(32),"meu",lookalike.send,1)).rejects.toBeInstanceOf(FetchError);
  }
  expect(aliexpressHost("https://aliexpress.com/x")).toBe(true);expect(aliexpressHost("https://s.click.aliexpress.com/e/_x")).toBe(true);
 });
 it("says why a link stays as it is",async()=>{
  expect(await convertLink(new URL("https://loja.example.com.br/p"),null)).toMatchObject({store:"other",affiliate:false,reason:"unsupported_store"});
  expect(await convertLink(new URL("https://www.amazon.com.br/dp/B0ABCDEF12"),null)).toMatchObject({store:"amazon",affiliate:false,reason:"not_configured"});
 });
 it("reads the product from the page and prices in minor units",()=>{
  const info=pageInfo(`<title>Velho</title><meta property="og:title" content="Fone &amp; Cia &lt;b&gt;"><meta property="og:image" content="/img.jpg"><script type="application/ld+json">{"@type":"Product","offers":{"price":"1299.90","priceCurrency":"brl"}}</script>`,new URL("https://loja.example.com.br/p"));
  expect(info).toEqual({title:"Fone & Cia",image:"https://loja.example.com.br/img.jpg",price:129990,currency:"BRL"});
  expect(priceCents("R$ 1.299,90")).toBe(129990);expect(priceCents("$19.99")).toBe(1999);expect(priceCents("N/A")).toBeNull();
  expect(pageInfo(`<meta property="og:image" content="http://insecure.invalid/a.jpg">`,new URL("https://a.example.com")).image).toBeNull();
 });
});

describe("Nuuvem through Rakuten",()=>{
 it("makes Rakuten's deep link with the person's ID and drops someone else's",()=>{
  expect(storeOf(new URL("https://www.nuuvem.com/br-pt/item/pine-hearts"))).toBe("nuuvem");
  expect(nuuvemLink(new URL("https://www.nuuvem.com/br-pt/item/pine-hearts?utm_source=x&ranSiteID=outro#top"),"*AbCdEfGh1")).toBe("https://click.linksynergy.com/deeplink?id=*AbCdEfGh1&mid=46796&murl=https%3A%2F%2Fwww.nuuvem.com%2Fbr-pt%2Fitem%2Fpine-hearts");
  expect(rakutenTarget(new URL("https://click.linksynergy.com/deeplink?id=*Outro0000&mid=46796&murl=https%3A%2F%2Fwww.nuuvem.com%2Fbr-pt%2Fitem%2Fpine-hearts"))?.href).toBe("https://www.nuuvem.com/br-pt/item/pine-hearts");
  expect(rakutenTarget(new URL("https://click.linksynergy.com/deeplink?murl=http%3A%2F%2Fx.invalid"))).toBeNull();
 });
 it("reads the game from its page without opening the other person's link",async()=>{
  await call("/affiliates/nuuvem",{method:"PUT",body:{siteId:"*AbCdEfGh1"}});
  expect((await call("/affiliates/nuuvem",{method:"PUT",body:{siteId:"<x>"}})).status).toBe(400);
  const {send,calls}=net({"www.nuuvem.com":()=>html('<meta property="og:title" content="Pine Hearts"><meta property="og:image" content="https://assets.nuuvem.com/x.jpg"><meta itemprop="priceCurrency" content="BRL" /><meta itemprop="price" content="38.99" />')});
  vi.stubGlobal("fetch",send);
  const preview=await(await call("/offers/preview",{method:"POST",body:{url:"https://click.linksynergy.com/deeplink?id=*Outro0000&mid=46796&murl=https%3A%2F%2Fwww.nuuvem.com%2Fbr-pt%2Fitem%2Fpine-hearts"}})).json();
  expect(preview).toEqual({store:"nuuvem",title:"Pine Hearts",image:"https://assets.nuuvem.com/x.jpg",price:3899,currency:"BRL",link:"https://click.linksynergy.com/deeplink?id=*AbCdEfGh1&mid=46796&murl=https%3A%2F%2Fwww.nuuvem.com%2Fbr-pt%2Fitem%2Fpine-hearts",affiliate:true});
  expect(calls.some(r=>new URL(r.url).hostname==="click.linksynergy.com")).toBe(false);
 });
});

describe("safe fetch",()=>{
 it.each(["http://example.com/","https://127.0.0.1/","https://10.0.0.5/","https://[::1]/","https://localhost/","https://cloud.getblup.io/api/me","https://169.254.169.254/latest","ftp://example.com/"])("refuses %s",async url=>{
  const {send,calls}=net({});await expect(safeFetch(url,{send})).rejects.toMatchObject({code:"FETCH_BLOCKED"});expect(calls.filter(c=>!c.url.startsWith("https://cloudflare-dns.com"))).toHaveLength(0);
 });
 it("refuses a name that points to a private address",async()=>{
  const {send,calls}=net({"shop.example.com":()=>html("x")},"10.1.2.3");
  await expect(safeFetch("https://shop.example.com/p",{send})).rejects.toMatchObject({code:"FETCH_BLOCKED"});expect(calls.some(c=>new URL(c.url).hostname==="shop.example.com")).toBe(false);
 });
 it("checks every redirect again and stops at a private one",async()=>{
  const {send}=net({"amzn.to":()=>new Response(null,{status:301,headers:{location:"https://127.0.0.1/admin"}})});
  await expect(safeFetch("https://amzn.to/abc",{send})).rejects.toBeInstanceOf(FetchError);
  const ok=net({"amzn.to":()=>new Response(null,{status:301,headers:{location:"https://www.amazon.com.br/dp/B0ABCDEF12"}}),"www.amazon.com.br":()=>html("<title>Fone</title>")});
  const page=await safeFetch("https://amzn.to/abc",{send:ok.send});expect(page.url.href).toBe("https://www.amazon.com.br/dp/B0ABCDEF12");expect(page.text).toContain("Fone");
  expect(ok.calls.filter(c=>!c.url.startsWith("https://cloudflare-dns.com")).every(c=>c.redirect==="manual")).toBe(true);
 });
 it("keeps only the first bytes and gives up after too many hops",async()=>{
  const big=net({"big.example.com":()=>html("a".repeat(100_000))});expect((await safeFetch("https://big.example.com/",{send:big.send,maxBytes:1000})).text).toHaveLength(1000);
  const loop=net({"loop.example.com":()=>new Response(null,{status:302,headers:{location:"https://loop.example.com/"}})});await expect(safeFetch("https://loop.example.com/",{send:loop.send})).rejects.toMatchObject({code:"FETCH_FAILED"});
 });
});

describe("secrets never come back",()=>{
 it("keeps store IDs and keys sealed and shows only a hint",async()=>{
  expect((await call("/affiliates/shopee",{method:"PUT",body:{appId:"123456",secret:"S3cretS3cretS3cretS3cret"}})).status).toBe(200);
  expect((await call("/affiliates/amazon",{method:"PUT",body:{tag:"minhatag-20"}})).status).toBe(200);
  expect((await call("/affiliates/amazon",{method:"PUT",body:{appId:"1",secret:"x"}})).status).toBe(400);
  const listed=await(await call("/affiliates")).text();
  expect(listed).not.toContain("S3cretS3cret");expect(listed).not.toContain("minhatag-20");
  expect(JSON.parse(listed).stores).toEqual([{store:"amazon",configured:true,hint:"…g-20",country:null},{store:"mercadolivre",configured:false,hint:null,country:null},{store:"shopee",configured:true,hint:"App ID …3456 · BR",country:"br"},{store:"aliexpress",configured:false,hint:null,country:null},{store:"nuuvem",configured:false,hint:null,country:null},{store:"magalu",configured:false,hint:null,country:null},{store:"terabyte",configured:false,hint:null,country:null}]);
  const rows=JSON.stringify(db.sqlite.prepare("SELECT * FROM deals_affiliate").all());expect(rows).not.toContain("S3cretS3cret");expect(rows).not.toContain("minhatag-20");
  expect(await(await call("/affiliates",{user:"bob"})).json()).toMatchObject({stores:[{configured:false},{configured:false},{configured:false},{configured:false},{configured:false},{configured:false},{configured:false}]});
 });
 it("never sends a channel's webhook back",async()=>{
  const id=await addDiscord();
  const listed=await(await call("/channels")).text();expect(listed).not.toContain("a".repeat(68));expect(JSON.parse(listed).channels).toEqual([expect.objectContaining({id,type:"discord",active:true,lastError:null})]);
  expect(JSON.stringify(db.sqlite.prepare("SELECT * FROM deals_channel").all())).not.toContain("a".repeat(68));
  expect(await(await call("/channels",{user:"bob"})).json()).toEqual({channels:[]});
  expect((await call(`/channels/${id}`,{method:"DELETE",user:"bob"})).status).toBe(404);
 });
});

describe("session, plan and CORS",()=>{
 it("needs a session and the plan to change anything",async()=>{
  for(const path of ["/channels","/affiliates","/offers","/hunts","/stats?days=7"])expect((await call(path,{user:null})).status).toBe(401);
  expect((await call("/channels",{method:"POST",user:"plain",body:{type:"discord",webhook:HOOK}})).status).toBe(403);
  expect((await call("/hunts",{method:"POST",user:"plain",body:{kind:"games_free",channelIds:["x"],filters:{}}})).status).toBe(403);
 });
 it("lets only the Deals panel read its routes across origins",()=>{
  const env={PULSE_APP_URL:"https://pulse.audit.invalid",DEALS_APP_URL:"https://deals.audit.invalid"};
  for(const path of ["/api/deals/channels","/api/deals/offers/preview","/api/deals/stats"])expect(panelCorsOrigin(env,env.DEALS_APP_URL,path)).toBe(env.DEALS_APP_URL);
  expect(panelCorsOrigin(env,env.PULSE_APP_URL,"/api/deals/channels")).toBeNull();
 });
 it("takes Telegram's webhook only with its secret",async()=>{
  expect((await call("/telegram/webhook",{method:"POST",user:null,body:{update_id:1}})).status).toBe(401);
  expect((await call("/telegram/webhook",{method:"POST",user:null,headers:{"x-telegram-bot-api-secret-token":"wrong"},body:{update_id:1}})).status).toBe(401);
 });
});

describe("Telegram linking",()=>{
 it("takes only a bot that is admin (and, in a channel, may post)",async()=>{
  const hook=(body:object)=>call("/telegram/webhook",{method:"POST",user:null,headers:{"x-telegram-bot-api-secret-token":"hook-secret"},body});
  const start=async(update:number,from:number)=>{const link=await(await call("/telegram/link",{method:"POST",body:{}})).json() as {id:string;url:string};await hook({update_id:update,message:{text:`/start ${/start=([a-f0-9]{48})$/.exec(link.url)?.[1]}`,chat:{id:from,type:"private"},from:{id:from,is_bot:false}}});return link.id;};
  const state=async(id:string)=>(await(await call(`/telegram/link/${id}`)).json() as {state:string}).state;
  let id=await start(100,777);
  await hook({update_id:101,my_chat_member:{chat:{id:-2001,type:"supergroup",title:"Grupo"},from:{id:777,is_bot:false},new_chat_member:{status:"member"}}});
  await hook({update_id:102,my_chat_member:{chat:{id:-2002,type:"group",title:"Grupo 2"},from:{id:777,is_bot:false},new_chat_member:{status:"restricted"}}});
  await hook({update_id:103,my_chat_member:{chat:{id:-1004,type:"channel",title:"Sem post"},from:{id:777,is_bot:false},new_chat_member:{status:"administrator"}}});
  await hook({update_id:104,my_chat_member:{chat:{id:-1005,type:"channel",title:"Negado"},from:{id:777,is_bot:false},new_chat_member:{status:"administrator",can_post_messages:false}}});
  expect(await state(id)).toBe("pending");
  await hook({update_id:105,my_chat_member:{chat:{id:-2001,type:"supergroup",title:"Grupo"},from:{id:777,is_bot:false},new_chat_member:{status:"administrator"}}});
  expect(await(await call(`/telegram/link/${id}`)).json()).toMatchObject({state:"candidate",candidate:{id:"-2001",label:"Grupo"}});
  const channel=(await(await call(`/telegram/link/${id}/confirm`,{method:"POST",body:{}})).json() as {id:string}).id;
  // Demoted to member later: the channel stops.
  await hook({update_id:106,my_chat_member:{chat:{id:-2001,type:"supergroup"},from:{id:777,is_bot:false},new_chat_member:{status:"member"}}});
  expect((await(await call("/channels")).json() as {channels:{id:string;active:boolean}[]}).channels.find(c=>c.id===channel)).toMatchObject({active:false});
  id=await start(107,778);
  await hook({update_id:108,my_chat_member:{chat:{id:-1006,type:"channel",title:"Pode postar"},from:{id:778,is_bot:false},new_chat_member:{status:"administrator",can_post_messages:true}}});
  expect(await state(id)).toBe("candidate");
 });
 it("links the channel where the same person added the bot as admin",async()=>{
  const link=await(await call("/telegram/link",{method:"POST",body:{}})).json() as {id:string;url:string};
  const code=/start=([a-f0-9]{48})$/.exec(link.url)?.[1];expect(link.url).toMatch(/^https:\/\/t\.me\/BlupDealsBot\?start=/);
  const hook=(body:object)=>call("/telegram/webhook",{method:"POST",user:null,headers:{"x-telegram-bot-api-secret-token":"hook-secret"},body});
  const reply=await(await hook({update_id:10,message:{text:`/start ${code}`,chat:{id:555,type:"private"},from:{id:555,is_bot:false}}})).json() as {method:string;reply_markup:{inline_keyboard:{url:string}[][]}};
  expect(reply.method).toBe("sendMessage");expect(reply.reply_markup.inline_keyboard.flat().map(b=>b.url)).toEqual(["https://t.me/BlupDealsBot?startchannel&admin=post_messages","https://t.me/BlupDealsBot?startgroup&admin=post_messages"]);
  // Someone else adding the bot somewhere is not this link's candidate.
  await hook({update_id:11,my_chat_member:{chat:{id:-1001,type:"channel",title:"Outro"},from:{id:999,is_bot:false},new_chat_member:{status:"administrator",can_post_messages:true}}});
  expect(await(await call(`/telegram/link/${link.id}`)).json()).toMatchObject({state:"pending",candidate:null});
  await hook({update_id:12,my_chat_member:{chat:{id:-1002,type:"channel",title:"Ofertas <b>"},from:{id:555,is_bot:false},new_chat_member:{status:"administrator",can_post_messages:true}}});
  await hook({update_id:12,my_chat_member:{chat:{id:-1003,type:"channel",title:"Repetido"},from:{id:555,is_bot:false},new_chat_member:{status:"administrator",can_post_messages:true}}});
  expect(await(await call(`/telegram/link/${link.id}`)).json()).toMatchObject({state:"candidate",candidate:{id:"-1002",label:"Ofertas"}});
  expect((await call(`/telegram/link/${link.id}`,{user:"bob"})).status).toBe(404);
  const confirmed=await call(`/telegram/link/${link.id}/confirm`,{method:"POST",body:{}});expect(confirmed.status).toBe(201);
  const {id}=await confirmed.json() as {id:string};expect((await(await call("/channels")).json() as {channels:{id:string;type:string;label:string}[]}).channels).toEqual([expect.objectContaining({id,type:"telegram",label:"Ofertas"})]);
  await hook({update_id:13,my_chat_member:{chat:{id:-1002,type:"channel"},from:{id:555,is_bot:false},new_chat_member:{status:"left"}}});
  expect((await(await call("/channels")).json() as {channels:{active:boolean;lastError:string|null}[]}).channels[0]).toMatchObject({active:false,lastError:"DESTINATION_DISABLED"});
 });
});

describe("the daily cap and dedupe",()=>{
 it("queues at most 100 posts a day per account",async()=>{
  const channel=await addDiscord(),now=Date.parse("2026-11-04T12:00:00Z");
  for(let i=0;i<100;i++){const o=await upsertOffer(db.asD1(),offer({key:`k${i}`}),now);expect(await enqueue(db.asD1(),{owner:"alice",offerId:o.id,channels:[channel],link:"https://x.invalid",affiliate:false,huntId:null,sendAt:now,now})).toBe(1);}
  const extra=await upsertOffer(db.asD1(),offer({key:"extra"}),now);
  expect(await enqueue(db.asD1(),{owner:"alice",offerId:extra.id,channels:[channel],link:"https://x.invalid",affiliate:false,huntId:null,sendAt:now,now})).toBe(0);
  expect(await enqueue(db.asD1(),{owner:"alice",offerId:extra.id,channels:[channel],link:"https://x.invalid",affiliate:false,huntId:null,sendAt:now+86_400_000,now})).toBe(1);
  db.sqlite.prepare("UPDATE deals_post SET day=? WHERE user_id='alice'").run(dayOf());
  vi.stubGlobal("fetch",net({}).send);
  expect((await call("/offers",{method:"POST",body:{url:"https://www.amazon.com.br/dp/B0ABCDEF12",channelIds:[channel]}})).status).toBe(429);
 });
 it("splits the cap across channels inside one batch",async()=>{
  const channels:string[]=[];for(let i=0;i<3;i++){const r=await call("/channels",{method:"POST",body:{type:"discord",webhook:`https://discord.com/api/webhooks/12345678901234567${i}/${"c".repeat(68)}`}});channels.push((await r.json() as {id:string}).id);}
  const now=Date.now();for(let i=0;i<99;i++){const o=await upsertOffer(db.asD1(),offer({key:`f${i}`}),now);await enqueue(db.asD1(),{owner:"alice",offerId:o.id,channels:[channels[0]],link:"https://x.invalid",affiliate:false,huntId:null,sendAt:now,now});}
  const last=await upsertOffer(db.asD1(),offer({key:"last"}),now);
  expect(await enqueue(db.asD1(),{owner:"alice",offerId:last.id,channels,link:"https://x.invalid",affiliate:false,huntId:null,sendAt:now,now})).toBe(1);
 });
 it("stores an offer once and its price only when it changes",async()=>{
  const a=await upsertOffer(db.asD1(),offer(),1000),b=await upsertOffer(db.asD1(),offer({price:9000}),2000),c=await upsertOffer(db.asD1(),offer({price:9000}),3000);
  expect(a).toEqual({id:a.id,isNew:true});expect(b).toEqual({id:a.id,isNew:false});expect(c.id).toBe(a.id);
  expect(db.sqlite.prepare("SELECT price FROM deals_price ORDER BY at").all().map(r=>r.price)).toEqual([10000,9000]);
  expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_offer").get()?.n).toBe(1);
 });
 it("never posts the same offer twice in a channel",async()=>{
  const channel=await addDiscord(),o=await upsertOffer(db.asD1(),offer());
  const input={owner:"alice",offerId:o.id,channels:[channel],link:"https://x.invalid",affiliate:false,huntId:null,sendAt:Date.now()};
  expect(await enqueue(db.asD1(),input)).toBe(1);expect(await enqueue(db.asD1(),input)).toBe(0);
 });
});

describe("pasting a link",()=>{
 it("previews with the person's ID and queues it to the chosen channels",async()=>{
  const channel=await addDiscord();await call("/affiliates/amazon",{method:"PUT",body:{tag:"minhatag-20"}});
  vi.stubGlobal("fetch",net({"amzn.to":()=>new Response(null,{status:301,headers:{location:"https://www.amazon.com.br/Fone/dp/B0ABCDEF12?tag=outro-20"}}),"www.amazon.com.br":()=>html(`<meta property="og:title" content="Fone Bom"><meta property="og:image" content="https://m.media-amazon.com/x.jpg"><meta property="product:price:amount" content="199.90"><meta property="product:price:currency" content="BRL">`)}).send);
  const preview=await(await call("/offers/preview",{method:"POST",body:{url:"https://amzn.to/abc"}})).json();
  expect(preview).toEqual({store:"amazon",title:"Fone Bom",image:"https://m.media-amazon.com/x.jpg",price:19990,currency:"BRL",link:"https://www.amazon.com.br/dp/B0ABCDEF12?tag=minhatag-20",affiliate:true});
  const created=await call("/offers",{method:"POST",body:{url:"https://amzn.to/abc",channelIds:[channel]}});expect(created.status).toBe(201);expect(await created.json()).toMatchObject({queued:1});
  expect((await call("/offers",{method:"POST",body:{url:"https://amzn.to/abc",channelIds:[channel]}})).status).toBe(409);
  expect((await call("/offers",{method:"POST",body:{url:"https://amzn.to/abc",channelIds:["not-mine"]}})).status).toBe(404);
  const posts=await(await call("/offers?state=queued")).json() as {posts:{title:string;link:string;state:string;clicks:number;lastError:string|null;sentAt:string|null}[]};
  expect(posts.posts).toEqual([expect.objectContaining({title:"Fone Bom",link:"https://www.amazon.com.br/dp/B0ABCDEF12?tag=minhatag-20",state:"queued",clicks:0,lastError:null,sentAt:null})]);
  expect((await call("/offers/preview",{method:"POST",body:{url:"https://192.168.0.1/x"}})).status).toBe(400);
 });
});

describe("games, sending and clicks",()=>{
 const gamerPower=[{id:1,title:"Jogo Grátis",worth:"$19.99",image:"https://www.gamerpower.com/a.jpg",open_giveaway_url:"https://www.gamerpower.com/open/jogo",platforms:"PC, Steam",end_date:"N/A",status:"Active",type:"Game"},{id:2,title:"Velho",worth:"N/A",open_giveaway_url:"https://www.gamerpower.com/open/velho",platforms:"PS5",end_date:"2020-01-01 00:00:00",status:"Active",type:"Game"}];
 it("parses the giveaways",()=>{
  expect(gamerPowerOffers(gamerPower,Date.parse("2026-11-04T00:00:00Z"))).toEqual([expect.objectContaining({key:"gamerpower:1",kind:"game_free",price:0,originalPrice:1999,platforms:["pc","steam"],credit:"gamerpower"})]);
 });
 it("credits the source with an active link on Telegram and Discord",()=>{
  const game={title:"Jogo <x>",image:null,price:0,originalPrice:1999,currency:"USD",discount:100,link:"https://deals.audit.invalid/r/ABCDEFGH",affiliate:false,kind:"game_free",credit:"gamerpower"};
  const text=telegramText(game,"pt");expect(text).toContain('<a href="https://www.gamerpower.com/">Via GamerPower</a>');expect(text).toContain("\nGrátis\n");expect(text).not.toContain("US$");expect(discordPayload(game,"en").embeds[0].description.split("\n")[0]).toBe("Free");expect(text).toContain("<b>Jogo &lt;x&gt;</b>");expect(text).not.toContain("#publi");
  const embed=discordPayload(game,"en").embeds[0];expect(embed.description).toContain("[Via GamerPower](https://www.gamerpower.com/)");expect(embed).not.toHaveProperty("footer");
  expect(telegramText({...game,credit:"unknown"},"pt")).not.toContain("Via");expect(telegramText({...game,credit:null},"pt")).not.toContain("Via");
 });
 it("reads each source once per round for everyone, first only filling the catalog",async()=>{
  const channel=await addDiscord(),bobChannel=await addDiscord("bob");
  const {send,calls}=net({"www.gamerpower.com":()=>Response.json(gamerPower)});
  const t0=Date.parse("2026-11-04T10:00:00Z");
  await collectDeals(state.env,send,t0);expect(calls).toHaveLength(1);expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_post").get()?.n).toBe(0);
  await collectDeals(state.env,send,t0+60_000);expect(calls).toHaveLength(1);
  db.sqlite.exec(`INSERT INTO deals_hunt(id,user_id,name,kind,source_id,filters,channels,created_at) VALUES('h1','alice','J','games_free','gamerpower','{"platforms":["steam"]}','["${channel}"]',0),('h2','bob','J','games_free','gamerpower','{"platforms":["xbox"]}','["${bobChannel}"]',0)`);
  const next=[...gamerPower,{...gamerPower[0],id:3,title:"Novo"}];
  const later=net({"www.gamerpower.com":()=>Response.json(next)});
  await collectDeals(state.env,later.send,t0+31*60_000);expect(later.calls).toHaveLength(1);
  expect(db.sqlite.prepare("SELECT p.user_id AS u,o.title FROM deals_post p JOIN deals_offer o ON o.id=p.offer_id").all()).toEqual([{u:"alice",title:"Novo"}]);
  expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_offer").get()?.n).toBe(2);
 });
 it("sends one post per channel at a time and counts real clicks",async()=>{
  const channel=await addDiscord(),now=Date.now();
  const a=await upsertOffer(db.asD1(),offer({key:"a",title:"Primeiro"}),now),b=await upsertOffer(db.asD1(),offer({key:"b",title:"Segundo"}),now);
  for(const o of [a,b])await enqueue(db.asD1(),{owner:"alice",offerId:o.id,channels:[channel],link:"https://www.amazon.com.br/dp/B000000001?tag=minhatag-20",affiliate:true,huntId:null,sendAt:now,now});
  const sent:{embeds:{title:string;url:string;description:string}[]}[]=[];
  const {send}=net({"discord.com":async req=>{sent.push(await req.json());return Response.json({id:"1"});}});
  await dispatchDeals(state.env,send,now);
  expect(sent).toHaveLength(1);expect(sent[0].embeds[0].url).toMatch(/^https:\/\/deals\.audit\.invalid\/r\/[A-Za-z0-9]{8}$/);expect(sent[0].embeds[0].description).toContain("#publi");
  await dispatchDeals(state.env,send,now+1000);expect(sent).toHaveLength(1);
  await dispatchDeals(state.env,send,now+3*60_000);expect(sent).toHaveLength(2);
  const code=/\/r\/([A-Za-z0-9]{8})$/.exec(sent[0].embeds[0].url)?.[1];
  const go=(ua:string)=>serveDealsLink(new Request(`https://deals.audit.invalid/r/${code}`,{headers:{"user-agent":ua}}),state.env);
  const res=await go("Mozilla/5.0");expect(res.status).toBe(302);expect(res.headers.get("location")).toBe("https://www.amazon.com.br/dp/B000000001?tag=minhatag-20");
  await go("TelegramBot (like TwitterBot)");await go("Mozilla/5.0");
  expect(db.sqlite.prepare("SELECT sum(clicks) AS n FROM deals_click_day").get()?.n).toBe(2);
  expect((await serveDealsLink(new Request("https://deals.audit.invalid/r/nope"),state.env)).status).toBe(404);
  const stats=await(await call("/stats?days=7")).json() as {days:{clicks:number;posts:number}[];top:{title:string;clicks:number}[]};
  expect(stats.days).toHaveLength(7);expect(stats.days.at(-1)).toMatchObject({clicks:2,posts:2});expect(stats.top[0]).toMatchObject({clicks:2});
 });
 it("stops a channel Discord no longer accepts and fails its posts",async()=>{
  const channel=await addDiscord(),now=Date.now(),o=await upsertOffer(db.asD1(),offer(),now);
  await enqueue(db.asD1(),{owner:"alice",offerId:o.id,channels:[channel],link:"https://x.invalid",affiliate:false,huntId:null,sendAt:now,now});
  await dispatchDeals(state.env,net({"discord.com":()=>new Response(null,{status:404})}).send,now);
  expect(db.sqlite.prepare("SELECT state,error FROM deals_post").get()).toEqual({state:"failed",error:"DESTINATION_DISABLED"});
  expect(db.sqlite.prepare("SELECT active,last_error AS e FROM deals_channel").get()).toEqual({active:0,e:"DESTINATION_DISABLED"});
 });
 it("sends a post scheduled days ahead when its time comes, and expires it only two days after",async()=>{
  const channel=await addDiscord(),now=Date.parse("2026-11-04T12:00:00Z"),at=now+3*86_400_000;
  vi.useFakeTimers({toFake:["Date"]});vi.setSystemTime(now);
  vi.stubGlobal("fetch",net({"www.amazon.com.br":()=>html("<title>Fone</title>")}).send);
  expect((await call("/offers",{method:"POST",body:{url:"https://www.amazon.com.br/dp/B0ABCDEF12",channelIds:[channel],sendAt:new Date(at).toISOString()}})).status).toBe(201);
  expect(db.sqlite.prepare("SELECT day,next_at AS nextAt FROM deals_post").get()).toEqual({day:"2026-11-07",nextAt:at});
  const sent:unknown[]=[],discord=net({"discord.com":async req=>{sent.push(await req.json());return Response.json({id:"1"});}}).send;
  vi.setSystemTime(at-60_000);await dispatchDeals(state.env,discord,at-60_000);expect(sent).toHaveLength(0);
  await dealsCleanup(state.env,90,at);expect(db.sqlite.prepare("SELECT state FROM deals_post").get()?.state).toBe("queued");
  vi.setSystemTime(at);await dispatchDeals(state.env,discord,at);expect(sent).toHaveLength(1);
  expect(db.sqlite.prepare("SELECT state FROM deals_post").get()?.state).toBe("sent");
  // Another one, due the same day, that could not go for two days.
  const o=await upsertOffer(db.asD1(),offer({key:"late"}),at);await enqueue(db.asD1(),{owner:"alice",offerId:o.id,channels:[channel],link:"https://x.invalid",affiliate:false,huntId:null,sendAt:at,now:at});
  const late=at+2*86_400_000;vi.setSystemTime(late);await dispatchDeals(state.env,discord,late);expect(sent).toHaveLength(2);
  db.sqlite.exec("UPDATE deals_post SET state='queued',next_at=0,claim_until=0 WHERE offer_id='"+o.id+"'");
  const expired=at+3*86_400_000;vi.setSystemTime(expired);await dispatchDeals(state.env,discord,expired);
  expect(db.sqlite.prepare("SELECT state,error FROM deals_post WHERE offer_id=?").get(o.id)).toEqual({state:"failed",error:"EXPIRED"});expect(sent).toHaveLength(2);
  vi.useRealTimers();
 });
 it("does not send for an account that lost the plan",async()=>{
  const channel=await addDiscord(),now=Date.now(),o=await upsertOffer(db.asD1(),offer(),now);
  await enqueue(db.asD1(),{owner:"alice",offerId:o.id,channels:[channel],link:"https://x.invalid",affiliate:false,huntId:null,sendAt:now,now});
  db.sqlite.exec("UPDATE subscription SET status='canceled' WHERE reference_id='alice'");
  const {send,calls}=net({});await dispatchDeals(state.env,send,now);
  expect(calls).toHaveLength(0);expect(db.sqlite.prepare("SELECT state,error FROM deals_post").get()).toEqual({state:"failed",error:"PLAN"});
 });
});

describe("account deletion",()=>{
 it("stops the short links at deletion and purges the account's Deals data, keeping the shared catalog",async()=>{
  Object.assign(state.env,{NOTES:new TestBucket().asR2(),SITES:new TestBucket().asR2(),SYNC_LIVE:{getByName:()=>({revoke:async()=>{}})}});
  const now=Date.now(),mine=await addDiscord("alice"),theirs=await addDiscord("bob");
  await call("/affiliates/shopee",{method:"PUT",body:{appId:"123456",secret:"S3cretS3cretS3cretS3cret"}});
  await call("/telegram/link",{method:"POST",body:{}});
  db.sqlite.exec(`INSERT INTO deals_hunt(id,user_id,name,kind,source_id,channels,created_at) VALUES('ha','alice','J','games_free','gamerpower','["${mine}"]',0),('hb','bob','J','games_free','gamerpower','["${theirs}"]',0)`);
  const solo=await upsertOffer(db.asD1(),offer({key:"solo",title:"Só da Alice"}),now),shared=await upsertOffer(db.asD1(),offer({key:"shared",title:"Dos dois"}),now),game=await upsertOffer(db.asD1(),offer({key:"g",kind:"game_free",title:"Jogo"}),now);
  for(const o of [solo,shared,game])await enqueue(db.asD1(),{owner:"alice",offerId:o.id,channels:[mine],link:"https://www.amazon.com.br/dp/B000000001?tag=a-20",affiliate:true,huntId:null,sendAt:now,now});
  for(const o of [shared,game])await enqueue(db.asD1(),{owner:"bob",offerId:o.id,channels:[theirs],link:"https://www.amazon.com.br/dp/B000000001?tag=b-20",affiliate:true,huntId:null,sendAt:now,now});
  const code=(owner:string,id:string)=>String(db.sqlite.prepare("SELECT code FROM deals_post WHERE user_id=? AND offer_id=?").get(owner,id)?.code);
  const go=(c:string)=>serveDealsLink(new Request(`https://deals.audit.invalid/r/${c}`,{headers:{"user-agent":"Mozilla/5.0"}}),state.env);
  expect((await go(code("alice",solo.id))).status).toBe(302);
  await softDelete(state.env,"alice",now);
  expect((await go(code("alice",solo.id))).status).toBe(404);expect((await go(code("bob",shared.id))).status).toBe(302);
  db.sqlite.exec("UPDATE resource_quota SET billing_cancel_pending=0 WHERE user_id='alice'");
  await purgeAccounts(state.env,now+31*LIFE_DAY);
  for(const table of ["deals_affiliate","deals_channel","deals_hunt","deals_post","deals_click_day","deals_telegram_link"])expect(db.sqlite.prepare(`SELECT count(*) AS n FROM ${table} WHERE user_id='alice'`).get()?.n,table).toBe(0);
  expect(db.sqlite.prepare("SELECT title FROM deals_offer ORDER BY title").all().map(r=>r.title)).toEqual(["Dos dois","Jogo"]);
  expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_price WHERE offer_id=?").get(solo.id)?.n).toBe(0);
  expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_post WHERE user_id='bob'").get()?.n).toBe(2);
  expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_channel WHERE user_id='bob'").get()?.n).toBe(1);
  expect((await go(code("bob",shared.id))).status).toBe(302);
 });
 it("stops a banned account's short links",async()=>{
  const channel=await addDiscord(),now=Date.now(),o=await upsertOffer(db.asD1(),offer(),now);
  await enqueue(db.asD1(),{owner:"alice",offerId:o.id,channels:[channel],link:"https://www.amazon.com.br/dp/B000000001",affiliate:false,huntId:null,sendAt:now,now});
  const c=String(db.sqlite.prepare("SELECT code FROM deals_post").get()?.code);
  db.sqlite.exec("UPDATE user SET banned=1 WHERE id='alice'");
  expect((await serveDealsLink(new Request(`https://deals.audit.invalid/r/${c}`),state.env)).status).toBe(404);
  expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_click_day").get()?.n).toBe(0);
 });
});

describe("losing the Deals plan",()=>{
 it("reads for 14 days without sending, comes back with the plan, then purges only the private data",async()=>{
  const t0=Date.parse("2026-11-04T12:00:00Z");vi.useFakeTimers({toFake:["Date"]});vi.setSystemTime(t0);
  const mine=await addDiscord("alice"),theirs=await addDiscord("bob");
  await call("/affiliates/amazon",{method:"PUT",body:{tag:"minhatag-20"}});
  const shared=await upsertOffer(db.asD1(),offer({key:"shared",title:"Dos dois"}),t0),solo=await upsertOffer(db.asD1(),offer({key:"solo",title:"Só da Alice"}),t0);
  for(const o of [shared,solo])await enqueue(db.asD1(),{owner:"alice",offerId:o.id,channels:[mine],link:"https://x.invalid",affiliate:false,huntId:null,sendAt:t0+3_600_000,now:t0});
  await enqueue(db.asD1(),{owner:"bob",offerId:shared.id,channels:[theirs],link:"https://x.invalid",affiliate:false,huntId:null,sendAt:t0,now:t0});
  // The paid period ends: Stripe cancels the subscription.
  db.sqlite.exec("UPDATE subscription SET status='canceled' WHERE reference_id='alice'");
  const usage=await(await call("/usage")).json() as {plan:null;readOnly:boolean;readOnlyUntil:string};
  expect(usage).toMatchObject({plan:null,readOnly:true,readOnlyUntil:new Date(t0+14*LIFE_DAY).toISOString()});
  expect(db.sqlite.prepare("SELECT state,error FROM deals_post WHERE user_id='alice'").all()).toEqual([{state:"failed",error:"PLAN"},{state:"failed",error:"PLAN"}]);
  expect((await call("/channels")).status).toBe(200);expect((await call("/channels",{method:"POST",body:{type:"discord",webhook:HOOK}})).status).toBe(403);
  const {send,calls}=net({"discord.com":()=>Response.json({id:"1"})});await dispatchDeals(state.env,send,t0+4_000_000);
  expect(calls.length).toBe(1);expect(db.sqlite.prepare("SELECT state FROM deals_post WHERE user_id='bob'").get()?.state).toBe("sent");
  // Back on the plan inside the window: nothing is lost.
  db.sqlite.exec("UPDATE subscription SET status='active' WHERE reference_id='alice'");
  expect(await dealsAccess(state.env,"alice",t0+LIFE_DAY)).toMatchObject({readOnly:false,readOnlyUntil:null});
  await dealsLifecycle(state.env,t0+20*LIFE_DAY);expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_channel WHERE user_id='alice'").get()?.n).toBe(1);
  // Lost again, and never opens the panel: the daily run starts the window and purges after it.
  db.sqlite.exec("UPDATE subscription SET status='canceled' WHERE reference_id='alice'");
  const t1=t0+21*LIFE_DAY;await dealsLifecycle(state.env,t1);
  expect(db.sqlite.prepare("SELECT read_only_until AS u FROM deals_account WHERE user_id='alice'").get()?.u).toBe(t1+14*LIFE_DAY);
  await dealsLifecycle(state.env,t1+13*LIFE_DAY);expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_affiliate WHERE user_id='alice'").get()?.n).toBe(1);
  await dealsLifecycle(state.env,t1+14*LIFE_DAY);await dealsLifecycle(state.env,t1+14*LIFE_DAY);
  for(const table of ["deals_affiliate","deals_channel","deals_hunt","deals_post","deals_click_day","deals_telegram_link","deals_account"])expect(db.sqlite.prepare(`SELECT count(*) AS n FROM ${table} WHERE user_id='alice'`).get()?.n,table).toBe(0);
  expect(db.sqlite.prepare("SELECT title FROM deals_offer ORDER BY title").all().map(r=>r.title)).toEqual(["Dos dois"]);
  expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_channel WHERE user_id='bob'").get()?.n).toBe(1);
  // The user and the Cloud account stay: only Deals went.
  expect(db.sqlite.prepare("SELECT count(*) AS n FROM user WHERE id='alice'").get()?.n).toBe(1);
  vi.useRealTimers();
 });
 it("leaves alone accounts that never had Deals and the owner's gift",async()=>{
  Object.assign(state.env,{STAFF_OWNER_EMAIL:"plain@audit.invalid"});
  await addDiscord("plain");await dealsAccess(state.env,"plain");
  await dealsLifecycle(state.env,Date.now()+60*LIFE_DAY);
  expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_channel WHERE user_id='plain'").get()?.n).toBe(1);
  expect(db.sqlite.prepare("SELECT read_only_until AS u FROM deals_account WHERE user_id='plain'").get()?.u).toBeNull();
 });
});

describe("the bot's sending budget",()=>{
 async function telegramChannels(n:number){
  const ids:string[]=[];
  for(let i=0;i<n;i++){const id=crypto.randomUUID();db.sqlite.prepare("INSERT INTO deals_channel(id,user_id,platform,name,target,target_hash,created_at) VALUES(?,?,?,?,?,?,0)").run(id,"alice","telegram",`Canal ${i}`,await seal(state.env,"alice",id,"telegram",`-100${i+1}`),`h${i}`);ids.push(id);}
  for(const [i,id] of ids.entries()){const o=await upsertOffer(db.asD1(),offer({key:`b${i}`}),Date.now());await enqueue(db.asD1(),{owner:"alice",offerId:o.id,channels:[id],link:"https://x.invalid",affiliate:false,huntId:null,sendAt:Date.now(),now:Date.now()});}
  db.sqlite.exec("UPDATE deals_post SET next_at=0");
  return ids;
 }
 const budget=()=>Number(db.sqlite.prepare("SELECT next_at AS n FROM pulse_delivery_budget WHERE key=?").get(DEALS_BUDGET.telegram.key)?.n);
 it("keeps Telegram messages at least 40 ms apart across channels",async()=>{
  await telegramChannels(3);const at:number[]=[];
  const {send}=net({"api.telegram.org":()=>{at.push(performance.now());return Response.json({ok:true,result:{message_id:1}});}});
  const now=Date.now();await dispatchDeals(state.env,send,now);
  expect(at).toHaveLength(3);for(let i=1;i<at.length;i++)expect(at[i]-at[i-1]).toBeGreaterThanOrEqual(35);
  expect(budget()).toBeGreaterThanOrEqual(now+120);
 });
 it("still sends every channel in the run when the timer wakes a little early",async()=>{
  await telegramChannels(4);let calls=0;
  const real=globalThis.setTimeout;vi.stubGlobal("setTimeout",((fn:()=>void,ms?:number)=>real(fn,Math.max(0,(ms??0)-1))) as unknown as typeof setTimeout);
  const {send}=net({"api.telegram.org":()=>{calls++;return Response.json({ok:true,result:{message_id:1}});}});
  const now=Date.now();await dispatchDeals(state.env,send,now);
  expect(calls).toBe(4);expect(budget()).toBeGreaterThanOrEqual(now+160);
 });
 it("runs on the clock it is given, whatever the real time",async()=>{
  await telegramChannels(2);let calls=0;
  const {send}=net({"api.telegram.org":()=>{calls++;return Response.json({ok:true,result:{message_id:1}});}});
  // An hour behind the real clock (and far ahead, below): the 45 s cut and
  // the send times follow the run's own clock.
  const behind=Date.now()-3_600_000;await dispatchDeals(state.env,send,behind);expect(calls).toBe(2);
  const sent=db.sqlite.prepare("SELECT sent_at AS at FROM deals_post WHERE state='sent'").all().map(r=>Number(r.at));
  for(const at of sent){expect(at).toBeGreaterThanOrEqual(behind);expect(at).toBeLessThan(behind+45_000);}
  db.sqlite.exec("UPDATE deals_post SET state='queued',next_at=0,claim_until=0");db.sqlite.exec("UPDATE deals_channel SET next_at=0");db.sqlite.exec("UPDATE pulse_delivery_budget SET next_at=0");
  await dispatchDeals(state.env,send,Date.now()+3_600_000);expect(calls).toBe(4);
 });
 it("pauses every Telegram channel after a global 429 until Retry-After",async()=>{
  await telegramChannels(3);let calls=0,limited=true;
  const {send}=net({"api.telegram.org":()=>{calls++;if(limited){limited=false;return Response.json({ok:false,error_code:429,parameters:{retry_after:30}},{status:429});}return Response.json({ok:true,result:{message_id:1}});}});
  const t=Date.now();
  await dispatchDeals(state.env,send,t);
  expect(calls).toBe(1);expect(budget()).toBeGreaterThanOrEqual(t+30_000);
  expect(db.sqlite.prepare("SELECT state,error FROM deals_post WHERE error IS NOT NULL").all()).toEqual([{state:"queued",error:"RATE_LIMIT"}]);
  await dispatchDeals(state.env,send,t+10_000);expect(calls).toBe(1);
  await dispatchDeals(state.env,send,t+31_000);expect(calls).toBe(3);
  expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_post WHERE state='sent'").get()?.n).toBe(2);
  // Discord has its own budget: a Telegram pause does not hold it.
  expect(db.sqlite.prepare("SELECT count(*) AS n FROM pulse_delivery_budget WHERE key=?").get(DEALS_BUDGET.discord.key)?.n).toBe(0);
 });
});

describe("a new TikTok video",()=>{
 const oembed=(id:string)=>()=>Response.json({version:"1.0",type:"video",provider_name:"TikTok",embed_product_id:id,embed_type:"video"});
 it("reads the video by its address and drops what the app adds",()=>{
  expect(videoOf(new URL("https://m.tiktok.com/@loja.da_ana/video/7430000000000000001?is_from_webapp=1&sender_device=pc"))).toEqual({id:"7430000000000000001",author:"loja.da_ana",url:"https://www.tiktok.com/@loja.da_ana/video/7430000000000000001"});
  for(const raw of ["https://www.tiktok.com/@ana/photo/7430000000000000001","https://tiktok.com.evil.invalid/@ana/video/7430000000000000001","https://www.tiktok.com/@ana"])expect(videoOf(new URL(raw))).toBeNull();
 });
 it("follows a short link, checks the video and queues the news to the channels",async()=>{
  const channel=await addDiscord();
  vi.stubGlobal("fetch",net({"vm.tiktok.com":()=>new Response(null,{status:301,headers:{location:"https://www.tiktok.com/@ana/video/7430000000000000001?_r=1"}}),"www.tiktok.com/oembed":oembed("7430000000000000001")}).send);
  expect(await(await call("/offers/video/preview",{method:"POST",body:{url:"https://vm.tiktok.com/ZMabc123/"}})).json()).toEqual({id:"7430000000000000001",author:"ana",url:"https://www.tiktok.com/@ana/video/7430000000000000001"});
  const created=await call("/offers/video",{method:"POST",body:{url:"https://vm.tiktok.com/ZMabc123/",title:"Fone sem fio",price:8990,channelIds:[channel]}});expect(created.status).toBe(201);
  expect((await call("/offers/video",{method:"POST",body:{url:"https://www.tiktok.com/@ana/video/7430000000000000001",title:"Outro",channelIds:[channel]}})).status).toBe(409);
  expect(db.sqlite.prepare("SELECT o.store,o.kind,o.title,o.price,o.currency,p.link,p.affiliate FROM deals_post p JOIN deals_offer o ON o.id=p.offer_id").get()).toEqual({store:"tiktok",kind:"video",title:"Fone sem fio",price:8990,currency:"BRL",link:"https://www.tiktok.com/@ana/video/7430000000000000001",affiliate:1});
  const sent:{embeds:{title:string;description:string}[]}[]=[];
  await dispatchDeals(state.env,net({"discord.com":async req=>{sent.push(await req.json());return Response.json({id:"1"});}}).send,Date.now());
  expect(sent[0].embeds[0].title).toBe("Vídeo novo: Fone sem fio");expect(sent[0].embeds[0].description).toContain("Ver o vídeo");expect(sent[0].embeds[0].description).toContain("#publi");expect(sent[0].embeds[0].description).not.toContain("link de afiliado");
 });
 it("refuses what is not a public TikTok video",async()=>{
  vi.stubGlobal("fetch",net({"www.tiktok.com/oembed":()=>new Response("{}",{status:400})}).send);
  expect(await(await call("/offers/video/preview",{method:"POST",body:{url:"https://www.amazon.com.br/dp/B0ABCDEF12"}})).json()).toEqual({code:"NOT_A_VIDEO"});
  const missing=await call("/offers/video/preview",{method:"POST",body:{url:"https://www.tiktok.com/@ana/video/7430000000000000009"}});expect(missing.status).toBe(404);expect(await missing.json()).toEqual({code:"VIDEO_NOT_FOUND"});
  expect((await call("/offers/video",{method:"POST",user:"plain",body:{url:"https://www.tiktok.com/@ana/video/7430000000000000001",title:"X",channelIds:["c"]}})).status).toBe(403);
 });
 it("shows the video above the text on Telegram",async()=>{
  const bodies:Record<string,unknown>[]=[];
  const send=vi.fn(async(_url:RequestInfo|URL,init?:RequestInit)=>{bodies.push(JSON.parse(String(init?.body)));return Response.json({ok:true,result:{message_id:1}});}) as unknown as typeof fetch;
  const post={title:"Fone",image:null,price:null,originalPrice:null,currency:null,discount:null,link:"https://deals.audit.invalid/r/ABCDEFGH",affiliate:true,kind:"video",credit:null};
  const text=telegramText(post,"pt");expect(text).toBe('<b>Vídeo novo: Fone</b>\n<a href="https://deals.audit.invalid/r/ABCDEFGH">Ver o vídeo</a>\n#publi');
  expect((await sendTelegram(state.env,"-1001",text,null,send,"https://www.tiktok.com/@ana/video/7430000000000000001")).sent).toBe(true);
  expect(bodies[0]).toMatchObject({link_preview_options:{url:"https://www.tiktok.com/@ana/video/7430000000000000001",prefer_large_media:true,show_above_text:true}});
 });
});

describe("Nuuvem's sales through Rakuten",()=>{
 const item=(sku:string,name:string,price:string,sale:string,slug=sku)=>`<item><mid>46796</mid><merchantname>Nuuvem</merchantname><linkid>46796${sku}</linkid><sku>${sku}</sku><productname>${name}</productname><price currency="BRL">${price}</price><saleprice currency="BRL">${sale}</saleprice><description><short>x</short></description><linkurl>https://click.linksynergy.com/link?id=BLUP&amp;offerid=1.2&amp;type=15&amp;murl=https%3A%2F%2Fwww.nuuvem.com%2Fbr-pt%2Fitem%2F${slug}</linkurl><imageurl>https://assets.nuuvem.com/i/${sku}.jpg</imageurl></item>`;
 const page=(items:string[],pages=1,n=1)=>`<result><TotalMatches>${items.length}</TotalMatches><TotalPages>${pages}</TotalPages><PageNumber>${n}</PageNumber>${items.join("")}</result>`;
 it("takes only the games on sale, with the store page and the list price before",()=>{
  const offers=nuuvemOffers(page([item("1","Ambulance Life","109.99","32.99","ambulance-life"),item("2","Pine Hearts","38.99","0"),item("3","Caro &amp; Bom","10","12"),`<item><mid>999</mid><sku>4</sku><productname>Outra loja</productname><price currency="BRL">10</price><saleprice currency="BRL">5</saleprice></item>`]),0);
  expect(offers).toEqual([expect.objectContaining({key:"nuuvem:1:3299",store:"nuuvem",title:"Ambulance Life",url:"https://www.nuuvem.com/br-pt/item/ambulance-life",imageUrl:"https://assets.nuuvem.com/i/1.jpg",price:3299,originalPrice:10999,currency:"BRL",kind:"nuuvem_deal",discount:70})]);
  expect(isCredit("EA SPORTS FC Mobile - 40 FC POINTS")).toBe(true);expect(isCredit("AFK Journey - 1554 Dragon Crystals")).toBe(true);expect(isCredit("Ambulance Life: A Paramedic Simulator")).toBe(false);
 });
 it("reads the catalog a few pages a minute and posts each new sale with the person's own ID",async()=>{
  Object.assign(state.env,{DEALS_RAKUTEN_CLIENT_ID:"client-id-123",DEALS_RAKUTEN_CLIENT_SECRET:"client-secret-123",DEALS_RAKUTEN_SID:"1234567"});
  const channel=await addDiscord();await call("/affiliates/nuuvem",{method:"PUT",body:{siteId:"*AbCdEfGh1"}});
  let sales:string[]=[item("1","Ambulance Life","109.99","32.99")];const pages=NUUVEM_PAGES_PER_RUN+2,reads:number[]=[];
  const {send}=net({"www.gamerpower.com":()=>Response.json([]),
   "api.linksynergy.com/token":req=>{expect(req.headers.get("authorization")).toBe(`Bearer ${btoa("client-id-123:client-secret-123")}`);return Response.json({token_type:"bearer",expires_in:3600,access_token:"a".repeat(32),refresh_token:"r".repeat(32)});},
   "api.linksynergy.com/productsearch/1.0":req=>{const n=Number(new URL(req.url).searchParams.get("pagenumber"));reads.push(n);return new Response(page(n===pages?sales:[],pages,n),{headers:{"content-type":"application/xml"}});}});
  const t0=Date.parse("2026-11-04T10:00:00Z");
  // The first full read only fills the catalog.
  await collectDeals(state.env,send,t0);expect(reads).toEqual(Array.from({length:NUUVEM_PAGES_PER_RUN},(_,i)=>i+1));
  await collectDeals(state.env,send,t0+30_000);expect(reads).toHaveLength(NUUVEM_PAGES_PER_RUN);
  await collectDeals(state.env,send,t0+60_000);expect(reads).toHaveLength(pages);
  expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_offer WHERE kind='nuuvem_deal'").get()?.n).toBe(1);
  expect((await call("/hunts",{method:"POST",user:"bob",body:{kind:"nuuvem_deals",channelIds:[await addDiscord("bob")],filters:{}}})).status).toBe(409);
  db.sqlite.exec(`INSERT INTO deals_hunt(id,user_id,name,kind,source_id,filters,channels,created_at) VALUES('n1','alice','N','nuuvem_deals','nuuvem','{"minDiscount":50}','["${channel}"]',0)`);
  sales=[item("1","Ambulance Life","109.99","32.99"),item("5","Hades","73.99","36.99","hades"),item("6","Jogo - 500 Coins","20","5")];
  // The next round starts 3 hours after the first one began.
  await collectDeals(state.env,send,t0+2*60*60_000);expect(reads).toHaveLength(pages);
  for(const at of [3,3.01,3.02])await collectDeals(state.env,send,t0+at*60*60_000);
  expect(db.sqlite.prepare("SELECT o.title,p.link,p.affiliate FROM deals_post p JOIN deals_offer o ON o.id=p.offer_id").all()).toEqual([{title:"Hades",link:"https://click.linksynergy.com/deeplink?id=*AbCdEfGh1&mid=46796&murl=https%3A%2F%2Fwww.nuuvem.com%2Fbr-pt%2Fitem%2Fhades",affiliate:1}]);
 });
});

describe("Nuuvem in other countries",()=>{
 it("links each country's page and gives the price only in Brazil",async()=>{
  expect(nuuvemPage("https://www.nuuvem.com/br-pt/item/hades","mx")).toBe("https://www.nuuvem.com/mx-es/item/hades");
  expect(nuuvemPage("https://www.nuuvem.com/br-pt/item/hades","us")).toBe("https://www.nuuvem.com/us-en/item/hades");
  const channel=await addDiscord();await call("/affiliates/nuuvem",{method:"PUT",body:{siteId:"*AbCdEfGh1"}});
  expect((await call("/hunts",{method:"POST",body:{kind:"nuuvem_deals",channelIds:[channel],filters:{country:"mx",maxPrice:5000}}})).status).toBe(400);
  expect((await call("/hunts",{method:"POST",body:{kind:"games_free",channelIds:[channel],filters:{country:"mx"}}})).status).toBe(400);
  const now=Date.now();
  await upsertOffer(db.asD1(),offer({key:"n:1",store:"nuuvem",title:"Hades",url:"https://www.nuuvem.com/br-pt/item/hades",price:3699,originalPrice:7399,kind:"nuuvem_deal",discount:50}),now);
  expect((await call("/hunts",{method:"POST",body:{kind:"nuuvem_deals",channelIds:[channel],filters:{country:"mx",minDiscount:40}}})).status).toBe(201);
  expect(db.sqlite.prepare("SELECT link,price_shown AS shown FROM deals_post").get()).toEqual({link:"https://click.linksynergy.com/deeplink?id=*AbCdEfGh1&mid=46796&murl=https%3A%2F%2Fwww.nuuvem.com%2Fmx-es%2Fitem%2Fhades",shown:0});
  const sent:{embeds:{description:string}[]}[]=[];
  await dispatchDeals(state.env,net({"discord.com":async req=>{sent.push(await req.json());return Response.json({id:"1"});}}).send,Date.now()+1000);
  expect(sent[0].embeds[0].description.split("\n")[0]).toBe("50% de desconto");expect(sent[0].embeds[0].description).not.toContain("R$");
 });
});

describe("Magalu and Terabyte",()=>{
 it("puts the person's promoter_id and showcase on Magalu and their p= on Terabyte",async()=>{
  expect(storeOf(new URL("https://www.magazineluiza.com.br/x/p/237849600/te/smxx/"))).toBe("magalu");expect(storeOf(new URL("https://especiais.magazineluiza.com.br/black"))).toBe("magalu");expect(storeOf(new URL("https://www.magazinevoce.com.br/magazineoutra/x/p/1/"))).toBe("magalu");expect(storeOf(new URL("https://www.terabyteshop.com.br/produto/1/x"))).toBe("terabyte");
  expect(magaluLink(new URL("https://www.magazineluiza.com.br/x/p/237849600/te/smxx/?promoter_id=999&utm_source=a"),"12345","magazineminhaloja")).toBe("https://www.magazineluiza.com.br/x/p/237849600/te/smxx/?promoter_id=12345&showcase=magazineminhaloja");
  expect(magaluLink(new URL("https://www.magazinevoce.com.br/magazineoutra/x/p/237849600/?promoter_id=999"),"12345","magazineminhaloja")).toBe("https://www.magazinevoce.com.br/magazineminhaloja/x/p/237849600/?promoter_id=12345&showcase=magazineminhaloja");
  expect(magaluLink(new URL("https://www.magazineluiza.com.br/x/p/1/"),"12345")).toBe("https://www.magazineluiza.com.br/x/p/1/?promoter_id=12345");
  expect(terabyteLink(new URL("https://www.terabyteshop.com.br/produto/123/placa?p=outro&utm_medium=x#a"),"meucodigo","afiliado","blup")).toBe("https://www.terabyteshop.com.br/produto/123/placa?p=meucodigo&utm_medium=afiliado&utm_campaign=blup");
  expect((await call("/affiliates/magalu",{method:"PUT",body:{promoterId:"12345",showcase:"magazineminhaloja"}})).status).toBe(200);
  expect((await call("/affiliates/magalu",{method:"PUT",body:{promoterId:"12345",showcase:"outraloja"}})).status).toBe(400);
  expect((await call("/affiliates/terabyte",{method:"PUT",body:{partner:"meucodigo"}})).status).toBe(200);
  const listed=await(await call("/affiliates")).json() as {stores:{store:string;configured:boolean;hint:string|null}[]};
  expect(listed.stores.filter(s=>s.configured).map(s=>[s.store,s.hint])).toEqual([["magalu","magazineminhaloja"],["terabyte","…digo"]]);
 });
});

describe("mirroring a public Telegram channel",()=>{
 const post=(id:number,text:string,link:string,at=new Date().toISOString())=>`<div class="tgme_widget_message_wrap js-widget_message_wrap"><div class="tgme_widget_message js-widget_message" data-post="promos/${id}"><a class="tgme_widget_message_photo_wrap 1" href="https://t.me/promos/${id}" style="width:800px;background-image:url('https://cdn4.telesco.pe/file/p${id}.jpg')"></a><div class="tgme_widget_message_text js-message_text" dir="auto">${text}<br/><a href="${link}" target="_blank" rel="noopener" onclick="return confirm('Open this link?\n\n'+this.href);">${link}</a></div><span class="tgme_widget_message_meta"><a class="tgme_widget_message_date" href="https://t.me/promos/${id}"><time datetime="${at}" class="time">10:00</time></a></span></div></div>`;
 const page=(...posts:string[])=>`<html><head><meta property="og:title" content="Promos &amp; Cia"></head><body><div class="tgme_channel_info"></div>${posts.join("")}</body></html>`;
 it("reads posts, prices and titles from the public preview",()=>{
  expect(channelName.parse("https://t.me/s/promos_br")).toBe("promos_br");expect(channelName.parse("@promos_br")).toBe("promos_br");expect(channelName.safeParse("https://t.me/+abcdef").success).toBe(false);
  const read=readChannel(page(post(7,"🔥 <b>Fone Bluetooth</b><br/>de R$ 199,90 por R$ 89,90","https://amzn.to/abc")),"promos");
  expect(read).toMatchObject({title:"Promos & Cia",posts:[{id:7,links:["https://amzn.to/abc"],photo:"https://cdn4.telesco.pe/file/p7.jpg"}]});
  expect(textPrices(read.posts[0].text)).toEqual({price:8990,before:19990});expect(textTitle(read.posts[0].text)).toBe("Fone Bluetooth");
 });
 it("posts each new product once with the person's own link, again only when cheaper",async()=>{
  const channel=await addDiscord();await call("/affiliates/amazon",{method:"PUT",body:{tag:"minhatag-20"}});
  const bobChannel=await addDiscord("bob");
  let html=page(post(5,"Velho","https://amzn.to/old"));
  const store=(price:string)=>()=>html_(`<meta property="og:title" content="Fone Bom"><meta property="product:price:amount" content="${price}"><meta property="product:price:currency" content="BRL">`);
  const html_=(b:string)=>new Response(b,{headers:{"content-type":"text/html; charset=utf-8"}});
  let amazonPrice="89.90";
  const {send}=net({"t.me/s/promos":()=>html_(html),"amzn.to":()=>new Response(null,{status:301,headers:{location:"https://www.amazon.com.br/Fone/dp/B0ABCDEF12?tag=dono-20"}}),"www.amazon.com.br":()=>store(amazonPrice)()});
  vi.stubGlobal("fetch",send);
  expect(await(await call("/hunts/mirror/check",{method:"POST",body:{channel:"https://t.me/promos"}})).json()).toEqual({name:"promos",title:"Promos & Cia",posts:1});
  expect((await call("/hunts",{method:"POST",body:{kind:"mirror",channel:"@promos",channelIds:[channel],filters:{}}})).status).toBe(201);
  expect((await call("/hunts",{method:"POST",body:{kind:"mirror",channel:"promos",channelIds:[channel],filters:{}}})).status).toBe(409);
  expect((await call("/hunts",{method:"POST",body:{kind:"mirror",channel:"promos",channelIds:[channel],filters:{platforms:["pc"]}}})).status).toBe(400);
  expect((await call("/hunts",{method:"POST",user:"bob",body:{kind:"mirror",channel:"promos",channelIds:[bobChannel],filters:{}}})).status).toBe(201);
  const t0=Date.now();
  await collectMirrors(state.env,send,t0);expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_post").get()?.n).toBe(0);
  html=page(post(5,"Velho","https://amzn.to/old"),post(6,"Fone Bom<br/>por R$ 89,90","https://amzn.to/abc"));
  await collectMirrors(state.env,send,t0+5*60_000);expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_post").get()?.n).toBe(0);
  await collectMirrors(state.env,send,t0+11*60_000);
  // Bob has no Amazon tag: nothing for him.
  expect(db.sqlite.prepare("SELECT p.user_id AS u,p.link,o.title,o.price FROM deals_post p JOIN deals_offer o ON o.id=p.offer_id").all()).toEqual([{u:"alice",link:"https://www.amazon.com.br/dp/B0ABCDEF12?tag=minhatag-20",title:"Fone Bom",price:8990}]);
  html=page(post(7,"Fone Bom de novo","https://amzn.to/abc"));
  await collectMirrors(state.env,send,t0+22*60_000);expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_post").get()?.n).toBe(1);
  amazonPrice="79.90";html=page(post(8,"Fone Bom mais barato","https://amzn.to/abc"));
  await collectMirrors(state.env,send,t0+33*60_000);expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_post").get()?.n).toBe(2);
  const listed=await(await call("/hunts")).json() as {hunts:{kind:string;mirror:{channel:string;title:string|null;error:string|null}|null}[]};
  expect(listed.hunts).toEqual([expect.objectContaining({kind:"mirror",mirror:{channel:"promos",title:"Promos & Cia",error:null}})]);
 });
 it("pauses a channel that is gone or refuses, with the reason",async()=>{
  const channel=await addDiscord();
  vi.stubGlobal("fetch",net({"t.me/s/promos":()=>new Response("<html><div class=\"tgme_channel_info\"></div></html>",{headers:{"content-type":"text/html"}})}).send);
  await call("/hunts",{method:"POST",body:{kind:"mirror",channel:"promos",channelIds:[channel],filters:{}}});
  await collectMirrors(state.env,net({"t.me/s/promos":()=>new Response("slow down",{status:429})}).send,Date.now()+60_000);
  expect(db.sqlite.prepare("SELECT error,failures FROM deals_source WHERE id='telegram:promos'").get()).toEqual({error:"RATE_LIMIT",failures:1});
  expect((await call("/hunts/mirror/check",{method:"POST",body:{channel:"nao_existe"}})).status).toBe(502);
 });
});

describe("store hunts on Shopee and AliExpress",()=>{
 const node=(id:number,price:string,rate:number|null,over:Record<string,unknown>={})=>({itemId:id,productName:`Produto ${id}`,imageUrl:`https://down-br.img.susercontent.com/file/${id}`,productLink:`https://shopee.com.br/product/1/${id}?sp_atk=x`,offerLink:`https://s.shopee.com.br/off${id}`,priceMin:price,priceDiscountRate:rate,...over});
 const shopee=(...nodes:unknown[])=>({data:{productOfferV2:{nodes}}});
 it("takes from Shopee only what is on sale, with its own links",()=>{
  const found=shopeeOffers(shopee(node(1,"89.90",30),node(2,"50.00",0),node(3,"10.00",20,{offerLink:"https://evil.invalid/x"}),node(4,"5.00",10,{imageUrl:"https://evil.invalid/i.png"})),1000);
  expect(found.map(f=>[f.offer.externalId,f.link,f.offer.price,f.offer.discount,f.offer.originalPrice,f.offer.imageUrl])).toEqual([["1","https://s.shopee.com.br/off1",8990,30,null,"https://down-br.img.susercontent.com/file/1"],["4","https://s.shopee.com.br/off4",500,10,null,null]]);
  expect(found[0].offer).toMatchObject({store:"shopee",kind:"store_deal",url:"https://shopee.com.br/product/1/1",currency:"BRL",key:"shopee:br:1:8990"});
  expect(()=>shopeeOffers({errors:[{message:"Invalid Signature"}]})).toThrow(ShopRefused);
 });
 it("takes from AliExpress the reais price, the price before and the tracked link",()=>{
  const product=(id:number,over:Record<string,unknown>={})=>({product_id:id,product_title:`Item ${id}`,product_main_image_url:`https://ae01.alicdn.com/kf/${id}.jpg`,promotion_link:`https://s.click.aliexpress.com/e/_${id}`,target_sale_price:"45.90",target_sale_price_currency:"BRL",target_original_price:"91.80",discount:"50%",...over});
  const found=aliexpressOffers({aliexpress_affiliate_product_query_response:{resp_result:{resp_code:200,result:{products:{product:[product(10),product(11,{target_sale_price_currency:"USD"}),product(12,{promotion_link:"https://aliexpress.com.evil.invalid/x"}),product(13,{target_original_price:"45.90"})]}}}}});
  expect(found.map(f=>[f.offer.externalId,f.link,f.offer.price,f.offer.originalPrice,f.offer.discount])).toEqual([["10","https://s.click.aliexpress.com/e/_10",4590,9180,50]]);
  expect(found[0].offer.url).toBe("https://www.aliexpress.com/item/10.html");
  expect(()=>aliexpressOffers({error_response:{code:"InvalidApiKey"}})).toThrow(ShopRefused);
  expect(()=>aliexpressOffers({type:"ISV",code:"IncompleteSignature",message:"x"})).toThrow(ShopRefused);
 });
 it("searches with the person's keys and posts the biggest discounts, again only when cheaper",async()=>{
  const channel=await addDiscord();
  const hunt={kind:"store_deals",channelIds:[channel],filters:{store:"shopee",keywords:["fone bluetooth","air fryer"],minDiscount:20,maxPrice:20000}};
  expect(await(await call("/hunts",{method:"POST",body:hunt})).json()).toEqual({code:"STORE_NOT_CONFIGURED"});
  expect((await call("/affiliates/shopee",{method:"PUT",body:{appId:"12345",secret:"s".repeat(32)}})).status).toBe(200);
  expect((await call("/hunts",{method:"POST",body:{...hunt,filters:{store:"shopee",minDiscount:20}}})).status).toBe(400);
  expect((await call("/hunts",{method:"POST",body:{...hunt,filters:{...hunt.filters,country:"br"}}})).status).toBe(400);
  expect((await call("/hunts",{method:"POST",body:{kind:"games_free",channelIds:[channel],filters:{keywords:["x1"]}}})).status).toBe(400);
  const created=await call("/hunts",{method:"POST",body:hunt});expect(created.status).toBe(201);const id=(await created.json() as {id:string}).id;
  let fone=node(1,"89.90",30);
  const asked:string[]=[];
  const {send,calls}=net({"open-api.affiliate.shopee.com.br":async req=>{const body=await req.json() as {variables:{keyword:string}};asked.push(body.variables.keyword);return Response.json(body.variables.keyword==="fone bluetooth"?shopee(fone,node(2,"300.00",60),node(3,"20.00",10)):shopee(node(4,"150.00",45),fone));}});
  const t0=Date.now()+1000;
  await collectShops(state.env,send,t0);
  expect(asked).toEqual(["fone bluetooth","air fryer"]);
  expect(calls[0].headers.get("authorization")).toMatch(/^SHA256 Credential=12345, Timestamp=[0-9]+, Signature=[0-9a-f]{64}$/);
  // Over the price limit (item 2) and under the discount (item 3) stay out.
  expect(db.sqlite.prepare("SELECT o.external_id AS id,p.link,p.affiliate FROM deals_post p JOIN deals_offer o ON o.id=p.offer_id ORDER BY o.discount DESC").all()).toEqual([{id:"4",link:"https://s.shopee.com.br/off4",affiliate:1},{id:"1",link:"https://s.shopee.com.br/off1",affiliate:1}]);
  await collectShops(state.env,send,t0+10*60_000);expect(asked).toHaveLength(2);
  await collectShops(state.env,send,t0+31*60_000);expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_post").get()?.n).toBe(2);
  fone=node(1,"79.90",38);
  await collectShops(state.env,send,t0+62*60_000);expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_post").get()?.n).toBe(3);
  // Refused keys show on the hunt; removing it takes its reading along.
  await collectShops(state.env,net({"open-api.affiliate.shopee.com.br":()=>Response.json({errors:[{message:"Invalid Credential"}]})}).send,t0+93*60_000);
  const listed=await(await call("/hunts")).json() as {hunts:{kind:string;shop:{error:string|null}|null}[]};
  expect(listed.hunts).toEqual([expect.objectContaining({kind:"store_deals",shop:{error:"STORE_REFUSED"}})]);
  expect((await call(`/hunts/${id}`,{method:"DELETE"})).status).toBe(200);
  expect(db.sqlite.prepare("SELECT count(*) AS n FROM deals_source WHERE id=?").get(`shop:${id}`)?.n).toBe(0);
 });
 it("sells in the money of each country: Shopee by the account, AliExpress by the hunt",async()=>{
  const channel=await addDiscord();
  expect((await call("/affiliates/shopee",{method:"PUT",body:{appId:"12345",secret:"s".repeat(32),country:"my"}})).status).toBe(200);
  expect((await call("/affiliates/aliexpress",{method:"PUT",body:{appKey:"123456",appSecret:"a".repeat(32),trackingId:"meutrack"}})).status).toBe(200);
  expect((await call("/hunts",{method:"POST",body:{kind:"store_deals",channelIds:[channel],filters:{store:"shopee",market:"us",keywords:["lamp"]}}})).status).toBe(400);
  expect((await call("/hunts",{method:"POST",body:{kind:"store_deals",channelIds:[channel],filters:{store:"shopee",keywords:["lamp"],minDiscount:10}}})).status).toBe(201);
  expect((await call("/hunts",{method:"POST",body:{kind:"store_deals",channelIds:[channel],filters:{store:"aliexpress",market:"jp",keywords:["lamp"],minDiscount:10,maxPrice:5000}}})).status).toBe(201);
  const aliAsked:URLSearchParams[]=[];
  const {send,calls}=net({
   "open-api.affiliate.shopee.com.my":()=>Response.json(shopee(node(7,"25.90",20,{productLink:"https://shopee.com.my/product/1/7",offerLink:"https://s.shopee.com.my/x7"}))),
   "api-sg.aliexpress.com":req=>{const q=new URL(req.url).searchParams;aliAsked.push(q);return Response.json({aliexpress_affiliate_product_query_response:{resp_result:{resp_code:200,result:{products:{product:[{product_id:55,product_title:"Lamp",promotion_link:"https://s.click.aliexpress.com/e/_55",target_sale_price:"1980",target_sale_price_currency:"JPY",target_original_price:"3960",discount:"50%"},{product_id:56,product_title:"Big lamp",promotion_link:"https://s.click.aliexpress.com/e/_56",target_sale_price:"6000",target_sale_price_currency:"JPY",target_original_price:"9000",discount:"33%"}]}}}}});},
  });
  await collectShops(state.env,send,Date.now()+1000);
  expect(calls.some(c=>c.url==="https://open-api.affiliate.shopee.com.my/graphql")).toBe(true);
  expect(Object.fromEntries(["target_currency","ship_to_country","target_language","tracking_id"].map(k=>[k,aliAsked[0].get(k)]))).toEqual({target_currency:"JPY",ship_to_country:"JP",target_language:"JA",tracking_id:"meutrack"});
  // The yen has no decimals: 1980 yen is 1980, and over the limit (5000 yen) stays out.
  expect(db.sqlite.prepare("SELECT o.store,o.price,o.original_price AS before,o.currency,p.link FROM deals_post p JOIN deals_offer o ON o.id=p.offer_id ORDER BY o.store").all()).toEqual([{store:"aliexpress",price:1980,before:3960,currency:"JPY",link:"https://s.click.aliexpress.com/e/_55"},{store:"shopee",price:2590,before:null,currency:"MYR",link:"https://s.shopee.com.my/x7"}]);
  expect(telegramText({title:"Lamp",price:1980,originalPrice:3960,currency:"JPY",discount:50,link:"https://deals.audit.invalid/r/x",affiliate:true,kind:"store_deal",credit:null,image:null},"en")).toContain("¥1,980");
 });
});
