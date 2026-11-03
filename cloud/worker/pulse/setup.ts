import {Hono} from "hono";
import type {PulseApp} from "./access";
// The first steps of an account, read from what it already has: a Pulse
// plan, a project, an error received, a monitor, an alert destination on a
// project and a status page. Nothing is marked by hand.
export const setupSteps=["plan","project","event","monitor","alert","page"] as const;
export type SetupStep=(typeof setupSteps)[number];
type Found={project:number;event:number;monitor:number;alert:number;page:number};
export const setupApi=new Hono<PulseApp>()
 .get("/",async c=>{
  const owner=c.get("session").user.id,db=c.env.DB;
  const found=await db.prepare(`SELECT
   EXISTS(SELECT 1 FROM pulse_project WHERE user_id=?1 AND deleted_at IS NULL) AS project,
   EXISTS(SELECT 1 FROM pulse_project WHERE user_id=?1 AND deleted_at IS NULL AND first_event_at IS NOT NULL) AS event,
   EXISTS(SELECT 1 FROM pulse_monitor WHERE user_id=?1) AS monitor,
   EXISTS(SELECT 1 FROM pulse_project_channel pc JOIN pulse_channel ch ON ch.id=pc.channel_id JOIN pulse_project p ON p.id=pc.project_id WHERE ch.user_id=?1 AND p.deleted_at IS NULL) AS alert,
   EXISTS(SELECT 1 FROM pulse_page WHERE user_id=?1) AS page`).bind(owner).first<Found>();
  let event=!!found?.event;
  // The first error is noted on the project the first time it is seen
  // here, so the project objects are asked only until then.
  if(found?.project&&!event){
   const projects=await db.prepare("SELECT id,object_name AS objectName FROM pulse_project WHERE user_id=? AND deleted_at IS NULL AND first_event_at IS NULL ORDER BY id LIMIT 10").bind(owner).all<{id:string;objectName:string}>();
   for(const p of projects.results){if(await c.env.PULSE_PROJECT.getByName(p.objectName).hasIssues()){await db.prepare("UPDATE pulse_project SET first_event_at=? WHERE id=? AND first_event_at IS NULL").bind(Date.now(),p.id).run();event=true;break;}}
  }
  const done:Record<SetupStep,boolean>={plan:!!c.get("access").plan,project:!!found?.project,event,monitor:!!found?.monitor,alert:!!found?.alert,page:!!found?.page};
  return c.json({steps:setupSteps.map(id=>({id,done:done[id]}))});
 });
