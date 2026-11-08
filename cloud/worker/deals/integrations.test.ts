import {it,expect,vi,describe} from "vitest";
vi.mock("cloudflare:workers",()=>({env:{},DurableObject:class{}}));
vi.mock("../auth",()=>({auth:{api:{getSession:async()=>null}}}));
import {shopeeOffers,aliexpressOffers,ShopRefused} from "./shops";
import {FetchError} from "./fetch";
import {telegramText,discordPayload,type Post} from "./message";
import {serveDealsLink} from "./redirect";
import {DetailPage} from "./site-detail";

// The integrations' edges, with the stores' and the platforms' answers
// made up here (no real API is called).
describe("store errors: a limit passes, refused keys stop the hunt",()=>{
 it("tells Shopee's rate limit and own failure from refused keys",()=>{
  const err=(code?:number)=>({errors:[{message:"x",...(code===undefined?{}:{extensions:{code,message:"x"}})}],data:null});
  expect(()=>shopeeOffers(err(10030))).toThrow(FetchError);
  expect(()=>shopeeOffers(err(10000))).toThrow(FetchError);
  expect(()=>shopeeOffers(err(10020))).toThrow(ShopRefused);
  expect(()=>shopeeOffers(err(10035))).toThrow(ShopRefused);
  expect(()=>shopeeOffers(err())).toThrow(ShopRefused);
 });
 it("tells AliExpress's call limit and gateway failure from refused keys",()=>{
  expect(()=>aliexpressOffers({type:"ISP",code:"ApiCallLimit",message:"x"})).toThrow(FetchError);
  expect(()=>aliexpressOffers({error_response:{type:"ISV",code:"AppCallLimit"}})).toThrow(FetchError);
  expect(()=>aliexpressOffers({type:"SYSTEM",code:"ServiceTimeout",message:"x"})).toThrow(FetchError);
  expect(()=>aliexpressOffers({type:"ISV",code:"IncompleteSignature",message:"x"})).toThrow(ShopRefused);
  expect(()=>aliexpressOffers({error_response:{code:"InvalidApiKey"}})).toThrow(ShopRefused);
  expect(aliexpressOffers({unrelated:true})).toEqual([]);
 });
});

const post=(over:Partial<Post>={}):Post=>({title:"Fone",image:null,price:9990,originalPrice:19990,currency:"BRL",discount:50,link:"https://deals.audit.invalid/r/AbCdEfGh",affiliate:true,kind:"product",credit:null,...over});
describe("what people write never becomes markup",()=>{
 it("escapes every text of a Telegram post (HTML)",()=>{
  const text=telegramText(post({title:'<a href="https://evil.invalid">x</a>',description:"</b><script>1</script>",coupon:"<i>CUPOM</i>",extra:"a & b <u>",socials:[{name:"<b>X</b>",url:'https://x.com/a"onmouseover="1'}],style:"classic"}),"pt");
  expect(text).not.toMatch(/<(?:script|i|u|a href="https:\/\/evil)/);
  expect(text).toContain("&lt;a href=&quot;https://evil.invalid&quot;&gt;x&lt;/a&gt;");
  expect(text).toContain("&lt;/b&gt;&lt;script&gt;1&lt;/script&gt;");
  expect(text).toContain("Cupom: &lt;i&gt;CUPOM&lt;/i&gt;");
  expect(text).toContain("a &amp; b &lt;u&gt;");
  expect(text).toContain('<a href="https://x.com/a&quot;onmouseover=&quot;1">&lt;b&gt;X&lt;/b&gt;</a>');
  // The only tags are Blup's own: bold title, struck price, links.
  expect([...text.matchAll(/<\/?([a-z]+)/g)].map(m=>m[1]).filter(t=>!["b","s","a"].includes(t!))).toEqual([]);
 });
 it("escapes Discord's markdown and mentions nobody",()=>{
  const payload=discordPayload(post({description:"[clique](https://evil.invalid) @everyone **x**"}),"pt");
  expect(payload.allowed_mentions).toEqual({parse:[]});
  expect(payload.embeds[0]!.description).toContain("\\[clique\\]\\(https://evil.invalid\\) @everyone \\*\\*x\\*\\*");
 });
});

describe("the short link /r/",()=>{
 const env=(link:string|null)=>({DB:{prepare:()=>({bind:()=>({first:async()=>link===null?null:{id:"p1",userId:"u1",link},run:async()=>({})})})},DEALS_PUBLIC_LIMIT:{limit:async()=>({success:true})}}) as unknown as Cloudflare.Env;
 const go=(e:Cloudflare.Env,path="/r/AbCdEfGh",method="GET")=>serveDealsLink(new Request(`https://deals.audit.invalid${path}`,{method,headers:{"user-agent":"Mozilla/5.0"}}),e);
 it("sends only to the post's own HTTPS link",async()=>{
  const ok=await go(env("https://www.amazon.com.br/dp/B000000001?tag=ana-20"));
  expect(ok.status).toBe(302);expect(ok.headers.get("location")).toBe("https://www.amazon.com.br/dp/B000000001?tag=ana-20");
  for(const link of ["http://www.amazon.com.br/dp/B1","javascript:alert(1)","data:text/html,x","not a url"])expect((await go(env(link))).status).toBe(404);
  expect((await go(env(null))).status).toBe(404);
 });
 it("takes no address from the request",async()=>{
  for(const path of ["/r/AbCdEfGh?to=https://evil.invalid","/r/https:%2F%2Fevil.invalid","/r/../evil","/r/short"]){
   const res=await go(env("https://loja.example/p"),path);
   if(res.status===302)expect(res.headers.get("location")).toBe("https://loja.example/p");else expect(res.status).toBe(404);
  }
  expect((await go(env("https://loja.example/p"),"/r/AbCdEfGh","POST")).status).toBe(404);
 });
});

describe("the deal page for search engines",()=>{
 it("gives the price in the currency's own units (the yen has no cents)",()=>{
  const site={slug:"ana",title:"Ana",description:"",language:"pt" as const,logos:{light:null,dark:null},showTitle:true,links:[],socials:[]};
  const deal=(price:number,currency:string)=>({code:"AbCdEfGh",title:"Fone",image:null,price,originalPrice:null,currency,discount:null,store:"aliexpress",coupon:null,description:null,category:null,sentAt:Date.now(),kind:"store_deal",platforms:[],endsAt:null,expired:false,prices:[]});
  const ld=(price:number,currency:string)=>{const html=String(DetailPage({site,lang:"pt",base:"/ana",origin:"https://deals.audit.invalid",links:"https://deals.audit.invalid",deals:[],query:null,category:null,categories:[],next:null,now:Date.now(),deal:deal(price,currency)}));return JSON.parse(/<script type="application\/ld\+json">([\s\S]*?)<\/script>/.exec(html)![1]!.replace(/\\u003c/g,"<")) as {offers:{price:string;priceCurrency:string}};};
  expect(ld(1500,"JPY").offers).toMatchObject({price:"1500",priceCurrency:"JPY"});
  expect(ld(4590,"BRL").offers).toMatchObject({price:"45.90",priceCurrency:"BRL"});
 });
});
