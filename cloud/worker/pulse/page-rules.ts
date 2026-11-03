import {z} from "zod";
export const reservedSlugs=["api","www","admin","cloud","staff","pulse","status","login","assets","support"];
export const slugShape=z.string().min(3).max(40).regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/);
export const pageSlug=slugShape.refine(v=>!reservedSlugs.includes(v));
