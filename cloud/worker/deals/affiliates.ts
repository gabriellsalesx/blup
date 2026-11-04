import {Hono} from "hono";
import {z} from "zod";
import {validate} from "../validation";
import {dealsWrite,type DealsApp} from "./access";
import {stores,storeSchema,affiliateBody,affiliateHint} from "./links";
import {seal} from "./store";

const storeParam=z.object({store:storeSchema});
// The body of each store, as the panel sends it (the store is in the path).
const body=z.union([
 z.object({tag:z.string().max(64)}).strict(),
 z.object({word:z.string().max(64),tool:z.string().max(20).optional()}).strict(),
 z.object({appId:z.string().max(20),secret:z.string().max(128)}).strict(),
 z.object({appKey:z.string().max(20),appSecret:z.string().max(128),trackingId:z.string().max(64)}).strict(),
]);
// The IDs and keys are sealed at rest. The panel only ever learns whether
// a store is configured and a few characters to recognize it.
export const affiliatesApi=new Hono<DealsApp>()
 .get("/",async c=>{
  const rows=await c.env.DB.prepare("SELECT store,hint,credentials IS NOT NULL AS configured FROM deals_affiliate WHERE user_id=?").bind(c.get("session").user.id).all<{store:string;hint:string|null;configured:number}>();
  const saved=new Map(rows.results.map(r=>[r.store,r]));
  return c.json({stores:stores.map(store=>({store,configured:!!saved.get(store)?.configured,hint:saved.get(store)?.hint??null}))});
 })
 .put("/:store",dealsWrite,validate("param",storeParam),validate("json",body),async c=>{
  const owner=c.get("session").user.id,store=c.req.valid("param").store,parsed=affiliateBody.safeParse({...c.req.valid("json"),store});
  if(!parsed.success)return c.json({code:"INVALID"},400);
  let sealed:string;try{sealed=await seal(c.env,owner,store,"affiliate",JSON.stringify(parsed.data));}catch{return c.json({code:"CHANNEL_UNAVAILABLE"},503);}
  const now=Date.now();
  await c.env.DB.prepare("INSERT INTO deals_affiliate(user_id,store,credentials,hint,created_at,updated_at) VALUES(?,?,?,?,?,?) ON CONFLICT(user_id,store) DO UPDATE SET credentials=excluded.credentials,hint=excluded.hint,tag=NULL,updated_at=excluded.updated_at").bind(owner,store,sealed,affiliateHint(parsed.data),now,now).run();
  return c.json({ok:true});
 })
 .delete("/:store",validate("param",storeParam),async c=>{
  await c.env.DB.prepare("DELETE FROM deals_affiliate WHERE user_id=? AND store=?").bind(c.get("session").user.id,c.req.valid("param").store).run();
  return c.json({ok:true});
 });
