import {Hono} from "hono";
import {z} from "zod";
import {validate} from "../validation";
import {sameSecret} from "../pulse/channel-crypto";
import {dealsWrite,type DealsApp} from "./access";
import {cleanLine} from "./page";
import {addChannel,targetHash} from "./store";
import {bridge,bridgeReady,signature,MIN_SECRET,type Answer} from "./bridge";

// WhatsApp Canais (newsletters) through the Bridge (cloud/bridge), the
// unofficial way the owner chose: Meta's API does not post in Canais. The
// Bridge keeps each account's WhatsApp connected and posts when asked;
// everything else (channels, queue, posts) stays here, in D1. Each request
// to it is signed (HMAC with time and nonce), and so is each notice it
// sends back. Only Canais: groups don't exist for the Bridge.
const states=["pairing","linking","connected","disconnected","expired","logged_out","banned","replaced","outdated"] as const;
const bridgeView=z.object({state:z.enum(states),qr:z.string().startsWith("data:image/png;base64,").max(200_000).optional(),qrExpiresAt:z.string().max(64).optional(),phoneEnd:z.string().regex(/^[0-9]{4}$/).optional()});
const bridgeChannels=z.object({channels:z.array(z.object({id:z.string().regex(/^[0-9]{6,32}$/),name:z.string().max(200),role:z.enum(["owner","admin"])})).max(200)});
// The states a person has to act on: the Canais wait until they reconnect.
const ended=new Set<string>(["expired","logged_out","banned","replaced","outdated"]);
export const whatsappError=(state:string)=>`WHATSAPP_${state.toUpperCase()}`;

type Row={id:string;state:string;phoneEnd:string|null};
// What the panel gets about the connection, always the same shape (none:
// never connected; ready: the Bridge is set up; reachable false: the Bridge
// did not answer now, so state is the last one known).
type State=(typeof states)[number];
const stateOf=(s:string):State=>z.enum(states).catch("logged_out").parse(s);
const out=(state:State|"none",ready:boolean,v:{phoneEnd?:string|null;qr?:string;qrExpiresAt?:string}={},reachable=true)=>({state,phoneEnd:v.phoneEnd??null,qr:v.qr??null,qrExpiresAt:v.qrExpiresAt??null,ready,reachable});
const rowOf=(db:D1Database,owner:string)=>db.prepare("SELECT id,state,phone_end AS phoneEnd FROM deals_whatsapp WHERE user_id=?").bind(owner).first<Row>();
// A new state from the Bridge: kept, and the Canais of that WhatsApp show
// it (or go back to normal once it connects again).
async function keep(db:D1Database,id:string,state:string,phoneEnd:string|null|undefined,now=Date.now()){
 await db.batch([
  db.prepare("UPDATE deals_whatsapp SET state=?,phone_end=coalesce(?,phone_end),updated_at=? WHERE id=?").bind(state,phoneEnd??null,now,id),
  state==="connected"
   ?db.prepare("UPDATE deals_channel SET last_error=NULL WHERE session_id=? AND last_error LIKE 'WHATSAPP_%'").bind(id)
   :ended.has(state)?db.prepare("UPDATE deals_channel SET last_error=? WHERE session_id=?").bind(whatsappError(state),id):db.prepare("SELECT 1"),
 ]);
}

// /api/deals/whatsapp: connecting the account's WhatsApp by QR and picking
// the Canais it posts in. The QR is the Bridge's picture, shown while it
// waits; of the number, only its last four digits.
export const whatsappApi=new Hono<DealsApp>()
 .get("/session",async c=>{
  const row=await rowOf(c.env.DB,c.get("session").user.id);
  const ready=bridgeReady(c.env);
  if(!row)return c.json(out("none",ready));
  if(!ready||ended.has(row.state))return c.json(out(stateOf(row.state),ready,row));
  let answer:Answer;try{answer=await bridge(c.env,"GET",`/v1/sessions/${row.id}`);}catch{return c.json(out(stateOf(row.state),true,row,false));}
  // The Bridge forgot it (its data was lost): connect again.
  const view=answer.status===404?{state:"logged_out" as const}:bridgeView.safeParse(answer.data).data;
  if(!view)return c.json(out(stateOf(row.state),true,row,false));
  const phoneEnd="phoneEnd" in view?view.phoneEnd:undefined;
  if(view.state!==row.state||(phoneEnd&&phoneEnd!==row.phoneEnd))await keep(c.env.DB,row.id,view.state,phoneEnd);
  return c.json(out(view.state,true,{...view,phoneEnd:phoneEnd??row.phoneEnd}));
 })
 // A new QR (the person accepted the risk). A connection that ended starts
 // over with a new name at the Bridge; one still alive is shown as it is,
 // unless fresh asks for a new start while it waits for the QR (only the
 // first QR of a connection lasts a minute; the next ones, 20 seconds).
 .post("/session",dealsWrite,validate("json",z.object({acceptRisk:z.literal(true),fresh:z.boolean().optional()}).strict()),async c=>{
  if(!bridgeReady(c.env))return c.json({code:"WHATSAPP_UNAVAILABLE"},503);
  const owner=c.get("session").user.id,now=Date.now();
  let row=await rowOf(c.env.DB,owner);
  if(row&&(ended.has(row.state)||(c.req.valid("json").fresh&&row.state==="pairing"))){
   try{await bridge(c.env,"DELETE",`/v1/sessions/${row.id}`);}catch{}
   const next=crypto.randomUUID();
   await c.env.DB.batch([
    c.env.DB.prepare("UPDATE deals_whatsapp SET id=?,state='pairing',risk_accepted_at=?,updated_at=? WHERE id=?").bind(next,now,now,row.id),
    c.env.DB.prepare("UPDATE deals_channel SET session_id=? WHERE session_id=?").bind(next,row.id),
   ]);
   row={...row,id:next,state:"pairing"};
  }
  if(!row){
   const id=crypto.randomUUID();
   await c.env.DB.prepare("INSERT INTO deals_whatsapp(id,user_id,state,risk_accepted_at,created_at,updated_at) VALUES(?,?,'pairing',?,?,?) ON CONFLICT(user_id) DO NOTHING").bind(id,owner,now,now,now).run();
   row=await rowOf(c.env.DB,owner);
   if(!row)return c.json({code:"WHATSAPP_UNAVAILABLE"},503);
  }
  let answer:Answer;try{answer=await bridge(c.env,"POST",`/v1/sessions/${row.id}`,{});}catch{return c.json({code:"WHATSAPP_UNAVAILABLE"},503);}
  if(answer.status===503)return c.json({code:"BRIDGE_BUSY"},503);
  const view=bridgeView.safeParse(answer.data);
  if(answer.status!==200||!view.success)return c.json({code:"WHATSAPP_UNAVAILABLE"},503);
  await keep(c.env.DB,row.id,view.data.state,view.data.phoneEnd,now);
  return c.json(out(view.data.state,true,view.data));
 })
 // Disconnecting: the Bridge logs the WhatsApp out and forgets it; its
 // Canais stay, waiting for a new connection.
 .delete("/session",async c=>{
  const owner=c.get("session").user.id,row=await rowOf(c.env.DB,owner);
  if(!row)return c.json({code:"NOT_FOUND"},404);
  if(bridgeReady(c.env)){try{await bridge(c.env,"DELETE",`/v1/sessions/${row.id}`);}catch{}}
  await keep(c.env.DB,row.id,"logged_out",null);
  return c.json({ok:true});
 })
 // The Canais the connected WhatsApp owns or runs (never the ones it only
 // follows), and which are already in Blup Deals.
 .get("/channels",async c=>{
  const owner=c.get("session").user.id,row=await rowOf(c.env.DB,owner);
  if(!row||row.state!=="connected"||!bridgeReady(c.env))return c.json({code:"NOT_CONNECTED"},409);
  const list=await bridgeList(c.env,row.id);
  if(!list)return c.json({code:"WHATSAPP_UNAVAILABLE"},503);
  const added=await c.env.DB.prepare("SELECT target_hash AS hash FROM deals_channel WHERE user_id=? AND platform='whatsapp'").bind(owner).all<{hash:string}>();
  const have=new Set(added.results.map(r=>r.hash));
  return c.json({channels:await Promise.all(list.map(async ch=>({id:ch.id,name:ch.name||"Canal",role:ch.role,added:have.has(await targetHash("whatsapp",ch.id))})))});
 })
 .post("/channels",dealsWrite,validate("json",z.object({channel:z.string().regex(/^[0-9]{6,32}$/)}).strict()),async c=>{
  const owner=c.get("session").user.id,row=await rowOf(c.env.DB,owner);
  if(!row||row.state!=="connected"||!bridgeReady(c.env))return c.json({code:"NOT_CONNECTED"},409);
  // The name comes from WhatsApp, never from the request.
  const list=await bridgeList(c.env,row.id),pick=list?.find(ch=>ch.id===c.req.valid("json").channel);
  if(!list)return c.json({code:"WHATSAPP_UNAVAILABLE"},503);
  if(!pick)return c.json({code:"NOT_FOUND"},404);
  let saved:Awaited<ReturnType<typeof addChannel>>;
  try{saved=await addChannel(c.env,owner,"whatsapp",pick.id,cleanLine(pick.name,80)||"WhatsApp",Date.now(),row.id);}catch{return c.json({code:"CHANNEL_UNAVAILABLE"},503);}
  if(!saved)return c.json({code:"QUOTA"},409);
  return c.json({id:saved.id},saved.created?201:200);
 });

async function bridgeList(env:Cloudflare.Env,id:string){
 try{const answer=await bridge(env,"GET",`/v1/sessions/${id}/channels`);const parsed=bridgeChannels.safeParse(answer.data);return answer.status===200&&parsed.success?parsed.data.channels:null;}catch{return null;}
}

// The Bridge's notice that a connection changed (signed with its own
// secret, within five minutes). Repeating one is harmless: it only sets
// the state again.
const notice=z.object({session:z.string().uuid(),state:z.enum(states),at:z.string().max(64)});
export const whatsappCallbackApi=new Hono<DealsApp>()
 .post("/callback",async c=>{
  const secret=c.env.DEALS_BRIDGE_CALLBACK_SECRET??"";
  if(secret.length<MIN_SECRET)return c.json({code:"NOT_FOUND"},404);
  const raw=await c.req.text();
  if(raw.length>4096)return c.json({code:"TOO_LARGE"},413);
  const at=c.req.header("x-bridge-timestamp")??"",nonce=c.req.header("x-bridge-nonce")??"",got=c.req.header("x-bridge-signature")??"";
  const when=Number(at),url=new URL(c.req.url);
  if(!/^[0-9]{1,12}$/.test(at)||Math.abs(Date.now()/1000-when)>300||!/^[0-9a-f]{32}$/.test(nonce))return c.json({code:"UNAUTHORIZED"},401);
  if(!sameSecret(got,await signature(secret,at,nonce,"POST",url.pathname+url.search,raw)))return c.json({code:"UNAUTHORIZED"},401);
  let body:unknown;try{body=JSON.parse(raw);}catch{return c.json({code:"BAD_REQUEST"},400);}
  const parsed=notice.safeParse(body);if(!parsed.success)return c.json({code:"BAD_REQUEST"},400);
  const row=await c.env.DB.prepare("SELECT id FROM deals_whatsapp WHERE id=?").bind(parsed.data.session).first();
  if(row)await keep(c.env.DB,parsed.data.session,parsed.data.state,undefined);
  return c.json({ok:true});
 });

