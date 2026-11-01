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
		const slug = slugOfHost(url.hostname, env.SITES_DOMAIN);
		const res = slug ? await serveSite(req, env, ctx, slug) : new Response("Not found", { status: 404 });
		// Set on the answer itself: a copy would lose that its body is already
		// gzipped and get it compressed twice.
		for (const [k, v] of Object.entries(HEADERS)) if (!res.headers.has(k)) res.headers.set(k, v);
		return res;
	},
} satisfies ExportedHandler<Cloudflare.Env>;
