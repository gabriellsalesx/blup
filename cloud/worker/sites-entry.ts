import { serveSite } from "./serve-site";
import { slugOfHost } from "./site-rules";

// The Worker of the published sites (<slug>.getblup.site), apart from the
// panel's: no assets, no auth, only the D1 rows of the sites and their files
// in R2 (wrangler.sites.jsonc).

const HEADERS: Record<string, string> = {
	"X-Content-Type-Options": "nosniff",
	"Referrer-Policy": "strict-origin-when-cross-origin",
	"Strict-Transport-Security": "max-age=31536000",
};

const domainCache=new Map<string,{slug:string;hostname:string;at:number}>();
async function resolveDomain(env:Cloudflare.Env,host:string,canonical:string){
 const cached=domainCache.get(host);if(cached&&Date.now()-cached.at<30_000)return cached;
 const row=await env.DB.prepare("SELECT s.slug,d.hostname FROM site_domain d JOIN site s ON s.id=d.site_id WHERE d.hostname IN (?,?) AND d.verified_at IS NOT NULL AND d.status='active' AND d.ssl='active' ORDER BY d.hostname=? DESC LIMIT 1").bind(host,canonical,host).first<{slug:string;hostname:string}>();
 if(!row)return null;
 if(domainCache.size>=1024){const first=domainCache.keys().next().value;if(first!==undefined)domainCache.delete(first);}
 const result={...row,at:Date.now()};domainCache.set(host,result);return result;
}

export default {
	async fetch(req: Request, env: Cloudflare.Env, ctx: ExecutionContext) {
		const url = new URL(req.url);
		const local = env.SITES_DOMAIN === "localhost";
		// Always HTTPS, and www.name.getblup.site is the same site as
		// name.getblup.site (one address, so search engines see one).
		const bare = url.hostname.startsWith("www.") ? url.hostname.slice(4) : url.hostname;
		if (!local && (url.protocol === "http:" || bare !== url.hostname) && slugOfHost(bare, env.SITES_DOMAIN)) {
			return new Response(null, { status: 301, headers: { location: `https://${bare}${url.pathname}${url.search}` } });
		}
		let custom=false;
		let slug = slugOfHost(url.hostname, env.SITES_DOMAIN);
		if(!slug){
			const host=url.hostname.toLowerCase();
			const canonical=host.startsWith("www.")?host.slice(4):host;
			const domain=await resolveDomain(env,url.hostname,canonical);
			if(domain){
				custom=true;
				if(host!==domain.hostname||url.protocol!=="https:")return new Response(null,{status:301,headers:{location:`https://${domain.hostname}${url.pathname}${url.search}`}});
				slug=domain.slug;
			}
		}
		const res = slug ? await serveSite(req, env, ctx, slug,custom) : new Response("Not found", { status: 404 });
		// Set on the answer itself: a copy would lose that its body is already
		// gzipped and get it compressed twice.
		for (const [k, v] of Object.entries(HEADERS)) if (!res.headers.has(k)) res.headers.set(k, v);
		return res;
	},
} satisfies ExportedHandler<Cloudflare.Env>;
