import {it,expect} from "vitest";
// Worker types intentionally omit Node. This import only runs in Vitest.
// @ts-expect-error test-only Node builtin, no additional dependency
import * as fs from "node:fs";
// The Workers runtime has no `redirect: "error"` (it throws before the
// request leaves; Cloudflare's Request docs): calls use "manual" and treat a
// 3xx as a failure. The tests mock fetch, so this rule is checked in the
// source itself.
const readFileSync=fs.readFileSync as (path:URL,encoding:string)=>string;
const readdirSync=fs.readdirSync as (path:URL,options:{withFileTypes:true})=>{name:string;isDirectory():boolean}[];
const moduleURL=(import.meta as unknown as {url:string}).url;
function sources(dir:URL):URL[]{return readdirSync(dir,{withFileTypes:true}).flatMap(entry=>entry.isDirectory()?sources(new URL(`${entry.name}/`,dir)):entry.name.endsWith(".ts")&&!entry.name.endsWith(".test.ts")?[new URL(entry.name,dir)]:[]);}
it("never asks the Workers fetch for redirect: error",()=>{
 const offenders=sources(new URL("./",moduleURL)).filter(file=>/redirect\s*:\s*["']error["']/.test(readFileSync(file,"utf8"))).map(file=>file.pathname);
 expect(offenders).toEqual([]);
});
