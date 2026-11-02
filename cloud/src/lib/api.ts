// The wire types come from the complete Hono route chain. Dates here are
// JSON strings, rather than the database's internal Date objects.
import {hc,parseResponse,type InferResponseType} from "hono/client";
import type {AppType} from "./rpc";
export const rpc=hc<AppType>("/",{init:{credentials:"same-origin"}}).api;
export type Me=InferResponseType<typeof rpc.me.$get,200>;
export type PlanInfo=InferResponseType<typeof rpc.plans.$get,200>[number];
export type Limits=Me["plan"]["limits"];
export type AIShare=Me["ai"];
export type Device=InferResponseType<typeof rpc.devices.$get,200>["devices"][number];
export type Site=InferResponseType<typeof rpc.sites.$get,200>["sites"][number];
export const api={
 me:()=>parseResponse(rpc.me.$get()),
 plans:()=>parseResponse(rpc.plans.$get()),
 devices:()=>parseResponse(rpc.devices.$get()),
 sites:()=>parseResponse(rpc.sites.$get()),
 takeDown:async(id:string)=>{await parseResponse(rpc.sites[":id"].$delete({param:{id}}));},
 disconnect:async(id:string)=>{await parseResponse(rpc.devices[":id"].$delete({param:{id}}));},
};

export const money = (cents: number, currency: string = "brl") => (cents / 100).toLocaleString("pt-BR", { style: "currency", currency: currency.toUpperCase() });
