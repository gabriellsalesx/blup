// Removing source maps (R2 objects and their D1 rows), apart from the
// routes so the clean-up can use it without the routes' imports.
export const SOURCEMAP_KEEP_MS=90*86_400_000;
// Maps of a removed project, or sent more than 90 days ago.
export async function dropSourcemaps(env:Cloudflare.Env,where:{project:string}|{before:number}){
 const rows=await ("project" in where?env.DB.prepare("SELECT project_id AS projectId,release,file,object_key AS objectKey FROM pulse_sourcemap WHERE project_id=? LIMIT 500").bind(where.project):env.DB.prepare("SELECT project_id AS projectId,release,file,object_key AS objectKey FROM pulse_sourcemap WHERE uploaded_at<? LIMIT 500").bind(where.before)).all<{projectId:string;release:string;file:string;objectKey:string}>();
 if(!rows.results.length)return;
 await env.PULSE_SOURCEMAPS.delete(rows.results.map(r=>r.objectKey));
 await env.DB.batch(rows.results.map(r=>env.DB.prepare("DELETE FROM pulse_sourcemap WHERE project_id=? AND release=? AND file=?").bind(r.projectId,r.release,r.file)));
}
