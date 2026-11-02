import {afterEach,describe,expect,it} from "vitest";
import {render} from "svelte/server";
import LanguagePicker from "./LanguagePicker.svelte";
import {i18n} from "../i18n.svelte";
afterEach(()=>{i18n.locale="en";i18n.staff=false;});
describe("site-style language selector",()=>{
 it.each(["pt","en","es"] as const)("renders accessible %s selection and native language names",locale=>{
  i18n.locale=locale;
  const {body}=render(LanguagePicker);
  expect(body).toContain(`lang="${locale}" aria-checked="true"`);
  for(const name of ["Português","English","Español"])expect(body).toContain(name);
  expect(body).toContain('role="menu"');expect(body).toContain("popovertarget=");
 });
});
