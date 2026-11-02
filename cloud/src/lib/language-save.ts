import type {Language} from "../../shared/language";
// Serialize account writes so a slow earlier request cannot replace the
// last manual choice. A stale page never writes into a newly signed-in user.
export function createLanguageSaver(write:(language:Language)=>Promise<unknown>,isCurrent:(id:string)=>boolean,onFailure:()=>void){
 let previousAccount="",previousKey="",pending=Promise.resolve();
 return {schedule(id:string,desired:Language,stored:Language):Promise<void>{
  const key=`${id}:${desired}`;
  if(key===previousKey)return pending;
  const initial=previousAccount!==id;
  previousAccount=id;previousKey=key;
  if(initial&&stored===desired)return pending;
  pending=pending.then(async()=>{if(isCurrent(id))await write(desired);}).catch(()=>{if(previousKey===key){previousKey="";onFailure();}});
  return pending;
 }};
}
