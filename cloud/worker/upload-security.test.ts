import {describe,it,expect,vi,afterEach} from "vitest";
// @ts-expect-error test-only installed Node builtin
import {createHash} from "node:crypto";
import {stageFile} from "./upload";
import {boundedBody} from "./validation";
import {PLANS} from "./plans";

// Node lacks workerd's two stream primitives. This test implementation
// hashes incrementally too; the bucket consumes without retaining bytes.
class TestDigest extends WritableStream<Uint8Array>{
	readonly digest:Promise<ArrayBuffer>;
	constructor(){
		let resolve!:(v:ArrayBuffer)=>void,reject!:(e:unknown)=>void;
		const result=new Promise<ArrayBuffer>((yes,no)=>{resolve=yes;reject=no;});
		const hash=createHash("sha256");
		super({write(chunk){hash.update(chunk);},close(){resolve(new Uint8Array(hash.digest()).buffer);},abort(e){reject(e);}});
		this.digest=result;
	}
}
class TestFixedLength extends TransformStream<Uint8Array,Uint8Array>{
	constructor(size:number){let n=0;super({transform(chunk,c){n+=chunk.byteLength;if(n>size)throw new Error("length");c.enqueue(chunk);},flush(){if(n!==size)throw new Error("length");}});}
}
afterEach(()=>vi.unstubAllGlobals());
describe("bounded file intake",()=>{
	it("streams a 100 MiB file with bounded backpressure",async()=>{
		vi.stubGlobal("crypto",{subtle:crypto.subtle,DigestStream:TestDigest});vi.stubGlobal("FixedLengthStream",TestFixedLength);
		const chunk=new Uint8Array(65536),total=100*1024*1024;let produced=0,consumed=0,ahead=0;
		const stream=new ReadableStream<Uint8Array>({pull(c){if(produced===total){c.close();return;}produced+=chunk.byteLength;ahead=Math.max(ahead,produced-consumed);c.enqueue(chunk);}});
		const bucket={put:async(_key:string,body:ReadableStream<Uint8Array>)=>{const reader=body.getReader();while(true){const {done,value}=await reader.read();if(done)break;consumed+=value.byteLength;await Promise.resolve();}}} as unknown as R2Bucket;
		const request=new Request("https://upload.audit.invalid/",{method:"PUT",headers:{"content-length":String(total)},body:stream,duplex:"half"} as RequestInit);
		const result=await stageFile(request,bucket,"fixture",total);
		expect(result.size).toBe(total);expect(consumed).toBe(total);expect(result.hash).toMatch(/^[a-f0-9]{64}$/);expect(ahead).toBeLessThan(10*chunk.byteLength);
	},15_000);
	it("rejects dishonest lengths while reading, even under the declared cap",async()=>{
		const req=new Request("https://upload.audit.invalid",{method:"POST",headers:{"content-length":"1"},body:new Uint8Array(10)});
		await expect(boundedBody(req,5)).rejects.toMatchObject({status:413});
	});
	it("uses the approved 10/25/50/100 MiB file caps",()=>{
		expect(PLANS.filter(p=>!p.pulse).map(p=>p.limits.maxFileMB)).toEqual([10,25,50,100]);
	});
});
