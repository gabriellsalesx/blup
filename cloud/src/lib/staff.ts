import {parseResponse,type InferResponseType} from "hono/client";
import {z} from "zod/mini";
import {rpc} from "./api";
import {money as formatMoney} from "./money";
const client=rpc.staff;
type StaffMe=InferResponseType<typeof client.me.$get,200>;
export type Role=StaffMe["role"];
export type Stats=InferResponseType<typeof client.stats.$get,200>;
export type Team=InferResponseType<typeof client.team.$get,200>;
export type AIModels=InferResponseType<typeof client.ai.$get,200>;
export type AIFeature=AIModels["features"][number]["feature"];
export type AccountDetail=InferResponseType<(typeof client.users)[":id"]["$get"],200>;
export type Addons=AccountDetail["addons"];
export class StaffError extends Error {
 constructor(public status:number,public code:string){super(code);}
}
async function call<T>(request:Promise<T>):Promise<T>{
 try{return await request;}
 catch(e){
  const parsed=z.object({statusCode:z.int(),detail:z.object({data:z.object({code:z.string()})})}).safeParse(e);
  if(parsed.success)throw new StaffError(parsed.data.statusCode,parsed.data.detail.data.code);
  throw e;
 }
}
export const staffApi={
 me:()=>call(parseResponse(client.me.$get())),
 stats:()=>call(parseResponse(client.stats.$get())),
 users:(q:string,cursor:string|null)=>call(parseResponse(client.users.$get({query:{q,...(cursor?{cursor}:{})}}))),
 user:(id:string)=>call(parseResponse(client.users[":id"].$get({param:{id}}))),
 revokeSessions:(id:string)=>call(parseResponse(client.users[":id"].sessions.revoke.$post({param:{id}}))),
 removePasskey:(id:string,pid:string)=>call(parseResponse(client.users[":id"].passkeys[":pid"].$delete({param:{id,pid}}))),
 ban:(id:string,reason:string)=>call(parseResponse(client.users[":id"].ban.$post({param:{id},json:{reason}}))),
 unban:(id:string)=>call(parseResponse(client.users[":id"].unban.$post({param:{id}}))),
 remove:(id:string)=>call(parseResponse(client.users[":id"].$delete({param:{id}}))),
 restore:(id:string)=>call(parseResponse(client.users[":id"].restore.$post({param:{id}}))),
 team:()=>call(parseResponse(client.team.$get())),
 setMember:(email:string,role:Role)=>call(parseResponse(client.team.$post({json:{email,role}}))),
 removeMember:(email:string)=>call(parseResponse(client.team[":email"].$delete({param:{email}}))),
 ai:()=>call(parseResponse(client.ai.$get())),
 setModels:(feature:AIFeature,models:string[],recommended:string|null)=>call(parseResponse(client.ai[":feature"].$put({param:{feature},json:{models,recommended}}))),
 log:(cursor:string|null)=>call(parseResponse(client.log.$get({query:cursor?{cursor}:{}}))),
};

// A staff page's address: /staff/accounts here, /accounts on staff.getblup.io
// (hooks.ts maps it back).
export const staffHref = (path: string) =>
	typeof location !== "undefined" && location.hostname.startsWith("staff.") ? path.replace(/^\/staff/, "") || "/" : path;

export const roleLabel = (r: Role) => (r === "admin" ? "Admin" : "Suporte");

export const actionLabel: Record<string, string> = {
	"sessions.revoke": "Encerrou as sessões",
	"passkey.remove": "Removeu uma passkey",
	"user.ban": "Bloqueou a conta",
	"user.unban": "Desbloqueou a conta",
	"user.delete": "Excluiu a conta",
	"user.restore": "Recuperou a conta",
	"team.set": "Mudou a equipe",
	"team.remove": "Tirou da equipe",
	"ai.models": "Mudou os modelos de IA",
	// Refused: only the owner (STAFF_OWNER_EMAIL) may do these.
	"denied.sessions.revoke": "Tentou encerrar as sessões (negado)",
	"denied.passkey.remove": "Tentou remover uma passkey (negado)",
	"denied.user.ban": "Tentou bloquear a conta (negado)",
	"denied.user.delete": "Tentou excluir a conta (negado)",
	"denied.team.set": "Tentou mudar um admin (negado)",
	"denied.team.remove": "Tentou tirar um admin da equipe (negado)",
};

// Stripe's money events, in the staff's words (and whether they need a look).
export const billingLabel: Record<string, { text: string; alert: boolean }> = {
	"invoice.paid": { text: "Pagamento recebido", alert: false },
	"invoice.payment_failed": { text: "Pagamento falhou", alert: true },
	"charge.refunded": { text: "Reembolso", alert: false },
	"charge.dispute.created": { text: "Contestação aberta", alert: true },
	"radar.early_fraud_warning.created": { text: "Alerta de fraude", alert: true },
};

export const money = (cents: number | null, currency: string | null) =>
	cents === null || currency === null ? "" : formatMoney(cents,currency);

export const when = (d: string | null) => d===null?"Sem data":new Date(d).toLocaleString("pt-BR", { day: "2-digit", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit" });
