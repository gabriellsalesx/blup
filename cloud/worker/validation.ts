import { z } from "zod";
import { zValidator } from "@hono/zod-validator";
import { HTTPException } from "hono/http-exception";
import type { ValidationTargets } from "hono";

export const opaqueId = z.string().regex(/^[A-Za-z0-9_-]{1,128}$/);
export const hashId = z.string().regex(/^[0-9a-f]{64}$/);
export const idParam = z.object({id:opaqueId});
export const safeInt = z.string().regex(/^(0|[1-9][0-9]*)$/).transform(Number).pipe(z.number().int().min(0).max(Number.MAX_SAFE_INTEGER-1));
export const validate = <T extends z.ZodType, K extends keyof ValidationTargets>(target: K, schema: T) =>
	zValidator(target,schema,(r,c) => {
		if (!r.success) return c.req.path.startsWith("/api/ai/")?c.json({code:"INVALID",error:{code:"INVALID",message:"Pedido inválido."}},400):c.json({code:"INVALID",fields:r.error.issues.map(i=>i.path.join("."))},400);
	});

export class BodyError extends HTTPException {
	constructor(status: 400 | 413 = 413) { super(status,{res:Response.json({code:status===413?"TOO_LARGE":"INVALID"},{status})}); }
}

// Bound the stream before JSON/multipart parsers consume it. In
// particular, do not make a second 34 MiB copy of a sealed manifest.
export function boundedStream(body:ReadableStream<Uint8Array>,max:number,expected?:number){
	if(expected!==undefined&&expected>max)throw new BodyError();
	let size=0;
	return body.pipeThrough(new TransformStream<Uint8Array,Uint8Array>({
		transform(chunk,c){size+=chunk.byteLength;if(size>max)throw new BodyError();c.enqueue(chunk);},
		flush(){if(expected!==undefined&&size!==expected)throw new BodyError(400);},
	}),{signal:AbortSignal.timeout(120_000)});
}

// Read no more than the bound, regardless of a forged/missing length.
// Chunked uploads are allowed for small bodies; large files stream to R2.
export async function boundedBody(req: Request, max: number): Promise<Uint8Array<ArrayBuffer>> {
	const length = req.headers.get("content-length");
	if (length !== null) {
		const parsed = safeInt.safeParse(length);
		if (!parsed.success) throw new BodyError(400);
		if (parsed.data > max) throw new BodyError();
	}
	if (!req.body) return new Uint8Array(0);
	const reader = req.body.getReader(), chunks: Uint8Array[] = [];
	let size = 0;
	try {
		while (true) {
			const part = await reader.read(); if (part.done) break;
			size += part.value.byteLength;
			if (size > max) { await reader.cancel(); throw new BodyError(); }
			chunks.push(part.value);
		}
	} finally { reader.releaseLock(); }
	const out = new Uint8Array(size);
	let at=0; for(const chunk of chunks){out.set(chunk,at);at+=chunk.byteLength;}
	return out;
}
