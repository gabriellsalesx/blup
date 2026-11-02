import { manifestKey,manifestSchema,HASH } from "./site-rules";
import { siteLock,unlockSite } from "./quota";
import { deletePrefix } from "./lifecycle";

// A gc: entry carries R2's opaque pagination cursor. New publication
// restarts the scan; it never deletes files referenced by a retained deploy.
export async function cleanSites(env:Cloudflare.Env,now=Date.now()) {
 const due=await env.DB.prepare("SELECT site_id AS id FROM retired_deploy WHERE delete_after<=? ORDER BY delete_after LIMIT 1").bind(now).all<{id:string}>();
 for(const {id} of due.results){
  const token=await siteLock(env.DB,id);if(!token)continue;
  try{
   const current=await env.DB.prepare("SELECT deploy FROM site WHERE id=?").bind(id).first<{deploy:string}>();
   if(!current){
    await deletePrefix(env.SITES,`s/${id}/`);
    await env.DB.prepare("DELETE FROM retired_deploy WHERE site_id=?").bind(id).run();continue;
   }
   const old=await env.DB.prepare("SELECT deploy FROM retired_deploy WHERE site_id=? AND delete_after<=? AND deploy NOT LIKE 'gc:%' LIMIT 25").bind(id,now).all<{deploy:string}>();
   for(const {deploy} of old.results){
    if(deploy!==current.deploy)await env.SITES.delete(manifestKey(id,deploy));
    await env.DB.prepare("DELETE FROM retired_deploy WHERE site_id=? AND deploy=?").bind(id,deploy).run();
   }
   const retained=await env.DB.prepare("SELECT deploy FROM retired_deploy WHERE site_id=? AND deploy NOT LIKE 'gc:%' LIMIT 100").bind(id).all<{deploy:string}>();
   if(retained.results.length>=100)continue;
   const keep=new Set<string>();let valid=true;
   for(const deploy of [current.deploy,...retained.results.map(r=>r.deploy)].filter(Boolean)){
    const obj=await env.SITES.get(manifestKey(id,deploy));
    const parsed=manifestSchema.safeParse(obj?await obj.json().catch(()=>null):null);
    if(!parsed.success){valid=false;break;}
    for(const [hash] of Object.values(parsed.data))keep.add(hash);
   }
   if(!valid)continue;
   const job=await env.DB.prepare("SELECT deploy FROM retired_deploy WHERE site_id=? AND deploy LIKE 'gc:%' AND delete_after<=? LIMIT 1").bind(id,now).first<{deploy:string}>();
   if(!job)continue;
   const cursor=job.deploy.slice(3)||undefined;
   const prefix=`s/${id}/`,filesPrefix=`s/${id}/f/`,tmpPrefix=`s/${id}/tmp/`;
   const page=await env.SITES.list({prefix,cursor,limit:99});
   const unused=page.objects.filter(o=>(o.key.startsWith(tmpPrefix)||o.key.startsWith(filesPrefix)&&HASH.test(o.key.slice(filesPrefix.length))&&!keep.has(o.key.slice(filesPrefix.length)))&&o.uploaded.getTime()<now-600_000);
   if(unused.length){
    const hashes=unused.filter(o=>o.key.startsWith(filesPrefix)).map(o=>o.key.slice(filesPrefix.length));
    const rows=hashes.length?await env.DB.prepare(`SELECT hash,size FROM site_blob WHERE site_id=? AND hash IN (${hashes.map(()=>"?").join(",")})`).bind(id,...hashes).all<{hash:string;size:number}>():{results:[]};
    await env.SITES.delete(unused.map(o=>o.key));
    if(hashes.length)await env.DB.batch([
     env.DB.prepare(`DELETE FROM site_blob WHERE site_id=? AND hash IN (${hashes.map(()=>"?").join(",")})`).bind(id,...hashes),
     env.DB.prepare("UPDATE site SET bytes=max(0,bytes-?) WHERE id=?").bind(rows.results.reduce((n,r)=>n+r.size,0),id),
    ]);
   }
   await env.DB.batch([
    env.DB.prepare("DELETE FROM retired_deploy WHERE site_id=? AND deploy=?").bind(id,job.deploy),
    ...(page.truncated?[env.DB.prepare("INSERT INTO retired_deploy(site_id,deploy,delete_after) VALUES (?,?,?)").bind(id,`gc:${page.cursor}`,now)]:[]),
   ]);
  }finally{await unlockSite(env.DB,id,token);}
 }
}
