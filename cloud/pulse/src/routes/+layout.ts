import {rpc,pulseApi,parseResponse} from "../lib/api";
import {panel} from "../lib/panel.svelte";
export const ssr=false;
// Who is signed in and their Pulse plan, once, before any page loads; the
// pages wait for it (`parent`) and stay empty when nobody is signed in.
export async function load(){
 if(!panel.ready){try{const [me,usage]=await Promise.all([parseResponse(rpc.me.$get()),parseResponse(pulseApi.usage.$get())]);panel.me=me;panel.usage=usage;}catch{panel.error=true;}panel.ready=true;}
 return {signedIn:!!panel.me};
}
