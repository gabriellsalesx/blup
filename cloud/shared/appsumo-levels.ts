import type {PlanName} from "./pulse-plans";
// AppSumo's lifetime deal: each code redeemed moves the account up one
// level, to APPSUMO_MAX, on this ladder (the Worker and the redeem page).
export const APPSUMO_MAX=5;
export const appsumoLevels=["pro","max","max_pulse","ultra","ultra_pulse_deals"] as const satisfies readonly PlanName[];
export type AppsumoPlan=typeof appsumoLevels[number];
// The lifetime row of the subscription table has the id appsumo-<user>.
// The one exact test of it, in JS and in SQL (substr, not LIKE, whose _
// and % would be wildcards).
export const isLifetimeId=(id:string|null|undefined)=>!!id&&id.startsWith("appsumo-");
export const LIFETIME_ID_SQL="substr(id,1,8)='appsumo-'";
