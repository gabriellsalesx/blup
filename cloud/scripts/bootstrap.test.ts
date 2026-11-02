import {it,expect,vi} from "vitest";
import {readFileSync} from "node:fs";
import ts from "typescript";
import {z} from "zod";
vi.mock("cloudflare:workers",()=>({DurableObject:class{}}));
import bootstrap from "../worker/bootstrap";
const config=z.object({name:z.string(),compatibility_date:z.string(),compatibility_flags:z.array(z.string()),migrations:z.array(z.object({tag:z.string(),new_sqlite_classes:z.array(z.string())})),durable_objects:z.object({bindings:z.array(z.object({name:z.string(),class_name:z.string()}))})});
const parse=(path:string)=>ts.parseConfigFileTextToJson(path,readFileSync(new URL(path,import.meta.url),"utf8")).config;
it("bootstraps exactly the final namespaces without routes, cron, assets or a circular service dependency",()=>{
 const raw=z.record(z.string(),z.unknown()).parse(parse("../wrangler.bootstrap.jsonc")),initial=config.parse(raw),final=config.extend({env:z.object({production:z.object({durable_objects:config.shape.durable_objects})})}).parse(parse("../wrangler.jsonc"));
 expect(initial.name).toBe(`${final.name}-production`);expect(initial.migrations).toEqual(final.migrations);expect(initial.durable_objects).toEqual(final.env.production.durable_objects);expect(initial.compatibility_flags).toEqual(final.compatibility_flags);expect(raw).toMatchObject({workers_dev:false,preview_urls:false,routes:[],triggers:{crons:[]}});expect(raw.services).toBeUndefined();expect(raw.assets).toBeUndefined();expect(raw.d1_databases).toBeUndefined();
});
it("keeps the bootstrap handler closed even if called directly",async()=>{
 const response=bootstrap.fetch();expect(response.status).toBe(503);expect(response.headers.get("cache-control")).toBe("no-store");expect(await response.text()).toBe("Service unavailable");
});
