import {describe,expect,it,vi} from "vitest";
import {refreshOnReturn} from "./refresh-on-return";

// A tab: its visibility, what has the focus and whether a dialog is open.
function tab(){
 const target=new EventTarget();
 const state={visibilityState:"visible" as DocumentVisibilityState,focused:false,dialog:false};
 const page={
  get visibilityState(){return state.visibilityState;},
  get activeElement(){return {matches:()=>state.focused} as unknown as Element;},
  querySelector:(selector:string)=>state.dialog&&selector==="dialog[open]"?({} as Element):null,
  addEventListener:target.addEventListener.bind(target),
  removeEventListener:target.removeEventListener.bind(target),
 } as unknown as Document;
 const show=(visibility:DocumentVisibilityState="visible")=>{state.visibilityState=visibility;target.dispatchEvent(new Event("visibilitychange"));};
 return {page,state,show};
}

describe("refresh when the tab is seen again",()=>{
 it("loads again only after the gap, when visible, on the pages that allow it",async()=>{
  let clock=0;const refresh=vi.fn(async()=>{}),{page,show}=tab();let allowed=true;
  const stop=refreshOnReturn(refresh,()=>allowed,page,30_000,()=>clock);
  clock=10_000;show();expect(refresh).not.toHaveBeenCalled();
  clock=40_000;show("hidden");expect(refresh).not.toHaveBeenCalled();
  show();expect(refresh).toHaveBeenCalledOnce();await Promise.resolve();await Promise.resolve();
  clock=50_000;show();expect(refresh).toHaveBeenCalledOnce();
  clock=90_000;allowed=false;show();expect(refresh).toHaveBeenCalledOnce();
  allowed=true;stop();show();expect(refresh).toHaveBeenCalledOnce();
 });
 it("never interrupts a dialog or a field being typed in",()=>{
  let clock=0;const refresh=vi.fn(async()=>{}),{page,state,show}=tab();
  refreshOnReturn(refresh,()=>true,page,30_000,()=>clock);
  clock=60_000;state.dialog=true;show();state.dialog=false;state.focused=true;show();expect(refresh).not.toHaveBeenCalled();
  state.focused=false;show();expect(refresh).toHaveBeenCalledOnce();
 });
});
