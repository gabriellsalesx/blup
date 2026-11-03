import {beforeEach,afterEach,it,expect,vi} from "vitest";
import {TestD1,seedUser} from "../security-test-support";
const state=vi.hoisted(()=>({env:{} as Cloudflare.Env,issues:false}));
vi.mock("cloudflare:workers",()=>({env:state.env,DurableObject:class{}}));
vi.mock("../auth",()=>({auth:{api:{getSession:async({headers}:{headers:Headers})=>{const id=headers.get("authorization")?.replace("Bearer ","");return id?{user:{id}}:null;}}}}));
import {pulseApi} from "./api";
let db:TestD1;
beforeEach(()=>{
 db=new TestD1();seedUser(db,"alice","max_pulse");seedUser(db,"free");state.issues=false;
 const hasIssues=vi.fn(async()=>state.issues);
 Object.assign(state.env,{DB:db.asD1(),STAFF_OWNER_EMAIL:"",PULSE_PROJECT:{getByName:()=>({hasIssues})}});
});
afterEach(()=>{db.close();vi.restoreAllMocks();});
const setup=async(user:string)=>(await(await pulseApi.fetch(new Request("https://cloud.audit.invalid/setup",{headers:{authorization:`Bearer ${user}`}}),state.env)).json() as {steps:{id:string;done:boolean}[]}).steps;
const done=(steps:{id:string;done:boolean}[])=>steps.filter(s=>s.done).map(s=>s.id);
it("starts empty, with the plan as the first step",async()=>{
 expect((await setup("free")).map(s=>s.id)).toEqual(["plan","project","event","monitor","alert","page"]);
 expect(done(await setup("free"))).toEqual([]);expect(done(await setup("alice"))).toEqual(["plan"]);
});
it("follows what the account has and notes the first error once",async()=>{
 db.sqlite.exec("INSERT INTO pulse_project(id,user_id,name,platform,object_name,created_at) VALUES('1','alice','App','go','one',0)");
 expect(done(await setup("alice"))).toEqual(["plan","project"]);
 state.issues=true;expect(done(await setup("alice"))).toEqual(["plan","project","event"]);
 expect(db.sqlite.prepare("SELECT first_event_at AS at FROM pulse_project WHERE id='1'").get()?.at).toBeTypeOf("number");
 state.issues=false;expect(done(await setup("alice"))).toContain("event");
 db.sqlite.exec("INSERT INTO pulse_monitor(id,user_id,project_id,name,url,interval_minutes,next_at,created_at) VALUES('m1','alice','1','Site','https://example.com',5,0,0)");
 db.sqlite.exec("INSERT INTO pulse_channel(id,user_id,type,target,label,created_at) VALUES('c1','alice','discord','x','Discord',0)");db.sqlite.exec("INSERT INTO pulse_project_channel VALUES('1','c1','discord')");
 db.sqlite.exec("INSERT INTO pulse_page(id,user_id,slug,title,created_at) VALUES('p1','alice','status','Status',0)");
 expect(done(await setup("alice"))).toEqual(["plan","project","event","monitor","alert","page"]);
 expect(done(await setup("free"))).toEqual([]);
});
