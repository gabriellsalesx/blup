// Local attack fixtures: real SQLite behind the D1/Drizzle interface.
// No network, production bindings, or personal data.
// Worker types intentionally omit Node. These imports only run in Vitest.
// @ts-expect-error test-only Node builtin, no additional dependency
import { DatabaseSync as SQLite } from "node:sqlite";
// @ts-expect-error test-only Node builtin, no additional dependency
import * as fs from "node:fs";

type Row = Record<string, unknown>;
type SQLiteDB = {
	exec(sql:string):void;
	prepare(sql:string):{all(...v:unknown[]):Row[];get(...v:unknown[]):Row|undefined;run(...v:unknown[]):unknown};
	close():void;
};
const DatabaseSync = SQLite as new (path:string) => SQLiteDB;
const readFileSync = fs.readFileSync as (path:URL,encoding:string)=>string;
const readdirSync = fs.readdirSync as (path:URL)=>string[];
const moduleURL = (import.meta as unknown as {url:string}).url;

export class TestD1 {
	readonly sqlite = new DatabaseSync(":memory:");
	readonly statements: { sql: string; bindings: number }[] = [];
	constructor(through?:string) {
		this.sqlite.exec("PRAGMA foreign_keys=ON");
		for (const name of readdirSync(new URL("../migrations/", moduleURL)).filter(n => n.endsWith(".sql")&&(!through||n<=through)).sort()) {
			this.applyMigration(name);
		}
	}
	applyMigration(name:string){this.sqlite.exec(readFileSync(new URL(`../migrations/${name}`,moduleURL),"utf8"));}
	prepare(sql: string) { return new TestStatement(this, sql); }
	async batch(statements: TestStatement[]) {
		this.sqlite.exec("BEGIN");
		try { const result = statements.map(s => s.execute()); this.sqlite.exec("COMMIT"); return result; }
		catch(e) { this.sqlite.exec("ROLLBACK"); throw e; }
	}
	close() { this.sqlite.close(); }
	asD1() { return this as unknown as D1Database; }
}

class TestStatement {
	values: unknown[] = [];
	constructor(readonly db: TestD1, readonly sql: string) {}
	bind(...values: unknown[]) { const s = new TestStatement(this.db, this.sql); s.values = values; return s; }
	execute() {
		if (this.values.length > 100) throw new Error("D1: too many bound parameters");
		this.db.statements.push({ sql: this.sql, bindings: this.values.length });
		const stmt = this.db.sqlite.prepare(this.sql);
		const results = stmt.all(...this.values as never[]);
		const { n } = this.db.sqlite.prepare("SELECT changes() AS n").get() as { n: number };
		return { success: true, results, meta: { changes: n } };
	}
	async all() { return this.execute(); }
	async run() { return this.execute(); }
	async first(column?: string) { const r = this.execute().results[0]; return column ? r?.[column] : r ?? null; }
	async raw() { return this.execute().results.map(r => Object.values(r)); }
}

export class TestBucket {
	readonly objects = new Map<string, Uint8Array>();
	async put(key: string, body: string | ArrayBuffer | Uint8Array | ReadableStream) {
		if(body instanceof ReadableStream){this.objects.set(key,new Uint8Array(await new Response(body).arrayBuffer()));return;}
		this.objects.set(key, typeof body === "string" ? new TextEncoder().encode(body) : new Uint8Array(body));
	}
	async head(key: string) { const bytes=this.objects.get(key);return bytes?{size:bytes.byteLength}:null; }
	async get(key: string) {
		const bytes = this.objects.get(key);
		if (!bytes) return null;
		return { body: new Blob([bytes as Uint8Array<ArrayBuffer>]).stream(), text: async () => new TextDecoder().decode(bytes), json: async () => JSON.parse(new TextDecoder().decode(bytes)) };
	}
	async delete(keys: string | string[]) { for (const key of Array.isArray(keys) ? keys : [keys]) this.objects.delete(key); }
	async list({prefix}: {prefix:string}) { return {objects:[...this.objects.keys()].filter(k=>k.startsWith(prefix)).map(key=>({key,uploaded:new Date(0)})),truncated:false}; }
	asR2() { return this as unknown as R2Bucket; }
}

export function seedUser(d1: TestD1, id: string, plan = "free") {
	d1.sqlite.prepare("INSERT INTO user (id,name,email,email_verified) VALUES (?,?,?,1)").run(id, id, `${id}@audit.invalid`);
	if (plan !== "free") d1.sqlite.prepare("INSERT INTO subscription (id,plan,reference_id,status) VALUES (?,?,?,'active')").run(`sub-${id}`,plan,id);
}
