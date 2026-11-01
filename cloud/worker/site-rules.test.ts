import { describe, expect, it } from "vitest";
import { cacheControl, deploySchema, pathAllowed, resolvePath, slugAllowed, slugOfHost, type Manifest } from "./site-rules";

const H = "a".repeat(64);

describe("site names", () => {
	it("are one DNS label, not kept by Blup", () => {
		expect(slugAllowed("revista-nori")).toBe(true);
		expect(slugAllowed("ab")).toBe(false);
		expect(slugAllowed("-ana")).toBe(false);
		expect(slugAllowed("ana-")).toBe(false);
		expect(slugAllowed("Ana")).toBe(false);
		expect(slugAllowed("a--b")).toBe(false);
		expect(slugAllowed("www")).toBe(false);
		expect(slugAllowed("cloud")).toBe(false);
	});
	it("come from the host only under the sites' domain", () => {
		expect(slugOfHost("ana.getblup.site", "getblup.site")).toBe("ana");
		expect(slugOfHost("ANA.getblup.site", "getblup.site")).toBe("ana");
		expect(slugOfHost("getblup.site", "getblup.site")).toBeNull();
		expect(slugOfHost("a.b.getblup.site", "getblup.site")).toBeNull();
		expect(slugOfHost("cloud.getblup.io", "getblup.site")).toBeNull();
		expect(slugOfHost("ana.localhost", "localhost")).toBe("ana");
		expect(slugOfHost("ana.getblup.site", "")).toBeNull();
	});
});

describe("paths", () => {
	it("refuse dot segments and special files", () => {
		expect(pathAllowed("/posts/oi/index.html")).toBe(true);
		expect(pathAllowed("posts/x")).toBe(false);
		expect(pathAllowed("/../etc")).toBe(false);
		expect(pathAllowed("/a/./b")).toBe(false);
		expect(pathAllowed("/_headers")).toBe(false);
		expect(pathAllowed("/a\\b")).toBe(false);
	});
	it("serve folders by their index, with the closing slash", () => {
		const m: Manifest = { "/index.html": [H, "text/html", 1], "/posts/oi/index.html": [H, "text/html", 1], "/style.css": [H, "text/css", 1], "/posts/olá/index.html": [H, "text/html", 1] };
		expect(resolvePath(m, "/")).toEqual({ file: "/index.html" });
		expect(resolvePath(m, "/posts/oi/")).toEqual({ file: "/posts/oi/index.html" });
		expect(resolvePath(m, "/posts/oi")).toEqual({ redirect: "/posts/oi/" });
		expect(resolvePath(m, "/style.css")).toEqual({ file: "/style.css" });
		expect(resolvePath(m, "/posts/ol%C3%A1/")).toEqual({ file: "/posts/olá/index.html" });
		expect(resolvePath(m, "/nada")).toBeNull();
		expect(resolvePath(m, "/%E0%A4%A")).toBeNull();
	});
});

describe("deploys", () => {
	it("take each path once, with a stored hash and a media type", () => {
		const ok = deploySchema.safeParse({ files: [{ path: "/index.html", hash: H, type: "text/html; charset=utf-8", gz: true }] });
		expect(ok.success).toBe(true);
		expect(deploySchema.safeParse({ files: [{ path: "/a", hash: H, type: "text/html", gz: true }, { path: "/a", hash: H, type: "text/html", gz: true }] }).success).toBe(false);
		expect(deploySchema.safeParse({ files: [{ path: "/a", hash: "x", type: "text/html", gz: true }] }).success).toBe(false);
		expect(deploySchema.safeParse({ files: [{ path: "/a", hash: H, type: "<script>", gz: true }] }).success).toBe(false);
		expect(deploySchema.safeParse({ files: [] }).success).toBe(false);
	});
});

describe("browser cache", () => {
	it("checks pages every time and keeps fingerprinted files for good", () => {
		expect(cacheControl("/index.html", "text/html; charset=utf-8")).toBe("public, max-age=0, must-revalidate");
		expect(cacheControl("/index.xml", "application/rss+xml")).toBe("public, max-age=0, must-revalidate");
		expect(cacheControl("/sw.js", "text/javascript")).toBe("public, max-age=0, must-revalidate");
		expect(cacheControl(`/css/site.min.${"b".repeat(64)}.css`, "text/css")).toBe("public, max-age=31536000, immutable");
		expect(cacheControl("/img/foto_hu_3f2a9c1b77aa.webp", "image/webp")).toBe("public, max-age=31536000, immutable");
		expect(cacheControl("/favicon.svg", "image/svg+xml")).toBe("public, max-age=3600");
	});
});
