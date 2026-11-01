import { z } from "zod";

// The rules of the sites published through Blup Cloud, apart from any
// binding so the tests can reach them: names, paths, the manifest of a
// deploy and how each file is cached.

// A site's name is its address: <slug>.getblup.site. One DNS label, lower
// case, 3 to 40 characters, no dash at either end.
export const SLUG = /^[a-z0-9](?:[a-z0-9-]{1,38}[a-z0-9])$/;

// Names Blup keeps for itself (or that would mislead a visitor).
const RESERVED = new Set([
	"www", "api", "app", "cloud", "staff", "admin", "blup", "getblup", "mail", "email", "smtp", "ftp",
	"status", "help", "support", "docs", "blog", "dashboard", "login", "signin", "signup", "account",
	"billing", "pay", "security", "static", "assets", "cdn", "test", "dev", "staging",
]);

export const slugAllowed = (s: string) => slugProblem(s) === null;

// Why a name can't be used: its shape, or Blup keeps it.
export function slugProblem(s: string): "invalid" | "reserved" | null {
	if (!SLUG.test(s) || s.includes("--")) return "invalid";
	return RESERVED.has(s) ? "reserved" : null;
}

// Names close to one that is taken, to offer instead (the ones free are
// checked against the sites).
export function slugCandidates(base: string): string[] {
	const b = base.replace(/-?\d+$/, "").slice(0, 34) || "site";
	const out = [`${b}-site`, `${b}-blog`, `meu-${b}`, `${b}-online`, ...[2, 3, 4, 5].map((n) => `${b}-${n}`)];
	return [...new Set(out)].filter((s) => s !== base && slugAllowed(s));
}

// The site a request is for: "ana" for ana.getblup.site (or ana.localhost on
// this computer); null for any other host.
export function slugOfHost(host: string, domain: string): string | null {
	if (!domain) return null;
	const h = host.toLowerCase();
	const suffix = `.${domain.toLowerCase()}`;
	if (!h.endsWith(suffix)) return null;
	const slug = h.slice(0, -suffix.length);
	return SLUG.test(slug) ? slug : null;
}

export const HASH = /^[0-9a-f]{64}$/;

// Where a site's things live in R2 (the SITES bucket), all under s/<site>/:
// f/<sha256> a file as stored, d/<deploy> the manifest of a deploy.
export const fileKey = (siteId: string, hash: string) => `s/${siteId}/f/${hash}`;
export const manifestKey = (siteId: string, deploy: string) => `s/${siteId}/d/${deploy}`;

// A path of the site as Hugo wrote it ("/posts/oi/index.html"): absolute,
// no dot segments, no control characters, not one of Cloudflare Pages'
// special files.
const SPECIAL = new Set(["/_headers", "/_redirects", "/_routes.json", "/_worker.js"]);
export function pathAllowed(p: string): boolean {
	if (!p.startsWith("/") || p.length > 1024 || SPECIAL.has(p)) return false;
	if (/[\u0000-\u001f\u007f\\]/.test(p)) return false;
	return !p.split("/").some((seg) => seg === "." || seg === "..");
}

// A deploy as the Studio sends it: each file's path, the hash of what is
// stored, its media type and whether it is stored gzipped.
export const deploySchema = z.object({
	files: z
		.array(
			z.object({
				path: z.string().refine(pathAllowed),
				hash: z.string().regex(HASH),
				type: z.string().regex(/^[a-z0-9.+-]+\/[a-z0-9.+-]+(?:; ?charset=utf-8)?$/i).max(100),
				gz: z.boolean(),
			}),
		)
		.min(1)
		.max(20_000)
		.refine((files) => new Set(files.map((f) => f.path)).size === files.length, "duplicate path"),
});
export type DeployFiles = z.infer<typeof deploySchema>["files"];

// The manifest kept in R2: path -> [hash, type, gzipped].
export type Manifest = Record<string, [string, string, 0 | 1]>;

export const toManifest = (files: DeployFiles): Manifest =>
	Object.fromEntries(files.map((f) => [f.path, [f.hash, f.type, f.gz ? 1 : 0] as [string, string, 0 | 1]]));

// What a request's path serves: the file, a redirect to the folder's
// address (with the closing slash, as Hugo links it) or nothing.
export type Resolved = { file: string } | { redirect: string } | null;

export function resolvePath(manifest: Manifest, pathname: string): Resolved {
	let p: string;
	try {
		p = decodeURIComponent(pathname);
	} catch {
		return null;
	}
	if (!pathAllowed(p)) return null;
	if (p.endsWith("/")) return manifest[`${p}index.html`] ? { file: `${p}index.html` } : null;
	if (manifest[p]) return { file: p };
	if (manifest[`${p}/index.html`]) return { redirect: `${pathname}/` };
	return null;
}

// How long browsers keep a file: pages, feeds and the service worker are
// checked every time (the ETag answers 304 when nothing changed); files
// with a fingerprint in the name never change; the rest, an hour.
const FINGERPRINT = /(?:\.[0-9a-f]{20,64}\.|_hu_?[0-9a-f]{8,}|_hu[0-9a-z]{6,}_)/;
export function cacheControl(path: string, type: string): string {
	if (/^(?:text\/html|application\/(?:rss\+)?xml|application\/json|application\/manifest\+json|text\/xml)/.test(type) || path.endsWith("/sw.js")) {
		return "public, max-age=0, must-revalidate";
	}
	if (FINGERPRINT.test(path)) return "public, max-age=31536000, immutable";
	return "public, max-age=3600";
}
