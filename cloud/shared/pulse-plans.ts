import {z} from "zod";
export const pulsePlanSchema=z.enum(["max_pulse","ultra_pulse"]);
export type PulsePlan=z.infer<typeof pulsePlanSchema>;
export const planNameSchema=z.enum(["free","pro","max","ultra","max_pulse","ultra_pulse"]);
export const paidPlanNameSchema=z.enum(["pro","max","ultra","max_pulse","ultra_pulse"]);
export type PlanName=z.infer<typeof planNameSchema>;
export type BasePlanName="free"|"pro"|"max"|"ultra";
export const basePlanName=(name:PlanName):BasePlanName=>name==="max_pulse"?"max":name==="ultra_pulse"?"ultra":name;
export const pulseLimits={
 max_pulse:{projects:3,events:50_000,retentionDays:30,monitors:10,intervalMinutes:5,pages:1,customDomain:false,projectBytes:256*1024*1024,burst:60},
 ultra_pulse:{projects:10,events:250_000,retentionDays:90,monitors:50,intervalMinutes:2,pages:3,customDomain:true,projectBytes:1024*1024*1024,burst:300},
} as const;
