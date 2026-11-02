import {afterEach,describe,expect,it,vi} from "vitest";
import {render} from "svelte/server";
import Toaster from "./Toaster.svelte";
import {toast,toasts} from "../toast.svelte";
import {i18n,t} from "../i18n.svelte";
afterEach(()=>{toasts.splice(0);i18n.locale="en";i18n.staff=false;vi.useRealTimers();});
describe("notification language",()=>{
 it("updates an existing message and its close action when the language changes",()=>{
  vi.useFakeTimers();i18n.locale="en";
  toast(()=>t("nameSaved"));
  const english=render(Toaster).body;expect(english).toContain("Name saved");expect(english).toContain('aria-label="Close"');
  i18n.locale="es";const spanish=render(Toaster).body;expect(spanish).toContain("Nombre guardado");expect(spanish).toContain('aria-label="Cerrar"');
  i18n.staff=true;expect(render(Toaster).body).toContain("Nome salvo");
 });
});
