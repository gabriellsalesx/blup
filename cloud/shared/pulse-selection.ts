import {z} from "zod";
const opaque=z.string().regex(/^[A-Za-z0-9_-]{1,128}$/);
export const keepSchema=z.object({projects:z.array(z.string().regex(/^[1-9][0-9]{0,15}$/)).max(10),monitors:z.array(opaque).max(50),pages:z.array(opaque).max(3)}).strict().refine(v=>[v.projects,v.monitors,v.pages].every(ids=>new Set(ids).size===ids.length));
