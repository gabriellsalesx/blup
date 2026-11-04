import {shortCodeSchema,dayOf} from "./store";

// Link previews (Telegram, Discord and others) open the link too; they
// are not people, so they do not count as clicks.
const robots=/bot|crawler|spider|preview|facebookexternalhit|embedly|slack|discord|telegram|whatsapp|curl|wget|python|headless/i;
// deals.getblup.site/r/:code: the post's link, and one more click on the
// day (per post and day; no IP or visitor kept). Only while the account
// may be served (as lifecycle.accountAllowed): a banned or deleted
// account's links stop at once, before the purge removes them.
export async function serveDealsLink(req:Request,env:Cloudflare.Env):Promise<Response>{
 const url=new URL(req.url),code=shortCodeSchema.safeParse(/^\/r\/([^/]+)\/?$/.exec(url.pathname)?.[1]);
 const notFound=()=>new Response("Not found",{status:404,headers:{"content-type":"text/plain; charset=utf-8","cache-control":"no-store"}});
 if(!code.success||!["GET","HEAD"].includes(req.method))return notFound();
 const post=await env.DB.prepare("SELECT p.id,p.user_id AS userId,p.link FROM deals_post p JOIN user u ON u.id=p.user_id LEFT JOIN resource_quota q ON q.user_id=u.id WHERE p.code=? AND u.banned=0 AND q.deleted_at IS NULL").bind(code.data).first<{id:string;userId:string;link:string}>();
 let target:URL;try{target=new URL(post?.link??"");}catch{return notFound();}
 if(!post||target.protocol!=="https:")return notFound();
 if(req.method==="GET"&&!robots.test(req.headers.get("user-agent")??""))
  await env.DB.prepare("INSERT INTO deals_click_day(post_id,user_id,day,clicks) VALUES(?,?,?,1) ON CONFLICT(post_id,day) DO UPDATE SET clicks=clicks+1").bind(post.id,post.userId,dayOf()).run();
 return new Response(null,{status:302,headers:{location:target.href,"cache-control":"no-store","referrer-policy":"no-referrer-when-downgrade"}});
}
