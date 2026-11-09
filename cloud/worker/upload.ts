import { BodyError, boundedBody, safeInt } from "./validation";

const hex = (bytes:ArrayBuffer) => [...new Uint8Array(bytes)].map(b=>b.toString(16).padStart(2,"0")).join("");

// Small chunked bodies are bounded in memory. Larger files require their
// length and use workerd's incremental digest with R2 backpressure. No tee:
// a fast digest must not queue the whole file while R2 writes slowly.
// With `expected` (a SHA-256 in hex), R2 itself refuses other bytes, so the
// object can go straight to its final, content-addressed name: a body that
// is not that file is never stored, and the answer carries its real hash.
export async function receiveFile(req:Request,bucket:R2Bucket,key:string,expected:string|undefined,max:number) {
	const raw=req.headers.get("content-length");
	if(raw===null) {
		const bytes=await boundedBody(req,Math.min(max,4*1024*1024));
		if(!bytes.byteLength) throw new BodyError(400);
		const hash=hex(await crypto.subtle.digest("SHA-256",bytes));
		if(expected!==undefined&&hash!==expected)return {size:bytes.byteLength,hash};
		await bucket.put(key,bytes,expected===undefined?undefined:{sha256:expected});
		return {size:bytes.byteLength,hash};
	}
	const parsed=safeInt.safeParse(raw);
	if(!parsed.success || parsed.data===0 || !req.body) throw new BodyError(400);
	const size=parsed.data;
	if(size>max) throw new BodyError();
	const digest=new crypto.DigestStream("SHA-256"),writer=digest.getWriter();
	// Digest rejection is observed even when an upload fails before flush.
	const result=digest.digest;
	void result.catch(()=>{});
	let received=0;
	const stream=new TransformStream<Uint8Array,Uint8Array>({
		async transform(chunk,controller){
			received+=chunk.byteLength;
			if(received>size || received>max) throw new BodyError();
			await writer.write(chunk);controller.enqueue(chunk);
		},
		async flush(){if(received!==size) throw new BodyError(400);await writer.close();},
	});
	try {
		await bucket.put(key,req.body.pipeThrough(stream,{signal:AbortSignal.timeout(120_000)}).pipeThrough(new FixedLengthStream(size)),expected===undefined?undefined:{sha256:expected});
		return {size,hash:hex(await result)};
	}catch(e){
		// The whole body arrived (the digest is done) and R2 refused it: its
		// checksum is not the expected one. Anything else is a real failure.
		await writer.abort().catch(()=>{});
		const got=await result.then(hex,()=>null);
		if(expected!==undefined&&got!==null&&got!==expected)return {size,hash:got};
		throw e;
	}
}
export const stageFile=(req:Request,bucket:R2Bucket,key:string,max:number)=>receiveFile(req,bucket,key,undefined,max);
