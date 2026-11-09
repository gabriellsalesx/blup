import {pulseApi,parseResponse} from "../../lib/api";
import {early} from "../../lib/load";
import {checksGET,type Monitor,type History} from "../../lib/monitor";
export async function load({parent}){
 const empty={monitors:[] as Monitor[],histories:{} as Record<string,History>};
 return early(async()=>{
  const monitors=(await parseResponse(pulseApi.monitors.$get())).monitors;
  const loaded=await Promise.all(monitors.map(async m=>[m.id,await parseResponse(checksGET({param:{monitor:m.id}}))] as const));
  return {monitors,histories:Object.fromEntries(loaded)};
 },parent,empty);
}
