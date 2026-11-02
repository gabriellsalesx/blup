import {describe,expect,it,vi} from "vitest";
import {createLanguageSaver} from "./language-save";
describe("account language persistence",()=>{
 it("keeps the last choice even when it matches stale /me data",async()=>{
  const writes:string[]=[];let release!:()=>void;
  const gate=new Promise<void>(r=>{release=r;});
  const saver=createLanguageSaver(async language=>{writes.push(language);if(language==="en")await gate;},()=>true,()=>{});
  const first=saver.schedule("account","en","pt");
  const last=saver.schedule("account","pt","pt");
  await Promise.resolve();expect(writes).toEqual(["en"]);release();await first;await last;expect(writes).toEqual(["en","pt"]);
 });
 it("skips a saved initial language and duplicate successful writes",async()=>{
  const write=vi.fn(async()=>{}),saver=createLanguageSaver(write,()=>true,()=>{});
  await saver.schedule("account","pt","pt");expect(write).not.toHaveBeenCalled();
  await saver.schedule("account","es","pt");await saver.schedule("account","es","pt");expect(write).toHaveBeenCalledOnce();
 });
 it("drops queued work after the page/account is no longer current",async()=>{
  const write=vi.fn(async()=>{}),saver=createLanguageSaver(write,()=>false,()=>{});
  await saver.schedule("old","es","pt");expect(write).not.toHaveBeenCalled();
 });
 it("does not report an older failed write after a newer choice is queued",async()=>{
  let fail!:(error:Error)=>void;const gate=new Promise<void>((_yes,no)=>{fail=no;}),error=vi.fn();
  const saver=createLanguageSaver(language=>language==="en"?gate:Promise.resolve(),()=>true,error);
  void saver.schedule("account","en","pt");const last=saver.schedule("account","pt","pt");
  fail(new Error("old request"));await last;expect(error).not.toHaveBeenCalled();
 });
 it("allows retrying a failed preference without rejecting the queue",async()=>{
  const write=vi.fn().mockRejectedValueOnce(new Error("offline")).mockResolvedValue(undefined),error=vi.fn();
  const saver=createLanguageSaver(write,()=>true,error);
  await saver.schedule("account","es","pt");await saver.schedule("account","es","pt");expect(write).toHaveBeenCalledTimes(2);expect(error).toHaveBeenCalledOnce();
 });
});
