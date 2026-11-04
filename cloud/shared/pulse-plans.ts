import {z} from "zod";
export const pulsePlanSchema=z.enum(["max_pulse","ultra_pulse"]);
export type PulsePlan=z.infer<typeof pulsePlanSchema>;
// Blup Deals exists only with Ultra, with or without Pulse.
export const dealsPlanSchema=z.enum(["ultra_deals","ultra_pulse_deals"]);
export type DealsPlan=z.infer<typeof dealsPlanSchema>;
export const planNameSchema=z.enum(["free","pro","max","ultra","max_pulse","ultra_pulse","ultra_deals","ultra_pulse_deals"]);
export const paidPlanNameSchema=planNameSchema.exclude(["free"]);
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
export const pulseLimits={
 max_pulse:{projects:3,events:50_000,retentionDays:30,monitors:10,intervalMinutes:5,pages:1,customDomain:false,projectBytes:256*1024*1024,burst:60},
 ultra_pulse:{projects:10,events:250_000,retentionDays:90,monitors:50,intervalMinutes:2,pages:3,customDomain:true,projectBytes:1024*1024*1024,burst:300},
} as const;
// Blup Deals: destination channels, hunts, mirrored channels, posts sent
// per day and days of history (offers, prices and clicks).
export const dealsLimits={channels:10,hunts:20,mirrors:10,postsPerDay:100,historyDays:90} as const;
export type DealsLimits=typeof dealsLimits;
