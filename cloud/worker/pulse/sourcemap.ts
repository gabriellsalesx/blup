import {z} from "zod";
import {cleanText} from "./event";
// Source maps (revision 3, the one Vite, esbuild, Rollup and webpack write):
// what is kept of one, and the original place of a minified line and
// column. Our own decoder, no dependency: the mappings are decoded line by
// line only as far as the asked line.

// What a stored map must be. Index maps ("sections") are not taken: the
// bundlers above write plain ones.
export const sourceMapSchema=z.object({
 version:z.literal(3),
 file:z.string().max(1024).optional(),
 sourceRoot:z.string().max(1024).optional(),
 sources:z.array(z.string().max(2048).nullable()).max(20_000),
 sourcesContent:z.array(z.string().nullable()).max(20_000).optional(),
 names:z.array(z.string().max(1024)).max(200_000).default([]),
 mappings:z.string(),
});
export type SourceMap=z.infer<typeof sourceMapSchema>;
export type OriginalFrame={file:string;line:number;column:number;function:string;inApp:boolean;context?:{pre:string[];line:string;post:string[]}};

const BASE64="ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
const digit=new Map([...BASE64].map((c,i)=>[c,i]));
type Segment=[column:number,source?:number,line?:number,sourceColumn?:number,name?:number];

// Decodes one Base64 VLQ group (a segment) starting at `at`; returns the
// values and where the next group starts.
function readSegment(text:string,at:number):{values:number[];next:number}{
 const values:number[]=[];let value=0,shift=0,i=at;
 for(;i<text.length;i++){
  const c=text.charAt(i);if(c===","||c===";")break;
  const d=digit.get(c);if(d===undefined)throw new Error("BAD_MAPPING");
  value+=(d&31)<<shift;
  if(d&32){shift+=5;if(shift>30)throw new Error("BAD_MAPPING");continue;}
  values.push(value&1?-(value>>>1):value>>>1);value=0;shift=0;
 }
 return {values,next:i};
}

// A map ready for lookups: its lines decoded as they are needed.
export class SourceMapLookup{
 private lines:Segment[][]=[];private at=0;
 private state={source:0,line:0,column:0,name:0};
 constructor(readonly map:SourceMap){}
 private decodeTo(line:number){
  const text=this.map.mappings;
  while(this.lines.length<=line&&this.at<=text.length){
   const segments:Segment[]=[];let column=0;
   while(this.at<text.length&&text[this.at]!==";"){
    if(text[this.at]===","){this.at++;continue;}
    const {values,next}=readSegment(text,this.at);this.at=next;
    const [first,source,sourceLine,sourceColumn,name]=values;
    if(first===undefined)continue;
    column+=first;
    if(source!==undefined&&sourceLine!==undefined&&sourceColumn!==undefined){this.state.source+=source;this.state.line+=sourceLine;this.state.column+=sourceColumn;if(name!==undefined)this.state.name+=name;segments.push(name!==undefined?[column,this.state.source,this.state.line,this.state.column,this.state.name]:[column,this.state.source,this.state.line,this.state.column]);}
    else segments.push([column]);
   }
   segments.sort((a,b)=>a[0]-b[0]);
   this.lines.push(segments);this.at++;
  }
 }
 // The original place of a generated line and column (both from 1, as the
 // SDKs send them), or null when the map does not cover it.
 originalOf(line:number,column:number):OriginalFrame|null{
  if(line<1||line>1_000_000)return null;
  this.decodeTo(line-1);
  const segments=this.lines[line-1];if(!segments?.length)return null;
  let lo=0,hi=segments.length-1,found=-1;const col=Math.max(0,column-1);
  while(lo<=hi){const mid=(lo+hi)>>1,at=segments[mid];if(at&&at[0]<=col){found=mid;lo=mid+1;}else hi=mid-1;}
  const seg=found>=0?segments[found]:null;if(!seg||seg[1]===undefined||seg[2]===undefined||seg[3]===undefined)return null;
  const raw=this.map.sources[seg[1]];if(raw===undefined||raw===null)return null;
  const file=sourcePath(raw,this.map.sourceRoot),content=this.map.sourcesContent?.[seg[1]];
  // The map names what is at the position (often the called function);
  // the frame's own function is the one around the line in the original
  // code, when it came with the map.
  const name=(content?enclosingFunction(content,seg[2]):"")||(seg[4]!==undefined?this.map.names[seg[4]]??"":"");
  return {file,line:seg[2]+1,column:seg[3]+1,function:cleanText(name,128),inApp:!/(?:^|\/)node_modules\//.test(file),...(content?{context:around(content,seg[2])}:{})};
 }
}

// A readable source path: no bundler scheme, no leading "./" or "../",
// no query; cleaned like any text.
export function sourcePath(raw:string,root?:string){
 const joined=root&&!/^[a-z]+:/i.test(raw)?`${root.replace(/\/+$/,"")}/${raw}`:raw;
 const path=joined.replace(/^[a-z][\w+.-]*:\/\/\/?/i,"").replace(/^(?:\.\.?\/)+/,"").split(/[?#]/)[0]??"";
 return cleanText(path,240);
}
// Five lines of the original code each side of a line, cleaned and cut.
function around(content:string,index:number){
 const lines=content.split(/\r?\n/),line=(v:string)=>cleanText(v,200);
 if(index<0||index>=lines.length)return undefined;
 return {pre:lines.slice(Math.max(0,index-5),index).map(line),line:line(lines[index]??""),post:lines.slice(index+1,index+6).map(line)};
}
// The name of the function a line is in, read upwards in the original code
// (at most 80 lines): a function declaration, a function or arrow given to
// a name, or a method. Empty when none is found.
const keywords=new Set(["if","for","while","switch","catch","return","function","else","do","with"]);
export function enclosingFunction(content:string,index:number){
 const lines=content.split(/\r?\n/);
 for(let i=Math.min(index,lines.length-1);i>=0&&i>=index-80;i--){
  const line=lines[i]??"";
  const declared=/\bfunction\s*\*?\s*([A-Za-z_$][\w$]*)\s*[<(]/.exec(line)?.[1];if(declared)return declared;
  const assigned=/\b([A-Za-z_$][\w$]*)\s*[:=]\s*(?:async\s+)?(?:function\b|\([^)]*\)\s*(?::[^=]+)?=>|[A-Za-z_$][\w$]*\s*=>)/.exec(line)?.[1];if(assigned&&!keywords.has(assigned))return assigned;
  const method=/^\s*(?:(?:async|static|public|private|protected|get|set)\s+)*([A-Za-z_$][\w$]*)\s*\([^)]*\)\s*(?::\s*[^{]+)?\{/.exec(line)?.[1];if(method&&!keywords.has(method))return method;
 }
 return "";
}

