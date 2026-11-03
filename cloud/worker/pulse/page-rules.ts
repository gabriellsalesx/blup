import {z} from "zod";
export const reservedSlugs=["api","www","admin","cloud","staff","pulse","status","login","assets","support"];
export const slugShape=z.string().min(3).max(40).regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/);
export const pageSlug=slugShape.refine(v=>!reservedSlugs.includes(v));
// How an incident leaves each service of its page.
export const serviceStatus=z.enum(["none","downtime","degraded","resolved"]);
// Where a maintenance stands, from its window and the time now.
export const maintenanceState=z.enum(["scheduled","active","finished"]);
export const maintenanceStateAt=(startsAt:number,endsAt:number,now:number):z.infer<typeof maintenanceState>=>now<startsAt?"scheduled":now<endsAt?"active":"finished";
// The longest window, and how far ahead one can be scheduled.
export const MAINTENANCE_MAX_MS=30*86_400_000,MAINTENANCE_AHEAD_MS=365*86_400_000;
