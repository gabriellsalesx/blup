// A logo the person picks for a public page (Pulse's status pages, Deals'
// page), PNG, JPEG or WebP up to 3 MB, is made small in their own browser
// before it is sent: at most 960 by 192 pixels (three
// times the header's height), as WebP, or PNG where the browser cannot
// make WebP (Safari). What reaches the server is a few kilobytes.
export const LOGO_INPUT_TYPES=["image/png","image/jpeg","image/webp"] as const;
export const LOGO_INPUT_MAX=3*1024*1024,LOGO_SEND_MAX=256*1024;
const MAX_WIDTH=960,MAX_HEIGHT=192;
export type LogoVariant="light"|"dark";
export type LogoProblem="type"|"size"|"unreadable"|"heavy";
export class LogoError extends Error{constructor(readonly problem:LogoProblem){super(problem);}}
export type MadeLogo={blob:Blob;type:"image/webp"|"image/png"};

const isInputType=(type:string):type is (typeof LOGO_INPUT_TYPES)[number]=>(LOGO_INPUT_TYPES as readonly string[]).includes(type);
const encode=(canvas:HTMLCanvasElement,type:string,quality?:number)=>new Promise<Blob|null>(done=>canvas.toBlob(done,type,quality));

export async function optimizeLogo(file:File):Promise<MadeLogo>{
 if(!isInputType(file.type))throw new LogoError("type");
 if(file.size>LOGO_INPUT_MAX)throw new LogoError("size");
 let bitmap:ImageBitmap;try{bitmap=await createImageBitmap(file);}catch{throw new LogoError("unreadable");}
 try{
  const scale=Math.min(1,MAX_WIDTH/bitmap.width,MAX_HEIGHT/bitmap.height);
  const canvas=document.createElement("canvas");canvas.width=Math.max(1,Math.round(bitmap.width*scale));canvas.height=Math.max(1,Math.round(bitmap.height*scale));
  const context=canvas.getContext("2d");if(!context)throw new LogoError("unreadable");
  context.imageSmoothingQuality="high";context.drawImage(bitmap,0,0,canvas.width,canvas.height);
  for(const quality of [0.9,0.8,0.7]){
   const blob=await encode(canvas,"image/webp",quality);
   // A browser that cannot make WebP hands back a PNG instead.
   if(blob?.type!=="image/webp")break;
   if(blob.size<=LOGO_SEND_MAX)return {blob,type:"image/webp"};
  }
  const png=await encode(canvas,"image/png");
  if(png?.type==="image/png"&&png.size<=LOGO_SEND_MAX)return {blob:png,type:"image/png"};
  throw new LogoError("heavy");
 }finally{bitmap.close();}
}

