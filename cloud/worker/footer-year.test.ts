import {it,expect} from "vitest";
import {renderStatusPage} from "./pulse/public-html";
import {SitePage} from "./deals/site-page";

// The public pages' footer year is the year the page is made: it turns on
// its own, with nothing to change at New Year.
const at=(year:number)=>Date.UTC(year,0,1,12);
it("turns the Pulse page's year",()=>{
 const page={title:"Status da Ana",language:"pt" as const,historyDays:30,components:[],incidents:[],maintenance:[],showTitle:true,links:[],logos:{light:null,dark:null}};
 expect(renderStatusPage(page,at(2026))).toContain("© 2026 Status da Ana");
 expect(renderStatusPage(page,at(2031))).toContain("© 2031 Status da Ana");
});
it("turns the Deals page's year",()=>{
 const props=(now:number)=>({now,site:{slug:"ana",title:"Ofertas da Ana",description:"",language:"pt" as const,logos:{light:null,dark:null},showTitle:true,links:[],socials:[]},lang:"pt" as const,base:"/ana",origin:"https://deals.getblup.site",links:"https://deals.getblup.site",deals:[],query:null,category:null,categories:[],next:null});
 expect(String(SitePage(props(at(2026))))).toContain("© 2026 Ofertas da Ana");
 expect(String(SitePage(props(at(2031))))).toContain("© 2031 Ofertas da Ana");
});
