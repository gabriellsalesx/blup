// The wire types come from the complete Hono route chain. Dates here are
// JSON strings, rather than the database's internal Date objects.
import {hc,parseResponse,type InferResponseType} from "hono/client";
import {language} from "./i18n.svelte";
import type {AppType} from "./rpc";
export const rpc=hc<AppType>("/",{init:{credentials:"same-origin"},headers:()=>({"x-blup-language":language()})}).api;
export type Me=InferResponseType<typeof rpc.me.$get,200>;
export type Limits=Me["plan"]["limits"];
export type AIShare=Me["ai"];
export type Device=InferResponseType<typeof rpc.devices.$get,200>["devices"][number];
export type Site=InferResponseType<typeof rpc.sites.$get,200>["sites"][number];
// AppSumo codes: the lifetime level of the account and its codes (last 4 only).
export type RedeemState=InferResponseType<typeof rpc.redeem.$get,200>;
export const api={
 config:()=>parseResponse(rpc.config.$get()),
 me:()=>parseResponse(rpc.me.$get()),
 setLanguage:(language:Me["language"])=>parseResponse(rpc.me.language.$put({json:{language}})),
 plans:()=>parseResponse(rpc.plans.$get()),
 devices:()=>parseResponse(rpc.devices.$get()),
 sites:()=>parseResponse(rpc.sites.$get()),
 selectRetained:(siteId:string)=>parseResponse(rpc.sites.retained.$put({json:{siteId}})),
 takeDown:async(id:string)=>{await parseResponse(rpc.sites[":id"].$delete({param:{id}}));},
 redeemState:()=>parseResponse(rpc.redeem.$get()),
 redeem:(code:string)=>parseResponse(rpc.redeem.$post({json:{code}})),
 disconnect:async(id:string)=>{await parseResponse(rpc.devices[":id"].$delete({param:{id}}));},
};

export { money } from "./money";
