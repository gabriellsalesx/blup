import {z} from "zod";
// How a public status page looks at the top: whether its title shows, the
// links of its header and its logos. The logos arrive already made small
// in the person's browser, as WebP (PNG where the browser cannot make
// WebP, as Safari); the server only checks that a file really is one of
// them, of a sane size, before keeping it.

// A header link: a short name and an https address.
export const navLink=z.object({
 label:z.string().trim().min(1).max(40),
 url:z.url({protocol:/^https$/,hostname:z.regexes.domain}).max(2048).refine(v=>{const u=new URL(v);return !u.username&&!u.password;}),
}).strict();
export const navLinks=z.array(navLink).max(5);
export type NavLink=z.infer<typeof navLink>;
// The stored list, or none when it is not one.
export function linksOf(stored:string):NavLink[]{try{const parsed=navLinks.safeParse(JSON.parse(stored));return parsed.success?parsed.data:[];}catch{return [];}}

// One logo for a light background and one for a dark one.
export const logoVariant=z.enum(["light","dark"]);
export type LogoVariant=z.infer<typeof logoVariant>;
export const LOGO_MAX_BYTES=256*1024,LOGO_MAX_SIDE=2048;
export const logoKey=(page:string,variant:LogoVariant)=>`pages/${page}/logo-${variant}`;
export const logoType=z.enum(["image/webp","image/png"]);
export type LogoType=z.infer<typeof logoType>;
// A logo's version (part of its public address, so a new one is fetched).
export const logoVersion=z.string().regex(/^[a-f0-9]{16}$/);

// The width and height of a WebP (lossy, lossless or extended), or null
// when the bytes are not one.
export function webpSize(b:Uint8Array):{width:number;height:number}|null{
 const text=(at:number,n:number)=>String.fromCharCode(...b.subarray(at,at+n));
 if(b.length<30||text(0,4)!=="RIFF"||text(8,4)!=="WEBP")return null;
 const riff=(b[4]|b[5]<<8|b[6]<<16|b[7]<<24)>>>0;if(riff+8!==b.length)return null;
 const chunk=text(12,4);
 if(chunk==="VP8 "){if(b[23]!==0x9d||b[24]!==0x01||b[25]!==0x2a)return null;return {width:(b[26]|b[27]<<8)&0x3fff,height:(b[28]|b[29]<<8)&0x3fff};}
 if(chunk==="VP8L"){if(b[20]!==0x2f)return null;const bits=(b[21]|b[22]<<8|b[23]<<16|b[24]<<24)>>>0;return {width:(bits&0x3fff)+1,height:((bits>>>14)&0x3fff)+1};}
 if(chunk==="VP8X")return {width:1+(b[24]|b[25]<<8|b[26]<<16),height:1+(b[27]|b[28]<<8|b[29]<<16)};
 return null;
}
// The width and height of a PNG, or null when the bytes are not one.
export function pngSize(b:Uint8Array):{width:number;height:number}|null{
 const signature=[137,80,78,71,13,10,26,10];if(b.length<33||signature.some((v,i)=>b[i]!==v)||String.fromCharCode(...b.subarray(12,16))!=="IHDR")return null;
 const view=new DataView(b.buffer,b.byteOffset,b.byteLength);return {width:view.getUint32(16),height:view.getUint32(20)};
}
// A logo that can be kept: really of the type it says, within the limits.
export function acceptableLogo(bytes:Uint8Array,type:LogoType){const size=type==="image/webp"?webpSize(bytes):pngSize(bytes);return !!size&&size.width>0&&size.height>0&&size.width<=LOGO_MAX_SIDE&&size.height<=LOGO_MAX_SIDE;}
