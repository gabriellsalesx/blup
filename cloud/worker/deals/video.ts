import {z} from "zod";
import {apiJSON,safeFetch,publicHTTPS,FetchError} from "./fetch";

// A TikTok video the person posted with a product in it. The channels get
// the news and the click goes to the video, where TikTok Shop sells and
// pays the creator. No TikTok API: the address is read by its shape (a
// short one followed safely to the video) and TikTok's public oEmbed says
// whether the video exists and is public.
export class VideoError extends Error{constructor(readonly code:"NOT_A_VIDEO"|"VIDEO_NOT_FOUND"){super(code);}}
export type Video={id:string;author:string;url:string};
const tiktokHost=/^(?:www\.|m\.)?tiktok\.com$/;
const shortHosts=new Set(["vm.tiktok.com","vt.tiktok.com"]);
const videoPath=/^\/@([A-Za-z0-9._]{1,64})\/video\/([0-9]{8,25})\/?$/;
// The video's own address, without what the app adds to a shared link.
export function videoOf(url:URL):Video|null{
 if(!tiktokHost.test(url.hostname.toLowerCase()))return null;
 const match=videoPath.exec(url.pathname);if(!match)return null;
 return {id:match[2],author:match[1],url:`https://www.tiktok.com/@${match[1]}/video/${match[2]}`};
}
const oembed=z.object({embed_product_id:z.string().regex(/^[0-9]{8,25}$/)}).loose();
export async function tiktokVideo(raw:string,send:typeof fetch=fetch):Promise<Video>{
 let url=publicHTTPS(raw);
 if(shortHosts.has(url.hostname.toLowerCase()))url=(await safeFetch(url.href,{body:false,until:next=>!shortHosts.has(next.hostname.toLowerCase()),send})).url;
 const video=videoOf(url);if(!video)throw new VideoError("NOT_A_VIDEO");
 let answer:unknown;
 try{answer=await apiJSON(`https://www.tiktok.com/oembed?url=${encodeURIComponent(video.url)}`,{headers:{accept:"application/json","user-agent":"BlupDeals/1.0 (+https://getblup.io)"}},send,64*1024);}
 catch(e){if(e instanceof FetchError)throw new VideoError("VIDEO_NOT_FOUND");throw e;}
 const parsed=oembed.safeParse(answer);
 if(!parsed.success||parsed.data.embed_product_id!==video.id)throw new VideoError("VIDEO_NOT_FOUND");
 return video;
}
