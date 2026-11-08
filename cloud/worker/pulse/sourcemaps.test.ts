import {beforeEach,afterEach,it,expect,vi} from "vitest";
// esbuild (installed with Vite) makes a real minified file and its map.
import {transformSync} from "esbuild";
import {TestD1,TestBucket,seedUser} from "../security-test-support";
import {SourceMapLookup,sourceMapSchema,sourcePath,enclosingFunction} from "./sourcemap";
const state=vi.hoisted(()=>({env:{} as Cloudflare.Env,samples:[] as unknown[]}));
vi.mock("cloudflare:workers",()=>({env:state.env,DurableObject:class{}}));
vi.mock("../auth",()=>({auth:{api:{getSession:async({headers}:{headers:Headers})=>{const id=headers.get("authorization")?.replace("Bearer ","");return id&&!/^[a-f0-9]{48}$/.test(id)?{user:{id}}:null;}}}}));
import {pulseApi} from "./api";
import {SOURCEMAP_BYTES_PER_PROJECT,SOURCEMAPS_PER_PROJECT} from "./sourcemaps";
const source=`export function total(cart: {items:{price:number}[]}, coupon: any) {\n  const subtotal = cart.items.reduce((s, i) => s + i.price, 0);\n  const rate = coupon.discount.rate;\n  return subtotal * (1 - rate);\n}\n`;
const built=transformSync(source,{loader:"ts",minify:true,sourcemap:"external",sourcefile:"src/checkout.ts",format:"esm"});
// Where the minified code reads the coupon's discount (1-based, as SDKs send).
const column=built.code.indexOf(".discount");
let db:TestD1,bucket:TestBucket;
beforeEach(()=>{
 db=new TestD1();bucket=new TestBucket();seedUser(db,"alice","max_pulse");seedUser(db,"bob","ultra_pulse");state.samples=[];
 db.sqlite.exec("INSERT INTO pulse_project(id,user_id,name,platform,object_name,created_at) VALUES('1','alice','Shop','javascript','one',0),('2','bob','Other','javascript','two',0)");
 Object.assign(state.env,{DB:db.asD1(),PULSE_SOURCEMAPS:bucket.asR2(),ABUSE_LIMIT:{limit:async()=>({success:true})},STAFF_OWNER_EMAIL:"",PULSE_PROJECT:{getByName:()=>({samples:async()=>state.samples})}});
});
afterEach(()=>{db.close();vi.restoreAllMocks();});
const call=(path:string,init:{method?:string;body?:string;auth?:string}={})=>pulseApi.fetch(new Request(`https://cloud.audit.invalid${path}`,{method:init.method??"GET",headers:{...(init.auth?{authorization:`Bearer ${init.auth}`}:{}),...(init.body!==undefined?{"content-type":"application/json"}:{})},body:init.body}),state.env);
it("finds the original file, line, name and code of a minified position",()=>{
 const lookup=new SourceMapLookup(sourceMapSchema.parse(JSON.parse(built.map)));
 const original=lookup.originalOf(1,column);
 expect(original).toMatchObject({file:"src/checkout.ts",line:3,inApp:true,function:"total"});
 expect(original?.context?.line).toBe("  const rate = coupon.discount.rate;");
 expect(original?.context?.pre).toHaveLength(2);
 expect(lookup.originalOf(99,1)).toBeNull();
 expect(enclosingFunction("class A {\n  async pay(order) {\n    if (x) {\n      go();",3)).toBe("pay");expect(enclosingFunction("const run = async (a) => {\n  a.b;",1)).toBe("run");expect(enclosingFunction("a.b;",0)).toBe("");
 expect(sourcePath("webpack:///./src/app.ts?abc")).toBe("src/app.ts");expect(sourcePath("../node_modules/lib/index.js")).toBe("node_modules/lib/index.js");
});
it("takes maps only with the project's upload token and shows originals when reading",async()=>{
 const minted=await call("/projects/1/upload-token",{method:"POST",body:"{}",auth:"alice"});expect(minted.status).toBe(201);const {token}=await minted.json() as {token:string};
 expect((await call("/projects/2/upload-token",{method:"POST",body:"{}",auth:"alice"})).status).toBe(404);
 const send=(query:string,body:string,auth=token)=>pulseApi.fetch(new Request(`https://cloud.audit.invalid/sourcemaps?${query}`,{method:"POST",headers:{authorization:`Bearer ${auth}`},body}),state.env);
 expect((await send("release=1.0.0&file=app.min.js",built.map,"f".repeat(48))).status).toBe(401);
 expect((await send("release=1.0.0&file=../app.js",built.map)).status).toBe(400);
 expect((await send("release=1.0.0&file=app.min.js","{\"version\":2}")).status).toBe(400);
 expect((await send("release=1.0.0&file=app.min.js",built.map)).status).toBe(201);
 const listed=await(await call("/projects/1/sourcemaps",{auth:"alice"})).json() as {token:{prefix:string}|null;maps:{release:string;file:string}[]};
 expect(listed.token?.prefix).toBe(token.slice(0,8));expect(listed.maps.map(m=>[m.release,m.file])).toEqual([["1.0.0","app.min.js"]]);
 expect(JSON.stringify(listed)).not.toContain(token);
 state.samples=[{id:"e1",receivedAt:Date.now(),event:{id:"e1",group:"g",title:"x",level:"error",platform:"javascript",environment:"production",release:"1.0.0",message:"",exceptions:[{type:"TypeError",value:"x",frames:[{filename:"app.min.js",function:"i",module:"",line:1,column,inApp:true}]}]}}];
 const events=await(await call("/projects/1/issues/"+"a".repeat(64)+"/events",{auth:"alice"})).json() as {events:{originals:Record<string,{file:string;line:number}>}[]};
 expect(events.events[0].originals["0:0"]).toMatchObject({file:"src/checkout.ts",line:3});
 // A new token replaces the old one.
 await call("/projects/1/upload-token",{method:"POST",body:"{}",auth:"alice"});expect((await send("release=1.0.1&file=app.min.js",built.map)).status).toBe(401);
});
it("keeps the per-project quota in count and bytes and rate limits each token",async()=>{
 const minted=await call("/projects/1/upload-token",{method:"POST",body:"{}",auth:"alice"});const {token}=await minted.json() as {token:string};
 const send=(query:string)=>pulseApi.fetch(new Request(`https://cloud.audit.invalid/sourcemaps?${query}`,{method:"POST",headers:{authorization:`Bearer ${token}`},body:built.map}),state.env);
 // Almost all the bytes already used: a new map does not fit, replacing it does.
 db.sqlite.exec(`INSERT INTO pulse_sourcemap(project_id,release,file,object_key,size,uploaded_at) VALUES('1','0.9.0','big.min.js','k',${SOURCEMAP_BYTES_PER_PROJECT-10},0)`);
 expect((await send("release=1.0.0&file=app.min.js")).status).toBe(409);
 expect((await send("release=0.9.0&file=big.min.js")).status).toBe(201);
 const row=db.sqlite.prepare("SELECT size FROM pulse_sourcemap WHERE release='0.9.0'").get() as {size:number};expect(row.size).toBe(new TextEncoder().encode(JSON.stringify(sourceMapSchema.parse(JSON.parse(built.map)))).byteLength);
 // The count: 300 maps kept, the 301st is refused.
 db.sqlite.exec("DELETE FROM pulse_sourcemap");
 const insert=db.sqlite.prepare("INSERT INTO pulse_sourcemap(project_id,release,file,object_key,size,uploaded_at) VALUES('1',?,'a.min.js','k',1,0)");for(let i=0;i<SOURCEMAPS_PER_PROJECT;i++)insert.run(`r${i}`);
 expect((await send("release=1.0.0&file=app.min.js")).status).toBe(409);
 // Over the token's rate: 429 before reading the database.
 const limit=vi.fn(async({key}:{key:string})=>({success:!key.startsWith("sourcemap:")}));Object.assign(state.env,{ABUSE_LIMIT:{limit}});
 const res=await send("release=r1&file=a.min.js");expect(res.status).toBe(429);expect(res.headers.get("retry-after")).toBe("60");
});
