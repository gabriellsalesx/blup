import {z} from "zod";

// What a product page says about itself: Open Graph tags first, then the
// schema.org Product in JSON-LD. Only text and numbers come out, never
// markup; the image must be an HTTPS address.
export type PageInfo={title:string|null;image:string|null;price:number|null;currency:string|null};
const entities:Record<string,string>={amp:"&",lt:"<",gt:">",quot:"\"",apos:"'",nbsp:" "};
export function decodeEntities(text:string){
 return text.replace(/&(#x[0-9a-f]{1,6}|#[0-9]{1,7}|[a-z]{2,6});/gi,(all,code:string)=>{
  if(code[0]==="#"){const n=code[1]==="x"||code[1]==="X"?parseInt(code.slice(2),16):parseInt(code.slice(1),10);return n>0&&n<=0x10ffff&&!(n>=0xd800&&n<=0xdfff)?String.fromCodePoint(n):"";}
  return entities[code.toLowerCase()]??all;
 });
}
// One line of plain text, without control characters, cut at max.
export function cleanLine(text:string,max:number){
 const line=decodeEntities(text).replace(/<[^>]*>/g," ").replace(/[\u0000-\u001f\u007f-\u009f\u200b-\u200f\u2028-\u202e\u2066-\u2069]/g," ").replace(/\s+/g," ").trim();
 return [...line].slice(0,max).join("");
}
function meta(html:string,names:string[]):string|null{
 for(const tag of html.match(/<meta\b[^>]{0,2048}>/gi)??[]){
  const key=/\b(?:property|name|itemprop)\s*=\s*["']([^"']{1,80})["']/i.exec(tag)?.[1]?.toLowerCase(),content=/\bcontent\s*=\s*(?:"([^"]{0,2048})"|'([^']{0,2048})')/i.exec(tag);
  if(key&&names.includes(key)&&content)return content[1]??content[2]??null;
 }
 return null;
}
// "1.299,90", "1,299.90", "1299.9" or "R$ 59" in minor units.
export function priceCents(raw:unknown):number|null{
 if(typeof raw==="number")return Number.isFinite(raw)&&raw>=0&&raw<1e9?Math.round(raw*100):null;
 if(typeof raw!=="string")return null;
 let v=raw.replace(/[^0-9.,]/g,"");if(!v)return null;
 const comma=v.lastIndexOf(","),dot=v.lastIndexOf(".");
 if(comma>dot)v=v.slice(0,comma).replace(/[.,]/g,"")+"."+v.slice(comma+1);
 else if(dot>comma&&v.length-dot-1!==3)v=v.slice(0,dot).replace(/[.,]/g,"")+"."+v.slice(dot+1);
 else v=v.replace(/[.,]/g,"");
 const n=Number(v);return Number.isFinite(n)&&n>=0&&n<1e9?Math.round(n*100):null;
}
const currency=z.string().trim().toUpperCase().regex(/^[A-Z]{3}$/);
const offer=z.object({price:z.union([z.string(),z.number()]).optional(),lowPrice:z.union([z.string(),z.number()]).optional(),priceCurrency:z.string().optional()}).loose();
const product=z.object({"@type":z.union([z.string(),z.array(z.string())]).optional(),name:z.string().optional(),image:z.union([z.string(),z.array(z.string()),z.object({url:z.string()}).loose()]).optional(),offers:z.union([offer,z.array(offer)]).optional()}).loose();
function jsonLd(html:string){
 for(const block of html.match(/<script\b[^>]*application\/ld\+json[^>]*>[\s\S]{0,65536}?<\/script>/gi)??[]){
  let value:unknown;try{value=JSON.parse(block.replace(/^<script\b[^>]*>|<\/script>$/gi,""));}catch{continue;}
  const items=(Array.isArray(value)?value:[value]).flatMap(v=>typeof v==="object"&&v!==null&&"@graph" in v&&Array.isArray(v["@graph"])?v["@graph"]:[v]);
  for(const item of items){const p=product.safeParse(item);if(p.success&&[p.data["@type"]].flat().includes("Product"))return p.data;}
 }
 return null;
}
const httpsImage=(raw:string|null|undefined,base:URL)=>{if(!raw)return null;try{const u=new URL(decodeEntities(raw.trim()),base);return u.protocol==="https:"&&u.href.length<=1024?u.href:null;}catch{return null;}};
export function pageInfo(html:string,base:URL):PageInfo{
 const ld=jsonLd(html),offers=ld?.offers?[ld.offers].flat()[0]:undefined;
 const title=meta(html,["og:title","twitter:title"])??ld?.name??/<title[^>]*>([^<]{1,512})<\/title>/i.exec(html)?.[1]??null;
 const ldImage=ld?.image?typeof ld.image==="string"?ld.image:Array.isArray(ld.image)?ld.image[0]:ld.image.url:null;
 const price=priceCents(meta(html,["product:price:amount","og:price:amount","price"])??offers?.price??offers?.lowPrice);
 const cur=currency.safeParse(meta(html,["product:price:currency","og:price:currency","pricecurrency"])??offers?.priceCurrency);
 const cleanTitle=title?cleanLine(title,200):"";
 return {title:cleanTitle||null,image:httpsImage(meta(html,["og:image","og:image:secure_url","twitter:image"])??ldImage,base),price,currency:price!==null&&cur.success?cur.data:null};
}
