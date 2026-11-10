// zod/mini: the panels import this file too (much smaller than full zod).
import {z} from "zod/mini";
export const pulsePlanSchema=z.enum(["max_pulse","ultra_pulse"]);
export type PulsePlan=z.infer<typeof pulsePlanSchema>;
// Blup Deals exists only with Ultra, with or without Pulse.
export const dealsPlanSchema=z.enum(["ultra_deals","ultra_pulse_deals"]);
export type DealsPlan=z.infer<typeof dealsPlanSchema>;
const paidPlanNames=["pro","max","ultra","max_pulse","ultra_pulse","ultra_deals","ultra_pulse_deals"] as const;
export const planNameSchema=z.enum(["free",...paidPlanNames]);
export const paidPlanNameSchema=z.enum(paidPlanNames);
export type PlanName=z.infer<typeof planNameSchema>;
export type BasePlanName="free"|"pro"|"max"|"ultra";
// What each complete variant is made of: its base plan, the Pulse limits it
// carries (by Pulse plan) and whether it has Deals.
const variants:Record<PlanName,{base:BasePlanName;pulse:PulsePlan|null;deals:boolean}>={
 free:{base:"free",pulse:null,deals:false},pro:{base:"pro",pulse:null,deals:false},max:{base:"max",pulse:null,deals:false},ultra:{base:"ultra",pulse:null,deals:false},
 max_pulse:{base:"max",pulse:"max_pulse",deals:false},ultra_pulse:{base:"ultra",pulse:"ultra_pulse",deals:false},
 ultra_deals:{base:"ultra",pulse:null,deals:true},ultra_pulse_deals:{base:"ultra",pulse:"ultra_pulse",deals:true},
};
export const basePlanName=(name:PlanName):BasePlanName=>variants[name].base;
export const pulsePlanOf=(name:PlanName):PulsePlan|null=>variants[name].pulse;
export const hasDeals=(name:PlanName):name is DealsPlan=>variants[name].deals;
// The plan with this base and these add-ons, or null when it does not exist
// (Deals only with Ultra, Pulse only with Max and Ultra).
export const planWith=(base:BasePlanName,addOns:{pulse:boolean;deals:boolean}):PlanName|null=>planNameSchema.options.find(n=>variants[n].base===base&&(variants[n].pulse!==null)===addOns.pulse&&variants[n].deals===addOns.deals)??null;
// The subscription plans that give these Pulse limits (or any Pulse).
export const plansWithPulse=(pulse?:PulsePlan):PlanName[]=>planNameSchema.options.filter(n=>pulse?variants[n].pulse===pulse:variants[n].pulse!==null);
export const plansWithDeals:PlanName[]=planNameSchema.options.filter(n=>variants[n].deals);
// A fixed list of plan names as SQL literals: the names are the constants
// above, never input.
export const sqlPlans=(names:readonly PlanName[])=>names.map(n=>`'${planNameSchema.parse(n)}'`).join(",");
// Which subscription wins when an account has more than one (a Stripe one
// and the AppSumo lifetime one): the most complete plan; on a tie the one
// with no end (the lifetime row has no period_end), then by id, so every
// reader picks the same row.
const planRanks:Record<z.infer<typeof paidPlanNameSchema>,number>={pro:1,max:2,max_pulse:3,ultra:4,ultra_deals:5,ultra_pulse:6,ultra_pulse_deals:7};
export const planRank=(name:string|null|undefined)=>{const parsed=paidPlanNameSchema.safeParse(name);return parsed.success?planRanks[parsed.data]:0;};
// Whether `target` has everything `plan` has and more: each of its add-ons
// (Pulse, Deals) and either a higher base plan or the same base with an
// add-on more. What a lifetime plan lets a person buy from Stripe.
export const supersetOf=(target:PlanName,plan:PlanName)=>{
 if(pulsePlanOf(plan)&&!pulsePlanOf(target)||hasDeals(plan)&&!hasDeals(target))return false;
 const base=planRank(basePlanName(target))-planRank(basePlanName(plan));
 return base>0||base===0&&(!pulsePlanOf(plan)&&!!pulsePlanOf(target)||!hasDeals(plan)&&hasDeals(target));
};
// The ORDER BY of a subscription read, for a table alias ("s") or none.
// The CASE is built from the constants above, never input.
export const planOrder=(alias="")=>{
 const p=alias?`${alias}.`:"";if(!/^[a-z]{0,8}$/.test(alias))throw new Error("Bad alias");
 return `CASE ${p}plan ${paidPlanNameSchema.options.map(n=>`WHEN '${n}' THEN ${planRanks[n]}`).join(" ")} ELSE 0 END DESC,${p}period_end IS NULL DESC,${p}id`;
};
// The same order in JS (staff lists), highest first.
export const byPlanOrder=(a:{plan:string;periodEnd:Date|number|null;id:string},b:{plan:string;periodEnd:Date|number|null;id:string})=>planRank(b.plan)-planRank(a.plan)||(a.periodEnd===null?0:1)-(b.periodEnd===null?0:1)||(a.id<b.id?-1:a.id>b.id?1:0);
export const pulseLimits={
 max_pulse:{projects:3,events:50_000,retentionDays:30,monitors:10,intervalMinutes:5,pages:1,customDomain:false,projectBytes:256*1024*1024,burst:60},
 ultra_pulse:{projects:10,events:250_000,retentionDays:90,monitors:50,intervalMinutes:2,pages:3,customDomain:true,projectBytes:1024*1024*1024,burst:300},
} as const;
// Blup Deals: destination channels, hunts, mirrored channels, posts sent
// per day and days of history (offers, prices and clicks).
export const dealsLimits={channels:10,hunts:20,mirrors:10,postsPerDay:100,historyDays:90} as const;
export type DealsLimits=typeof dealsLimits;
