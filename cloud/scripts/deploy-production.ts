// Production deploy of the Blup Cloud from this checkout, without the
// account's values ever touching the repository.
//
// The wrangler*.jsonc files in git keep placeholders (zeroed D1 id, test
// Turnstile key, empty zone). The real values live outside the repo, in
// ~/.config/blup-deploy (or BLUP_DEPLOY_DIR):
// - vars.json: { "<config file>": { "<dotted path>": value } }, applied over
//   each config;
// - cloud.secrets.json: the Worker secrets; empty values are skipped.
// The merged configs are written next to the originals as .deploy-*.jsonc
// (ignored by git, so relative paths still resolve) and removed at the end,
// together with the temporary secrets files.
//
//   node scripts/deploy-production.ts check              dry-run of every Worker
//   node scripts/deploy-production.ts migrate            D1 migrations (remote)
//   node scripts/deploy-production.ts deploy             build and publish
//   node scripts/deploy-production.ts deploy --bootstrap first deploy on an empty account
import { chmodSync, existsSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";
import { spawnSync } from "node:child_process";
import ts from "typescript";

type Json = null | boolean | number | string | Json[] | { [key: string]: Json };
type Overrides = Record<string, Record<string, Json>>;
// vars.json may also carry build_env: variables for the panels' build
// (VITE_SENTRY_DSN, the Pulse project of the panels).
const BUILD_ENV = "build_env";

const CONFIGS = ["wrangler.jsonc", "wrangler.sites.jsonc", "wrangler.edge.jsonc", "wrangler.pulse.jsonc", "wrangler.deals.jsonc"] as const;
// The sites Worker only gets what it reads; auth, Stripe, AI and CF_API_TOKEN stay in the Cloud Worker.
const SITES_SECRETS = ["STAFF_OWNER_EMAIL", "SENTRY_DSN"];
const PLACEHOLDERS: Record<string, string> = {
	"00000000-0000-0000-0000-000000000000": "D1 database_id zerado",
	"1x00000000000000000000AA": "chave de teste do Turnstile",
};

const dir = process.env.BLUP_DEPLOY_DIR || join(homedir(), ".config", "blup-deploy");
const created: string[] = [];

function fail(message: string): never {
	throw new Error(message);
}

function readJson<T>(path: string): T {
	if (!existsSync(path)) fail(`Falta ${path}`);
	return JSON.parse(readFileSync(path, "utf8")) as T;
}

function parseJsonc(file: string): { [key: string]: Json } {
	const parsed = ts.parseConfigFileTextToJson(file, readFileSync(file, "utf8"));
	if (parsed.error) fail(`${file}: ${ts.flattenDiagnosticMessageText(parsed.error.messageText, "\n")}`);
	return parsed.config as { [key: string]: Json };
}

// Sets a dotted path ("env.production.vars.CF_ZONE_ID", array indexes as
// numbers). The path must already exist in the config, so a renamed key in
// the template fails here instead of deploying without the value.
function setPath(config: { [key: string]: Json }, path: string, value: Json, file: string) {
	const keys = path.split(".");
	let node: Json = config;
	for (const key of keys.slice(0, -1)) {
		node = Array.isArray(node) ? (node[Number(key)] ?? null) : node && typeof node === "object" ? (node[key] ?? null) : null;
		if (node === null || typeof node !== "object") fail(`${file}: o caminho ${path} não existe`);
	}
	const last = keys[keys.length - 1]!;
	if (Array.isArray(node) ? !(Number(last) in node) : !(last in (node as object))) fail(`${file}: o caminho ${path} não existe`);
	if (Array.isArray(node)) node[Number(last)] = value;
	else (node as { [key: string]: Json })[last] = value;
}

function checkProduction(config: { [key: string]: Json }, file: string) {
	const production = (config.env as { [key: string]: Json } | undefined)?.production;
	if (!production) return;
	const text = JSON.stringify(production);
	for (const [value, label] of Object.entries(PLACEHOLDERS)) if (text.includes(value)) fail(`${file}: a produção ainda tem ${label}. Preencha em ${join(dir, "vars.json")}`);
	const vars = (production as { [key: string]: Json }).vars as Record<string, Json> | undefined;
	for (const key of ["CF_ZONE_ID", "SITES_CNAME_TARGET", "DEALS_BRIDGE_URL"]) if (vars && key in vars && vars[key] === "") fail(`${file}: ${key} vazio em produção`);
}

function writePrivate(path: string, content: string) {
	writeFileSync(path, content, { mode: 0o600 });
	chmodSync(path, 0o600);
	created.push(path);
}

function prepareConfigs(): Record<string, string> {
	const overrides = readJson<Overrides>(join(dir, "vars.json"));
	for (const name of Object.keys(overrides)) if (name !== BUILD_ENV && !(CONFIGS as readonly string[]).includes(name)) fail(`vars.json: ${name} não é um dos arquivos de deploy`);
	const out: Record<string, string> = {};
	for (const name of CONFIGS) {
		const config = parseJsonc(name);
		for (const [path, value] of Object.entries(overrides[name] ?? {})) setPath(config, path, value, name);
		checkProduction(config, name);
		const target = `.deploy-${name}`;
		writePrivate(target, JSON.stringify(config, null, "\t"));
		out[name] = target;
	}
	return out;
}

function prepareSecrets(): { cloud: string; sites: string } {
	const all = readJson<Record<string, string>>(join(dir, "cloud.secrets.json"));
	const filled = Object.fromEntries(Object.entries(all).filter(([, v]) => typeof v === "string" && v.trim() !== ""));
	for (const key of ["BETTER_AUTH_SECRET", "TURNSTILE_SECRET_KEY", "STRIPE_SECRET_KEY", "STRIPE_WEBHOOK_SECRET", "STAFF_OWNER_EMAIL", "ACCESS_TEAM_DOMAIN", "ACCESS_AUD", "PULSE_CHANNEL_KEY"]) {
		if (!(key in filled)) fail(`cloud.secrets.json: falta ${key}`);
	}
	const cloud = join(dir, ".tmp-cloud.secrets.json");
	const sites = join(dir, ".tmp-sites.secrets.json");
	writePrivate(cloud, JSON.stringify(filled));
	writePrivate(sites, JSON.stringify(Object.fromEntries(SITES_SECRETS.filter((k) => k in filled).map((k) => [k, filled[k]!]))));
	console.log(`Secrets do Cloud: ${Object.keys(filled).length} preenchidos; vazios ficam de fora: ${Object.keys(all).filter((k) => !(k in filled)).join(", ") || "nenhum"}`);
	return { cloud, sites };
}

function run(cmd: string, args: string[], env: Record<string, string> = {}) {
	console.log(`\n> ${cmd} ${args.join(" ")}`);
	const r = spawnSync(cmd, args, { stdio: "inherit", env: { ...process.env, ...env } });
	if (r.status !== 0) fail(`Falhou: ${cmd} ${args.join(" ")}`);
}

const wrangler = (...args: string[]) => run("npx", ["wrangler", ...args]);

// The panels' build, with build_env from vars.json.
function build() {
	const extra = readJson<Overrides>(join(dir, "vars.json"))[BUILD_ENV] ?? {};
	const env = Object.fromEntries(Object.entries(extra).filter((e): e is [string, string] => typeof e[1] === "string"));
	console.log(`Build com: ${Object.keys(env).join(", ") || "nenhuma variável extra"}`);
	run("npm", ["run", "build"], env);
}

function main() {
	const [command, ...flags] = process.argv.slice(2);
	const configs = prepareConfigs();
	const cloud = configs["wrangler.jsonc"]!, sites = configs["wrangler.sites.jsonc"]!;
	const pulse = configs["wrangler.pulse.jsonc"]!, deals = configs["wrangler.deals.jsonc"]!, edge = configs["wrangler.edge.jsonc"]!;
	switch (command) {
		case "check":
			build();
			for (const c of [sites, cloud, edge, pulse, deals]) wrangler("deploy", "-c", c, "--env", "production", "--dry-run");
			break;
		case "migrate":
			wrangler("d1", "migrations", "apply", "DB", "--remote", "--env", "production", "-c", cloud);
			break;
		case "deploy": {
			const secrets = prepareSecrets();
			if (flags.includes("--bootstrap")) wrangler("deploy", "-c", "wrangler.bootstrap.jsonc");
			wrangler("d1", "migrations", "apply", "DB", "--remote", "--env", "production", "-c", cloud);
			build();
			wrangler("deploy", "-c", sites, "--env", "production", "--secrets-file", secrets.sites);
			wrangler("deploy", "-c", cloud, "--env", "production", "--secrets-file", secrets.cloud);
			// After the Cloud's Worker, which it calls: cloud.getblup.io and
			// staff.getblup.io move to it (the first time wrangler asks to take
			// them from blup-cloud-production: answer yes).
			wrangler("deploy", "-c", edge, "--env", "production");
			wrangler("deploy", "-c", pulse, "--env", "production");
			wrangler("deploy", "-c", deals, "--env", "production");
			break;
		}
		default:
			fail("Use: node scripts/deploy-production.ts check | migrate | deploy [--bootstrap]");
	}
}

try {
	main();
} catch (err) {
	console.error(`\n${err instanceof Error ? err.message : String(err)}`);
	process.exitCode = 1;
} finally {
	for (const path of created) rmSync(path, { force: true });
}
