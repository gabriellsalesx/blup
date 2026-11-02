import {z} from "zod";
export const pageSlug=z.string().min(3).max(40).regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/).refine(v=>!["api","www","admin","cloud","staff","pulse","status","login","assets","support"].includes(v));
