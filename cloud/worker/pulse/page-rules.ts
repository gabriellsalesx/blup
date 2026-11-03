import {z} from "zod";
export const reservedSlugs=["api","www","admin","cloud","staff","pulse","status","login","assets","support"];
export const slugShape=z.string().min(3).max(40).regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/);
export const pageSlug=slugShape.refine(v=>!reservedSlugs.includes(v));
// How an incident leaves each service of its page.
export const serviceStatus=z.enum(["none","downtime","degraded","resolved"]);
