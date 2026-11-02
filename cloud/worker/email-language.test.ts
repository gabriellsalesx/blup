import {afterEach,beforeEach,describe,expect,it,vi} from "vitest";
import {TestD1,seedUser} from "./security-test-support";
import {downgradeNotices} from "./lifecycle";
import {sendCode,sendReminder,sendDowngrade} from "./email";
import {languageTag} from "../shared/language";
let db:TestD1,env:Cloudflare.Env;
// The fixture supplies only bindings exercised by the mail functions.
let sent:{subject:string;text:string;html:string}[];
beforeEach(()=>{db=new TestD1();seedUser(db,"mail");sent=[];env={DB:db.asD1(),APP_URL:"https://cloud.audit.invalid",EMAIL_MODE:"send",MAIL_FROM:"Blup <fixture@audit.invalid>",EMAIL:{send:async(message:{subject:string;text:string;html:string})=>{sent.push(message);}}} as unknown as Cloudflare.Env;vi.stubGlobal("fetch",vi.fn(()=>{throw new Error("External network forbidden");}));});
afterEach(()=>{db.close();vi.unstubAllGlobals();});
describe("transactional mail account preference",()=>{
 it("preserves Portuguese for existing accounts when migration 0014 is applied",()=>{
  const legacy=new TestD1("0013_webhook_stripe_atual.sql");
  try{seedUser(legacy,"legacy");legacy.applyMigration("0014_idioma_da_conta.sql");expect(legacy.sqlite.prepare("SELECT language FROM user WHERE id='legacy'").get()?.language).toBe("pt");}
  finally{legacy.close();}
 });

 it("keeps the saved preference when the recipient address has uppercase letters",async()=>{
  db.sqlite.exec("UPDATE user SET language='es' WHERE id='mail'");
  await sendCode(env,"MAIL@AUDIT.INVALID","123456","sign-in","en");expect(sent[0]?.html).toContain('lang="es"');
 });
 it("uses the persisted account language in the actual downgrade cron",async()=>{
  const deadline=Date.now()+14*86400000;
  db.sqlite.exec("UPDATE user SET language='es' WHERE id='mail'; INSERT INTO sync_account(user_id,wrapped_key,storage_bytes) VALUES ('mail','{}',1073741824)");
  db.sqlite.prepare("INSERT INTO resource_quota(user_id,last_plan,grace_until,storage_grace_limit) VALUES ('mail','max',?,5368709120)").run(deadline);
  await downgradeNotices(env);expect(sent).toHaveLength(1);expect(sent[0]?.subject).toBe("Sobre el cambio de tu plan de Blup");
 });

 it.each(["pt","en","es"] as const)("uses saved %s for sign-in, reminders and plan notices",async language=>{
  db.sqlite.prepare("UPDATE user SET language=? WHERE id='mail'").run(language);
  await sendCode(env,"mail@audit.invalid","012345","sign-in",language==="en"?"pt":"en");
  expect(await sendReminder(env,"mail@audit.invalid")).toBe(true);
  expect(await sendDowngrade(env,"mail@audit.invalid",{sites:{limit:1,deadline:Date.UTC(2026,10,16)},domainDisableAt:Date.UTC(2026,10,16),storage:{usedBytes:2**30,limitBytes:50*2**20,deadline:Date.UTC(2026,10,16)}})).toBe(true);
  expect(sent).toHaveLength(3);for(const mail of sent){expect(mail.html).toContain(`lang="${languageTag[language]}"`);expect(mail.text).not.toMatch(/[—–]/);expect(mail.html).toContain("/email/blup.png");}
  expect(sent[0]?.subject).toBe({pt:"Seu código para entrar no Blup",en:"Your Blup sign-in code",es:"Tu código para entrar en Blup"}[language]);
  expect(sent[0]?.text).toContain("012345");expect(sent[2]?.text).toContain("UTC");expect(sent[2]?.text).not.toMatch(/\{\w+\}/);
 });
 it("uses request language only before the email belongs to an account",async()=>{
  await sendCode(env,"new@audit.invalid","123456","change-email","es");expect(sent[0]?.subject).toContain("Confirma");expect(sent[0]?.html).toContain('lang="es"');
 });
 it("escapes unexpected values in email HTML",async()=>{
  await sendCode(env,"mail@audit.invalid",'<script>alert("x")</script>',"sign-in");expect(sent[0]?.html).not.toContain("<script>");expect(sent[0]?.html).toContain("&lt;script&gt;");
 });
 it("does not send to a passkey placeholder address",async()=>{
  await sendCode(env,"fixture@sem-email.blup.invalid","123456","sign-in","en");expect(await sendReminder(env,"fixture@sem-email.blup.invalid")).toBe(false);expect(sent).toEqual([]);
 });
});
