import {beforeEach,afterEach,it,expect,vi} from "vitest";
import {TestD1,seedUser} from "../security-test-support";
const state=vi.hoisted(()=>({env:{} as Cloudflare.Env}));
vi.mock("cloudflare:workers",()=>({env:state.env,DurableObject:class{}}));
vi.mock("../auth",()=>({auth:{api:{getSession:async({headers}:{headers:Headers})=>{const id=headers.get("authorization")?.replace("Bearer ","");return id?{user:{id}}:null;}}}}));
import {dealsAccess} from "./access";
import {dealsApi} from "./api";
import {pulseAccess} from "../pulse/limits";
import {planWith,hasDeals,pulsePlanOf,plansWithPulse,paidPlanNameSchema,dealsLimits} from "../../shared/pulse-plans";
import {PLANS,planNamed} from "../plans";
import {planLabel} from "../../shared/plan-i18n";

let db:TestD1;
beforeEach(()=>{
 db=new TestD1();
 for(const [id,plan] of [["deals","ultra_deals"],["both","ultra_pulse_deals"],["ultra","ultra"],["pulse","ultra_pulse"],["max","max_pulse"],["free","free"],["owner","free"]])seedUser(db,id,plan);
 Object.assign(state.env,{DB:db.asD1(),STAFF_OWNER_EMAIL:"owner@audit.invalid"});
});
afterEach(()=>{db.close();vi.restoreAllMocks();});
const usage=async(user?:string)=>dealsApi.fetch(new Request("https://cloud.audit.invalid/usage",{headers:user?{authorization:`Bearer ${user}`}:{}}),state.env);

it("exists only with Ultra, with or without Pulse",()=>{
 expect(planWith("ultra",{pulse:false,deals:true})).toBe("ultra_deals");
 expect(planWith("ultra",{pulse:true,deals:true})).toBe("ultra_pulse_deals");
 for(const base of ["free","pro","max"] as const)expect(planWith(base,{pulse:false,deals:true})).toBeNull();
 expect(planWith("max",{pulse:true,deals:true})).toBeNull();
 expect(paidPlanNameSchema.options).toEqual(expect.arrayContaining(["ultra_deals","ultra_pulse_deals"]));
 expect(paidPlanNameSchema.safeParse("max_deals").success).toBe(false);
});

it("keeps Ultra's Cloud limits and sells each variant by its own Stripe price",()=>{
 for(const name of ["ultra_deals","ultra_pulse_deals"] as const){
  const plan=planNamed(name);
  expect(plan.name).toBe(name);expect(plan.basePlan).toBe("ultra");expect(plan.limits).toEqual(planNamed("ultra").limits);expect(plan.deals).toEqual(dealsLimits);
  expect(plan.lookupKey).toBe(`blup_${name}_mensal`);expect(plan.yearlyLookupKey).toBe(`blup_${name}_anual`);
 }
 expect(planNamed("ultra_pulse_deals").pulse).toEqual(planNamed("ultra_pulse").pulse);expect(planNamed("ultra_deals").pulse).toBeNull();
 expect(PLANS.filter(p=>p.deals).map(p=>p.name)).toEqual(["ultra_deals","ultra_pulse_deals"]);
 expect(planLabel("ultra_pulse_deals","en")).toBe("Ultra + Pulse + Deals");expect(planLabel("ultra_deals","pt")).toBe("Ultra + Deals");
 expect(dealsLimits).toEqual({channels:10,hunts:20,mirrors:10,postsPerDay:100,historyDays:90});
});

it("gives Deals to Ultra + Deals subscriptions and to the owner only",async()=>{
 expect(await dealsAccess(state.env,"deals")).toEqual({plan:"ultra_deals",limits:dealsLimits,readOnly:false,readable:true,readOnlyUntil:null});
 expect(await dealsAccess(state.env,"both")).toEqual({plan:"ultra_pulse_deals",limits:dealsLimits,readOnly:false,readable:true,readOnlyUntil:null});
 expect((await dealsAccess(state.env,"owner")).plan).toBe("ultra_pulse_deals");
 for(const id of ["ultra","pulse","max","free","nobody"])expect(await dealsAccess(state.env,id)).toEqual({plan:null,limits:null,readOnly:true,readable:false,readOnlyUntil:null});
});

it("ends with the subscription and with a banned or deleted account",async()=>{
 db.sqlite.exec("UPDATE subscription SET status='past_due' WHERE reference_id='deals'");expect((await dealsAccess(state.env,"deals")).readOnly).toBe(false);
 db.sqlite.exec("UPDATE subscription SET status='canceled' WHERE reference_id='deals'");expect((await dealsAccess(state.env,"deals")).plan).toBeNull();
 db.sqlite.exec("UPDATE user SET banned=1 WHERE id='both'");expect((await dealsAccess(state.env,"both")).plan).toBeNull();
});

it("keeps Pulse for Ultra + Pulse + Deals, not for Ultra + Deals",async()=>{
 expect(pulsePlanOf("ultra_pulse_deals")).toBe("ultra_pulse");expect(hasDeals("ultra_pulse")).toBe(false);
 expect(plansWithPulse("ultra_pulse")).toEqual(["ultra_pulse","ultra_pulse_deals"]);
 expect(await pulseAccess(state.env,"both")).toMatchObject({plan:"ultra_pulse",writable:true});
 expect(await pulseAccess(state.env,"pulse")).toMatchObject({plan:"ultra_pulse",writable:true});
 expect(await pulseAccess(state.env,"deals")).toMatchObject({plan:null,writable:false});
});

it("answers the usage only with a session, read only without the plan",async()=>{
 expect((await usage()).status).toBe(401);
 const none=await(await usage("ultra")).json();expect(none).toMatchObject({plan:null,limits:null,readOnly:true,used:{channels:0,hunts:0,mirrors:0,postsToday:0}});
 db.sqlite.exec("INSERT INTO deals_channel(id,user_id,platform,name,target,created_at) VALUES('c1','deals','telegram','Canal','-100',0),('c2','free','telegram','Outro','-200',0)");
 db.sqlite.exec("INSERT INTO deals_hunt(id,user_id,name,kind,created_at) VALUES('h1','deals','Jogos','games',0),('h2','deals','Espelho','mirror',0),('h3','deals','Velha','games',0)");
 db.sqlite.exec(`UPDATE deals_hunt SET retire_at=${Date.now()-1} WHERE id='h3'`);
 const day=new Date().toISOString().slice(0,10);
 db.sqlite.exec(`INSERT INTO deals_post(id,user_id,channel_id,offer_id,code,link,next_at,day,created_at) VALUES('p1','deals','c1','o1','abc','https://x.invalid',0,'${day}',0),('p2','deals','c1','o2','abd','https://x.invalid',0,'2020-01-01',0)`);
 const body=await(await usage("deals")).json() as {resetsAt:string};
 expect(body).toMatchObject({plan:"ultra_deals",limits:dealsLimits,readOnly:false,used:{channels:1,hunts:1,mirrors:1,postsToday:1}});
 expect(Date.parse(body.resetsAt)).toBeGreaterThan(Date.now());
});
